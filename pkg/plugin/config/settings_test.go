package config_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/cloudeteer/grafana-pdf-report-app/pkg/plugin/config"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSettings(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name       string
		config     string
		secretsMap map[string]string
		expected   config.Config
	}{
		{
			"empty config",
			`{}`,
			nil,
			func() config.Config {
				conf := config.DefaultConfig

				return conf
			}(),
		},
		{
			"layout config",
			`{"layout": "grid"}`,
			nil,
			func() config.Config {
				conf := config.DefaultConfig
				conf.Layout = "grid"

				return conf
			}(),
		},
		{
			"with secrets",
			`{"layout": "grid"}`,
			map[string]string{
				"saToken": "superSecretToken",
			},
			func() config.Config {
				conf := config.DefaultConfig
				conf.Layout = "grid"
				conf.Token = "superSecretToken"

				return conf
			}(),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			conf, err := config.Load(context.Background(), backend.AppInstanceSettings{JSONData: json.RawMessage(tc.config), DecryptedSecureJSONData: tc.secretsMap})

			require.NoError(t, err)

			assert.Equal(t, tc.expected.HTTPClientOptions.TLS, conf.HTTPClientOptions.TLS)

			tc.expected.HTTPClientOptions = httpclient.Options{}
			conf.HTTPClientOptions = httpclient.Options{}
			assert.Equal(t, tc.expected, conf)
		})
	}
}

func TestSettingsUsingEnvVars(t *testing.T) {
	// Setup env vars
	t.Setenv("GF_REPORTER_PLUGIN_APP_URL", "https://localhost:3000")
	t.Setenv("GF_REPORTER_PLUGIN_SKIP_TLS_CHECK", "true")
	t.Setenv("GF_REPORTER_PLUGIN_REPORT_THEME", "light")
	t.Setenv("GF_REPORTER_PLUGIN_REPORT_ORIENTATION", "landscape")
	t.Setenv("GF_REPORTER_PLUGIN_REPORT_LAYOUT", "grid")
	t.Setenv("GF_REPORTER_PLUGIN_REPORT_DASHBOARD_MODE", "full")
	t.Setenv("GF_REPORTER_PLUGIN_REPORT_TIMEZONE", "America/New_York")
	t.Setenv("GF_REPORTER_PLUGIN_REPORT_LOGO", "encodedLogo")
	t.Setenv("GF_REPORTER_PLUGIN_REMOTE_CHROME_URL", "ws://localhost:5333")

	const configJSON = `{}`
	configData := json.RawMessage(configJSON)
	conf, err := config.Load(context.Background(), backend.AppInstanceSettings{JSONData: configData})

	require.NoError(t, err)

	assert.Equal(t, "https://localhost:3000", conf.AppURL)
	assert.Equal(t, true, conf.SkipTLSCheck)
	assert.Equal(t, "light", conf.Theme)
	assert.Equal(t, "landscape", conf.Orientation)
	assert.Equal(t, "grid", conf.Layout)
	assert.Equal(t, "full", conf.DashboardMode)
	assert.Equal(t, "America/New_York", conf.TimeZone)
	assert.Equal(t, "encodedLogo", conf.EncodedLogo)
	assert.Equal(t, 2, conf.MaxBrowserWorkers)
	assert.Equal(t, 2, conf.MaxRenderWorkers)
	assert.Equal(t, "ws://localhost:5333", conf.RemoteChromeURL)
}

func TestSettingsUsingConfigAndEnvVars(t *testing.T) {
	// Setup env vars
	t.Setenv("GF_REPORTER_PLUGIN_SKIP_TLS_CHECK", "true")
	t.Setenv("GF_REPORTER_PLUGIN_REPORT_THEME", "light")
	t.Setenv("GF_REPORTER_PLUGIN_REPORT_ORIENTATION", "landscape")
	t.Setenv("GF_REPORTER_PLUGIN_REPORT_LAYOUT", "grid")
	t.Setenv("GF_REPORTER_PLUGIN_REPORT_TIMEZONE", "America/New_York")
	t.Setenv("GF_REPORTER_PLUGIN_REPORT_LOGO", "encodedLogo")
	t.Setenv("GF_REPORTER_PLUGIN_REMOTE_CHROME_URL", "ws://localhost:5333")

	const configJSON = `{"appUrl": "https://localhost:3000","dashboardMode": "full"}`
	configData := json.RawMessage(configJSON)
	conf, err := config.Load(context.Background(), backend.AppInstanceSettings{JSONData: configData})

	require.NoError(t, err)

	assert.Equal(t, "https://localhost:3000", conf.AppURL)
	assert.Equal(t, true, conf.SkipTLSCheck)
	assert.Equal(t, "light", conf.Theme)
	assert.Equal(t, "landscape", conf.Orientation)
	assert.Equal(t, "grid", conf.Layout)
	assert.Equal(t, "full", conf.DashboardMode)
	assert.Equal(t, "America/New_York", conf.TimeZone)
	assert.Equal(t, "encodedLogo", conf.EncodedLogo)
	assert.Equal(t, 2, conf.MaxBrowserWorkers)
	assert.Equal(t, 2, conf.MaxRenderWorkers)
	assert.Equal(t, "ws://localhost:5333", conf.RemoteChromeURL)
}

func TestSettingsUsingConfigAndOverridingEnvVars(t *testing.T) {
	// Setup env vars
	t.Setenv("GF_REPORTER_PLUGIN_APP_URL", "https://example.grafana.com")
	t.Setenv("GF_REPORTER_PLUGIN_SKIP_TLS_CHECK", "true")
	t.Setenv("GF_REPORTER_PLUGIN_REPORT_THEME", "light")
	t.Setenv("GF_REPORTER_PLUGIN_REPORT_ORIENTATION", "landscape")
	t.Setenv("GF_REPORTER_PLUGIN_REPORT_LAYOUT", "grid")
	t.Setenv("GF_REPORTER_PLUGIN_REPORT_TIMEZONE", "America/New_York")
	t.Setenv("GF_REPORTER_PLUGIN_REPORT_LOGO", "encodedLogo")
	t.Setenv("GF_REPORTER_PLUGIN_REMOTE_CHROME_URL", "ws://localhost:5333")

	const configJSON = `{"appUrl": "https://localhost:3000","theme": "dark", "dashboardMode": "full"}`
	configData := json.RawMessage(configJSON)
	conf, err := config.Load(context.Background(), backend.AppInstanceSettings{JSONData: configData})

	require.NoError(t, err)

	assert.Equal(t, "https://example.grafana.com", conf.AppURL)
	assert.Equal(t, true, conf.SkipTLSCheck)
	assert.Equal(t, "light", conf.Theme)
	assert.Equal(t, "landscape", conf.Orientation)
	assert.Equal(t, "grid", conf.Layout)
	assert.Equal(t, "full", conf.DashboardMode)
	assert.Equal(t, "America/New_York", conf.TimeZone)
	assert.Equal(t, "encodedLogo", conf.EncodedLogo)
	assert.Equal(t, 2, conf.MaxBrowserWorkers)
	assert.Equal(t, 2, conf.MaxRenderWorkers)
	assert.Equal(t, "ws://localhost:5333", conf.RemoteChromeURL)
}
