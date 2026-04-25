# Architecture

## Overview

sonar-migration-tool is a Go CLI application built with [Cobra](https://github.com/spf13/cobra). It compiles to a single static binary with no runtime dependencies. Its purpose is to migrate configurations from SonarQube Server to SonarQube Cloud.

## Project Structure

The repository contains two Go modules:

```
sonar-migration-tool/
├── go/                          # Migration tool (main binary)
│   ├── main.go                  # Entry point
│   ├── cmd/                     # Cobra command definitions
│   │   ├── root.go              # Root command + global flags
│   │   ├── wizard.go            # Interactive guided migration
│   │   ├── extract.go           # Phase 1: Extract from SonarQube Server
│   │   ├── structure.go         # Phase 2: Generate org/project structure
│   │   ├── mappings.go          # Phase 3: Generate entity mappings
│   │   ├── migrate.go           # Phase 4: Push to SonarQube Cloud
│   │   ├── reset.go             # Delete all migrated content
│   │   ├── report.go            # Maturity/migration reports
│   │   └── analysis_report.go   # API call outcome summary
│   └── internal/
│       ├── common/              # Shared utilities
│       │   ├── rawclient.go     # HTTP client with auth + retry
│       │   ├── store.go         # DataStore (JSONL read/write)
│       │   ├── writer.go        # ChunkWriter (batched JSONL output)
│       │   ├── planner.go       # Topological sort task planner
│       │   ├── edition.go       # SonarQube edition detection
│       │   └── helpers.go       # CSV, JSON, string utilities
│       ├── extract/             # Server data extraction (67 tasks)
│       │   ├── extract.go       # Orchestrator
│       │   ├── planner.go       # Task dependency graph
│       │   └── tasks_*.go       # Typed extraction tasks by category
│       ├── structure/           # Org/project/profile/gate mapping
│       │   ├── structure.go     # Structure command logic
│       │   ├── csv.go           # CSV load/export
│       │   └── types.go         # Organization, Project, etc.
│       ├── migrate/             # Cloud migration (44+ tasks)
│       │   ├── migrate.go       # Orchestrator
│       │   ├── planner.go       # Task dependency graph
│       │   ├── reset.go         # Delete/reset tasks
│       │   └── tasks_*.go       # Typed migration tasks by category
│       ├── wizard/              # Interactive wizard (6 phases)
│       │   ├── wizard.go        # Phase loop + resume logic
│       │   ├── phases.go        # Phase implementations
│       │   ├── state.go         # WizardState persistence (JSON)
│       │   ├── prompter.go      # Prompter interface
│       │   ├── cli_prompter.go  # Terminal UI (survey library)
│       │   └── helpers.go       # Phase sequence, validation
│       ├── report/              # Report generation
│       │   ├── common/          # Data loaders (JSONL → report rows)
│       │   ├── maturity/        # SonarQube maturity report
│       │   ├── migration/       # Migration readiness report
│       │   ├── markdown.go      # Markdown rendering
│       │   └── jsonpath.go      # JSON path extraction
│       └── analysis/            # API call analysis (requests.log → CSV)
├── lib/
│   └── sq-api-go/               # Typed SonarQube API binding library
│       ├── client.go            # Client factory (Server + Cloud)
│       ├── auth.go              # Auth strategies (Basic, Bearer, mTLS)
│       ├── pagination.go        # Generic paginator
│       ├── retry.go             # Exponential backoff
│       ├── server/              # SonarQube Server API methods
│       ├── cloud/               # SonarQube Cloud API methods
│       └── types/               # Shared response structs
├── examples/                    # JSON config file examples
├── scripts/                     # Automation scripts
└── docs/                        # Documentation
```

## Task Engine

Both `extract` and `migrate` use a typed task engine with topological sort planning.

### How It Works

1. **Task registration** — Each task is a Go function with typed dependencies and a defined execution function. Tasks declare which other tasks they depend on.

2. **Dependency resolution** — `common.Planner` builds a directed acyclic graph (DAG) from task dependencies and produces an ordered execution plan using topological sort.

3. **Phased execution** — Tasks are grouped into phases where all tasks in a phase can run concurrently. Each phase completes before the next begins.

4. **Data flow** — Tasks read input from a `DataStore` (which loads JSONL files from previous tasks) and write output via a `ChunkWriter` (which produces JSONL files for downstream tasks).

### Extract Tasks (67 tasks)

Organized by category in `go/internal/extract/tasks_*.go`:
- **System** — Server version, edition, plugins
- **Projects** — Project list, tags, settings, branches, pull requests
- **Profiles** — Quality profiles, rules, inheritance
- **Gates** — Quality gates, conditions, associations
- **Users/Groups** — Users, groups, memberships
- **Templates** — Permission templates, associated groups/users
- **Views** — Portfolios, applications (Enterprise+ only)
- **Issues** — Accepted issues, safe hotspots
- **Webhooks** — Global and project-level webhooks

### Migrate Tasks (44+ tasks)

Organized by category in `go/internal/migrate/tasks_*.go`:
- **Create** — Projects, groups, quality gates, quality profiles, permission templates, portfolios
- **Configure** — Gate conditions, project settings, new code periods, default profiles/gates
- **Associate** — Profile-to-project, gate-to-project, group memberships
- **Permissions** — Template permissions, project permissions
- **Rules** — Custom rule activation
- **ALM** — DevOps platform binding detection
- **Delete/Reset** — Cleanup tasks for the `reset` command

## Data Flow

```
SonarQube Server API
    | extract (typed tasks → JSONL)
    v
JSONL files in files/<extract_id>/
    | structure
    v
organizations.csv + projects.csv
    | (user fills in sonarcloud_org_key)
    | mappings
    v
gates.csv, profiles.csv, groups.csv, templates.csv, portfolios.csv
    | migrate (typed tasks → Cloud API)
    v
SonarQube Cloud API
```

## API Binding Library (sq-api-go)

The `lib/sq-api-go/` module provides typed Go methods for SonarQube Server and Cloud APIs. Key features:

- **Dual client** — `NewServerClient()` for Server, `NewCloudClient()` for Cloud
- **Version-aware auth** — Basic auth (Server < 10), Bearer token (Server 10+, Cloud)
- **mTLS support** — Client certificate authentication
- **Automatic pagination** — Handles `p`/`ps` pagination parameters
- **Retry with backoff** — 3 attempts with exponential backoff

## Version Detection

The tool auto-detects SonarQube Server version and edition:

- **Server < 10:** Basic authentication (username:token)
- **Server >= 10:** Bearer token authentication
- **Edition-aware:** Tasks are filtered by edition — portfolio-related tasks only run on Enterprise and Data Center editions

## Configuration

Commands accept flags, positional arguments, or a JSON config file (`--config path/to/config.json`). CLI flags override config file values. See `docs/CONFIG.md` for details.

## Testing

- **Framework:** stdlib `testing` + `net/http/httptest` for HTTP mocking
- **Run tests:** `cd go && go test ./... -count=1`
- **Coverage:** `cd go && go test ./... -coverprofile=coverage.out`
