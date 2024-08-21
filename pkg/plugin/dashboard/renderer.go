package dashboard

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/cloudeteer/grafana-pdf-report-app/pkg/plugin/worker"
)

func (d *Dashboard) FetchPNG(ctx context.Context, panel Panel, timeRange TimeRange) (PanelImage, error) {
	panelPNGURL, err := d.getPanelPNGURL(panel, timeRange)
	if err != nil {
		return PanelImage{}, fmt.Errorf("error getting panel PNG URL: %w", err)
	}

	wg := sync.WaitGroup{}

	var panelImage PanelImage

	d.workerPools[worker.Renderer].Do(func() {
		defer wg.Done()

		panelImage, err = d.fetchPNGFromGrafanaAPI(ctx, panelPNGURL)
	})

	wg.Wait()

	if err != nil {
		return PanelImage{}, fmt.Errorf("error fetching panel PNG: %w", err)
	}

	panelImage.Panel = panel

	return panelImage, nil
}

func (d *Dashboard) getPanelPNGURL(panel Panel, timeRange TimeRange) (string, error) {
	dashURL, err := url.Parse(d.grafanaBaseURL)
	if err != nil {
		return "", fmt.Errorf("error parsing Grafana base URL: %w", err)
	}

	dashURL = dashURL.JoinPath("render/d-solo", d.uid, "_")
	dashURL.RawQuery = d.values

	dashURL.Query().Add("theme", d.conf.Theme)
	dashURL.Query().Add("panelId", strconv.Itoa(panel.ID))
	dashURL.Query().Add("from", strconv.FormatInt(timeRange.From, 10))
	dashURL.Query().Add("to", strconv.FormatInt(timeRange.To, 10))

	// If using a grid layout we use 100px for width and 36px for height scaling.
	// Grafana panels are fitted into 24 units width and height units are said to
	// 30px in docs but 36px seems to be better.
	//
	// In simple layout we create panels with 1000x500 resolution always and include
	// them one in each page of report
	if d.conf.Layout == "grid" {
		width := int(panel.GridPos.W * 100)
		height := int(panel.GridPos.H * 36)
		dashURL.Query().Add("width", strconv.Itoa(width))
		dashURL.Query().Add("height", strconv.Itoa(height))
	} else {
		dashURL.Query().Add("width", "1000")
		dashURL.Query().Add("height", "500")
	}

	// Get Panel API endpoint
	return dashURL.String(), nil
}

func (d *Dashboard) fetchPNGFromGrafanaAPI(ctx context.Context, panelURL string) (PanelImage, error) {
	// Create a new request for panel
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, panelURL, nil)
	if err != nil {
		return PanelImage{}, fmt.Errorf("error creating request for %s: %w", panelURL, err)
	}

	// Add the Authorization header
	req.Header.Add("Authorization", "Bearer "+d.saToken)

	// Send the request
	resp, err := d.httpClient.Do(req)
	if err != nil {
		return PanelImage{}, fmt.Errorf("error sending request: %w", err)
	}

	// Do multiple tries to get panel before giving up
	for retries := 1; retries < 3 && resp.StatusCode != http.StatusOK; retries++ {
		if err = resp.Body.Close(); err != nil {
			return PanelImage{}, fmt.Errorf("error closing response body: %w", err)
		}

		time.Sleep(10 * time.Second * time.Duration(retries))

		resp, err = d.httpClient.Do(req)
		if err != nil {
			return PanelImage{}, fmt.Errorf("error executing retry request for %s: %w", panelURL, err)
		}
	}

	// Close the response body
	defer func(body io.ReadCloser) {
		if err := body.Close(); err != nil {
			d.logger.Error("error closing response body", "error", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)

		return PanelImage{}, fmt.Errorf(
			"%w: URL: %s. Status: %s, message: %s",
			ErrImageRendererHTTPError,
			panelURL,
			resp.Status,
			string(body),
		)
	}

	sb := &bytes.Buffer{}
	sb.Grow(base64.StdEncoding.EncodedLen(int(resp.ContentLength)))

	encoder := base64.NewEncoder(base64.StdEncoding, sb)

	if _, err = io.Copy(encoder, resp.Body); err != nil {
		return PanelImage{}, fmt.Errorf("error reading response body of panel PNG: %w", err)
	}

	return PanelImage{
		Image:    sb.String(),
		MimeType: "image/png",
	}, nil
}
