[![ci](https://github.com/cloudeteer/grafana-pdf-report-app/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/cloudeteer/grafana-pdf-report-app/actions/workflows/ci.yml?query=branch%3Amain) [![docs](https://img.shields.io/badge/docs-passing-green?style=flat&link=https://github.com/cloudeteer/grafana-pdf-report-app/blob/main/src/README.md)](https://github.com/cloudeteer/grafana-pdf-report-app/blob/main/src/README.md) [![Release](https://img.shields.io/github/v/release/cloudeteer/grafana-pdf-report-app.svg?include_prereleases)](https://github.com/cloudeteer/grafana-pdf-report-app/releases/latest) [![GitHub License](https://img.shields.io/github/license/cloudeteer/grafana-pdf-report-app)](https://gitlab.com/cloudeteer/grafana-pdf-report-app) [![Go Report Card](https://goreportcard.com/badge/github.com/cloudeteer/grafana-pdf-report-app)](https://goreportcard.com/report/github.com/cloudeteer/grafana-pdf-report-app) [![code style](https://img.shields.io/badge/code%20style-gofmt-blue.svg)](https://pkg.go.dev/cmd/gofmt)

# Grafana Dashboard Reporter

This Grafana plugin app can create PDF reports of a given dashboard using headless `chromium`
and [`grafana-image-renderer`](https://github.com/grafana/grafana-image-renderer).

![Sample report](https://github.com/cloudeteer/grafana-pdf-report-app/blob/main/docs/pngs/sample_report.png)

This plugin is based on the original work
[grafana-reporter](https://github.com/IzakMarais/reporter) and 
[grafana-dashboard-reporter-app](https://github.com/mahendrapaipuri/grafana-dashboard-reporter-app)

The core of the plugin is heavily inspired from the above stated work with some
improvements and modernization.

- The current plugin uses HTML templates and headless chromium to generate reports
  instead of LaTeX. `grafana-image-renderer` is a prerequisite for both current and
  original plugins.

- The current plugin app exposes the reporter as a custom API end point of Grafana instance without
  needing to run the [grafana-reporter](https://github.com/IzakMarais/reporter)
  as a separate web service. The advantage of the plugin approach is the authenticated
  access to the reporter app is guaranteed by Grafana auth.

- The plugin is capable of including all the repeated rows and/or panels in the
  generated report.

- The plugin can be configured by Admins and users either from
  [Configuration Page](./src/img/light.png) or query parameters to the report API.


A Grafana plugin to create PDF reports from dashboard panels. This app has been
heavily inspired from the original work [grafana-reporter](https://github.com/IzakMarais/reporter).
The core backend follows very closely to the original work. Instead of using LaTeX
to generate reports, the current plugin generates it from HTML templates using headless
chromium similar the reporting app in Grafana Enterprise offering. The plugin app also integrates
the frontend components to be able to configure the reporter from the Configuration page.

By default the user needs to be authenticated with Grafana to access the service and
must have role `Viewer` on the dashboard that user wants to create a PDF report.

## Prerequisites

This plugin app depends on following:

- Grafana >= 11

- Another Grafana plugin [`grafana-image-renderer`](https://github.com/grafana/grafana-image-renderer) to render panels into PNG files.

- If `grafana-image-renderer` is installed as Grafana plugin, no other external
dependencies are required for the plugin to work. `grafana-image-renderer` ships the
plugin with a standalone instance of `chromium` and the same `chromium` will be used
to render PDF reports. If `grafana-image-renderer` is deployed as a service on a
different host, `chromium` must be installed on the host where Grafana is installed.

> [!IMPORTANT]
> `grafana-image-renderer` advises to install `chromium` to ensure that all the
dependent libraries of the `chromium` are available on the host.

## Installation

### Installation via `grafana-cli`

Grafana Enterprise offers a very similar plugin [reports](https://grafana.com/docs/grafana/latest/dashboards/create-reports/#export-dashboard-as-pdf)
and hence, their plugin policies do not allow to publish the current plugin in their
official catalog.

It is important to note that the current plugin does not offer all the functionalities
offered by Enterprise plugin and it is only relevant if users would like to create a
PDF report of a given dashboard. If users needs more advanced functionalities like
generating and sending reports automatically, they should look into official plugin.

However, it is still possible to install this plugin on on-premise Grafana installations
as an unsigned plugin. The installation procedure is briefed in
[Local installation](#local-installation) section below.

### Local installation

Download the [latest Grafana Dashboard Reporter](https://github.com/cloudeteer/grafana-pdf-report-app/releases/latest).

Create a directory for grafana to access your custom-plugins
_e.g._ `/var/lib/grafana/plugins/cloudeteer-dashboardreporter-app`.

The following shell script downloads and extracts the latest plugin source
code into the the current working directory. Run the following inside your grafana
plugin directory:

```bash
cd /var/lib/grafana/plugins
curl https://raw.githubusercontent.com/cloudeteer/grafana-pdf-report-app/main/scripts/bootstrap-dashboard-reporter-app.sh | bash
```

This will install the latest release of plugin in the `/var/lib/grafana/plugins` folder
and upon Grafana restart, the plugin will be loaded.

If user wants to install the latest nightly release, it is enough to add a environment
variable `NIGHTLY` to `bash`

```bash
cd /var/lib/grafana/plugins
curl https://raw.githubusercontent.com/cloudeteer/grafana-pdf-report-app/main/scripts/bootstrap-dashboard-reporter-app.sh | NIGHTLY=1 bash
```

> [!IMPORTANT]
> The final step is to _whitelist_ the plugin as it is an unsigned plugin and Grafana,
by default, does not load any unsigned plugins even if they are installed. In order to
whitelist the plugin, we need to add following to the Grafana configuration file

```ini
[plugins]
allow_loading_unsigned_plugins = cloudeteer-dashboardreporter-app
```

Once this configuration is added, restart the Grafana server and it should load the
plugin. The loading of plugin can be verified by the following log lines

```bash
logger=plugin.signature.validator t=2024-03-21T11:16:54.738077851Z level=warn msg="Permitting unsigned plugin. This is not recommended" pluginID=cloudeteer-dashboardreporter-app
logger=plugin.loader t=2024-03-21T11:16:54.738166325Z level=info msg="Plugin registered" pluginID=cloudeteer-dashboardreporter-app

```

### Install with Docker-compose

There is a docker compose file provided in the repo. Create a directory `dist` in the
root of the repo and extract the latest version of the plugin app into this folder `dist`.
Once this is done, starting a Grafana server with plugin installed can be done
as follows:

```bash
docker-compose -f docker-compose.yaml up
```

## Configuring the plugin

After successful installation of the plugin, it will be, by default, disabled. We can
enable it in different ways.

- From Grafana UI, navigating to `Apps > Dashboard Reporter App > Configuration` will
show [this page](https://github.com/cloudeteer/grafana-pdf-report-app/blob/main/src/img/light.png)
and plugin can be enabled there. The configuration page can also be
accessed by URL `<Grafana URL>/plugins/cloudeteer-dashboardreporter-app`.

> [!NOTE]
> The warning about `Invalid plugin signature` is not fatal and it is simply saying
that plugin has not been signed by Grafana Labs.

- By using [Grafana Provisioning](https://grafana.com/docs/grafana/latest/administration/provisioning/).
An example provision config is provided in the [repo](https://github.com/cloudeteer/grafana-pdf-report-app/blob/main/provisioning/plugins/app.yaml)
and it can be installed at `/etc/grafana/provisioning/plugins/reporter.yml`. After installing
this YAML file, by restarting Grafana server, the plugin will be enabled with config
settings used in the `reporter.yml` file.

Grafana Provisioning is a programmatic way of configuring the plugin app. Some of the
configuration settings can be set from the environment variables too. Note that any
configured environment variable takes precedence over configuration file settings. Thus,
the plugin app can be configured at install time using either provisioning through YAML
file or using environment variables or mix of both. It is possible to modify these
settings at the runtime using Grafana UI.

To resume, the configuration settings can be set in the following ways:

- Using provisioning through a YAML file at install time
- Using environment variables set on Grafana server at install time
- Using Grafana UI at runtime

The configuration options set in the above stated methods are applied `Org` wide
in Grafana acting as baseline configuration for the plugin. Hence, these settings can
only be changed by a user with a `Admin` role using Grafana UI.

Different configuration settings are explained below. As each configuration option can
be set with different sources, the name of the option in each source is identified as
well. `file` stands for provisioning through YAML file, `env` stands for environment
variable and `ui` stands for name in Grafana UI. When a source is emitted, it means
that it is not possible to set that configuration option using that specific source.

### Authentication settings

This config section allows to configure authentication related settings.

- `file:saToken; ui:Service Account Token`: A service account token that will be used
   to generate reports _via_ API requests. More details on how to use it is briefed in
  [Using Grafana API](#using-grafana-api) section.

### Report settings

This config section allows to configure report related settings.

- `file:theme; env:GF_REPORTER_PLUGIN_REPORT_THEME; ui: Theme`: Theme of the panels in
  the report.

- `file:layout; env:GF_REPORTER_PLUGIN_REPORT_LAYOUT; ui:Layout`: Layout of the report.
  Using grid layout renders the report as it is rendered in the browser. A simple
  layout will render the report with one panel per row. Available options: `simple`
  and `grid`.

- `file:orientation; env:GF_REPORTER_PLUGIN_REPORT_ORIENTATION; ui:Orientation`: Orientation
  of the report. Available options: `portrait` and `landscape`.

- `file:dashboardMode; env:GF_REPORTER_PLUGIN_REPORT_DASHBOARD_MODE; ui:Dashboard Mode`:
  Whether to render default dashboard or full dashboard. In default mode, collapsed rows
  are ignored and only visible panels are included in the report. Whereas in full mode,
  rows are un collapsed and all the panels are included in the report. Available options:
  `default` and `full`.

- `file:timeZone; env:GF_REPORTER_PLUGIN_REPORT_TIMEZONE; ui:Time Zone`: The time zone
  that will be used in the report. It has to conform to the
  [IANA format](https://www.iana.org/time-zones). By default, local Grafana server's
  time zone will be used.

- `file:logo; env: GF_REPORTER_PLUGIN_REPORT_LOGO; ui:Branding Logo`: This parameter
  takes a base64 encoded image that will be included in the footer of each page in the
  report. Typically, operators can include their organization logos to have "customized"
  reports. Images of format PNG and JPG are accepted. **There is no need to add the base64 header**.
  Based on the content, Mime type will be detected and appropriate header will be added.

### Additional settings

The following configuration settings allow more control over plugin's functionality.

- `file:appUrl; env: GF_REPORTER_PLUGIN_APP_URL; ui: Grafana Hostname`: The URL at which
  Grafana is running. By default, `http://localhost:3000` is used which should work for
  most of the deployments.

- `file:skipTlsCheck; env: GF_REPORTER_PLUGIN_SKIP_TLS_CHECK; ui: Skip TLS Verification`:
  If Grafana instance is configured to use TLS with self signed certificates
  set this parameter to `true` to skip TLS certificate check.

- `file:remoteChromeUrl; env: GF_REPORTER_PLUGIN_REMOTE_CHROME_URL; ui: Remote Chrome URL`:
  A URL of a running remote chrome instance which will be used in report generation. Grafana
  running on k8s can opt to use this option when installing `chromium` inside Grafana
  container is not desired. An example [docker-compose file](https://github.com/cloudeteer/grafana-pdf-report-app/blob/main/docker-compose.yaml) shows how to run `chromium` in an `init` container. When remote chrome instance is being used, ensure
  that `appUrl` is accessible to remote chrome.

- `file:maxBrowserWorkers; env: GF_REPORTER_PLUGIN_MAX_BROWSER_WORKERS; ui: Maximum Browser Workers`:
  Maximum number of workers for interacting with chrome browser.

- `file:maxRenderWorkers; env: GF_REPORTER_PLUGIN_MAX_RENDER_WORKERS; ui: Maximum Render Workers`:
  Maximum number of workers for generating panel PNGs.

> [!NOTE]
> Starting from `v1.4.0`, config parameter `dataPath` is not needed anymore as the plugin
will get the Grafana's data path based on its own executable path. If the existing provisioned
configs have this parameter set, it will be ignored while loading the plugin's configuration.

#### Overriding global report settings

Although configuration settings can only be modified by users with `Admin` role for whole `Org`
of Grafana, it is possible to override the global defaults for a particular report
by using query parameters. It is enough to add query parameters to dashboard report URL
to set these values. Currently, the supported query parameters are:

- Query field for theme is `theme` and it takes either `light` or `dark` as value.
  Example is `<grafanaAppUrl>/api/plugins/cloudeteer-dashboardreporter-app/resources/report?dashUid=<UID of dashboard>&theme=dark`

- Query field for layout is `layout` and it takes either `simple` or `grid` as value.
  Example is `<grafanaAppUrl>/api/plugins/cloudeteer-dashboardreporter-app/resources/report?dashUid=<UID of dashboard>&layout=grid`

- Query field for orientation is `orientation` and it takes either `portrait` or `landscape`
  as value. Example is `<grafanaAppUrl>/api/plugins/cloudeteer-dashboardreporter-app/resources/report?dashUid=<UID of dashboard>&orientation=landscape`

- Query field for dashboard mode is `dashboardMode` and it takes either `default` or `full`
  as value. Example is `<grafanaAppUrl>/api/plugins/cloudeteer-dashboardreporter-app/resources/report?dashUid=<UID of dashboard>&dashboardMode=full`

- Query field for dashboard mode is `timeZone` and it takes a value in [IANA format](https://www.iana.org/time-zones)
  as value. **Note** that it should be encoded to escape URL specific characters. For example
  to use `America/New_York` query parameter should be
  `<grafanaAppUrl>/api/plugins/cloudeteer-dashboardreporter-app/resources/report?dashUid=<UID of dashboard>&timeZone=America%2FNew_York`

Besides there are two special query parameters available namely:

- `includePanelID`: This can be used to include only panels with IDs set in the query in
  the generated report. An example can be
  `<grafanaAppUrl>/api/plugins/cloudeteer-dashboardreporter-app/resources/report?dashUid=<UID of dashboard>&includePanelID=1&includePanelID=5&includePanelID=8`.
  This request will only include the panels `1`, `5` and `8` in the report and ignoring the rest.
  When `grid` layout is used with `includePanelID`, the report layout will leave the gaps
  in the place of panels that are not included in the report.

- `excludePanelID`: This can be used to exclude any unwanted panels in
  the generated report. An example can be
  `<grafanaAppUrl>/api/plugins/cloudeteer-dashboardreporter-app/resources/report?dashUid=<UID of dashboard>&excludePanelID=2&excludePanelID=7`.
  This request will only exclude panels `2`, and `7` in the report and including the rest.
  When `grid` layout is used with `excludePanelID`, the report layout will leave the gaps
  in the place of panels that are excluded in the report.

> [!NOTE]
> If a given panel ID is set in both `includePanelID` and `excludePanelID` query parameter,
  it will be **included** in the report. Query parameter `includePanelID` has more
  precedence over `excludePanelID`.

### Grafana API Token

The plugin needs to make API requests to Grafana to fetch resources like dashboard models,
panels, _etc._  Depending on the Grafana version the operators need to perform some
extra configuration to get an API token from Grafana.

- `Grafana <= 10.4.3`: Until Grafana 10.4.3, Grafana was forwarding the user cookies to
  plugin apps and the plugin will use the same user cookie to make API requests to Grafana.
  Thus, if `Grafana <= 10.4.3` is being used, there is no need to provide any API token
  to the the plugin.

- `Grafana > 10.4.3`: For these Grafana deployments, the plugin needs an API token from
  Grafana to make API requests to Grafana. This can be done automatically by enabling
  feature flag `externalServiceAccounts`, which will create a service account and
  provision a service account token automatically for the plugin. To enable this feature,
  it is necessary to set `enable = externalServiceAccounts` in `feature_toggles` section
  of Grafana configuration.

> [!NOTE]
> If the operators do not wish or cannot use `externalServiceAccounts` feature flag on
their Grafana deployment, it is possible to manually create an API token and set it in
the [plugin configuration options](#authentication-settings).

## Using plugin

### Using Grafana web UI

The prerequisite is the user must have at least `Viewer` role on the dashboard that they
want to create a PDF report. After the user authenticates with Grafana, creating a
dashboard report is done by visiting the following end point

```bash
<grafanaAppUrl>/api/plugins/cloudeteer-dashboardreporter-app/resources/report?dashUid=<UID of dashboard>
```

In addition to `dashUid` query parameter, it is possible to pass time range query
parameters `from`, `to` and also dashboard variables that have `var-` prefix. This
permits to integrate the dashboard reporter app into Dashboard links.

The layout and orientation options can be passed by query parameters which will override
the global values set by admins in the plugin configuration. `layout` will take either
`simple` or `grid` as query parameter and `orientation` will take `portrait` or
`landscape` as parameters.

Following steps will configure a dashboard link to create PDF report for that dashboard

- Go to Settings of Dashboard
- Go to Links in the side bar and click on `Add Dashboard Link`
- Use Report for `Title` field, set `Type` to `Link`
- Now set `URL` to `<grafanaAppUrl>/api/plugins/cloudeteer-dashboardreporter-app/resources/report?dashUid=<UID of dashboard>`
- Set `Tooltip` to `Create a PDF report` and set `Icon` to `doc`
- By checking `Include current time range` and `Include current template variables values`,
  time range and dashboard variables will be added to query parameters while creating
  PDF report.

Now there should be link in the right top corner of dashboard named `Report` and clicking
this link will create a new PDF report of the dashboard.

### Using Grafana API

The plugin can generate reports programmatically using Grafana API by using
[Grafana service accounts](https://grafana.com/docs/grafana/latest/administration/service-accounts/).

Once a service account is created with appropriate permissions by following
[Grafana docs](https://grafana.com/docs/grafana/latest/administration/service-accounts/#to-create-a-service-account),
generate an [API token](https://grafana.com/docs/grafana/latest/administration/service-accounts/#add-a-token-to-a-service-account-in-grafana)
from the service account. If `externalServiceAccounts` feature flag is not enabled,
either the same or another API token must be added to the
[plugin configuration](#authentication-settings) as well. Once the token has been
generated and configured in the plugin, reports can be created using

```bash
curl --output=report.pdf -H "Authorization: Bearer <supersecrettoken>" "https://example.grafana.com/api/plugins/cloudeteer-dashboardreporter-app/resources/report?dashUid=<UID of dashboard>"
```

The above example shows on how to generate report using `curl` but this can be done with
any HTTP client of your favorite programming language.

## Security

### `Grafana <= 10.4.3`

When reports are generated from browser, there is minimal to no security risks as the
plugin forward the current Grafana cookie in the request to make API requests to other
Grafana resources. This ensures that user will not be able to generate reports for
the dashboards that contains data sources that they do not have permissions to query. The
plugin _always_ prioritizes the cookie for authentication when found. Disabling basic auth
for Grafana will force the users to generate reports from browser, thus forcing them to
use cookie for authentication.

### `Grafana > 10.4.3`

Starting from `Grafana 10.4.4`, user cookies are not forwarded to the plugin apps anymore.
When the user cookie is not found and the plugin needs a manually configured service account
token or `externalServiceAccounts` feature must be enabled (for Grafana >= 10.3.0). If the
configured service account token has broader permissions than the user making the request,
the user _may_ generate reports of dashboards on the data sources that they do not
have permissions to.

## Examples

Here are the example reports that are generated out of the test dashboards

- [Report with portrait orientation, simple layout and full dashboard mode](https://github.com/cloudeteer/grafana-pdf-report-app/blob/main/docs/reports/report_portrait_simple_full.pdf)
- [Report with landscape orientation, simple layout and full dashboard mode](https://github.com/cloudeteer/grafana-pdf-report-app/blob/main/docs/reports/report_landscape_simple_full.pdf)
- [Report with portrait orientation, grid layout and full dashboard mode](https://github.com/cloudeteer/grafana-pdf-report-app/blob/main/docs/reports/report_portrait_grid_full.pdf)
- [Report with landscape orientation, grid layout and full dashboard mode](https://github.com/cloudeteer/grafana-pdf-report-app/blob/main/docs/reports/report_landscape_grid_full.pdf)
- [Report with portrait orientation, grid layout and default dashboard mode](https://github.com/cloudeteer/grafana-pdf-report-app/blob/main/docs/reports/report_portrait_grid_default.pdf)

## Troubleshooting

- When TLS is enabled on Grafana server, `grafana-image-renderer` tends to throw
certificate errors even when the TLS certificates are signed by well-known CA. Typical
error messages will be as follows:

  ```bash
  logger=plugin.grafana-image-renderer t=2024-05-09T10:46:00.117454724+02:00 level=error msg="Browser request failed" url="https://localhost/d-solo/f5a26bea-adf2-4f2c-8522-79159ba26c0f/_?from=now-24h&height=500&panelId=6&theme=light&to=now&width=1000&render=1" method=GET failure=net::ERR_CERT_COMMON_NAME_INVALID
  logger=plugin.grafana-image-renderer t=2024-05-09T10:46:00.118784778+02:00 level=error msg="Error while trying to prepare page for screenshot" url="https://localhost:443/d-solo/f5a26bea-adf2-4f2c-8522-79159ba26c0f/_?from=now-24h&height=500&panelId=6&theme=light&to=now&width=1000&render=1" err="Error: net::ERR_CERT_COMMON_NAME_INVALID"
  ```

  To solve this issue set environment variables `GF_RENDERER_PLUGIN_IGNORE_HTTPS_ERRORS=true`
  and `IGNORE_HTTPS_ERRORS=true` for the `grafana-image-renderer` service.

- If `chromium` fails to run, it suggests that there are missing dependent libraries on
the host. In that case, we advise to install `chromium` on the machine which will
install all the dependent libraries.

## Development

See [DEVELOPMENT.md](https://github.com/cloudeteer/grafana-pdf-report-app/blob/main/DEVELOPMENT.md)
