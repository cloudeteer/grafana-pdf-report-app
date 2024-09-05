
# Release Process for `grafana-pdf-report-app`

This document outlines the steps to follow when releasing a new version of the `grafana-pdf-report-app`.

## Release Steps

1. **Create a Git Tag**
   - Before releasing a new version, create a git tag to mark the release point in the repository. This should follow [semantic versioning](https://semver.org/) (e.g., `v1.0.0`).
   ```bash
   git tag vX.X.X
   git push origin vX.X.X
   ```

2. **Draft a Release on GitHub**
   - Once the git tag is created, a release draft will automatically appear at:
     [https://github.com/cloudeteer/grafana-pdf-report-app/releases](https://github.com/cloudeteer/grafana-pdf-report-app/releases)
   - In draft mode, the release artifacts are **not publicly** accessible.

3. **Edit and Publish the Release**
   - Go to the [GitHub Releases page](https://github.com/cloudeteer/grafana-pdf-report-app/releases).
   - Once ready, publish the release to make the artifacts publicly available.
