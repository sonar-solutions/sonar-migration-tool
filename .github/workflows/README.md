# GitHub Actions Workflows

## Active Workflows

### 1. `build.yml` - Test + Release
**Triggers:**
- Push to `main` or `kilo` branches — runs tests and SonarQube scan
- Push of a version tag (`v*`) — runs tests, then builds and publishes cross-platform binaries

**What it does:**
- Runs Go library and migration tool tests with coverage
- Runs SonarQube Cloud analysis
- On tagged releases: builds static binaries for 6 platform/arch combinations and uploads them as GitHub Release assets

**Release binaries produced:**
| Platform | Architecture | Filename |
|----------|-------------|----------|
| Linux    | x64         | `sonar-migration-tool-linux-amd64` |
| Linux    | ARM64       | `sonar-migration-tool-linux-arm64` |
| macOS    | x64         | `sonar-migration-tool-darwin-amd64` |
| macOS    | ARM64       | `sonar-migration-tool-darwin-arm64` |
| Windows  | x64         | `sonar-migration-tool-windows-amd64.exe` |
| Windows  | ARM64       | `sonar-migration-tool-windows-arm64.exe` |

### 2. `test.yml` - Manual Test Run
**Trigger:** Manual dispatch (`workflow_dispatch`)
**Purpose:** On-demand test run with SonarQube Cloud scan
