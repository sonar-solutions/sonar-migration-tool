# SonarQube Migration Tool

Migrates SonarQube Server configurations to SonarQube Cloud — quality gates, quality profiles, groups, permissions, projects, and portfolios.

## Migration Workflow

```
┌─────────────┐    ┌─────────────┐    ┌──────────────┐    ┌─────────────┐    ┌─────────────┐
│   EXTRACT   │───►│  STRUCTURE  │───►│   MAPPINGS   │───►│   MIGRATE   │───►│  PIPELINES  │
│   Phase 1   │    │   Phase 2   │    │   Phase 3    │    │   Phase 4   │    │   Phase 5   │
└─────────────┘    └─────────────┘    └──────────────┘    └─────────────┘    └─────────────┘
```

---

## Prerequisites

- **Docker** installed and running
- **SonarQube Server token** with: Administer System, Administer Quality Gates, Administer Quality Profiles, Browse all projects
- **SonarQube Cloud token** with enterprise admin + organization admin permissions for all target orgs
- **SonarQube Cloud enterprise** with target organizations already created

---

## What Gets Migrated

| Migrated | NOT Migrated |
|----------|-------------|
| Projects, Quality Gates, Quality Profiles | Historical analysis data |
| Groups, Permissions, Permission Templates | Issues and their history |
| Portfolios | Source code (re-scan after migration) |

---

## Interactive Wizard (Recommended)

> **Recommended for most users. No scripting required.**

```bash
docker run -it -v ./files:/app/files ghcr.io/sonar-solutions/sonar-reports:latest wizard
```

The wizard guides you through all five phases with prompts. Key features:
- **Resume support** — saves state and picks up from the last completed phase if interrupted
- **mTLS support** — prompts for client certificate details when needed
- **Validation** — confirms all organizations are mapped before migrating

---

## Manual CLI Method

> For scripting, automation, or advanced users. Most users should use the wizard above.

### 1. Extract

```bash
docker run -v ./files:/app/files ghcr.io/sonar-solutions/sonar-reports:latest \
  extract <URL> <TOKEN> [--concurrency 25] [--timeout 60] [--extract_id <id>]
```

- `--extract_id` — resume a previous extraction
- For mTLS: add `--pem_file_path`, `--key_file_path`, `--cert_password`
- For multiple servers: run `extract` once per server; `structure` aggregates all results

### 2. Structure

```bash
docker run -v ./files:/app/files ghcr.io/sonar-solutions/sonar-reports:latest structure
```

Then edit `files/organizations.csv` — fill in `sonarcloud_org_key` for each row before continuing.

### 3. Mappings

```bash
docker run -v ./files:/app/files ghcr.io/sonar-solutions/sonar-reports:latest mappings
```

Outputs `gates.csv`, `profiles.csv`, `groups.csv`, `templates.csv`, `portfolios.csv`.

### 4. Migrate

```bash
docker run -v ./files:/app/files ghcr.io/sonar-solutions/sonar-reports:latest \
  migrate <TOKEN> <ENTERPRISE_KEY> [--run_id <id>] [--skip_profiles]
```

- `--run_id` — resume a failed migration from the last completed task

### 5. Pipelines (Optional)

```bash
docker run -v ./files:/app/files ghcr.io/sonar-solutions/sonar-reports:latest \
  pipelines <SECRETS_FILE> <SONAR_TOKEN> <SONAR_URL>
```

Updates CI/CD pipeline files to point to SonarQube Cloud. Supports GitHub, GitLab, Azure DevOps, Bitbucket; scanners: CLI, Maven, Gradle, .NET.

---

## Additional Commands

**Analysis Report** — parse `requests.log` into a CSV summary of API call outcomes:
```bash
docker run -v ./files:/app/files ghcr.io/sonar-solutions/sonar-reports:latest \
  analysis_report <RUN_ID>
```

**Report** — generate a migration readiness or maturity report:
```bash
docker run -v ./files:/app/files ghcr.io/sonar-solutions/sonar-reports:latest \
  report [--report_type migration|maturity]
```

**Reset** — ⚠️ deletes all content in every org in the enterprise:
```bash
docker run -v ./files:/app/files ghcr.io/sonar-solutions/sonar-reports:latest \
  reset <TOKEN> <ENTERPRISE_KEY>
```

---

## Post-Migration

1. Verify projects appear in SonarQube Cloud and are linked to repositories
2. Verify quality gates and profiles are correct
3. Re-scan all projects — historical data does not transfer
4. If pipelines phase was run, confirm `SONAR_TOKEN` and `SONAR_HOST_URL` are set in your CI/CD platform

---

## Troubleshooting

**Token does not have sufficient permissions** — ensure the SonarQube Server token has Administer System, Administer Quality Gates, and Administer Quality Profiles permissions.

**Organization not found** — verify `sonarcloud_org_key` in `organizations.csv` matches an existing org in your enterprise.

**SSL / connection errors** — mount and pass client certificate files:
```bash
docker run -v ./files:/app/files -v ./certs:/app/certs \
  ghcr.io/sonar-solutions/sonar-reports:latest \
  extract <URL> <TOKEN> \
  --pem_file_path /app/certs/client.pem \
  --key_file_path /app/certs/client.key
```

**Migration fails midway** — re-run the same `migrate` command with `--run_id <id>` (shown in output). Completed tasks are skipped automatically.

**Large instances** — reduce concurrency with `--concurrency 10` and increase timeout with `--timeout 120`.

---

## Best Practices

- Create a dedicated migration user in SonarQube Cloud with enterprise admin permissions
- Test with a subset first using `--target_task` to migrate specific entities
- Review CSV mappings before running migrate
- Monitor `files/<run_id>/requests.log` for API errors

---

## Version Support

Supports SonarQube Server 6.3+. Authentication auto-detects version (Basic auth < 10, Bearer token ≥ 10). Edition-aware: Community, Developer, Enterprise, Data Center.

---

## License

See [LICENSE](LICENSE) for details.
