package dashboard

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"maps"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/cdproto/browser"
	"github.com/chromedp/chromedp"
	"github.com/cloudeteer/grafana-pdf-report-app/pkg/plugin/chrome"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

const (
	// javascriptExpandRows is a javascript code that will expand all rows in the dashboard.
	javascriptExpandRows = `[...document.querySelectorAll("[data-testid='dashboard-row-container']")].map((e) => [...e.querySelectorAll("[aria-expanded=false]")].map((e) => e.click()))`

	// javascriptScrollToBottom is a javascript code that will scroll to the bottom of the page.
	javascriptScrollToBottom = `
const scrollContainer = document.getElementById('page-scrollbar') ?? document.querySelectorAll('#pageContent > div > div > .scrollbar-view')[0];
const scrollHeight = document.body.scrollHeight;
const totalScrollHeight = scrollContainer.scrollHeight;
const promises = [];

for (let currentHeight = scrollHeight, delay = 0; currentHeight < totalScrollHeight; currentHeight += scrollHeight) {
    promises.push(new Promise((resolve) => {
        setTimeout(() => {
            scrollContainer.scrollTo(0, currentHeight);
            resolve();
        }, currentHeight);
    }));
}

Promise.all(promises);`

	javascriptPanelData = `
[...document.getElementsByClassName('react-grid-item')].map(e => {
	props = e[Object.keys(e).filter(k => k.includes("Props"))[0]];
	return {
		"width": e.style.width,
		"height": e.style.height,
		"transform": e.style.transform,
		"id": parseInt(e.getAttribute("data-panelid")),
		"type": props.children[0].props.panel.type,
		"title": e.querySelector('h2')?.innerText,
    }
})`

	javascriptTimePickerMouseEnter = `document.querySelector('button[aria-controls="TimePickerContent"]').dispatchEvent(new Event('mouseenter'));`

	javascriptGetTimeRange = `
let timeRange = document.querySelector('[role="tooltip"]').innerText.split("\n");
let output = {
    from: Math.floor(Date.parse(timeRange[0]+" "+timeRange[3].replace('Local browser time', '')) / 1000), 
    to: Math.floor(Date.parse(timeRange[2]+" "+timeRange[3].replace('Local browser time', '')) / 1000),
}; output
`

	selPageScrollbar              = `#page-scrollbar`
	selTimePickerTimeRangeToolTip = `div[role="tooltip"]`
	selTimePickerButton           = `button[aria-controls="TimePickerContent"]`

	selDownloadCSVButton                             = `div[aria-label="Panel inspector Data content"] button[type="button"][aria-disabled="false"]`
	selInspectPanelDataTabExpandDataOptions          = `div[role='dialog'] button[aria-expanded=false]`
	selInspectPanelDataTabApplyTransformationsToggle = `div[data-testid="dataOptions"] input:not(#excel-toggle):not(#formatted-data-toggle) + label`
)

func (d *Dashboard) fetchBrowser(ctx context.Context, expandRows bool) (BrowserData, error) {
	dashURL, err := url.Parse(d.grafanaBaseURL)
	if err != nil {
		return BrowserData{}, fmt.Errorf("error parsing Grafana base URL: %w", err)
	}

	dashURL = dashURL.JoinPath("d", d.uid, "_")

	dashURLValues := maps.Clone(d.values)
	dashURLValues.Set("theme", d.conf.Theme)

	dashURL.RawQuery = dashURLValues.Encode()

	browserData, err := d.fetchPanelDataFromBrowser(ctx, dashURL.String(), expandRows)
	if err != nil {
		return BrowserData{}, fmt.Errorf("error fetching browser data: %w", err)
	}

	return browserData, nil
}

// fetchPanelDataFromBrowser fetches the panel data for a dashboard using a browser.
// It fetch data of all grafana panels, including repeating one, which are not visible via API.
// Additionally, it fetches the absolute time range of the dashboard.
// Asking for absolute time range avoids additional logic to parse time ranges like 'now-30d/M'.
//
//nolint:cyclop
func (d *Dashboard) fetchPanelDataFromBrowser(_ context.Context, dashURL string, expandRows bool) (BrowserData, error) {
	tab := d.chromeInstance.NewTab(d.logger, d.conf)
	tab.WithTimeout(5 * time.Minute)

	defer tab.Close(d.logger)

	// Set the OAuth token in the headers
	headers := map[string]any{backend.OAuthIdentityTokenHeaderName: "Bearer " + d.saToken}

	d.logger.Debug("Navigating to dashboard via browser", "url", dashURL)

	// Navigate to the dashboard
	if err := tab.NavigateAndWaitFor(dashURL, headers, "networkIdle"); err != nil {
		return BrowserData{}, fmt.Errorf("NavigateAndWaitFor: %w", err)
	}

	// Expand all rows, if requested
	if expandRows {
		if err := tab.RunWithTimeout(5*time.Second, chromedp.Evaluate(javascriptExpandRows, nil)); err != nil {
			return BrowserData{}, fmt.Errorf("error uncollapsing rows: %w", err)
		}
	}

	// Check if the page has a scrollbar
	if err := tab.RunWithTimeout(5*time.Second, chromedp.WaitReady(selPageScrollbar, chromedp.ByQuery)); err != nil {
		return BrowserData{}, fmt.Errorf("error waiting for #page-scrollbar: %w", err)
	}

	if err := tab.RunWithTimeout(30*time.Second, chromedp.Evaluate(javascriptScrollToBottom, nil, chrome.WithAwaitPromise)); err != nil {
		return BrowserData{}, fmt.Errorf("error scrolling to bottom: %w", err)
	}

	// Fetch dashboard data
	var dashboardData BrowserData

	// JS that will fetch dashboard model
	if err := tab.RunWithTimeout(30*time.Second, chromedp.Evaluate(javascriptPanelData, &dashboardData.PanelData)); err != nil {
		return BrowserData{}, fmt.Errorf("error fetching panel data: %w", err)
	}

	if len(dashboardData.PanelData) == 0 {
		return BrowserData{}, ErrJavaScriptReturnedNoPanels
	}

	// Check if the page has a time picker button
	if err := tab.RunWithTimeout(5*time.Second, chromedp.WaitReady(selTimePickerButton, chromedp.ByQuery)); err != nil {
		return BrowserData{}, fmt.Errorf("error waiting for #page-scrollbar: %w", err)
	}

	// To get the time range, we need to hover over the time picker button
	if err := tab.RunWithTimeout(5*time.Second, chromedp.Evaluate(javascriptTimePickerMouseEnter, nil)); err != nil {
		return BrowserData{}, fmt.Errorf("error mouse entering time picker: %w", err)
	}

	// Check if the page has a time picker button
	if err := tab.RunWithTimeout(5*time.Second, chromedp.WaitReady(selTimePickerTimeRangeToolTip, chromedp.ByQuery)); err != nil {
		return BrowserData{}, fmt.Errorf("error waiting for time picker tooltip: %w", err)
	}

	// Fetch the time range data
	if err := tab.RunWithTimeout(5*time.Second, chromedp.Evaluate(javascriptGetTimeRange, &dashboardData.TimeRange)); err != nil {
		return BrowserData{}, fmt.Errorf("error fetching time range: %w", err)
	}

	dashboardData.TimeRange.FromTime = time.Unix(dashboardData.TimeRange.From, 0)
	dashboardData.TimeRange.ToTime = time.Unix(dashboardData.TimeRange.To, 0)

	return dashboardData, nil
}

func (d *Dashboard) FetchTable(ctx context.Context, panel Panel) (PanelTable, error) {
	dashURL, err := url.Parse(d.grafanaBaseURL)
	if err != nil {
		return PanelTable{}, fmt.Errorf("error parsing Grafana base URL: %w", err)
	}

	dashURL = dashURL.JoinPath("d", d.uid, "_")

	dashURLValues := maps.Clone(d.values)
	dashURLValues.Set("theme", d.conf.Theme)
	dashURLValues.Set("viewPanel", strconv.Itoa(panel.ID))
	dashURLValues.Set("inspect", strconv.Itoa(panel.ID))
	dashURLValues.Set("inspectTab", "data")

	dashURL.RawQuery = dashURLValues.Encode()

	data, err := d.fetchTableData(ctx, dashURL.String())
	if err != nil {
		return PanelTable{}, fmt.Errorf("error fetching browser data: %w", err)
	}

	return PanelTable{
		Title: panel.Title,
		Data:  data,
	}, nil
}

// fetchTableData fetches the CSV data for a panel using a browser.
// The panel URL is the same as the dashboard URL, but with additional query parameters.
// It will navigate to the panel URL and download the CSV data.
// The CSV data is generated by the Grafana frontend, so we need to use a browser to fetch it.
// The browser will navigate to the panel URL, click the "Download CSV" button, and capture the CSV data.
//
//nolint:cyclop
func (d *Dashboard) fetchTableData(_ context.Context, panelURL string) (PanelTableData, error) {
	tab := d.chromeInstance.NewTab(d.logger, d.conf)
	tab.WithTimeout(1 * time.Minute)

	defer tab.Close(d.logger)

	// Set the OAuth token in the headers
	headers := map[string]any{backend.OAuthIdentityTokenHeaderName: "Bearer " + d.saToken}

	d.logger.Debug("fetch table data via browser", "url", panelURL)

	err := tab.NavigateAndWaitFor(panelURL, headers, "networkIdle")
	if err != nil {
		return nil, fmt.Errorf("NavigateAndWaitFor: %w", err)
	}

	// this will be used to capture the blob URL of the CSV download
	blobURLCh := make(chan string, 1)

	// If an error occurs on the way to fetching the CSV data, it will be sent to this channel
	errCh := make(chan error, 1)

	// Listen for download events. Downloading from JavaScript won't emit any network events.
	chromedp.ListenTarget(tab.Context(), func(event interface{}) {
		if eventDownloadWillBegin, ok := event.(*browser.EventDownloadWillBegin); ok {
			d.logger.Debug("got CSV download URL", "url", eventDownloadWillBegin.URL)
			// once we have the download URL, we can fetch the CSV data via JavaScript.
			blobURLCh <- eventDownloadWillBegin.URL
		}
	})

	task := chromedp.Action(
		browser.SetDownloadBehavior(browser.SetDownloadBehaviorBehaviorAllowAndName).
			WithDownloadPath("/dev/null").
			WithEventsEnabled(true),
	)
	if err = tab.RunWithTimeout(2*time.Second, task); err != nil {
		return nil, fmt.Errorf("error setting download behavior: %w", err)
	}

	if err = tab.RunWithTimeout(2*time.Second, chromedp.WaitVisible(selDownloadCSVButton, chromedp.ByQuery)); err != nil {
		return nil, fmt.Errorf("error waiting for download CSV button: %w", err)
	}

	if err = tab.RunWithTimeout(2*time.Second, chromedp.Click(selInspectPanelDataTabExpandDataOptions, chromedp.ByQuery)); err != nil {
		return nil, fmt.Errorf("error clicking on expand data options: %w", err)
	}

	if err = tab.RunWithTimeout(1*time.Second, chromedp.Click(selInspectPanelDataTabApplyTransformationsToggle, chromedp.ByQuery)); err != nil && !errors.Is(err, context.DeadlineExceeded) {
		return nil, fmt.Errorf("error clicking on apply transformations toggle: %w", err)
	}

	if err = tab.RunWithTimeout(1*time.Second, chromedp.Click(selInspectPanelDataTabApplyTransformationsToggle, chromedp.ByQuery)); err != nil && !errors.Is(err, context.DeadlineExceeded) {
		return nil, fmt.Errorf("error clicking on apply transformations toggle: %w", err)
	}

	// Run all tasks in a goroutine.
	// If an error occurs, it will be sent to the errCh channel.
	// If a element can't be found, a timeout will occur and the context will be canceled.
	go func() {
		task = chromedp.Click(selDownloadCSVButton, chromedp.ByQuery)
		if err := tab.Run(task); err != nil {
			errCh <- fmt.Errorf("error fetching dashboard URL from browser %s: %w", panelURL, err)
		}
	}()

	var blobURL string

	select {
	case blobURL = <-blobURLCh:
		if blobURL == "" {
			return nil, fmt.Errorf("error fetching CSV data from URL from browser %s: %w", panelURL, ErrEmptyBlobURL)
		}

		blobURLObject, err := url.Parse(strings.TrimPrefix(blobURL, "blob:"))
		if err != nil {
			return nil, fmt.Errorf("error parsing blob URL %s: %w", blobURL, err)
		}

		blobURLObject.Host = strings.TrimPrefix(strings.TrimPrefix(d.grafanaBaseURL, "http://"), "https://")

	case err := <-errCh:
		return nil, fmt.Errorf("error fetching CSV data from URL from browser %s: %w", panelURL, err)
	case <-tab.Context().Done():
		return nil, fmt.Errorf("error fetching CSV data from URL from browser %s: %w", panelURL, tab.Context().Err())
	}

	close(blobURLCh)
	close(errCh)

	var buf []byte

	task = chromedp.Evaluate(
		// fetch the CSV data from the blob URL, using Javascript.
		fmt.Sprintf("fetch('%s').then(r => r.blob()).then(b => new Response(b).text()).then(t => t)", blobURL),
		&buf,
		chrome.WithAwaitPromise,
	)

	if err := tab.RunWithTimeout(45*time.Second, task); err != nil {
		return nil, fmt.Errorf("error fetching CSV data from URL from browser %s: %w", panelURL, err)
	}

	if len(buf) == 0 {
		return nil, fmt.Errorf("error fetching CSV data from URL from browser %s: %w", panelURL, ErrEmptyCSVData)
	}

	csvStringData, err := strconv.Unquote(string(buf))
	if err != nil {
		return nil, fmt.Errorf("error unquoting CSV data: %w", err)
	}

	reader := csv.NewReader(strings.NewReader(csvStringData))

	csvData, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("error reading CSV data: %w", err)
	}

	return csvData, nil
}
