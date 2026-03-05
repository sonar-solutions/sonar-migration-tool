# SonarQube to SonarCloud Migration Tool

Migrate your SonarQube Server configurations to SonarQube Cloud — quality gates, quality profiles, groups, permissions, projects, and portfolios.

## What Gets Migrated

| Migrated | NOT Migrated |
|----------|-------------|
| Quality Gates (with conditions) | Historical analysis data |
| Quality Profiles (with rules) | Issues and their history |
| Groups and Permissions | Code coverage history |
| Permission Templates | Security hotspots |
| Projects (settings, tags) | Source code (you need to re-scan) |
| Portfolios | |

---

## Quick Start

The fastest way to migrate: download the executable, create a config file, run one command.

### 1. Download the Executable

Grab the pre-built binary for your platform from the [Releases page](https://github.com/sonar-solutions/sonar-reports/releases):

| Platform | Download |
|----------|----------|
| macOS (Apple Silicon) | `sonar-reports-macos-arm64` |
| macOS (Intel) | `sonar-reports-macos-x86_64` |
| Linux (x86_64) | `sonar-reports-linux-x86_64` |
| Linux (ARM64) | `sonar-reports-linux-arm64` |
| Windows (x86_64) | `sonar-reports-windows-x86_64.exe` |
| Windows (ARM64) | `sonar-reports-windows-arm64.exe` |

Make it runnable (macOS/Linux):

```bash
chmod +x sonar-reports-*
```

### 2. Create a Config File

```bash
cp examples/migration-config.example.json migration-config.json
```

Edit `migration-config.json` with your details:

```json
{
  "sonarqube": {
    "url": "https://your-sonarqube-server.com",
    "token": "YOUR_SONARQUBE_ADMIN_TOKEN"
  },
  "sonarcloud": {
    "url": "https://sonarcloud.io/",
    "token": "YOUR_SONARCLOUD_ADMIN_TOKEN",
    "enterprise_key": "YOUR_ENTERPRISE_KEY",
    "org_key": "YOUR_TARGET_ORG_KEY"
  },
  "settings": {
    "export_directory": "./files",
    "concurrency": 10,
    "timeout": 60
  }
}
```

**You will need:**
- A SonarQube Server admin token (with system admin, quality gate admin, and quality profile admin permissions)
- A SonarCloud admin token (with enterprise-level and organization-level admin permissions)
- Your SonarCloud enterprise key and target organization key

See [docs/SECURITY.md](docs/SECURITY.md) for token handling best practices.

### 3. Run the Migration

```bash
./sonar-reports-macos-arm64 full-migrate migration-config.json
```

Replace the binary name with the one you downloaded. On Windows, use `sonar-reports-windows-x86_64.exe` instead.

The tool will automatically extract data, generate mappings, and migrate everything to SonarCloud.

> **Migrating to multiple organizations?** The `full-migrate` command maps all projects to a single org. For multi-org migrations, use the [step-by-step method](docs/MANUAL-MIGRATION.md) where you can edit `organizations.csv` to assign different target orgs.

---

## Other Ways to Run

| Method | Best For | Guide |
|--------|----------|-------|
| **Docker** | No local installs needed | [docs/DOCKER.md](docs/DOCKER.md) |
| **Python** | `pip install -r requirements.txt && python src/main.py <command>` | [docs/MANUAL-MIGRATION.md](docs/MANUAL-MIGRATION.md) |
| **Interactive Wizard** | Guided step-by-step prompts | See below |
| **Shell Scripts** | Automated/scripted migrations | `scripts/execute_full_migration.sh` or `scripts/execute_migration_with_binary.sh` |
| **Build from Source** | Custom builds | [docs/BUILD.md](docs/BUILD.md) |

---

## Interactive Wizard

For a guided experience with prompts at each step:

```bash
./sonar-reports-macos-arm64 wizard
```

Or with Docker:

```bash
docker run -it -v ./files:/app/files ghcr.io/sonar-solutions/sonar-reports:latest wizard
```

The wizard walks you through extraction, organization mapping, validation, migration, and optional pipeline updates. It saves progress so you can resume if interrupted.

---

## Step-by-Step Manual Migration

If you need more control (e.g., migrating to multiple orgs, extracting from multiple servers, or running specific tasks), see the full manual guide:

**[docs/MANUAL-MIGRATION.md](docs/MANUAL-MIGRATION.md)**

The manual process follows these phases:

```
Extract → Structure → Edit Mappings → Generate Mappings → Migrate → Pipelines (optional)
```

---

## After Migration

1. **Verify** in SonarCloud — check quality gates, profiles, groups, and projects
2. **Re-scan your projects** — historical data is not migrated, so run new scans
3. **Set up CI/CD integration** — configure your DevOps platform (GitHub, GitLab, Azure DevOps, Bitbucket)

---

## Troubleshooting

**Migration failed?** Here are the most common issues:

| Problem | Solution |
|---------|----------|
| Token permission errors | Ensure your SonarQube token has system admin permissions and your SonarCloud token has enterprise admin permissions |
| Organization not found | Check that `sonarcloud_org_key` in `organizations.csv` matches an existing org in your enterprise |
| Timeout errors | Increase `--timeout` (e.g., `120`) or reduce `--concurrency` (e.g., `5`) |
| Docker can't reach localhost | Use `host.docker.internal` instead of `localhost` |
| Migration failed midway | Resume with `--run_id` — the tool tracks completed tasks |

For the full troubleshooting guide, see **[docs/TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md)**.

---

## All Available Commands

| Command | Description |
|---------|-------------|
| `full-migrate` | End-to-end migration from a single config file |
| `wizard` | Interactive guided migration |
| `extract` | Pull data from SonarQube Server |
| `structure` | Generate organization/project mappings |
| `mappings` | Generate entity mapping CSVs |
| `migrate` | Push configurations to SonarCloud |
| `pipelines` | Update CI/CD pipeline files (optional) |
| `report` | Generate migration or maturity reports |
| `analysis_report` | Generate CSV summary of API calls from a migration run |
| `reset` | Reset a SonarCloud enterprise to original state (**destructive**) |

---

## Documentation

| Guide | Description |
|-------|-------------|
| [docs/MANUAL-MIGRATION.md](docs/MANUAL-MIGRATION.md) | Step-by-step migration with binary, Docker, or Python |
| [docs/DOCKER.md](docs/DOCKER.md) | Docker-specific usage guide |
| [docs/CONFIG.md](docs/CONFIG.md) | Configuration file reference |
| [docs/BUILD.md](docs/BUILD.md) | Building executables from source |
| [docs/TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md) | Common errors and solutions |
| [docs/SECURITY.md](docs/SECURITY.md) | Token handling and security best practices |
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | Internal architecture for contributors |

---

## Version Support

- SonarQube Server 6.3+
- Supports Community, Developer, Enterprise, and Data Center editions
- SonarQube Cloud Enterprise and Teams editions

## License

See [LICENSE](LICENSE) for details.
