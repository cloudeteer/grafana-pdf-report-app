package dashboard

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// Regex for parsing X and Y co-ordinates from CSS
// Scales for converting width and height to Grafana units.
var translateRegex = regexp.MustCompile(`translate\((?P<X>\d+)px, (?P<Y>\d+)px\)`)

const (
	scaleWidth  = 30
	scaleHeight = 36
)

//nolint:cyclop
func (d *Dashboard) collectPanelsFromData(apiData APIDashboardData, browserData BrowserData) ([]Panel[string], error) {
	panels := make([]Panel[string], 0, len(browserData.PanelData))

	if browserData.PanelData == nil {
		return nil, errors.New("apiData.RowOrPanels or browserData.PanelData is nil")
	}

	for _, browserPanel := range browserData.PanelData {
		apiIDString, _, _ := strings.Cut(browserPanel.ID, "-")

		apiID, err := strconv.Atoi(apiIDString)
		if err != nil {
			return nil, fmt.Errorf("failed to convert panel ID to int for panel ID %s: %w", apiIDString, err)
		}

		if len(d.conf.IncludePanelIDs) > 0 && slices.Contains(d.conf.IncludePanelIDs, apiID) ||
			len(d.conf.ExcludePanelIDs) > 0 && !slices.Contains(d.conf.ExcludePanelIDs, apiID) {
			continue
		}

		for _, apiPanel := range apiData.RowOrPanels {
			if apiPanel.ID == apiID {
				browserPanel.Type = apiPanel.Type
				break
			}
		}

		if browserPanel.Type == "row" {
			continue
		}

		panelWidth, err := strconv.ParseFloat(strings.TrimSuffix(browserPanel.Width, "px"), 64)
		if err != nil {
			return nil, fmt.Errorf("failed to convert width to float for panel ID %d: %w", browserPanel.ID, err)
		}

		panelHeight, err := strconv.ParseFloat(strings.TrimSuffix(browserPanel.Height, "px"), 64)
		if err != nil {
			return nil, fmt.Errorf("failed to convert height to float for panel ID %d: %w", browserPanel.ID, err)
		}

		matches := translateRegex.FindStringSubmatch(browserPanel.Transform)
		if len(matches) != 3 {
			return nil, fmt.Errorf("failed to parse X and Y co-ordinates from CSS for panel ID %d: %s", browserPanel.ID, browserPanel.Transform)
		}

		panelX, err := strconv.Atoi(matches[translateRegex.SubexpIndex("X")])
		if err != nil {
			return nil, fmt.Errorf("failed to convert X co-ordinate to int for panel ID %d: %w", browserPanel.ID, err)
		}

		panelY, err := strconv.Atoi(matches[translateRegex.SubexpIndex("Y")])
		if err != nil {
			return nil, fmt.Errorf("failed to convert Y co-ordinate to int for panel ID %d: %w", browserPanel.ID, err)
		}

		panels = append(panels, Panel[string]{
			ID:    browserPanel.ID,
			Title: browserPanel.Title,
			Type:  browserPanel.Type,
			GridPos: GridPos{
				H: float64(int64(panelHeight) / scaleHeight),
				W: float64(int64(panelWidth) / scaleWidth),
				X: float64(panelX / scaleWidth),
				Y: float64(panelY / scaleHeight),
			},
		})
	}

	return panels, nil
}
