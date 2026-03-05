# Architecture

## Overview

sonar-reports is a Python CLI application built with Click. It can run as a standalone PyInstaller executable, Docker container, or directly with Python. Its primary purpose is to facilitate migration from SonarQube Server to SonarQube Cloud (formerly SonarCloud).

## Project Structure

```
src/
├── main.py                # CLI entry point (Click commands)
├── config.py              # JSON config file loader
├── plan.py                # Task plan generation (DAG)
├── execute.py             # Task execution engine
├── dependencies.py        # Dependency loading strategies
├── constants.py           # Built-in repos, task lists
├── validate.py            # Pre-migration validation
├── utils.py               # CSV/JSON utilities
├── parser.py              # Field/value parsing
├── analysis_report.py     # Migration analysis report
├── operations/            # Data transformation operations
│   ├── http_request/      # HTTP request operations
│   ├── apply_filter.py
│   ├── expand_list.py
│   ├── extend_list.py
│   ├── get_date.py
│   ├── join_string.py
│   ├── match_devops_platform.py
│   ├── process_array.py
│   └── set_key.py
├── structure/             # Org/project structure mapping
├── wizard/                # Interactive wizard
├── pipelines/             # CI/CD pipeline updater
├── report/                # Report generators
└── tasks/                 # JSON task definitions
    ├── export/            # 72+ extraction tasks
    ├── migrate/           # 46+ migration tasks
    └── delete/            # Cleanup/reset tasks
```

## Task Engine

The core of the tool is a JSON-driven task execution engine. Tasks are declarative definitions that describe how to extract data from SonarQube Server, transform it, and load it into SonarQube Cloud.

### Task Definitions

Tasks are JSON files in `src/tasks/` organized by SonarQube version (e.g., `4.3.json`, `10.4.json`, `cloud.json`). Each task specifies:

- **`dependencies`** -- prerequisite tasks with loading strategies (`each`, `chunk`, `all`, `map`, `none`)
- **`editions`** -- which SonarQube editions support this task (`community`, `developer`, `enterprise`, `datacenter`)
- **`operations`** -- an ordered pipeline of data transformations

### Dependency Loading Strategies

The dependency system controls how prerequisite task output is fed into downstream tasks:

- **`each`** -- yield each object individually. The task executes once per dependency object.
- **`chunk`** -- group objects into fixed-size chunks. Useful for batch API calls.
- **`all`** -- load all dependency data into memory at once. Suitable for lookups and joins.
- **`map`** -- group objects by a key field. Enables efficient key-based access.
- **`none`** -- no dependency data needed. The task runs independently.

### Operations

Operations are pluggable transformations that form the processing pipeline within each task:

- **`http_request`** -- make API calls with payload mapping and result extraction
- **`apply_filter`** -- filter objects based on conditions
- **`set_key`** -- add or modify fields on the current object
- **`process_array`** -- array transformations
- **`expand_list`** / **`extend_list`** -- list expansion and concatenation operations
- **`join_string`** -- concatenate fields into a single string
- **`match_devops_platform`** -- detect the DevOps platform (GitHub, GitLab, Azure DevOps, Bitbucket)
- **`get_date`** -- date parsing and formatting operations

### Execution Flow

1. **`get_available_task_configs()`** loads task configurations based on the detected SonarQube version and edition.
2. **`generate_task_plan()`** builds a directed acyclic graph (DAG) and resolves all dependencies.
3. **`plan_tasks()`** organizes tasks into sequential execution phases, respecting dependency order.
4. **`execute_plan()`** runs phases in order.
5. **`execute_task()`** loads dependencies, runs the operation pipeline, and exports results as NDJSON.

## Data Flow

The migration proceeds through a well-defined sequence of stages:

```
SonarQube Server API
    | extract (get* tasks)
    v
NDJSON files in files/<extract_id>/
    | structure
    v
organizations.csv + projects.csv
    | (user fills in sonarcloud_org_key)
    | mappings
    v
gates.csv, profiles.csv, groups.csv, templates.csv, portfolios.csv
    | migrate (create*/set*/add* tasks)
    v
SonarQube Cloud API
    | pipelines (optional)
    v
CI/CD pipeline PRs
```

**Extract phase:** The tool queries SonarQube Server APIs and writes each entity (projects, quality gates, quality profiles, users, groups, permissions, etc.) to NDJSON files on disk.

**Structure phase:** Extracted data is organized into CSV mapping files that define how SonarQube Server entities correspond to SonarQube Cloud organizations and projects. Users review and fill in required fields such as `sonarcloud_org_key`.

**Migrate phase:** Using the completed mapping files, the tool creates corresponding entities in SonarQube Cloud via its API.

**Pipelines phase (optional):** The tool can generate pull requests to update CI/CD pipeline configurations (e.g., GitHub Actions, GitLab CI, Azure Pipelines) to point to the new SonarQube Cloud instance.

## Configuration System

- `config.py` loads JSON configuration files that define connection details and migration parameters.
- CLI arguments override config file values, allowing per-invocation customization.
- `export_directory` is restricted to the working directory or `/tmp` for security -- this prevents path traversal attacks.

## Version Detection

The tool auto-detects SonarQube Server version and selects appropriate API endpoints and authentication methods:

- **Server < 10:** Basic authentication (username:token)
- **Server >= 10:** Bearer token authentication
- **Edition-aware:** The tool detects whether the source instance is Community, Developer, Enterprise, or Data Center edition. Tasks are filtered based on edition support -- for example, portfolio-related tasks only run against Enterprise and Data Center editions.

## Runtime Dependencies

| Dependency    | Purpose                          |
|---------------|----------------------------------|
| httpx         | Async HTTP client for API calls  |
| click         | CLI framework                    |
| markdown      | Markdown rendering               |
| tenacity      | Retry logic for transient errors |
| PyNaCl        | Cryptographic operations         |
| ruamel.yaml   | YAML parsing (pipeline updates)  |
| bashlex       | Bash script parsing              |
| lxml          | XML parsing                      |

## Build System

- **PyInstaller** creates standalone executables from the spec file `sonar-reports.spec`.
- Builds target **6 platforms**: macOS (x86_64, ARM64), Linux (x86_64, ARM64), Windows (x86_64, ARM64).
- **Docker image** is based on Python 3.14-slim for minimal footprint.
- **CI/CD** uses GitHub Actions workflows for automated testing, Docker image publishing, and release builds.

## Testing

- **pytest** is the test runner, with **pytest-asyncio** for async test support, **pytest-mock** for mocking, and **respx** for HTTP request mocking.
- Coverage is configured via `.coveragerc`.
- SonarQube Cloud analysis is configured through `sonar-project.properties` for continuous code quality tracking of the tool itself.
