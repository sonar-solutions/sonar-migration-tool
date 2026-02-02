# SonarQube Migration Tool

A CLI tool for migrating SonarQube Server configurations to SonarQube Cloud. Extract data from one or more SonarQube Server instances, map organizations, and migrate quality profiles, quality gates, groups, permissions, and projects to your cloud environment.

## Migration Workflow

```
┌─────────────┐    ┌─────────────┐    ┌──────────────┐    ┌─────────────┐    ┌─────────────┐
│   EXTRACT   │───►│  STRUCTURE  │───►│   MAPPINGS   │───►│   MIGRATE   │───►│  PIPELINES  │
│   Phase 1   │    │   Phase 2   │    │   Phase 3    │    │   Phase 4   │    │   Phase 5   │
└─────────────┘    └─────────────┘    └──────────────┘    └─────────────┘    └─────────────┘
      │                  │                   │                  │                  │
      ▼                  ▼                   ▼                  ▼                  ▼
   Pull data         Generate            Create             Push configs       Update CI/CD
   from SQS          org/project         mapping            to SonarQube       pipelines
   via API           CSV files           CSV files          Cloud              (optional)
```

**Phase 1 - Extract**: Pull all configuration data from your SonarQube Server instance(s)\
**Phase 2 - Structure**: Generate organization and project mappings based on DevOps bindings\
**Phase 3 - Mappings**: Create detailed mapping files for gates, profiles, groups, and templates\
**Phase 4 - Migrate**: Push all configurations to SonarQube Cloud\
**Phase 5 - Pipelines**: Automatically update CI/CD pipeline files to use SonarQube Cloud (optional)

---

## Prerequisites

### Required Software

- **Docker** installed and running on your machine

### SonarQube Server Requirements

- Admin user token with the following permissions:
  - Administer System
  - Administer Quality Gates
  - Administer Quality Profiles
  - Browse all projects

### SonarQube Cloud Requirements

- Enterprise license with organizations already created
- Organizations added to the enterprise
- Admin user token with permissions at the enterprise level and all target organizations

### Token Permissions Summary

| Environment | Token Type | Required Permissions |
|-------------|------------|---------------------|
| SonarQube Server | Admin Token | System admin, Quality Gates admin, Quality Profiles admin |
| SonarQube Cloud | User Token | Enterprise admin + Organization admin for all target orgs |

---

## Quick Start

This example walks through a complete migration from a single SonarQube Server instance to SonarQube Cloud.

### Step 1: Create a working directory

```bash
mkdir sonar-migration
cd sonar-migration
```

### Step 2: Extract data from SonarQube Server

```bash
docker run -v ./files:/app/files ghcr.io/sonar-solutions/sonar-reports:latest \
  extract https://your-sonarqube-server.com YOUR_SQS_ADMIN_TOKEN
```

**Expected output**: A timestamped directory under `./files/` containing extracted data in NDJSON format (e.g., `./files/1706745600/`)

**Verify**: Check that `./files/<timestamp>/extract.json` exists and contains your server version

### Step 3: Generate organization structure

```bash
docker run -v ./files:/app/files ghcr.io/sonar-solutions/sonar-reports:latest structure
```

**Expected output**: `organizations.csv` and `projects.csv` in `./files/`

### Step 4: Map your SonarQube Cloud organizations

Open `./files/organizations.csv` and fill in the `sonarcloud_org_key` column with your SonarQube Cloud organization keys:

```csv
alm,server_url,org_name,sonarcloud_org_key
github,https://your-sonarqube-server.com,my-github-org,my-cloud-org-key
```

### Step 5: Generate entity mappings

```bash
docker run -v ./files:/app/files ghcr.io/sonar-solutions/sonar-reports:latest mappings
```

**Expected output**: `gates.csv`, `profiles.csv`, `groups.csv`, `templates.csv`, `portfolios.csv` in `./files/`

### Step 6: Run the migration

```bash
docker run -v ./files:/app/files ghcr.io/sonar-solutions/sonar-reports:latest \
  migrate YOUR_SQC_ENTERPRISE_TOKEN your-enterprise-key
```

**Expected output**: Configurations created in SonarQube Cloud

### Step 7: Verify migration

Log in to your SonarQube Cloud organization and verify:
- Quality Gates were created with correct conditions
- Quality Profiles were restored with rules
- Groups exist with proper permissions
- Projects are linked to repositories

---

## Detailed Workflow

### Phase 1: Extract

Extract configuration data from your SonarQube Server instance via the REST API.

#### Command

```bash
docker run -v ./files:/app/files ghcr.io/sonar-solutions/sonar-reports:latest \
  extract <URL> <TOKEN> [OPTIONS]
```

#### Arguments

| Argument | Description |
|----------|-------------|
| `URL` | Full URL of your SonarQube Server (e.g., `https://sonar.example.com`) |
| `TOKEN` | Admin user token with system administration permissions |

#### Options

| Option | Description | Default |
|--------|-------------|---------|
| `--export_directory` | Output directory for extracted data | `/app/files/` |
| `--concurrency` | Maximum concurrent API requests | 25 |
| `--timeout` | Request timeout in seconds | 60 |
| `--extract_id` | Resume a previous extraction | (new extraction) |
| `--extract_type` | Type of extraction to run | (full) |
| `--target_task` | Run only a specific task and its dependencies | (all tasks) |
| `--pem_file_path` | Client certificate PEM file (for mTLS) | - |
| `--key_file_path` | Client certificate key file (for mTLS) | - |
| `--cert_password` | Password for client certificate | - |

#### What Gets Extracted

The extraction runs 70 tasks organized by dependencies:

| Category | Data Extracted |
|----------|----------------|
| **Projects** | Project list, settings, bindings, measures, tags, links |
| **Branches** | Branch information, pull request data |
| **Quality Gates** | Gate definitions, conditions, user/group permissions |
| **Quality Profiles** | Profiles, rules, backups, inheritance, permissions |
| **Rules** | Rule definitions, plugin rules, custom rules |
| **Permissions** | Groups, users, project permissions |
| **Templates** | Permission templates, default templates |
| **Portfolios** | Portfolio definitions, project assignments |
| **Applications** | Application definitions |
| **Server** | Server info, settings, webhooks, plugins |

#### Output Files

```
files/
└── <extract_id>/           # Unix timestamp (e.g., 1706745600)
    ├── extract.json        # Metadata: version, edition, URL
    ├── requests.log        # HTTP request/response log
    └── <task_name>/        # One folder per task
        └── results.1.jsonl # NDJSON output files
```

#### Resuming a Failed Extraction

If extraction fails partway through, resume using the extract ID:

```bash
docker run -v ./files:/app/files ghcr.io/sonar-solutions/sonar-reports:latest \
  extract https://your-server.com YOUR_TOKEN --extract_id 1706745600
```

---

### Phase 2: Structure

Generate organization and project mappings based on DevOps platform bindings.

#### Command

```bash
docker run -v ./files:/app/files ghcr.io/sonar-solutions/sonar-reports:latest \
  structure [OPTIONS]
```

#### Options

| Option | Description | Default |
|--------|-------------|---------|
| `--export_directory` | Directory containing extracted data | `/app/files/` |

#### What It Does

1. Reads project bindings from the extraction
2. Groups projects by DevOps platform (GitHub, GitLab, Azure DevOps, Bitbucket)
3. Identifies unique organizations based on repository ownership
4. Creates mapping files for user review

#### Output Files

**organizations.csv**

| Column | Description | User Action |
|--------|-------------|-------------|
| `alm` | DevOps platform (github, gitlab, azure, bitbucket) | Read-only |
| `server_url` | Original SonarQube Server URL | Read-only |
| `org_name` | DevOps organization/namespace | Read-only |
| `sonarcloud_org_key` | Target SonarQube Cloud org key | **Fill this in** |

**projects.csv**

| Column | Description |
|--------|-------------|
| `key` | Original project key |
| `name` | Project name |
| `server_url` | Source server URL |
| `repository` | Linked repository |
| `sonarqube_org_key` | Mapped organization key |

#### Required Manual Step

After running `structure`, you must edit `organizations.csv` and fill in the `sonarcloud_org_key` column with your SonarQube Cloud organization keys before proceeding to the next phase.

---

### Phase 3: Mappings

Create detailed mapping files for all entities that will be migrated.

#### Command

```bash
docker run -v ./files:/app/files ghcr.io/sonar-solutions/sonar-reports:latest \
  mappings [OPTIONS]
```

#### Options

| Option | Description | Default |
|--------|-------------|---------|
| `--export_directory` | Directory containing extracted data | `/app/files/` |

#### Output Files

**gates.csv** - Quality Gate mappings

| Column | Description |
|--------|-------------|
| `name` | Gate name |
| `server_url` | Source server |
| `sonarcloud_org_key` | Target organization |
| `is_default` | Default gate flag |
| `conditions` | Gate conditions (JSON) |

**profiles.csv** - Quality Profile mappings

| Column | Description |
|--------|-------------|
| `name` | Profile name |
| `language` | Programming language |
| `server_url` | Source server |
| `sonarcloud_org_key` | Target organization |
| `is_default` | Default profile flag |
| `parent_name` | Parent profile for inheritance |

**groups.csv** - Group mappings

| Column | Description |
|--------|-------------|
| `name` | Group name |
| `description` | Group description |
| `server_url` | Source server |
| `sonarcloud_org_key` | Target organization |

**templates.csv** - Permission Template mappings

| Column | Description |
|--------|-------------|
| `name` | Template name |
| `server_url` | Source server |
| `sonarcloud_org_key` | Target organization |
| `is_default` | Default template flag |

**portfolios.csv** - Portfolio mappings

| Column | Description |
|--------|-------------|
| `key` | Portfolio key |
| `name` | Portfolio name |
| `server_url` | Source server |
| `sonarcloud_org_key` | Target organization |

---

### Phase 4: Migrate

Execute the migration by pushing configurations to SonarQube Cloud.

#### Command

```bash
docker run -v ./files:/app/files ghcr.io/sonar-solutions/sonar-reports:latest \
  migrate <TOKEN> <ENTERPRISE_KEY> [OPTIONS]
```

#### Arguments

| Argument | Description |
|----------|-------------|
| `TOKEN` | SonarQube Cloud token with enterprise and organization admin permissions |
| `ENTERPRISE_KEY` | Key of your SonarQube Cloud enterprise |

#### Options

| Option | Description | Default |
|--------|-------------|---------|
| `--url` | SonarQube Cloud URL | `https://sonarcloud.io/` |
| `--edition` | SonarQube Cloud license edition | - |
| `--export_directory` | Directory containing mapping files | `/app/files/` |
| `--concurrency` | Maximum concurrent API requests | 25 |
| `--run_id` | Resume a previous migration | (new migration) |
| `--target_task` | Run only a specific task and its dependencies | (all tasks) |

#### Migration Tasks

The migration executes 44 tasks in dependency order:

| Category | What Gets Created |
|----------|-------------------|
| **Groups** | User groups with descriptions |
| **Permissions** | Organization-level group permissions |
| **Templates** | Permission templates with user/group assignments |
| **Quality Gates** | Gates with conditions, default assignments |
| **Quality Profiles** | Profiles restored from backup, inheritance set, defaults configured |
| **Projects** | Projects created/linked, gates assigned, profiles assigned, tags set |
| **Portfolios** | Portfolios created with project assignments |

#### Resuming a Failed Migration

If migration fails partway through, resume using the run ID:

```bash
docker run -v ./files:/app/files ghcr.io/sonar-solutions/sonar-reports:latest \
  migrate YOUR_TOKEN your-enterprise-key --run_id 1706745600
```

The tool tracks completed tasks and resumes from the last completed task.

---

### Phase 5: Pipelines (Optional)

Automatically update CI/CD pipeline configurations to use SonarQube Cloud.

#### Command

```bash
docker run -v ./files:/app/files ghcr.io/sonar-solutions/sonar-reports:latest \
  pipelines <SECRETS_FILE> <SONAR_TOKEN> <SONAR_URL> [OPTIONS]
```

#### Arguments

| Argument | Description |
|----------|-------------|
| `SECRETS_FILE` | Path to JSON file containing org-key to DevOps-token mappings (relative to `--input_directory`) |
| `SONAR_TOKEN` | SonarQube Cloud token to set as organization secret |
| `SONAR_URL` | SonarQube Cloud URL to set as organization secret |

#### Options

| Option | Description | Default |
|--------|-------------|---------|
| `--input_directory` | Directory containing migration files (must include `generateOrganizationMappings` task output) | `/app/files/` |
| `--output_directory` | Directory to place pipeline update output files | Same as `--input_directory` |

#### What It Does

1. Creates organization secrets (`SONARQUBE_CLOUD_TOKEN`, `SONARQUBE_CLOUD_URL`)
2. Fetches pipeline files from repositories
3. Parses and updates YAML configurations
4. Updates scanner configuration files
5. Creates pull requests with changes

#### Supported Platforms

- GitHub (primary)
- GitLab (extensible)
- Azure DevOps (extensible)
- Bitbucket (extensible)

#### Supported Scanners

- SonarQube CLI Scanner
- Maven (`pom.xml`)
- Gradle (`build.gradle`)
- .NET Scanner

---

## Additional Commands

### Report

Generate markdown reports based on extracted data.

```bash
docker run -v ./files:/app/files ghcr.io/sonar-solutions/sonar-reports:latest \
  report [OPTIONS]
```

#### Options

| Option | Description | Default |
|--------|-------------|---------|
| `--export_directory` | Directory containing extracted data | `/app/files/` |
| `--report_type` | Type of report to generate (`migration` or `maturity`) | `migration` |
| `--filename` | Custom filename for the generated report (without extension) | Same as `--report_type` |

Report types include migration readiness and maturity assessments.

### Reset

Reset a SonarQube Cloud enterprise to its original state.

```bash
docker run -v ./files:/app/files ghcr.io/sonar-solutions/sonar-reports:latest \
  reset <TOKEN> <ENTERPRISE_KEY> [OPTIONS]
```

**Warning**: This deletes everything in every organization within the enterprise.

#### Arguments

| Argument | Description |
|----------|-------------|
| `TOKEN` | SonarQube Cloud token with enterprise and organization admin permissions |
| `ENTERPRISE_KEY` | Key of the SonarQube Cloud enterprise that will be reset |

#### Options

| Option | Description | Default |
|--------|-------------|---------|
| `--url` | SonarQube Cloud URL | `https://sonarcloud.io/` |
| `--edition` | SonarQube Cloud license edition | `enterprise` |
| `--concurrency` | Maximum concurrent API requests | `25` |
| `--export_directory` | Directory for interim files | `/app/files/` |

---

## Output Files Reference

### Extraction Output

| File | Purpose |
|------|---------|
| `<extract_id>/extract.json` | Extraction metadata (version, edition, URL) |
| `<extract_id>/requests.log` | HTTP request/response log for debugging |
| `<extract_id>/<task>/results.*.jsonl` | Extracted data in NDJSON format |

### Structure Output

| File | Purpose | Requires Editing |
|------|---------|------------------|
| `organizations.csv` | Organization mappings | Yes - fill in `sonarcloud_org_key` |
| `projects.csv` | Project mappings | No |

### Mappings Output

| File | Purpose | Requires Editing |
|------|---------|------------------|
| `gates.csv` | Quality Gate mappings | No |
| `profiles.csv` | Quality Profile mappings | No |
| `groups.csv` | Group mappings | No |
| `templates.csv` | Permission Template mappings | No |
| `portfolios.csv` | Portfolio mappings | No |

---

## Troubleshooting

### Common Errors

#### "Token does not have sufficient permissions"

**Cause**: The admin token lacks required permissions.

**Solution**: Ensure your SonarQube Server token has:
- Administer System permission
- Administer Quality Gates permission
- Administer Quality Profiles permission

#### "Organization not found"

**Cause**: The `sonarcloud_org_key` in `organizations.csv` doesn't match an existing organization.

**Solution**:
1. Verify the organization exists in SonarQube Cloud
2. Check for typos in the organization key
3. Ensure the organization is part of your enterprise

#### "Request timeout"

**Cause**: Large datasets or slow network connection.

**Solution**: Increase the timeout value:
```bash
docker run -v ./files:/app/files ghcr.io/sonar-solutions/sonar-reports:latest \
  extract https://your-server.com YOUR_TOKEN --timeout 120
```

#### "Connection refused" or SSL errors

**Cause**: Network issues or certificate problems.

**Solutions**:
- Verify the URL is accessible from your machine
- For self-signed certificates, use client certificate options:
  ```bash
  docker run -v ./files:/app/files -v ./certs:/app/certs ghcr.io/sonar-solutions/sonar-reports:latest \
    extract https://your-server.com YOUR_TOKEN \
    --pem_file_path /app/certs/client.pem \
    --key_file_path /app/certs/client.key
  ```

#### Migration task fails midway

**Cause**: API error, rate limiting, or network interruption.

**Solution**: Resume the migration using the run ID shown in the output:
```bash
docker run -v ./files:/app/files ghcr.io/sonar-solutions/sonar-reports:latest \
  migrate YOUR_TOKEN your-enterprise-key --run_id <FAILED_RUN_ID>
```

### Finding Logs

- **Request logs**: `./files/<extract_id>/requests.log` contains all HTTP requests and responses
- **Docker logs**: `docker logs <container_id>` shows CLI output

### Reducing Memory Usage

For large SonarQube instances (50,000+ projects), reduce concurrency:

```bash
docker run -v ./files:/app/files ghcr.io/sonar-solutions/sonar-reports:latest \
  extract https://your-server.com YOUR_TOKEN --concurrency 10
```

---

## Docker Deployment

### Basic Usage

Mount a local directory to persist data between commands:

```bash
docker run -v /path/to/local/files:/app/files ghcr.io/sonar-solutions/sonar-reports:latest <command>
```

### With Client Certificates

Mount your certificate files:

```bash
docker run \
  -v ./files:/app/files \
  -v ./certs:/app/certs \
  ghcr.io/sonar-solutions/sonar-reports:latest \
  extract https://your-server.com YOUR_TOKEN \
  --pem_file_path /app/certs/client.pem \
  --key_file_path /app/certs/client.key \
  --cert_password YOUR_CERT_PASSWORD
```

### Docker Compose

```yaml
version: '3.8'
services:
  sonar-migration:
    image: ghcr.io/sonar-solutions/sonar-reports:latest
    volumes:
      - ./files:/app/files
      - ./certs:/app/certs  # Optional: for client certificates
```

Run commands with:

```bash
docker compose run sonar-migration extract https://your-server.com YOUR_TOKEN
docker compose run sonar-migration structure
docker compose run sonar-migration mappings
docker compose run sonar-migration migrate YOUR_SQC_TOKEN your-enterprise-key
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| (none currently) | The tool uses command-line arguments | - |

---

## Multi-Server Migration

To migrate from multiple SonarQube Server instances:

### Step 1: Extract from each server

```bash
# Server 1
docker run -v ./files:/app/files ghcr.io/sonar-solutions/sonar-reports:latest \
  extract https://sonar1.example.com TOKEN1

# Server 2
docker run -v ./files:/app/files ghcr.io/sonar-solutions/sonar-reports:latest \
  extract https://sonar2.example.com TOKEN2
```

### Step 2: Run structure (aggregates all extractions)

```bash
docker run -v ./files:/app/files ghcr.io/sonar-solutions/sonar-reports:latest structure
```

### Step 3: Edit organizations.csv

Map organizations from all servers to your cloud organizations.

### Step 4: Continue with mappings and migrate

The remaining steps work the same as a single-server migration.

---

## Best Practices

1. **Create a dedicated migration user** in SonarQube Cloud with enterprise admin permissions
2. **Test with a subset first** using `--target_task` to migrate specific entities
3. **Review CSV mappings** before running migrate to ensure accuracy
4. **Monitor the requests.log** for API errors during extraction and migration
5. **Use lower concurrency** for large instances to avoid overwhelming the server
6. **Back up existing configurations** before running reset on your cloud enterprise

---

## Version Support

The tool automatically detects your SonarQube Server version and uses the appropriate API endpoints:

- Supports SonarQube Server 6.3+
- Authentication adapts based on version:
  - Server < 10: Basic authentication
  - Server >= 10: Bearer token authentication
- Edition-aware: Handles Community, Developer, Enterprise, and Data Center editions

---

## License

See [LICENSE](LICENSE) for details.
