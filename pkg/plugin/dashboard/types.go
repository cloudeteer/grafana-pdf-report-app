package dashboard

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/cloudeteer/grafana-pdf-report-app/pkg/plugin/chrome"
	"github.com/cloudeteer/grafana-pdf-report-app/pkg/plugin/config"
	"github.com/cloudeteer/grafana-pdf-report-app/pkg/plugin/worker"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

type PanelType int

func (p PanelType) String() string {
	return [...]string{
		"singlestat",
		"text",
		"graph",
		"table",
	}[p]
}

const (
	SingleStat PanelType = iota
	Text
	Graph
	Table
)

type Dashboard struct {
	logger         log.Logger
	conf           config.Config
	httpClient     *http.Client
	chromeInstance chrome.Instance
	workerPools    worker.Pools

	grafanaBaseURL string
	uid            string
	values         url.Values
	saToken        string
}

type Data struct {
	Title     string
	TimeRange TimeRange
	Panels    []Panel
}

type BrowserData struct {
	TimeRange TimeRange
	PanelData []BrowserPanelData
}

type BrowserPanelData struct {
	ID        int    `json:"id"`
	Width     string `json:"width"`
	Height    string `json:"height"`
	Title     string `json:"title"`
	Type      string `json:"type"`
	Transform string `json:"transform"`
}

type TimeRange struct {
	From     int64 `json:"from"`
	To       int64 `json:"to"`
	FromTime time.Time
	ToTime   time.Time
}

type APIData struct {
	Dashboard APIDashboardData `json:"dashboard"`
}

type APIDashboardData struct {
	Title          string       `json:"title"`
	Description    string       `json:"description"`
	VariableValues string       // Not present in the Grafana JSON structure. Enriched data passed used by the Tex templating
	RowOrPanels    []RowOrPanel `json:"panels"`
}

// RowOrPanel represents a container for Panels.
type RowOrPanel struct {
	Panel
	Collapsed bool    `json:"collapsed"`
	Panels    []Panel `json:"panels"`
}

// Panel represents a Grafana dashboard panel.
type Panel struct {
	ID      int     `json:"id"`
	Type    string  `json:"type"`
	Title   string  `json:"title"`
	GridPos GridPos `json:"gridPos"`
}

// GridPos represents a Grafana dashboard panel position.
type GridPos struct {
	H float64 `json:"h"`
	W float64 `json:"w"`
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// IsSingleStat returns true if panel is of type SingleStat.
func (p Panel) IsSingleStat() bool {
	return p.Is(SingleStat)
}

// IsPartialWidth If panel has width less than total allowable width.
func (p Panel) IsPartialWidth() bool {
	return (p.GridPos.W < 24)
}

// Width returns the width of the panel.
func (p Panel) Width() float64 {
	return float64(p.GridPos.W) * 0.04
}

// Height returns the height of the panel.
func (p Panel) Height() float64 {
	return float64(p.GridPos.H) * 0.04
}

// Is returns true if panel is of type t.
func (p Panel) Is(t PanelType) bool {
	return p.Type == t.String()
}

type PanelImage struct {
	Panel
	Image    string
	MimeType string
}

func (p PanelImage) String() string {
	return fmt.Sprintf("data:%s;base64,%s", p.MimeType, p.Image)
}

type PanelTable struct {
	Title string
	Data  PanelTableData
}

type PanelTableData [][]string
