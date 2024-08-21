package plugin

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/cloudeteer/grafana-pdf-report-app/pkg/plugin/dashboard"
	"github.com/cloudeteer/grafana-pdf-report-app/pkg/plugin/report"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

// Get dashboard variables via query parameters
func getDashboardVariables(r *http.Request) url.Values {
	variables := url.Values{}
	for k, v := range r.URL.Query() {
		if strings.HasPrefix(k, "var-") || k == "from" || k == "to" {
			for _, singleV := range v {
				variables.Add(k, singleV)
			}
		}
	}

	return variables
}

// handleReport handles creating a PDF report from a given dashboard UID
// GET /api/plugins/cloudeteer-pdfreport-app/resources/report
func (app *App) handleReport(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

		return
	}

	var err error

	// Always start with an instance of current app's config
	conf := app.conf

	// Get context logger which we will use everywhere
	ctxLogger := log.DefaultLogger.FromContext(req.Context())

	// Get config from context
	pluginConfig := backend.PluginConfigFromContext(req.Context())
	currentUser := pluginConfig.User.Login

	ctxLogger = ctxLogger.With("user", currentUser)

	if !(pluginConfig.User.Role == "Admin" || pluginConfig.User.Role == conf.RequiredPermission || conf.RequiredPermission == "Viewer") {
		ctxLogger.Debug("user does not have required permission", "user_role", pluginConfig.User.Role)
		http.Error(w, "Query parameter dashUid not found", http.StatusForbidden)

		return
	}

	// Get Dashboard ID
	dashboardUID := req.URL.Query().Get("dashUid")
	if dashboardUID == "" {
		ctxLogger.Debug("Query parameter dashUid not found")
		http.Error(w, "Query parameter dashUid not found", http.StatusBadRequest)

		return
	}

	grafanaConfig := backend.GrafanaConfigFromContext(req.Context())

	if req.URL.Query().Has("theme") {
		conf.Theme = req.URL.Query().Get("theme")
		if conf.Theme != "light" && conf.Theme != "dark" {
			ctxLogger.Debug("invalid theme parameter: " + conf.Theme)
			http.Error(w, "invalid theme parameter: "+conf.Theme, http.StatusBadRequest)

			return
		}
	}

	if req.URL.Query().Has("layout") {
		conf.Layout = req.URL.Query().Get("layout")
		if conf.Layout != "simple" && conf.Layout != "grid" {
			ctxLogger.Debug("invalid layout parameter: " + conf.Layout)
			http.Error(w, "invalid layout parameter: "+conf.Layout, http.StatusBadRequest)

			return
		}
	}

	if req.URL.Query().Has("orientation") {
		conf.Orientation = req.URL.Query().Get("orientation")
		if conf.Orientation != "portrait" && conf.Orientation != "landscape" {
			ctxLogger.Debug("invalid orientation parameter: " + conf.Orientation)
			http.Error(w, "invalid orientation parameter: "+conf.Orientation, http.StatusBadRequest)

			return
		}
	}

	if req.URL.Query().Has("dashboardMode") {
		conf.DashboardMode = req.URL.Query().Get("dashboardMode")
		if conf.DashboardMode != "default" && conf.DashboardMode != "full" {
			ctxLogger.Warn("invalid dashboardMode parameter: " + conf.DashboardMode)
			http.Error(w, "invalid dashboardMode parameter: "+conf.DashboardMode, http.StatusBadRequest)

			return
		}
	}

	if req.URL.Query().Has("timeZone") {
		conf.TimeZone = req.URL.Query().Get("timeZone")
	}

	if req.URL.Query().Has("includePanelID") {
		conf.IncludePanelIDs = make([]int, len(req.URL.Query()["includePanelID"]))

		for i, stringID := range req.URL.Query()["includePanelID"] {
			conf.IncludePanelIDs[i], err = strconv.Atoi(stringID)
			if err != nil {
				ctxLogger.Debug("invalid includePanelID parameter: " + err.Error())
				http.Error(w, "invalid includePanelID parameter: "+err.Error(), http.StatusBadRequest)

				return
			}
		}
	}

	if req.URL.Query().Has("excludePanelID") {
		conf.ExcludePanelIDs = make([]int, len(req.URL.Query()["excludePanelID"]))

		for i, stringID := range req.URL.Query()["excludePanelID"] {
			conf.ExcludePanelIDs[i], err = strconv.Atoi(stringID)
			if err != nil {
				ctxLogger.Debug("invalid includePanelID parameter: " + err.Error())
				http.Error(w, "invalid excludePanelID parameter: "+err.Error(), http.StatusBadRequest)

				return
			}
		}
	}

	ctxLogger.Info(fmt.Sprintf("generate report using config: %s", conf.String()))

	var grafanaAppURL string
	if conf.AppURL != "" {
		grafanaAppURL = conf.AppURL
	} else {
		grafanaAppURL, err = grafanaConfig.AppURL()
		if err != nil {
			ctxLogger.Error("failed to get app URL", "err", err)
			http.Error(w, "failed to get app URL", http.StatusInternalServerError)

			return
		}
	}

	grafanaAppURL = strings.TrimSuffix(grafanaAppURL, "/")
	saToken, err := grafanaConfig.PluginAppClientSecret()
	if err != nil {
		ctxLogger.Error("failed to get plugin app client secret", "err", err)
		http.Error(w, "failed to get plugin app client secret", http.StatusInternalServerError)

		return
	}

	grafanaDashboard := dashboard.New(
		ctxLogger,
		conf,
		app.httpClient,
		app.chromeInstance,
		app.workerPools,
		grafanaAppURL,
		dashboardUID,
		req.URL.RawQuery,
		saToken,
	)

	// Make app new Grafana client to get dashboard JSON model and Panel PNGs
	pdfReport := report.New(
		ctxLogger,
		conf,
		app.httpClient,
		app.chromeInstance,
		app.workerPools,
		grafanaDashboard,
	)

	ctxLogger.Info(fmt.Sprintf("generate report using %s chrome", app.chromeInstance.Name()))

	// Generate report
	if err = pdfReport.Generate(req.Context(), w); err != nil {
		ctxLogger.Error("error generating report", "err", err)
		http.Error(w, "error generating report", http.StatusInternalServerError)

		return
	}

	ctxLogger.Info("report generated", "dash_uid", dashboardUID)
}

// handleHealth is an example HTTP GET resource that returns an OK response.
func (app *App) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Add("Content-Type", "text/plan")
	if _, err := w.Write([]byte("OK")); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// registerRoutes takes a *http.ServeMux and registers some HTTP handlers.
func (app *App) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/report", app.handleReport)
	mux.HandleFunc("/healthz", app.handleHealth)
}
