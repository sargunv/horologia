# Releasing

Horologia releases are published manually from GitHub Actions.

## What the workflow publishes

- A Git tag using CalVer in the form `v0.YYYYMMDD.x`
- A GitHub Release containing `horo` CLI archives from GoReleaser
- A multi-platform container image in GitHub Container Registry

Mobile artifacts are intentionally excluded for now.

## Versioning

The release workflow computes the version automatically in UTC:

- `YYYYMMDD` is the current UTC date on the GitHub Actions runner
- `x` starts at `0`
- `x` increments for additional releases created on the same UTC date

Examples:

- `v0.20260413.0`
- `v0.20260413.1`

## Running a release

1. Open the `Release` workflow in GitHub Actions.
2. Use `Run workflow` on the branch or commit you want to publish.
3. Wait for the workflow to finish.

The workflow:

1. Checks out the selected ref with full history and tags.
2. Runs `mise run ci`.
3. Computes the next CalVer tag.
4. Builds and publishes the Docker image to `ghcr.io/<owner>/<repo>`.
5. Pushes the release tag.
6. Runs GoReleaser to create the GitHub Release and upload CLI archives.

## Release outputs

Docker image tags:

- `ghcr.io/<owner>/<repo>:v0.YYYYMMDD.x`
- `ghcr.io/<owner>/<repo>:latest`

CLI release assets:

- `horologia_<version>_Darwin_x86_64.tar.gz`
- `horologia_<version>_Darwin_arm64.tar.gz`
- `horologia_<version>_Linux_x86_64.tar.gz`
- `horologia_<version>_Linux_arm64.tar.gz`
- `horologia_<version>_Windows_x86_64.zip`
- `horologia_<version>_Windows_arm64.zip`
