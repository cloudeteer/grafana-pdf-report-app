package report

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"sync"

	"github.com/cloudeteer/grafana-pdf-report-app/pkg/plugin/chrome"
	"github.com/cloudeteer/grafana-pdf-report-app/pkg/plugin/config"
	"github.com/cloudeteer/grafana-pdf-report-app/pkg/plugin/dashboard"
	"github.com/cloudeteer/grafana-pdf-report-app/pkg/plugin/worker"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

type Report struct {
	logger         log.Logger
	conf           config.Config
	httpClient     *http.Client
	chromeInstance chrome.Instance
	pools          worker.Pools
	dashboard      *dashboard.Dashboard
}

// Embed the entire directory.
//
//go:embed templates
var templateFS embed.FS

// Base64 content signatures.
var popularSignatures = map[string]string{
	"JVBERi0":     "application/pdf",
	"R0lGODdh":    "image/gif",
	"R0lGODlh":    "image/gif",
	"iVBORw0KGgo": "image/png",
	"/9j/":        "image/jpg",
	"Qk02U":       "image/bmp",
}

func New(logger log.Logger, conf config.Config, httpClient *http.Client, chromeInstance chrome.Instance,
	pools worker.Pools, dashboard *dashboard.Dashboard,
) *Report {
	return &Report{
		logger,
		conf,
		httpClient,
		chromeInstance,
		pools,
		dashboard,
	}
}

func (r *Report) Generate(ctx context.Context, writer http.ResponseWriter) error {
	dashboardData, err := r.dashboard.GetData(ctx, r.conf.DashboardMode == "full")
	if err != nil {
		return fmt.Errorf("failed to get dashboard data: %w", err)
	}

	panelTables := make([]dashboard.PanelTable, len(dashboardData.Panels))
	panelPNGs := make([]dashboard.PanelImage, len(dashboardData.Panels))
	errorCh := make(chan error, len(dashboardData.Panels)*2)

	wg := sync.WaitGroup{}

	for idx, panel := range dashboardData.Panels {
		if panel.Type == dashboard.Table.String() {
			wg.Add(1)

			r.pools[worker.Browser].Do(func() {
				defer wg.Done()

				panelTable, err := r.dashboard.FetchTable(ctx, panel)
				if err != nil {
					errorCh <- fmt.Errorf("failed to fetch CSV data for panel %d: %w", panel.ID, err)
				}

				panelTables[idx] = panelTable
			})
		}

		wg.Add(1)

		r.pools[worker.Renderer].Do(func() {
			defer wg.Done()

			panelPNG, err := r.dashboard.FetchPNG(ctx, panel)
			if err != nil {
				errorCh <- fmt.Errorf("failed to fetch PNG data for panel %d: %w", panel.ID, err)
			}

			panelPNGs[idx] = panelPNG
		})
	}

	wg.Wait()
	close(errorCh)

	errs := make([]error, 0, len(dashboardData.Panels)*2)

	for err := range errorCh {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to generate report: %w", errors.Join(errs...))
	}

	panelTables = slices.DeleteFunc(panelTables, func(panelTable dashboard.PanelTable) bool {
		return panelTable.Data == nil
	})

	// Sanitize title to escape non ASCII characters
	// Ref: https://stackoverflow.com/questions/62705546/unicode-characters-in-attachment-name
	// Ref: https://medium.com/@JeremyLaine/non-ascii-content-disposition-header-in-django-3a20acc05f0d
	filename := url.PathEscape(dashboardData.Title)
	header := fmt.Sprintf(`inline; filename*=UTF-8''%s.pdf`, filename)
	writer.Header().Add("Content-Disposition", header)

	htmlReport, err := r.generateHTMLFile(dashboardData, panelTables, panelPNGs)
	if err != nil {
		return fmt.Errorf("failed to generate HTML file: %w", err)
	}

	if err = r.renderPDF(htmlReport, writer); err != nil {
		return fmt.Errorf("failed to render PDF: %w", err)
	}

	return nil
}

// renderPDF renders HTML page into PDF using Chromium.
func (r *Report) renderPDF(htmlReport HTML, writer io.Writer) error {
	// Create a new tab
	tab := r.chromeInstance.NewTab(r.logger, r.conf)
	defer tab.Close(r.logger)

	err := tab.PrintToPDF(chrome.PDFOptions{
		Header:      htmlReport.Header,
		Body:        htmlReport.Body,
		Footer:      htmlReport.Footer,
		Orientation: r.conf.Orientation,
	}, writer)
	if err != nil {
		return fmt.Errorf("error rendering PDF: %w", err)
	}

	return nil
}
