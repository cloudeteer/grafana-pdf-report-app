package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

func (d *Dashboard) fetchAPI(ctx context.Context) (APIData, error) {
	dashURL, err := url.Parse(d.grafanaBaseURL)
	if err != nil {
		return APIData{}, fmt.Errorf("error parsing Grafana base URL: %w", err)
	}

	dashURL = dashURL.JoinPath("api/dashboards/uid", d.uid)
	dashURL.RawQuery = d.values

	// Create a new GET request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, dashURL.String(), nil)
	if err != nil {
		return APIData{}, fmt.Errorf("error creating request for %s: %w", dashURL.String(), err)
	}

	// Add the Authorization header
	req.Header.Add("Authorization", "Bearer "+d.saToken)

	// Send the request
	resp, err := d.httpClient.Do(req)
	if err != nil {
		return APIData{}, fmt.Errorf("error sending request: %w", err)
	}

	// Close the response body
	defer func(body io.ReadCloser) {
		if err := body.Close(); err != nil {
			d.logger.Error("error closing response body", "error", err)
		}
	}(resp.Body)

	// Check if the response status code is not 200
	if resp.StatusCode != http.StatusOK {
		// ignore the response body error if the status code is not 200
		body, _ := io.ReadAll(resp.Body)

		return APIData{}, fmt.Errorf(
			"%w: URL: %s. Status: %s, message: %s",
			ErrDashboardHTTPError,
			dashURL.String(),
			resp.Status,
			string(body),
		)
	}

	// Decode the response body
	var data APIData

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return APIData{}, fmt.Errorf("error decoding response body: %w", err)
	}

	return data, nil
}
