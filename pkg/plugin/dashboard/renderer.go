package dashboard

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

func (d *Dashboard) FetchPNG(ctx context.Context, panel Panel[string]) (PanelImage, error) {
	panelPNGURL, err := d.getPanelPNGURL(panel)
	if err != nil {
		return PanelImage{}, fmt.Errorf("error getting panel PNG URL: %w", err)
	}

	panelImage, err := d.fetchPNGFromGrafanaAPI(ctx, panelPNGURL)
	if err != nil {
		return PanelImage{}, fmt.Errorf("error fetching panel PNG: %w", err)
	}

	panelImage.Panel = panel

	return panelImage, nil
}

func (d *Dashboard) getPanelPNGURL(panel Panel[string]) (string, error) {
	dashURL, err := url.Parse(d.grafanaBaseURL)
	if err != nil {
		return "", fmt.Errorf("error parsing Grafana base URL: %w", err)
	}

	dashURL = dashURL.JoinPath("render/d-solo", d.uid, "_")

	dashURLValues := maps.Clone(d.values)
	dashURLValues.Set("theme", d.conf.Theme)
	dashURLValues.Set("panelId", panel.ID)

	// If using a grid layout we use 100px for width and 36px for height scaling.
	// Grafana panels are fitted into 24 units width and height units are said to
	// 30px in docs but 36px seems to be better.
	//
	// In simple layout we create panels with 1000x500 resolution always and include
	// them one in each page of report
	if d.conf.Layout == "grid" {
		width := int(panel.GridPos.W * 100)
		height := int(panel.GridPos.H * 36)

		dashURLValues.Set("width", strconv.Itoa(width))
		dashURLValues.Set("height", strconv.Itoa(height))
	} else {
		dashURLValues.Set("width", "1000")
		dashURLValues.Set("height", "500")
	}

	dashURL.RawQuery = dashURLValues.Encode()

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

	d.logger.Debug("fetching panel PNG", "url", panelURL)

	// Send the request
	resp, err := d.httpClient.Do(req) //nolint:bodyclose //https://github.com/timakin/bodyclose/issues/30
	if err != nil {
		return PanelImage{}, fmt.Errorf("error sending request: %w", err)
	}

	// Do multiple tries to get panel before giving up
	for retries := 1; retries < 3 && resp.StatusCode != http.StatusOK; retries++ {
		if err = resp.Body.Close(); err != nil {
			return PanelImage{}, fmt.Errorf("error closing response body: %w", err)
		}

		time.Sleep(10 * time.Second * time.Duration(retries))

		resp, err = d.httpClient.Do(req) //nolint:bodyclose //https://github.com/timakin/bodyclose/issues/30
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
