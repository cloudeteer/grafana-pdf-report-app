package dashboard

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/cloudeteer/grafana-pdf-report-app/pkg/plugin/chrome"
	"github.com/cloudeteer/grafana-pdf-report-app/pkg/plugin/config"
	"github.com/cloudeteer/grafana-pdf-report-app/pkg/plugin/worker"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

// New creates a new instance of the Dashboard struct.
func New(logger log.Logger, conf config.Config, httpClient *http.Client, chromeInstance chrome.Instance,
	pools worker.Pools, grafanaBaseURL string, uid string, values url.Values, saToken string,
) *Dashboard {
	return &Dashboard{
		logger,
		conf,
		httpClient,
		chromeInstance,
		pools,
		grafanaBaseURL,
		uid,
		values,
		saToken,
	}
}

func (d *Dashboard) GetData(ctx context.Context, expandRows bool) (Data, error) {
	apiData, err := d.fetchAPI(ctx)
	if err != nil {
		d.logger.Error("error fetching dashboard from API", "error", err)

		return Data{}, fmt.Errorf("error fetching dashboard from API: %w", err)
	}

	browserData, err := d.fetchBrowser(ctx, expandRows)
	if err != nil {
		d.logger.Error("error fetching dashboard from API", "error", err)

		return Data{}, fmt.Errorf("error fetching dashboard from API: %w", err)
	}

	panels, err := d.collectPanelsFromData(browserData)
	if err != nil {
		d.logger.Error("error collecting panels from data", "error", err)

		return Data{}, fmt.Errorf("error collecting panels from data: %w", err)
	}

	return Data{
		Title:     apiData.Title,
		TimeRange: browserData.TimeRange,
		Panels:    panels,
	}, err
}
