# Manual Migration Guide

## Overview

This guide walks you through migrating from SonarQube Server to SonarQube Cloud using the manual (step-by-step) approach. The manual method gives you more control over the process by letting you run each phase separately, inspect intermediate files, and make adjustments along the way.

If you prefer a more guided experience, there is also a `wizard` command that walks you through everything interactively. See the [Interactive Wizard Alternative](#interactive-wizard-alternative) section at the bottom of this guide.

---

## Prerequisites

Before you begin, make sure you have the following:

- **A SonarQube Server admin token** -- This token needs the following permissions:
  - Administer System
  - Quality Gates (read/write)
  - Quality Profiles (read/write)
  - Browse on all projects you want to migrate

- **A SonarQube Cloud enterprise account with an admin token** -- This token must have:
  - Enterprise-level access
  - Admin access to all target organizations in SonarQube Cloud

### Token Permissions Summary

| Token              | Required Permissions                                                        |
|--------------------|-----------------------------------------------------------------------------|
| SonarQube Server   | Administer System, Quality Gates, Quality Profiles, Browse (all projects)   |
| SonarQube Cloud    | Enterprise-level access, Admin on all target organizations                  |

---

## Migration Workflow Diagram

The migration follows this sequence of phases:

```
EXTRACT --> STRUCTURE --> MAPPINGS --> MIGRATE --> PIPELINES (optional)
```

1. **Extract** -- Pull data out of your SonarQube Server instance.
2. **Structure** -- Generate an organization structure file from the extracted data.
3. **Mappings** -- Create entity mapping files (gates, profiles, groups, etc.).
4. **Migrate** -- Push everything into SonarQube Cloud.
5. **Pipelines** -- Optionally update your CI/CD pipeline configurations.

---

## Step-by-Step Guide

### Step 1: Create a Working Directory

Start by creating a fresh directory to hold all migration files:

```bash
mkdir sonar-migration && cd sonar-migration
```

All subsequent commands assume you are running them from inside this directory.

---

### Step 2: Extract Data from SonarQube Server

This step connects to your SonarQube Server and exports all the data needed for migration.

**Using Binary:**

```bash
./sonar-reports-<platform> extract http://localhost:9000 YOUR_TOKEN --export_directory=./files
```

**Using Docker:**

```bash
docker run --rm -v "$(pwd)/files:/app/files" ghcr.io/sonar-solutions/sonar-reports:latest extract http://your-server:9000 YOUR_TOKEN
```

> **Note:** If your SonarQube Server is running on `localhost`, use `host.docker.internal` instead of `localhost` when running via Docker. For example: `http://host.docker.internal:9000`.

**Using Python:**

```bash
python src/main.py extract http://localhost:9000 YOUR_TOKEN --export_directory=./files
```

#### Extract Options

| Option              | Description                                              | Default |
|---------------------|----------------------------------------------------------|---------|
| `--concurrency`     | Number of concurrent API requests                        | 25      |
| `--timeout`         | Request timeout in seconds                               | 60      |
| `--extract_id`      | Resume a previously started extraction by its ID         | --      |
| `--extract_type`    | Type of extraction to perform                            | --      |
| `--target_task`     | Run a specific extraction task only                      | --      |
| `--pem_file_path`   | Path to a PEM client certificate file                    | --      |
| `--key_file_path`   | Path to a private key file for client certificate auth   | --      |
| `--cert_password`   | Password for the client certificate                      | --      |

---

### Step 3: Generate Organization Structure

This step reads the extracted data and generates an `organizations.csv` file that maps your SonarQube Server projects to SonarQube Cloud organizations.

**Using Binary:**

```bash
./sonar-reports-<platform> structure --export_directory=./files
```

**Using Docker:**

```bash
docker run --rm -v "$(pwd)/files:/app/files" ghcr.io/sonar-solutions/sonar-reports:latest structure
```

**Using Python:**

```bash
python src/main.py structure --export_directory=./files
```

The `--export_directory` option tells the tool where to find the extracted data and where to write the output.

---

### Step 4: Edit organizations.csv

Open the generated file at `files/organizations.csv` in any spreadsheet editor or text editor. You need to fill in the `sonarcloud_org_key` column with the key of the SonarQube Cloud organization where each group of projects should be migrated.

Here is an example of what the file looks like:

```csv
server_url,sonarcloud_org_key
http://localhost:9000,my-cloud-org-key
```

Each row represents a SonarQube Server instance. Fill in the `sonarcloud_org_key` value with the organization key from SonarQube Cloud where you want projects from that server to land.

Save the file when you are done.

---

### Step 5: Generate Entity Mappings

This step generates mapping files that control how quality gates, quality profiles, groups, permission templates, and portfolios are migrated.

**Using Binary:**

```bash
./sonar-reports-<platform> mappings --export_directory=./files
```

**Using Docker:**

```bash
docker run --rm -v "$(pwd)/files:/app/files" ghcr.io/sonar-solutions/sonar-reports:latest mappings
```

**Using Python:**

```bash
python src/main.py mappings --export_directory=./files
```

This produces the following files in your `files/` directory:

- `gates.csv` -- Quality Gate mappings
- `profiles.csv` -- Quality Profile mappings
- `groups.csv` -- Group mappings
- `templates.csv` -- Permission Template mappings
- `portfolios.csv` -- Portfolio mappings

You can open and review (or edit) any of these files before proceeding to the migration step.

---

### Step 6: Run the Migration

Now it is time to push everything to SonarQube Cloud. You will need your SonarQube Cloud admin token and your enterprise key.

**Using Binary:**

```bash
./sonar-reports-<platform> migrate YOUR_CLOUD_TOKEN YOUR_ENTERPRISE_KEY --export_directory=./files
```

**Using Docker:**

```bash
docker run --rm -v "$(pwd)/files:/app/files" ghcr.io/sonar-solutions/sonar-reports:latest migrate YOUR_CLOUD_TOKEN YOUR_ENTERPRISE_KEY
```

**Using Python:**

```bash
python src/main.py migrate YOUR_CLOUD_TOKEN YOUR_ENTERPRISE_KEY --export_directory=./files
```

#### Migrate Options

| Option              | Description                                               | Default                   |
|---------------------|-----------------------------------------------------------|---------------------------|
| `--url`             | SonarQube Cloud URL                                       | https://sonarcloud.io/    |
| `--edition`         | Target edition                                            | --                        |
| `--concurrency`     | Number of concurrent API requests                         | 25                        |
| `--run_id`          | Resume a previously started migration by its run ID       | --                        |
| `--target_task`     | Run a specific migration task only                        | --                        |
| `--skip_profiles`   | Skip migrating quality profiles                           | --                        |

---

### Step 7: Update CI/CD Pipelines (Optional)

If you want to automatically update your CI/CD pipeline configurations to point to SonarQube Cloud, you can use the `pipelines` command. This step is optional.

You will need a secrets file (containing credentials for your CI/CD platform), your SonarQube Cloud token, and the SonarQube Cloud URL.

**Using Binary:**

```bash
./sonar-reports-<platform> pipelines SECRETS_FILE SONAR_TOKEN SONAR_URL --export_directory=./files
```

**Using Docker:**

```bash
docker run --rm -v "$(pwd)/files:/app/files" ghcr.io/sonar-solutions/sonar-reports:latest pipelines SECRETS_FILE SONAR_TOKEN SONAR_URL
```

**Using Python:**

```bash
python src/main.py pipelines SECRETS_FILE SONAR_TOKEN SONAR_URL --export_directory=./files
```

Supported CI/CD platforms:

- GitHub Actions
- GitLab CI/CD
- Azure DevOps
- Bitbucket Pipelines

---

### Step 8: Verify

Once the migration is complete, take a few minutes to verify everything landed correctly:

1. **Log in to SonarQube Cloud** and check:
   - Quality Gates -- Are they all present and configured correctly?
   - Quality Profiles -- Do they match what you had on SonarQube Server?
   - Groups -- Are all user groups created with the right memberships?
   - Projects -- Are all projects visible and assigned to the correct organization?

2. **Generate a migration report** to get a summary of what was migrated:

   ```bash
   ./sonar-reports-<platform> report --export_directory=./files --report_type=migration
   ```

3. **Generate an analysis report** for a specific migration run (use the run ID from Step 6):

   ```bash
   ./sonar-reports-<platform> analysis_report <RUN_ID>
   ```

---

## Multi-Server Migration

If you are migrating from multiple SonarQube Server instances, follow this approach:

1. **Extract from each server separately** -- Run the `extract` command once for each SonarQube Server instance, each time pointing to a different server URL and token.

2. **Run the `structure` command** -- This step automatically aggregates data from all extractions into a single `organizations.csv` file.

3. **Edit `organizations.csv`** -- Fill in the `sonarcloud_org_key` for each server row.

4. **Continue with `mappings` and `migrate`** as described above. The tool will handle all servers in one pass.

---

## Resuming Failed Operations

Things do not always go perfectly. If a step fails partway through, you can pick up where you left off instead of starting over:

- **Resuming an extraction:** Use the `--extract_id` flag with the ID from the failed extraction run.

  ```bash
  ./sonar-reports-<platform> extract http://localhost:9000 YOUR_TOKEN --extract_id=<PREVIOUS_EXTRACT_ID>
  ```

- **Resuming a migration:** Use the `--run_id` flag with the run ID from the failed migration.

  ```bash
  ./sonar-reports-<platform> migrate YOUR_CLOUD_TOKEN YOUR_ENTERPRISE_KEY --run_id=<PREVIOUS_RUN_ID>
  ```

---

## Output Files Reference

Here is a summary of all the files generated during the migration process:

| File                     | Description                                                    |
|--------------------------|----------------------------------------------------------------|
| `extract.json`           | Metadata about the extraction (timestamps, server info, etc.)  |
| `requests.log`           | Log of all API requests made during extraction                 |
| `results.*.jsonl`        | Raw extracted data in JSON Lines format (one file per entity)  |
| `organizations.csv`      | Server-to-organization mapping (you edit this)                 |
| `projects.csv`           | List of all extracted projects                                 |
| `gates.csv`              | Quality Gate mappings                                          |
| `profiles.csv`           | Quality Profile mappings                                       |
| `groups.csv`             | Group mappings                                                 |
| `templates.csv`          | Permission Template mappings                                   |
| `portfolios.csv`         | Portfolio mappings                                             |

---

## Interactive Wizard Alternative

If you would rather not run each command separately, the `wizard` command provides an interactive, guided experience that walks you through the entire migration process.

**Using Binary:**

```bash
./sonar-reports-<platform> wizard
```

**Using Docker:**

```bash
docker run -it -v ./files:/app/files ghcr.io/sonar-solutions/sonar-reports:latest wizard
```

**Using Python:**

```bash
python src/main.py wizard
```

The wizard includes the following features:

- **Resume support** -- If a previous wizard session was interrupted, it can pick up where it left off.
- **Client certificate prompts** -- If your SonarQube Server requires client certificate authentication, the wizard will prompt you for the necessary file paths and passwords.
- **Progress display** -- See real-time progress as each phase runs.
- **Validation** -- The wizard validates your inputs (tokens, URLs, file paths) before proceeding to each step.

This is a great option if you are running a migration for the first time and want a bit of hand-holding through the process.
