package report

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/cloudeteer/grafana-pdf-report-app/pkg/plugin/dashboard"
)

// generateHTMLFile generates HTML files for PDF.
//
//nolint:cyclop
func (r *Report) generateHTMLFile(dashboardData dashboard.Data, panelTables []dashboard.PanelTable, panelPNGs []dashboard.PanelImage) (HTML, error) {
	var (
		err  error
		html HTML
		tmpl *template.Template
	)

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
					return template.URL(fmt.Sprintf("data:%s;base64,%s", mimeType, base64Content)) //nolint:gosec
				}
			}

			return template.URL(base64Content) //nolint:gosec
		},

		"url": func(url string) template.URL {
			return template.URL(url) //nolint:gosec
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
		return HTML{}, fmt.Errorf("error executing PDF template: %w", err)
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
