package report

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

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

// Base64 content signatures
var popularSignatures = map[string]string{
	"JVBERi0":     "application/pdf",
	"R0lGODdh":    "image/gif",
	"R0lGODlh":    "image/gif",
	"iVBORw0KGgo": "image/png",
	"/9j/":        "image/jpg",
	"Qk02U":       "image/bmp",
}

func New(logger log.Logger, conf config.Config, httpClient *http.Client, chromeInstance chrome.Instance,
	pools worker.Pools, dashboard *dashboard.Dashboard) *Report {

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

			panelPNG, err := r.dashboard.FetchPNG(ctx, panel, dashboardData.TimeRange)
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
		return fmt.Errorf("failed to generate report: %v", errors.Join(errs...))
	}

	// Sanitize title to escape non ASCII characters
	// Ref: https://stackoverflow.com/questions/62705546/unicode-characters-in-attachment-name
	// Ref: https://medium.com/@JeremyLaine/non-ascii-content-disposition-header-in-django-3a20acc05f0d
	filename := url.PathEscape(dashboardData.Titel)
	header := fmt.Sprintf(`inline; filename*=UTF-8''%s.pdf`, filename)
	writer.Header().Add("Content-Disposition", header)

	htmlReport, err := r.generateHTMLFile(dashboardData, panelTables, panelPNGs)

	if err = r.renderPDF(htmlReport, writer); err != nil {
		return fmt.Errorf("failed to render PDF: %w", err)
	}

	return nil
}

// generateHTMLFile generates HTML files for PDF
func (r *Report) generateHTMLFile(dashboardData dashboard.Data, panelTables []dashboard.PanelTable, panelPNGs []dashboard.PanelImage) (HTML, error) {
	var tmpl *template.Template
	var err error
	var html HTML

	// Template functions
	funcMap := template.FuncMap{
		// The name "inc" is what the function will be called in the template text.
		"inc": func(i float64) float64 {
			return i + 1
		},

		"mult": func(i int) int {
			return i*30 + 5
		},

		"embed": func(base64Content string) template.URL {
			for signature, mimeType := range popularSignatures {
				if strings.HasPrefix(base64Content, signature) {
					return template.URL(fmt.Sprintf("data:%s;base64,%s", mimeType, base64Content))
				}
			}
			return template.URL(base64Content)
		},

		"url": func(url string) template.URL {
			return template.URL(url)
		},

		"formatDate": func(dateTime time.Time) string {
			return dateTime.Format(time.RFC850)
		},
	}

	// Make a new template for Body of the PDF
	if r.conf.FooterTemplate != "" {
		tmpl, err = template.New("report").Funcs(funcMap).Parse(fmt.Sprintf(`{{define "report.gohtml"}}%s{{end}}`, r.conf.ReportTemplate))
	} else {
		tmpl, err = template.New("report").Funcs(funcMap).ParseFS(templateFS, "templates/report.gohtml")
	}

	if err != nil {
		return HTML{}, fmt.Errorf("error parsing PDF template: %w", err)
	}

	// Template data
	data := templateData{
		time.Now().Format(time.RFC850),
		dashboardData,
		panelTables,
		panelPNGs,
		r.conf,
	}

	// Render the template for Body of the PDF
	bufBody := &bytes.Buffer{}
	if err = tmpl.ExecuteTemplate(bufBody, "report.gohtml", data); err != nil {
		return HTML{}, fmt.Errorf("error executing PDF template: %v", err)
	}
	html.Body = bufBody.String()

	// Make a new template for Header of the PDF
	if r.conf.HeaderTemplate != "" {
		tmpl, err = template.New("header").Funcs(funcMap).Parse(fmt.Sprintf(`{{define "header.gohtml"}}%s{{end}}`, r.conf.HeaderTemplate))
	} else {
		tmpl, err = template.New("header").Funcs(funcMap).ParseFS(templateFS, "templates/header.gohtml")
	}

	if err != nil {
		return HTML{}, fmt.Errorf("error parsing Header template: %w", err)
	}

	// Render the template for Header of the PDF
	bufHeader := &bytes.Buffer{}
	if err = tmpl.ExecuteTemplate(bufHeader, "header.gohtml", data); err != nil {
		return HTML{}, fmt.Errorf("error executing Header template: %w", err)
	}
	html.Header = bufHeader.String()

	// Make a new template for Footer of the PDF
	if r.conf.FooterTemplate != "" {
		tmpl, err = template.New("footer").Funcs(funcMap).Parse(fmt.Sprintf(`{{define "footer.gohtml"}}%s{{end}}`, r.conf.FooterTemplate))
	} else {
		tmpl, err = template.New("footer").Funcs(funcMap).ParseFS(templateFS, "templates/footer.gohtml")
	}

	if err != nil {
		return HTML{}, fmt.Errorf("error parsing Footer template: %w", err)
	}

	// Render the template for Footer of the PDF
	bufFooter := &bytes.Buffer{}
	if err = tmpl.ExecuteTemplate(bufFooter, "footer.gohtml", data); err != nil {
		return HTML{}, fmt.Errorf("error executing Footer template: %w", err)
	}

	html.Footer = bufFooter.String()

	return html, nil
}

// renderPDF renders HTML page into PDF using Chromium
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
