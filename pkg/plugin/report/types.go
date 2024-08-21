package report

import (
	"github.com/cloudeteer/grafana-pdf-report-app/pkg/plugin/config"
	"github.com/cloudeteer/grafana-pdf-report-app/pkg/plugin/dashboard"
)

type HTML struct {
	Header string
	Body   string
	Footer string
}

// Data structures used inside HTML template
type templateData struct {
	Date string

	Dashboard   dashboard.Data
	PanelTables []dashboard.PanelTable
	PanelPNGs   []dashboard.PanelImage
	Conf        config.Config
}
