# SonarQube to SonarCloud Migration Tool

Migrate your SonarQube Server projects to SonarCloud with a single command. No Python or Docker installation required.

## Table of Contents

- [Quick Start](#quick-start)
- [What Gets Migrated](#what-gets-migrated)
- [Alternative Methods](#alternative-methods)
- [Run Commands Manually](#run-commands-manually)
- [Post-Migration Steps](#post-migration-steps)
- [Troubleshooting](#troubleshooting)
- [Advanced Usage](#advanced-usage)
- [Security](#security)

---

## Quick Start

The simplest way to migrate - download the executable, create a config file, and run one command.

### Prerequisites

- **SonarQube Server** with admin token
- **SonarCloud** account with:
  - Enterprise subscription
  - Admin token (enterprise-level permissions)
  - Target organization created and added to your enterprise

### Step 1: Download the Executable

Download the pre-built executable for your platform from the [Releases page](https://github.com/sonar-solutions/sonar-reports/releases):

- **macOS (Apple Silicon)**: `sonar-reports-macos-arm64`
- **macOS (Intel)**: `sonar-reports-macos-x86_64`
- **Linux (x86_64)**: `sonar-reports-linux-x86_64`
- **Windows (x86_64)**: `sonar-reports-windows-x86_64.exe`

Make the executable runnable (macOS/Linux):
```bash
chmod +x sonar-reports-*
```

### Step 2: Create Configuration File

```bash
cp examples/migration-config.example.json migration-config.json
```

Edit `migration-config.json` with your credentials:

```json
{
  "sonarqube": {
    "url": "http://localhost:9000",
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
    "timeout": 60,
    "skip_profiles": false
  }
}
```

> Please note that this assumes that you are migrating all of your projects on your SonarQube Server instance to a single SonarCloud organization. If you need to migrate to multiple organizations, please refer to the [Run Commands Manually](#run-commands-manually) section below, where you can edit the `organizations.csv` file to specify different target organizations for different projects.

### Step 3: Run Migration

Run the executable with your config file:

**macOS/Linux:**
```bash
./sonar-reports-<platform> full-migrate migration-config.json
```

**Windows:**
```cmd
sonar-reports-windows-x86_64.exe full-migrate migration-config.json
```

Replace `<platform>` with your downloaded executable name (e.g., `macos-arm64`, `macos-x86_64`, or `linux-x86_64`).

That's it! The tool automatically:
1. ✅ Extracts all data from SonarQube Server
2. ✅ Generates organization structure
3. ✅ Creates mappings (profiles, gates, groups)
4. ✅ Migrates everything to SonarCloud
5. ✅ Verifies the migration

---

## What Gets Migrated

### ✅ Migrated Items

- **Projects**: Configurations, settings, tags, new code definitions
- **Quality Profiles**: Including custom rules
- **Quality Gates**: Including all conditions
- **Users & Permissions**: Groups, permissions, templates
- **Portfolios**: Structure, projects, permissions

### ⚠️ NOT Migrated

- Historical analysis data
- Issues and their history
- Code coverage history
- Security hotspots
- Source code (you'll need to re-scan)

---

## Alternative Methods

### Option 1: Build the Executable Yourself

If you prefer to build the executable from source:

**macOS/Linux:**
```bash
git clone https://github.com/YOUR_REPO/sonar-reports.git
cd sonar-reports
./scripts/build.sh
```

**Windows:**
```cmd
git clone https://github.com/YOUR_REPO/sonar-reports.git
cd sonar-reports
scripts\build.bat
```

**Docker (Linux x86_64):**
```bash
git clone https://github.com/YOUR_REPO/sonar-reports.git
cd sonar-reports
docker buildx build --platform linux/amd64 -f docker/Dockerfile.linux-build -t sonar-reports-linux-builder --load .
docker run --rm -v "$(pwd)/dist:/output" sonar-reports-linux-builder cp /app/dist/sonar-reports-linux-x86_64 /output/
```

The built executable will be in the `dist/` directory. Then follow steps 2-3 from the Quick Start guide.

### Option 2: Binary with Shell Script

Use a shell script wrapper for the binary:

```bash
./scripts/execute_migration_with_binary.sh migration-config.json
```

This script automatically detects your platform and runs the appropriate binary.

### Option 3: Docker (No Installation)

If you prefer Docker, you can use the automated script for a quick migration:

1. **Configure the script:**
   ```bash
   # Edit scripts/execute_full_migration.sh with your values
   SONARQUBE_URL="http://localhost:9000"
   SONARQUBE_TOKEN="your-token"
   SONARCLOUD_URL="https://sonarcloud.io/"
   SONARCLOUD_TOKEN="your-token"
   SONARCLOUD_ENTERPRISE_KEY="your-enterprise-key"
   SONARCLOUD_ORG_KEY="your-org"
   ```

2. **Run:**
   ```bash
   chmod +x scripts/execute_full_migration.sh
   ./scripts/execute_full_migration.sh
   ```

For step-by-step manual commands with Docker, see [Run Commands Manually](#run-commands-manually).

### Option 4: Python CLI

For maximum control, use the Python CLI directly:

1. **Install dependencies:**
   ```bash
   pip install -r requirements.txt
   ```

2. **Run full migration with config file:**
   ```bash
   python src/main.py full-migrate migration-config.json
   ```

For step-by-step manual commands with Python, see [Run Commands Manually](#run-commands-manually).

### Option 5: Using Config Files with Individual Commands

You can use JSON config files with any command:

```bash
# Extract with config
./dist/sonar-reports-macos-arm64 extract --config extract-config.json

# Migrate with config
./dist/sonar-reports-macos-arm64 migrate --config migrate-config.json
```

See [docs/CONFIG.md](docs/CONFIG.md) for configuration file documentation.

---

## Run Commands Manually

If you prefer to run commands step-by-step for more control, you can execute the migration process in stages using any of these methods.

### Using Executable Binary

**1. Extract data from SonarQube:**
```bash
./sonar-reports-<platform> extract \
  http://localhost:9000 \
  YOUR_SONARQUBE_TOKEN \
  --export_directory=./files \
  --concurrency=10 \
  --timeout=60
```

Optional extract parameters:
- `--extract_type=all` - Extract all data (default)
- `--target_task=getProjects` - Extract specific data only
- `--extract_id=123456` - Resume a previous extract
- `--pem_file_path` - Path to client certificate
- `--key_file_path` - Path to certificate key
- `--cert_password` - Certificate password

**2. Generate organization structure:**
```bash
./sonar-reports-<platform> structure \
  --export_directory=./files
```

**3. Update organizations.csv:**
Edit `files/organizations.csv` and add your SonarCloud organization key to the `sonarcloud_org_key` column.

**4. Generate mappings:**
```bash
./sonar-reports-<platform> mappings \
  --export_directory=./files
```

**5. Migrate to SonarCloud:**
```bash
./sonar-reports-<platform> migrate \
  YOUR_SONARCLOUD_TOKEN \
  YOUR_ENTERPRISE_KEY \
  --url=https://sonarcloud.io/ \
  --export_directory=./files \
  --concurrency=10
```

Optional migrate parameters:
- `--run_id=123456` - Resume a previous migration
- `--target_task=createProjects` - Migrate specific task only
- `--edition=enterprise` - SonarCloud edition (default: enterprise)
- `--skip_profiles` - Skip quality profile migration/provisioning

**6. Generate a migration report (optional):**
```bash
./sonar-reports-<platform> report \
  --export_directory=./files \
  --report_type=migration \
  --filename=migration-report
```

**7. Reset SonarCloud (CAUTION - deletes everything):**
```bash
./sonar-reports-<platform> reset \
  YOUR_SONARCLOUD_TOKEN \
  YOUR_ENTERPRISE_KEY \
  --url=https://sonarcloud.io/ \
  --export_directory=./files
```

Replace `<platform>` with your downloaded executable name (e.g., `macos-arm64`, `macos-x86_64`, `linux-x86_64`, or `windows-x86_64.exe`).

### Using Docker

First, build the Docker image:
```bash
docker build -t sonar-reports:local .
```

**Note:** When running from Docker, use `host.docker.internal` instead of `localhost` to access services on your host machine.

**1. Extract data from SonarQube:**
```bash
docker run --rm \
  -v "$(pwd)/files:/app/files" \
  sonar-reports:local extract \
  http://host.docker.internal:9000 \
  YOUR_SONARQUBE_TOKEN \
  --export_directory=/app/files \
  --concurrency=10 \
  --timeout=60
```

Optional extract parameters:
- `--extract_type=all` - Extract all data (default)
- `--target_task=getProjects` - Extract specific data only
- `--extract_id=123456` - Resume a previous extract
- `--pem_file_path` - Path to client certificate
- `--key_file_path` - Path to certificate key
- `--cert_password` - Certificate password

**2. Generate organization structure:**
```bash
docker run --rm \
  -v "$(pwd)/files:/app/files" \
  sonar-reports:local structure \
  --export_directory=/app/files
```

**3. Update organizations.csv:**
Edit `files/organizations.csv` and add your SonarCloud organization key to the `sonarcloud_org_key` column.

**4. Generate mappings:**
```bash
docker run --rm \
  -v "$(pwd)/files:/app/files" \
  sonar-reports:local mappings \
  --export_directory=/app/files
```

**5. Migrate to SonarCloud:**
```bash
docker run --rm \
  -v "$(pwd)/files:/app/files" \
  sonar-reports:local migrate \
  YOUR_SONARCLOUD_TOKEN \
  YOUR_ENTERPRISE_KEY \
  --url=https://sonarcloud.io/ \
  --export_directory=/app/files \
  --concurrency=10
```

Optional migrate parameters:
- `--run_id=123456` - Resume a previous migration
- `--target_task=createProjects` - Migrate specific task only
- `--edition=enterprise` - SonarCloud edition (default: enterprise)
- `--skip_profiles` - Skip quality profile migration/provisioning

**6. Generate a migration report (optional):**
```bash
docker run --rm \
  -v "$(pwd)/files:/app/files" \
  sonar-reports:local report \
  --export_directory=/app/files \
  --report_type=migration \
  --filename=migration-report
```

**7. Reset SonarCloud (CAUTION - deletes everything):**
```bash
docker run --rm \
  -v "$(pwd)/files:/app/files" \
  sonar-reports:local reset \
  YOUR_SONARCLOUD_TOKEN \
  YOUR_ENTERPRISE_KEY \
  --url=https://sonarcloud.io/ \
  --export_directory=/app/files
```

### Using Python

First, install dependencies:
```bash
pip install -r requirements.txt
```

**1. Extract data from SonarQube:**
```bash
python src/main.py extract \
  http://localhost:9000 \
  YOUR_SONARQUBE_TOKEN \
  --export_directory=./files \
  --concurrency=10 \
  --timeout=60
```

Optional extract parameters:
- `--extract_type=all` - Extract all data (default)
- `--target_task=getProjects` - Extract specific data only
- `--extract_id=123456` - Resume a previous extract
- `--pem_file_path` - Path to client certificate
- `--key_file_path` - Path to certificate key
- `--cert_password` - Certificate password

**2. Generate organization structure:**
```bash
python src/main.py structure \
  --export_directory=./files
```

**3. Update organizations.csv:**
Edit `files/organizations.csv` and add your SonarCloud organization key to the `sonarcloud_org_key` column.

**4. Generate mappings:**
```bash
python src/main.py mappings \
  --export_directory=./files
```

**5. Migrate to SonarCloud:**
```bash
python src/main.py migrate \
  YOUR_SONARCLOUD_TOKEN \
  YOUR_ENTERPRISE_KEY \
  --url=https://sonarcloud.io/ \
  --export_directory=./files \
  --concurrency=10
```

Optional migrate parameters:
- `--run_id=123456` - Resume a previous migration
- `--target_task=createProjects` - Migrate specific task only
- `--edition=enterprise` - SonarCloud edition (default: enterprise)
- `--skip_profiles` - Skip quality profile migration/provisioning

**6. Generate a migration report (optional):**
```bash
python src/main.py report \
  --export_directory=./files \
  --report_type=migration \
  --filename=migration-report
```

**7. Reset SonarCloud (CAUTION - deletes everything):**
```bash
python src/main.py reset \
  YOUR_SONARCLOUD_TOKEN \
  YOUR_ENTERPRISE_KEY \
  --url=https://sonarcloud.io/ \
  --export_directory=./files \
  --concurrency=10
```

---

## Post-Migration Steps

### 1. Verify Projects

Visit your SonarCloud organization:
```
https://sonarcloud.io/organizations/YOUR_ORG/projects
```

Or verify via API:
```bash
curl -H "Authorization: Bearer YOUR_TOKEN" \
  "https://sonarcloud.io/api/projects/search?organization=YOUR_ORG&ps=500" | jq
```

### 2. Check Quality Gates

```
https://sonarcloud.io/organizations/YOUR_ORG/quality_gates
```

### 3. Verify Quality Profiles

```
https://sonarcloud.io/organizations/YOUR_ORG/quality_profiles
```

### 4. Re-scan Your Projects

Historical analysis data is NOT migrated. You need to run new scans:

**Maven:**
```bash
mvn sonar:sonar \
  -Dsonar.projectKey=YOUR_PROJECT_KEY \
  -Dsonar.organization=YOUR_ORG \
  -Dsonar.host.url=https://sonarcloud.io \
  -Dsonar.token=YOUR_TOKEN
```

**Gradle:**
```bash
./gradlew sonar \
  -Dsonar.projectKey=YOUR_PROJECT_KEY \
  -Dsonar.organization=YOUR_ORG \
  -Dsonar.host.url=https://sonarcloud.io \
  -Dsonar.token=YOUR_TOKEN
```

**Other scanners:** See [SonarCloud documentation](https://docs.sonarcloud.io/)

### 5. Configure DevOps Integration

Set up automatic analysis by configuring ALM integration:
- GitHub
- Azure DevOps
- GitLab
- Bitbucket

See [SonarCloud ALM Integration](https://docs.sonarcloud.io/advanced-setup/ci-based-analysis/)

---

## Troubleshooting

### Migration Fails During Extract

**Check logs:**
```bash
tail -100 files/*/requests.log
```

**Common issues:**
- SonarQube URL not accessible
- Token lacks admin permissions
- Docker: use `host.docker.internal` instead of `localhost`

### Migration Fails During Upload

**Common issues:**
- SonarCloud token lacks enterprise admin permissions
- Organization not added to enterprise
- Incorrect organization key

**Review logs:**
```bash
ls -lah files/*/requests.log
```

### No Projects Extracted

- Verify SonarQube token has admin permissions
- Check that projects exist in SonarQube
- Review `files/<extract-id>/requests.log` for API errors

### Authentication Errors

- Verify tokens are valid and not expired
- Required permissions:
  - **SonarQube**: Admin or global analysis permissions
  - **SonarCloud**: Enterprise-level admin permissions

### API Rate Limiting

Reduce concurrency in your config file:
```json
{
  "settings": {
    "concurrency": 5,
    "timeout": 120
  }
}
```

### Resume Failed Migration

If migration fails partway through:

```bash
# Find the run ID from directory names in files/
ls files/

# Resume with run_id
./dist/sonar-reports-macos-arm64 migrate --config migrate-config.json --run_id=<RUN_ID>
```

---

## Advanced Usage

### Migrate Specific Organizations Only

1. Run extract and structure commands
2. Edit `files/organizations.csv` and remove rows you don't want
3. Run mappings and migrate commands

### Migrate Specific Tasks Only

```bash
./dist/sonar-reports-macos-arm64 migrate \
  YOUR_TOKEN YOUR_ENTERPRISE_KEY \
  --export_directory=./files \
  --target_task=createProjects
```

### Skip Quality Profile Migration

If quality profiles are not important to migrate, or if profile provisioning causes issues, you can skip them entirely:

```bash
./dist/sonar-reports-macos-arm64 migrate \
  YOUR_TOKEN YOUR_ENTERPRISE_KEY \
  --export_directory=./files \
  --skip_profiles
```

For `full-migrate`, set `skip_profiles` in the config file:
```json
{
  "settings": {
    "skip_profiles": true
  }
}
```

This skips all profile-related tasks: creating profiles, restoring profile rules, setting default profiles, profile parent relationships, profile group permissions, and project profile assignments.

### Extract Specific Data Only

```bash
./dist/sonar-reports-macos-arm64 extract \
  http://localhost:9000 YOUR_TOKEN \
  --export_directory=./files \
  --target_task=getProjects
```

### Custom Export Directory

Specify a different directory:
```json
{
  "settings": {
    "export_directory": "/path/to/your/directory"
  }
}
```

### Adjust Concurrency and Timeout

For slower connections or rate limiting:
```json
{
  "settings": {
    "concurrency": 5,
    "timeout": 120
  }
}
```

### Docker Testing

Test the Linux binary in a clean container environment:

```bash
# Build the test container
docker build -f docker/Dockerfile.test-linux -t sonar-reports-test .

# Run a test migration (requires SonarQube accessible from Docker)
docker run --rm \
  -v "$(pwd):/app" \
  -w /app \
  --add-host=host.docker.internal:host-gateway \
  sonar-reports-test
```

**Note:** When running in Docker, use `host.docker.internal` instead of `localhost` in your config file to access services on the host machine.

---

## Security

⚠️ **Important Security Considerations:**

### Never Commit Secrets

- Never commit `migration-config.json` to version control
- Config files with secrets are already in `.gitignore`
- Only `.example.json` files should be committed

### Protect Your Tokens

- Store tokens securely (environment variables, secret managers)
- Tokens have full admin access - treat them as highly sensitive
- Consider using temporary tokens that expire after migration

### Use File Permissions

Restrict access to config files:
```bash
chmod 600 migration-config.json
```

### Environment Variables

For CI/CD, use environment variables:
```bash
export SONAR_TOKEN="your-token"
cat > migration-config.json <<EOF
{
  "sonarqube": {
    "token": "$SONAR_TOKEN"
  }
}
EOF
```

---

## Additional Documentation

- **[docs/QUICK-START.md](docs/QUICK-START.md)** - Detailed step-by-step guide (if available)
- **[docs/BUILD.md](docs/BUILD.md)** - Building executables for all platforms
- **[docs/CONFIG.md](docs/CONFIG.md)** - Configuration file reference
- **[docs/OLD_README.rst](docs/OLD_README.rst)** - Legacy technical documentation

---

## Support

For issues or questions:

1. Check log files in `files/*/requests.log`
2. Review this documentation and linked guides
3. Check for similar issues in the repository
4. Open a new issue with:
   - Your command/config (redact tokens!)
   - Error messages
   - Relevant log excerpts

---

## License

See LICENSE file for details.
