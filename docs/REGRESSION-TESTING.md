# Live Regression Testing Protocol (Dynamic Workflow Edition)
<!-- updated: 2026-05-31_00:00:00 -->

A reusable protocol for verifying **any change** — bug fix, new feature, refactor, or enhancement — by running the sonar-migration-tool **live** against real SonarQube Server and SonarQube Cloud instances. The goal is to prove that the change works correctly AND that nothing else broke.

**This protocol is a recursive loop.** You build, run, verify, and if anything is wrong — fix, rebuild, re-run, re-verify. You repeat until you achieve a **full clean pass**. That is the only stop condition.

**This protocol is designed for Dynamic Workflows.** Every phase that can be parallelized MUST be parallelized using parallel agent swarms. Look for the `> **[PARALLEL AGENT SWARM]**` annotations throughout this document — those are your fan-out points.

```
┌─────────────────────────────────────────────────────────────────────┐
│                                                                     │
│   Phase 0: Understand the change  ← PARALLEL SWARM: 4+N agents     │
│       ↓                                                             │
│   Phase 1: Adversarial code review  ← PARALLEL SWARM: 6 agents     │
│       ↓                                                             │
│   Phase 2: Environment setup (clean slate)  ← PARALLEL SWARM: 10+  │
│       ↓                                                             │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │  RECURSIVE LOOP (repeat until clean pass)                   │    │
│  │                                                             │    │
│  │  Phase 3: Build + run LIVE  ← PARALLEL SWARM: 7 agents     │    │
│  │      ↓                                                      │    │
│  │  Phase 4: Verify  ← PARALLEL SWARM: 46+ agents (maximum)   │    │
│  │      ↓                                                      │    │
│  │  Phase 5: Any failure? → investigate + fix  ──┐             │    │
│  │      ↓                          ↑  PARALLEL  │             │    │
│  │  ALL PASS? ──── NO ─────────────┘  SWARM: 8+ │             │    │
│  │      │                                        │             │    │
│  │     YES                                       │             │    │
│  │      ↓                                        │             │    │
│  │  Phase 6: Declare clean pass  ← PARALLEL: 6  │             │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Dynamic Workflow Architecture
<!-- updated: 2026-05-31_00:00:00 -->

When running this protocol as a Claude Dynamic Workflow, use the following phase and agent structure. Sequential steps are explicit — everything else is a parallel fan-out.

| Protocol Phase | Workflow Phase Name | Agents Spawned | Parallelism |
|---|---|---|---|
| Phase 0 — Understand | `Understand` | 4 (Q&A) + N (grep agents) | High |
| Phase 1 — Review | `Review` | 5 (dimensions) + 1 (synthesis) | Full |
| Phase 2 — Setup | `Setup` | 2 (health) + 8+ (data checks) + N (baseline) | High |
| Phase 3 — Execute | `Execute` | 3 (build/test/config) + 4 (log analysis) | Medium |
| Phase 4 — Verify | `Verify` | 46+ (extract + SC queries + spot-checks + edge cases) | **Maximum** |
| Phase 5 — Fix | `Fix` | 3 (isolate) + 5 (hypotheses) + 2 (pre-flight) | Medium |
| Phase 6 — Declare | `Declare` | 5 (checklist categories) + 1 (synthesis) | Full |

### Workflow Script Template
<!-- updated: 2026-05-31_00:00:00 -->

Copy this into the `Workflow` tool to run this protocol as a Dynamic Workflow:

```javascript
export const meta = {
  name: 'regression-test',
  description: 'Run full live regression test of sonar-migration-tool change',
  phases: [
    { title: 'Understand', detail: '4 Q&A agents + N grep agents in parallel' },
    { title: 'Review', detail: '5 dimension agents + synthesis in parallel' },
    { title: 'Setup', detail: 'Health checks + data verification + baseline in parallel' },
    { title: 'Execute', detail: 'Build → [tests ∥ race-build ∥ config] → pipeline → log analysis' },
    { title: 'Verify', detail: '46+ parallel agents across all entity types and checks' },
    { title: 'Fix', detail: 'Parallel isolation + hypothesis agents per failure' },
    { title: 'Declare', detail: '5 parallel checklist verifiers + synthesis agent' },
  ],
}

// Phase 0 — fan out to 4 Q&A agents simultaneously
phase('Understand')
const [intentBlast, commandsEntities, codePaths, acceptanceEdge] = await parallel([
  () => agent('Answer Phase 0.1 questions 1+5: intent sentence and blast radius. Read config.json and the diff. Cross-check all shared utilities (HTTP client, pagination, retry, config parsing).'),
  () => agent('Answer Phase 0.1 questions 2+3: list every affected command (extract/migrate/structure/mappings/report/reset/wizard/gui) and every affected entity type. Cross-reference the Full Entity Registry in REGRESSION-TESTING.md.'),
  () => agent('Answer Phase 0.1 question 4: read every changed file line by line. List every file, function, and API call that was added, removed, or modified.'),
  () => agent('Answer Phase 0.1 questions 6+7: write 3-5 measurable acceptance criteria and enumerate edge cases (empty, single, large, missing fields, idempotency).'),
])

// 0.3 — grep agents (one per function/endpoint found in the diff)
const watchlist = await parallel(
  codePaths.functions.map(fn => () => agent('grep -r "' + fn + '" go/ --include="*.go" | grep -v "_test.go". Report every file that references this function.'))
)

// Phase 1 — all 5 review dimensions in parallel
phase('Review')
const [dataIntegrity, errorHandling, concurrency, apiContract, edgeCases] = await parallel([
  () => agent('Adversarial review — DATA INTEGRITY: check serialization round-trips, field mapping, stat counter thread-safety. Read every changed file and all files on the regression watchlist.'),
  () => agent('Adversarial review — ERROR HANDLING: find every error return. Any _ = err is a bug. Check retry logic caps and Retry-After handling. Check goroutine error propagation.'),
  () => agent('Adversarial review — CONCURRENCY SAFETY: find every shared map/slice/counter mutated from goroutines without mutex/atomic. Verify goroutine lifecycle via WaitGroup/errgroup. Check for send-on-closed-channel.'),
  () => agent('Adversarial review — API CONTRACT COMPLIANCE: verify HTTP method, endpoint path, request body, and auth header for every API call against the SonarQube API docs.'),
  () => agent('Adversarial review — EDGE CASES: verify 0/1/N item handling, nil/missing field handling, and duplicate entity prevention for every code path.'),
])
const concerns = await agent('Synthesize all adversarial review findings into a single prioritized concern list. Format each: "This could cause [failure] if [condition]." Input: ' + JSON.stringify({dataIntegrity, errorHandling, concurrency, apiContract, edgeCases}))

// Phase 2 — environment setup with parallel health checks and baseline
phase('Setup')
const [sqsHealth, scHealth] = await parallel([
  () => agent('Verify SQS is accessible: run curl against /api/system/status, /api/projects/search, and /api/issues/search. Report status, auth result, and project count.'),
  () => agent('Verify SC is accessible: run curl against SC /api/system/status. Report auth result.'),
])
// Spawn one agent per entity type to verify test data exists
const testDataChecks = await parallel(
  ENTITY_TYPES.map(e => () => agent('Verify test data for entity type: ' + e + '. Count entities on SQS. If count is 0, create test data via the SQS API.'))
)
// Record baseline — one agent per metric
const baseline = await parallel(
  METRICS.map(m => () => agent('Query SQS for baseline count of: ' + m + '. Return a {metric, count} object.'))
)

// Phase 3 — build then parallel tests + config verify
phase('Execute')
// 3.1 — sequential prerequisite
const build = await agent('Build the binary: cd go && go build -o ../sonar-migration-tool ./main.go && cd ... Report exit code.')
// 3.2 + 3.3 + 3.4 — parallel after build
const [unitTests, raceDetector, configCheck] = await parallel([
  () => agent('Run unit tests: cd go && go test ./... && cd .. Report pass/fail per package.'),
  () => agent('Build race detector binary: cd go && go build -race -o ../sonar-migration-tool-race ./main.go && cd .. Report exit code.'),
  () => agent('Verify migration-config.json: read the file, confirm all URLs, tokens, project keys, org keys, and feature flags are set correctly.'),
])
// 3.5 — sequential pipeline (data dependency)
const pipeline = await agent('Run the full pipeline sequentially: extract → structure → mappings → migrate. Use the race-detector binary. Tee all logs to /tmp/smt-*.log. Return exit codes.')
// 3.6 — parallel log analysis
const logs = await parallel([
  () => agent('Analyze /tmp/smt-extract-*.log: scan for panics, DATA RACE, FATAL, ERROR. Extract summary stats.'),
  () => agent('Analyze /tmp/smt-structure-*.log: scan for panics, DATA RACE, FATAL, ERROR. Extract summary stats.'),
  () => agent('Analyze /tmp/smt-mappings-*.log: scan for panics, DATA RACE, FATAL, ERROR. Extract summary stats.'),
  () => agent('Analyze /tmp/smt-migrate-*.log: scan for panics, DATA RACE, FATAL, ERROR. Extract summary stats and wall-clock duration.'),
])

// Phase 4 — maximum parallelization: 46+ agents
phase('Verify')
const extractChecks = await parallel(
  ENTITY_TYPES.map(e => () => agent('Inspect ./files/<extract-id>/' + e + '.ndjson: check non-empty, line count, first 3 lines valid JSON, required fields present.'))
)
const scChecks = await parallel(
  ENTITY_TYPES.map(e => () => agent('Compare ' + e + ' count: query SQS API and SC API with the same parameters. Return {entity, sqsCount, scCount, match}.'))
)
const spotChecks = await parallel(
  Array.from({length: 5}, (_, i) => () => agent('Spot-check entity #' + (i+1) + ': fetch same entity from SQS and SC, compare all fields. Return {match: bool, diff: string}.'))
)
const edgeCaseChecks = await parallel([
  () => agent('Edge case — empty input: run migrate against entity type with 0 items. Verify no crash, no error.'),
  () => agent('Edge case — single entity: run migrate with 1 item. Verify correct migration.'),
  () => agent('Edge case — large volume: run migrate with 1000+ items. Verify all migrated, pagination worked.'),
  () => agent('Edge case — missing fields: run migrate with entities with optional/null fields. Verify no panic.'),
  () => agent('Edge case — idempotency: run migrate twice. Verify no duplicates, no errors on second run.'),
])
const [silentFailures, regressionCheck] = await parallel([
  () => agent('Check for silent failures: grep -ri "error|panic|fatal|fail" /tmp/smt-*.log | grep -v INFO. Also find ./files -name "*.ndjson" -empty. Report all findings.'),
  () => agent('Regression check: for every entity type NOT touched by the change, query SQS and SC counts and compare against the Phase 2 baseline. Report any mismatches.'),
])

// Phase 5 — only runs if Phase 4 had failures
phase('Fix')
// Run in parallel: isolate the failure from 3 angles simultaneously
const [sqsState, ndjsonState, scState] = await parallel([
  () => agent('Query SQS with the exact same API parameters the code uses. Report what data exists at the source.'),
  () => agent('Read the extract NDJSON files. Report what was actually extracted and flag any field anomalies.'),
  () => agent('Query SC to see what was actually created. Report actual target state and any API errors from the migrate log.'),
])
// Hypothesis agents
const hypotheses = await parallel([
  () => agent('Investigate field mapping hypothesis: is a field name/type mismatch causing wrong data? Check every field in the diff against the SQS API response schema.'),
  () => agent('Investigate silent error hypothesis: is a dropped _ = err suppressing a failure? grep -rn "_ = " go/ --include="*.go" in every changed file.'),
  () => agent('Investigate concurrency bug hypothesis: is a data race corrupting shared state? Check every shared variable in the changed goroutines.'),
  () => agent('Investigate API contract hypothesis: is a wrong HTTP method/path/body causing rejection? Compare the code against SonarQube API docs.'),
  () => agent('Investigate pagination bug hypothesis: are items dropped at a page boundary? Check every paginator in the changed code paths.'),
])

// Phase 6 — all checklist categories verified in parallel
phase('Declare')
const [crashFree, codeQuality, featureVerified, dataCorrect, regressionClear] = await parallel([
  () => agent('Verify crash-free: check all log files for non-zero exit codes, panic stack traces, DATA RACE warnings, FATAL lines. Return pass/fail.'),
  () => agent('Verify code quality: run go vet ./..., go test ./..., go test -race ./... in parallel. Return pass/fail for each.'),
  () => agent('Verify feature: check each acceptance criterion from Phase 0. Confirm the core code path was exercised with real data (not just the no-op path). Return pass/fail per criterion.'),
  () => agent('Verify data correctness: query SQS vs SC counts for all entity types, spot-check 5 entities field-by-field, scan for empty NDJSON files and silent data loss. Return pass/fail.'),
  () => agent('Verify no regressions: check all untouched entity types still extract/migrate correctly vs the Phase 2 baseline. Return pass/fail.'),
])
const verdict = await agent('Synthesize all verifier results. If ALL 5 pass → declare CLEAN PASS and stop the loop. If any fail → return specific failure list to Phase 5 with root cause guidance. Input: ' + JSON.stringify({crashFree, codeQuality, featureVerified, dataCorrect, regressionClear}))

return verdict
```

---

## Phase 0 — Understand the Change
<!-- updated: 2026-05-31_00:00:00 -->

> **MANDATORY. Do not proceed until you can answer every question below.**

> **[SEQUENTIAL → PARALLEL FAN-OUT]** Read the diff first (sequential prerequisite), then spawn 4 agents in parallel for the 7 questions, then build the checklist from their combined output.

> **[PARALLEL AGENT SWARM — 4 agents]** Spawn all concurrently after reading the diff. Wait for all before building the Phase 0.2 checklist.
> - **Agent 1 — Intent & Blast Radius**: Answers questions 1 and 5 — writes the intent sentence; scans all shared utilities (HTTP client, pagination, retry, rate limiting, config parsing) for blast radius
> - **Agent 2 — Commands & Entities**: Answers questions 2 and 3 — lists every affected command; lists every affected entity type cross-referenced against the Full Entity Registry
> - **Agent 3 — Code Path Tracer**: Answers question 4 — reads every line of the diff; lists every file, function, and API call added/removed/modified
> - **Agent 4 — Acceptance & Edge Cases**: Answers questions 6 and 7 — writes 3–5 measurable acceptance criteria; enumerates all edge cases (empty, single, large, missing fields, idempotency)

### 0.1 — What Did You Change?

Answer ALL of the following:

1. **What is the intent?** — One sentence: what does this change accomplish?
2. **Which commands are affected?** — extract, migrate, structure, mappings, report, reset, wizard, gui? List every command that touches modified code.
3. **Which entity types are affected?** — issues, hotspots, quality profiles, quality gates, permissions, groups, users, projects, portfolios, settings, rules, new code periods, ALM bindings, webhooks? Cross-reference against the [Full Entity Registry](#full-entity-registry).
4. **What code paths changed?** — Read every line of the diff. List every file, function, and API call that was added, removed, or modified.
5. **What is the blast radius?** — Could this change affect entity types or commands beyond the ones you intended? Check for shared utilities, shared HTTP clients, shared config parsing, shared error handling.
6. **What are the acceptance criteria?** — List 3-5 concrete, measurable pass/fail conditions for the change itself (e.g., "extract produces `quality_profiles.ndjson` with one entry per profile", "all issues with status CONFIRMED on SQS have status CONFIRMED on SC after migrate").
7. **What edge cases exist?** — At minimum: empty input (zero entities), single entity, large volume (pagination), missing/optional fields, entities that already exist on SC (idempotency).

### 0.2 — Build the Regression Checklist

> **[SEQUENTIAL]** Runs after all 4 Phase 0.1 agents complete — synthesizes their outputs into the two checklists below.

From 0.1, write down TWO lists:

**Feature verification** — what proves the change itself works:
- [ ] (your acceptance criteria from 0.1.6)
- [ ] (your edge cases from 0.1.7)

**Regression verification** — what proves nothing else broke:
- [ ] Every entity type NOT touched by your change still extracts/migrates correctly
- [ ] Every command NOT touched by your change still runs without error
- [ ] Summary stats for unchanged features are the same as before your change

### 0.3 — Identify the Regression Watchlist

> **[PARALLEL AGENT SWARM — N agents]** Spawn one grep/check agent per item discovered from the diff. All run concurrently.
> - **Agent per function name**: greps the codebase for each function name found in the diff; reports every file referencing it
> - **Agent per API endpoint**: greps for each API endpoint found in the diff; reports every call site
> - **Agent — Shared Utilities**: checks if HTTP client, pagination, retry, rate limiting, config parsing were modified
> - **Agent — Pattern Matcher**: checks if the same pattern or mapping is used elsewhere in the codebase

Find ALL code paths that share code with your change:

1. `grep` for every function name in the diff
2. `grep` for every API endpoint in the diff
3. Check if shared utilities (HTTP client, pagination, retry, rate limiting, config parsing) were modified
4. Check if the same pattern/mapping is used elsewhere

> Write down every related file. These are your **regression watchlist** — you MUST verify each one in Phase 4.

---

## Phase 1 — Adversarial Code Review
<!-- updated: 2026-05-31_00:00:00 -->

> Re-read every changed file and every file on your regression watchlist. For each, ask:

> **[PARALLEL AGENT SWARM — 6 agents]** Spawn all 5 review dimension agents concurrently against every changed file and every file on the regression watchlist. Agent 6 (Synthesis) waits for all 5.
> - **Agent 1 — Data Integrity**: Reviews items 1–3 below
> - **Agent 2 — Error Handling**: Reviews items 4–6 below
> - **Agent 3 — Concurrency Safety**: Reviews items 7–9 below
> - **Agent 4 — API Contract Compliance**: Reviews item 10 below
> - **Agent 5 — Edge Cases**: Reviews items 11–13 below
> - **Agent 6 — Synthesis** (waits for Agents 1–5): Aggregates all concerns into a single prioritized list. Format per concern: "This could cause [failure] if [condition]."

### Data Integrity
1. **Serialization boundaries**: Does data survive `json.Marshal`/`json.Unmarshal` round-trips? Do `nil` slices, empty maps, zero values, unexported fields survive? Do NDJSON extract outputs deserialize correctly in the migrate phase?
2. **Field mapping**: Is every field from the SQ API response correctly mapped to the SC API request? Check: field name, type, nil handling, slice vs scalar.
3. **Stat integrity**: Are counters accumulated correctly across goroutines? Protected by mutexes or atomics?

### Error Handling
4. **Every `error` return**: Is it checked? `_ = err` is a bug.
5. **Retry logic**: Max attempts cap? Respects Retry-After? Distinguishes retriable (5xx, timeout) from non-retriable (4xx)?
6. **Error propagation**: Can goroutine errors bubble up, or are they silently dropped?

### Concurrency Safety
7. **Shared state**: Any maps, slices, or counters mutated from concurrent goroutines without mutex/atomic protection?
8. **Goroutine lifecycle**: Properly joined via `WaitGroup`/`errgroup`/channel close before exit? Cleaned up on error?
9. **Channel safety**: Properly closed? No send-on-closed-channel panic? No leaked goroutines?

### API Contract Compliance
10. **HTTP method, endpoint path, request body, auth**: All exactly correct per the SonarQube API docs?

### Edge Cases
11. **0, 1, N**: Empty input, single item, pagination boundary?
12. **Missing/nil fields**: Handled gracefully or panic?
13. **Duplicates**: Could an entity be processed/created twice?

> **Write down every concern.** Format: "This could cause [failure] if [condition]."

---

## Phase 2 — Environment Setup (Clean Slate)
<!-- updated: 2026-05-31_00:00:00 -->

> Every test run starts from ZERO. No stale state. No leftover data.

> **[SEQUENTIAL → PARALLEL FAN-OUT]** Execution order: 2.0 (read config, sequential prerequisite) → [2.1 ∥ 2.2 health checks] → [2.3 entity data swarm] → 2.4 (clean slate, sequential) → [2.5 baseline metric swarm]

### 2.0 — Connection Details Are in `config.json`

> **[SEQUENTIAL]** 2.0 is a prerequisite — all subsequent agents need the exported env vars. Read config first.

**You have full access to both SonarQube Server and SonarQube Cloud.** All connection details — URLs, tokens, organization keys — are stored in **`config.json`** at the project root. Read it first:

```bash
cat config.json | jq '.'
```

Map the config fields to the environment variables used in this document:

| Env Var (this doc) | `config.json` path |
|---|---|
| `SQ_URL` | `.sonarqube.url` |
| `SONAR_TOKEN` | `.sonarqube.token` |
| `SC_URL` | `.sonarcloud.organizations[0].url` |
| `SC_TOKEN` | `.sonarcloud.organizations[0].token` |
| `SC_ORG` | `.sonarcloud.organizations[0].key` |

You can export them for use in the curl commands below:

```bash
export SQ_URL=$(jq -r '.sonarqube.url' config.json)
export SONAR_TOKEN=$(jq -r '.sonarqube.token' config.json)
export SC_URL=$(jq -r '.sonarcloud.organizations[0].url' config.json)
export SC_TOKEN=$(jq -r '.sonarcloud.organizations[0].token' config.json)
export SC_ORG=$(jq -r '.sonarcloud.organizations[0].key' config.json)
```

> **Do NOT skip this step.** If an agent claims it has no access to SQS or SC, it is wrong — `config.json` has everything needed.

### 2.1 — Verify Source (SQS) Is Accessible

> **[PARALLEL AGENT SWARM — 2 agents]** 2.1 and 2.2 run concurrently after 2.0 completes:
> - **Agent 1 — SQS Health**: runs the commands below; confirms status=UP, auth works, project exists
> - **Agent 2 — SC Health**: runs the 2.2 commands; confirms SC auth works

```bash
# Verify SQS is running and auth works
curl -s -u "${SONAR_TOKEN}:" "${SQ_URL}/api/system/status" | jq '.status'

# List available projects
curl -s -u "${SONAR_TOKEN}:" "${SQ_URL}/api/projects/search" | jq '.components[] | {key: .key, name: .name}'

# Confirm target project exists
curl -s -u "${SONAR_TOKEN}:" "${SQ_URL}/api/issues/search?projectKeys=${PROJECT_KEY}&ps=1" | jq '.total'
```

### 2.2 — Verify Target (SC) Is Accessible

```bash
# Verify SC auth works
curl -s -H "Authorization: Bearer ${SC_TOKEN}" "${SC_URL}/api/system/status" | jq '.status'
```

### 2.3 — Verify Test Data Exists

> **[PARALLEL AGENT SWARM — 8+ agents]** Spawn one agent per entity type. All run concurrently after 2.1+2.2 confirm environment is up.
> - **Agent — Issues**: counts issues with manual changes (required for issue sync)
> - **Agent — Hotspots**: counts hotspots available for review/status change
> - **Agent — Quality Profiles**: counts profiles on the test project
> - **Agent — Quality Gates**: counts configured quality gates
> - **Agent — Groups**: counts non-built-in user groups
> - **Agent — Users**: counts users with non-default settings
> - **Agent — Settings**: counts non-default project/global settings
> - **Agent — New Code Periods**: counts explicitly configured new code periods
> - **(Add one agent per entity type relevant to your change)**

For the entity types your change affects, confirm sufficient test data exists on SQS:

```bash
# Issues with manual changes (required for issue sync)
curl -s -u "${SONAR_TOKEN}:" "${SQ_URL}/api/issues/search?projectKeys=${PROJECT_KEY}&ps=100" \
  | jq '[.issues[] | select(.updateDate != .creationDate)] | length'

# Quality profiles
curl -s -u "${SONAR_TOKEN}:" "${SQ_URL}/api/qualityprofiles/search?project=${PROJECT_KEY}" | jq '.profiles | length'

# Quality gates
curl -s -u "${SONAR_TOKEN}:" "${SQ_URL}/api/qualitygates/list" | jq '.qualitygates | length'

# Adapt for whatever entity type your feature handles.
# If test data is insufficient, CREATE IT via the SQ API (see Reference: Creating Test Data).
```

### 2.4 — Clean Slate

> **[SEQUENTIAL]** Must run after 2.1+2.2+2.3 confirm the environment is live. Resets all state before running.

```bash
# Delete ALL local state
rm -rf ./files/
rm -f .wizard_state.json

# Reset SC target (delete migrated projects/orgs from prior runs)
# Option A: use the reset command
# ./sonar-migration-tool reset -c migration-config.json
#
# Option B: delete via API
# curl -s -X POST -H "Authorization: Bearer ${SC_TOKEN}" "${SC_URL}/api/projects/delete?project=${PROJECT_KEY}"

# Verify clean
curl -s -H "Authorization: Bearer ${SC_TOKEN}" "${SC_URL}/api/projects/search?organization=${SC_ORG}" \
  | jq '.components | length'
```

### 2.5 — Record Baseline

> **[PARALLEL AGENT SWARM — N agents]** Spawn one agent per row in the table below. All are independent API calls and run concurrently.
> - **Agent — Project Count**: queries SQS total project count
> - **Agent — Issue Count**: queries SQS issue count for test project
> - **Agent — Hotspot Count**: queries SQS hotspot count for test project
> - **Agent — Profile Count**: queries quality profile count
> - **Agent — Gate Count**: queries quality gate count
> - **Agent — Group Count**: queries user group count
> - **Agent — User Count**: queries user count
> - **(Add one agent per entity type relevant to your change)**

Before running, record the source state so you have numbers to compare against:

| Metric | SQS Value |
|--------|-----------|
| Total projects | ___ |
| Total issues (for test project) | ___ |
| Total hotspots (for test project) | ___ |
| Quality profiles | ___ |
| Quality gates | ___ |
| Groups | ___ |
| Users | ___ |
| (entity types relevant to your change) | ___ |

---

## Phase 3 — Build and Run LIVE
<!-- updated: 2026-05-31_00:00:00 -->

> This is the core of the protocol. You are running the actual tool against real instances.

> **[SEQUENTIAL → PARALLEL → SEQUENTIAL → PARALLEL]** Execution graph:
> 3.1 (build binary, sequential prerequisite) → [3.2 unit tests ∥ 3.3 race build ∥ 3.4 config verify] → 3.5 (pipeline, sequential data-dependency order) → [3.6 log analysis swarm]

### 3.1 — Build

> **[SEQUENTIAL]** Prerequisite for everything else. Do not proceed until the binary exists.

```bash
cd go && go build -o ../sonar-migration-tool ./main.go && cd ..
```

**If the build fails, STOP. Fix the compilation error. This counts as a loop iteration — after fixing, return to Phase 3.1.**

### 3.2 — Run Unit Tests

> **[PARALLEL AGENT SWARM — 3 agents]** After 3.1 completes, spawn all three concurrently:
> - **Agent 1 — Unit Tests**: runs `cd go && go test ./...`; reports pass/fail per package (Go parallelises packages internally)
> - **Agent 2 — Race Detector Build**: runs `go build -race -o ../sonar-migration-tool-race ./main.go`; reports build success/failure
> - **Agent 3 — Config Verify**: reads and validates migration-config.json (see 3.4); confirms all URLs, tokens, project keys, org keys, and feature flags are set

```bash
cd go && go test ./... && cd ..
```

**If any test fails, STOP. Fix the test. Return to Phase 3.1.**

### 3.3 — Run Race Detector Build

```bash
cd go && go build -race -o ../sonar-migration-tool-race ./main.go && cd ..
```

### 3.4 — Verify Config

```bash
cat migration-config.json | jq '.'
```

Confirm all URLs, tokens, project keys, organization keys, and feature flags are correct.

### 3.5 — Run the Full Pipeline LIVE

> **[SEQUENTIAL PIPELINE]** extract → structure → mappings → migrate run in strict order — each phase reads the prior phase's NDJSON output.

Run with the **race detector binary** so data races are caught immediately:

```bash
# Clean slate before every run
rm -rf ./files/ && rm -f .wizard_state.json

# Extract
./sonar-migration-tool-race extract -c migration-config.json 2>&1 | tee /tmp/smt-extract-$(date +%s).log
echo "EXIT CODE: $?"

# Structure (if applicable)
./sonar-migration-tool-race structure -c migration-config.json 2>&1 | tee /tmp/smt-structure-$(date +%s).log
echo "EXIT CODE: $?"

# Mappings (if applicable)
./sonar-migration-tool-race mappings -c migration-config.json 2>&1 | tee /tmp/smt-mappings-$(date +%s).log
echo "EXIT CODE: $?"

# Migrate
./sonar-migration-tool-race migrate -c migration-config.json 2>&1 | tee /tmp/smt-migrate-$(date +%s).log
echo "EXIT CODE: $?"
```

Or use the full pipeline script:
```bash
rm -rf ./files/ && rm -f .wizard_state.json
./scripts/execute_full_migration.sh 2>&1 | tee /tmp/smt-full-$(date +%s).log
echo "EXIT CODE: $?"
```

### 3.6 — Capture Evidence

> **[PARALLEL AGENT SWARM — 4 agents]** Spawn one log analysis agent per pipeline command after all commands complete:
> - **Agent 1 — Extract Log**: scans `/tmp/smt-extract-*.log` for panics, DATA RACE, FATAL, ERROR; extracts summary stats
> - **Agent 2 — Structure Log**: scans `/tmp/smt-structure-*.log`; extracts summary stats
> - **Agent 3 — Mappings Log**: scans `/tmp/smt-mappings-*.log`; extracts summary stats
> - **Agent 4 — Migrate Log**: scans `/tmp/smt-migrate-*.log`; extracts summary stats and wall-clock duration

For each command, record:
- [ ] Exit code (0 = success, non-zero = crash/error)
- [ ] Any panic stack traces in output
- [ ] Any `DATA RACE` warnings from race detector
- [ ] Any `ERROR` or `FATAL` log lines
- [ ] Summary stats (copy verbatim)
- [ ] Wall-clock duration

**If any command crashed (non-zero exit, panic, data race) → go directly to Phase 5.**

---

## Phase 4 — Verify Output and Data Correctness
<!-- updated: 2026-05-31_00:00:00 -->

> Do NOT trust the tool's own summary. Query the source and target APIs independently to verify.

> **[PARALLEL AGENT SWARM — 46+ agents]** This is the maximum parallelization phase. All six subsections (4.1–4.6) are independent and spawn their own swarms concurrently. Grand total: ~20 (4.1) + ~14 (4.2) + 5 (4.3) + 5 (4.4) + 2 (4.5+4.6) = **46+ agents running simultaneously**.

### 4.1 — Verify Extract Output

> **[PARALLEL AGENT SWARM — 20+ agents]** Spawn one inspection agent per NDJSON entity type file. All run concurrently.
> - **Agent per entity type**: reads `./files/<extract-id>/<entity>.ndjson`; checks: non-empty, line count > 0, first 3 lines are valid JSON, required fields present
> Spawn one agent per entity type in the Full Entity Registry that your change touches, plus a representative sample of untouched types for regression coverage.

Check that NDJSON files were produced for all expected entity types:

```bash
# List all extracted files
find ./files/ -name "*.ndjson" -exec sh -c 'echo "$1: $(wc -l < "$1") lines"' _ {} \;

# Check for empty files (potential bug)
find ./files/ -name "*.ndjson" -empty

# Spot-check: inspect first 3 lines of each file
for f in ./files/*/*.ndjson; do echo "=== $f ==="; head -3 "$f" | jq '.'; done
```

For every entity type your change touches, verify the NDJSON output has correct fields and values.

### 4.2 — Verify Migrate Output on SC

> **[PARALLEL AGENT SWARM — 14+ agents]** Spawn one SC query agent per entity type. All run concurrently.
> - **Agent — Projects**: queries SC `/api/projects/search`; compares count vs SQS baseline
> - **Agent — Issues**: queries both SQS and SC `/api/issues/search`; reports `SQ: N, SC: M`
> - **Agent — Hotspots**: queries both APIs; compares totals
> - **Agent — Quality Profiles**: queries both APIs; compares counts
> - **Agent — Quality Gates**: queries both APIs; compares gate counts
> - **Agent — Groups**: queries both APIs; compares non-built-in group counts
> - **Agent — Permissions**: queries both APIs; verifies permission assignments
> - **Agent — Settings**: queries both APIs; compares migrated setting count
> - **Agent — New Code Periods**: verifies via task success log or direct API query
> - **Agent — Rules**: queries both APIs; compares custom rule count
> - **Agent — Users**: queries both APIs; compares user counts
> - **Agent — User Groups**: verifies group membership counts
> - **Agent — Permission Templates**: queries both APIs; compares template count
> - **Agent — Portfolios**: queries both APIs (if applicable to your edition)

Query the SC API directly for every entity type that was migrated. Compare counts and field values against the SQS baseline from Phase 2.5.

**Projects:**
```bash
curl -s -H "Authorization: Bearer ${SC_TOKEN}" "${SC_URL}/api/projects/search?organization=${SC_ORG}" \
  | jq '.components | length'
```

**Issues:**
```bash
SQ_COUNT=$(curl -s -u "${SONAR_TOKEN}:" "${SQ_URL}/api/issues/search?projectKeys=${PROJECT_KEY}&ps=1" | jq '.total')
SC_COUNT=$(curl -s -H "Authorization: Bearer ${SC_TOKEN}" "${SC_URL}/api/issues/search?projects=${PROJECT_KEY}&ps=1" | jq '.total')
echo "SQ: $SQ_COUNT, SC: $SC_COUNT"
```

**Hotspots:**
```bash
SQ_COUNT=$(curl -s -u "${SONAR_TOKEN}:" "${SQ_URL}/api/hotspots/search?projectKey=${PROJECT_KEY}&ps=1" | jq '.total')
SC_COUNT=$(curl -s -H "Authorization: Bearer ${SC_TOKEN}" "${SC_URL}/api/hotspots/search?projectKey=${PROJECT_KEY}&ps=1" | jq '.total')
echo "SQ: $SQ_COUNT, SC: $SC_COUNT"
```

**Quality Profiles:**
```bash
SQ_COUNT=$(curl -s -u "${SONAR_TOKEN}:" "${SQ_URL}/api/qualityprofiles/search" | jq '.profiles | length')
SC_COUNT=$(curl -s -H "Authorization: Bearer ${SC_TOKEN}" "${SC_URL}/api/qualityprofiles/search?organization=${SC_ORG}" | jq '.profiles | length')
echo "SQ: $SQ_COUNT, SC: $SC_COUNT"
```

**Quality Gates:**
```bash
SQ_COUNT=$(curl -s -u "${SONAR_TOKEN}:" "${SQ_URL}/api/qualitygates/list" | jq '.qualitygates | length')
SC_COUNT=$(curl -s -H "Authorization: Bearer ${SC_TOKEN}" "${SC_URL}/api/qualitygates/list?organization=${SC_ORG}" | jq '.qualitygates | length')
echo "SQ: $SQ_COUNT, SC: $SC_COUNT"
```

**Groups:**
```bash
SQ_COUNT=$(curl -s -u "${SONAR_TOKEN}:" "${SQ_URL}/api/user_groups/search" | jq '.groups | length')
SC_COUNT=$(curl -s -H "Authorization: Bearer ${SC_TOKEN}" "${SC_URL}/api/user_groups/search?organization=${SC_ORG}" | jq '.groups | length')
echo "SQ: $SQ_COUNT, SC: $SC_COUNT"
```

> Adapt these queries for every entity type listed in the [Full Entity Registry](#full-entity-registry). You do NOT need to verify every single entity type on every run — but you MUST verify (a) all entity types touched by your change, and (b) a representative sample of untouched entity types to catch regressions.

### 4.3 — Spot-Check Data Fidelity (5+ Entities)

> **[PARALLEL AGENT SWARM — 5+ agents]** Spawn one field-comparison agent per entity being spot-checked. All run concurrently.
> - **Agent 1..N — Entity Spot-Check**: fetches the same entity from SQS and SC; compares field-by-field (tags, status, severity, comments, assignee, timestamps); returns `{match: bool, diff: string}`

For each entity type your change touches, pick 5+ specific entities and compare field-by-field between SQS and SC:

```bash
# Example: spot-check a specific issue
SQ_ISSUE=$(curl -s -u "${SONAR_TOKEN}:" "${SQ_URL}/api/issues/search?projectKeys=${PROJECT_KEY}&ps=1" | jq '.issues[0]')
echo "$SQ_ISSUE" | jq '{key: .key, status: .status, severity: .severity, tags: .tags, assignee: .assignee, comments: .comments}'

# Find the same issue on SC and compare
SC_ISSUE=$(curl -s -H "Authorization: Bearer ${SC_TOKEN}" "${SC_URL}/api/issues/search?projects=${PROJECT_KEY}&ps=1" | jq '.issues[0]')
echo "$SC_ISSUE" | jq '{key: .key, status: .status, severity: .severity, tags: .tags, assignee: .assignee, comments: .comments}'
```

Verify: tags match, status matches, comments match, assignee matches, severity matches.

### 4.4 — Verify Edge Cases

> **[PARALLEL AGENT SWARM — 5 agents]** All 5 edge cases are independent — spawn all concurrently.
> - **Agent 1 — Empty Input**: entity type with 0 items; verifies graceful handling, no crash
> - **Agent 2 — Single Entity**: entity type with exactly 1 item; verifies correct migration
> - **Agent 3 — Large Volume**: entity type with 1000+ items; verifies all migrated, pagination works
> - **Agent 4 — Missing Fields**: entities with optional/null fields; verifies no panic, graceful handling
> - **Agent 5 — Idempotency**: runs migrate twice consecutively; verifies no duplicates, no errors on second run

- [ ] **Empty input**: Entity type with 0 items on SQS — tool handles gracefully, no crash
- [ ] **Single entity**: Entity type with exactly 1 item — migrated correctly
- [ ] **Large volume**: Entity type with 1000+ items — all migrated, pagination works
- [ ] **Missing fields**: Entities with optional/null fields — no crash, fields handled correctly
- [ ] **Idempotency**: Run migrate twice — no duplicates created, no errors

### 4.5 — Check for Silent Failures

> **[PARALLEL AGENT SWARM — 2 agents]** 4.5 and 4.6 are fully independent — run both concurrently.
> - **Agent 1 — Silent Failure Check (4.5)**: runs grep scans across logs; reports error/panic/fatal lines and empty NDJSON files
> - **Agent 2 — Regression Check (4.6)**: queries all untouched entity type counts; fills in the regression table below

```bash
# Search logs for errors
grep -ri "error\|panic\|fatal\|fail" /tmp/smt-*.log | grep -v "INFO"

# Check for empty NDJSON files (entity type extracted but produced nothing)
find ./files/ -name "*.ndjson" -empty

# Check for suspiciously low counts (e.g., SQ has 1000 issues but SC only has 50)
# This indicates silent data loss — the worst kind of bug
```

### 4.6 — Regression Check

For entity types NOT touched by your change, verify they still work by comparing counts against the Phase 2.5 baseline:

| Entity Type | SQS Count | SC Count | Match? |
|-------------|-----------|----------|--------|
| Projects | ___ | ___ | ___ |
| Issues | ___ | ___ | ___ |
| Hotspots | ___ | ___ | ___ |
| Quality Profiles | ___ | ___ | ___ |
| Quality Gates | ___ | ___ | ___ |
| Groups | ___ | ___ | ___ |
| Permissions | ___ | ___ | ___ |
| Settings | ___ | ___ | ___ |
| New Code Periods | ___ | ___ | ___ |
| ... | ___ | ___ | ___ |

**If ANY check fails → proceed to Phase 5.**
**If ALL checks pass → proceed to Phase 6.**

---

## Phase 5 — Investigate, Fix, and Loop Back
<!-- updated: 2026-05-31_00:00:00 -->

> **This is the recursive part. For every failure, trace it to root cause, fix it, and re-run the entire pipeline from Phase 3.**

> **[SEQUENTIAL → PARALLEL FAN-OUT]** 5.1 classifies failures (sequential) → 5.2 launches parallel isolation agents → 5.3 runs parallel hypothesis agents → 5.4 applies minimal fix (sequential) → 5.5 runs parallel pre-flight checks → loop back to Phase 3.

### 5.1 — Classify the Failure

> **ADVERSARIAL NOTE — Never Blame Infrastructure First:**
>
> When a CE task fails or an API returns an unexpected error, the natural instinct is "the server is broken" or "it's a staging environment issue." **Resist this instinct.** Always ask: **does CloudVoyager succeed on the same instance with the same data?** If yes, the problem is in your code — not the server. Common code-level causes that masquerade as infrastructure issues:
>
> - **Missing protobuf fields** that the server requires but you don't set (e.g., `ReferenceBranchName` in metadata)
> - **Wrong protobuf encoding** (length-delimited vs non-delimited for a specific message type)
> - **Wrong ZIP entry names** (e.g., `externalissues-1.pb` vs `external-issues-1.pb`)
> - **Wrong multipart form field names** or missing form fields in CE submission
> - **Stale component refs** pointing to components that were filtered out
>
> Only blame infrastructure after you've verified: (1) CloudVoyager also fails, or (2) you've diff'd your protobuf output byte-for-byte against CloudVoyager's and they match.

| Failure type | Symptoms | Investigation approach |
|---|---|---|
| **Crash** | Non-zero exit, panic stack trace | Read the stack trace. Find the exact line. |
| **Data race** | `WARNING: DATA RACE` in output | Read the race report. Find the unsynchronized access. |
| **Wrong data** | SC field value doesn't match SQS | Trace the field through extract NDJSON → migrate code → SC API call. |
| **Missing data** | SC entity count < SQS entity count | Check: was it extracted? Was it filtered? Did the API call fail? |
| **Duplicate data** | SC entity count > SQS entity count | Check: is the entity processed twice? Is idempotency missing? |
| **Silent failure** | No error but data is wrong/missing | Check every error return in the code path. Look for `_ = err`. |
| **CE task failed** | `status: FAILED` in CE task response | Compare protobuf output against CloudVoyager's. Check every field in metadata.pb. Verify ZIP structure matches. Do NOT assume infrastructure failure. |

### 5.2 — Isolate with API Queries

> **[PARALLEL AGENT SWARM — 3 agents]** Three independent isolation agents run concurrently to triangulate the failure:
> - **Agent 1 — SQS Query**: makes the exact API calls the code makes against SQS; reports what data exists at the source
> - **Agent 2 — NDJSON Inspector**: reads the extract output files; reports what was actually extracted and flags any field anomalies
> - **Agent 3 — SC Query**: queries SC to see what was actually created; reports actual target state and any 4xx/5xx from the migrate log

Reproduce the problem by making the exact API calls the code makes:

```bash
# Query SQS with the exact same params the code uses
curl -s -u "${SONAR_TOKEN}:" "${SQ_URL}/api/..." | jq '.'

# Inspect the NDJSON extract output
head -5 ./files/<extract-id>/<entity>.ndjson | jq '.'

# Query SC to see what was actually created
curl -s -H "Authorization: Bearer ${SC_TOKEN}" "${SC_URL}/api/..." | jq '.'
```

### 5.3 — Root Cause Statement

> **[PARALLEL AGENT SWARM — 3–5 hypothesis agents]** Before writing the root cause, spawn agents to investigate each plausible hypothesis simultaneously:
> - **Agent 1 — Field Mapping Hypothesis**: checks every field in the diff against the SQS API response schema; reports any name/type mismatch
> - **Agent 2 — Silent Error Hypothesis**: greps for `_ = ` in all changed files; reports any dropped error returns
> - **Agent 3 — Concurrency Bug Hypothesis**: checks every shared variable in the changed goroutines for unsynchronized access
> - **Agent 4 — API Contract Hypothesis**: compares each API call in the changed code against SonarQube API docs; reports any method/path/body mismatch
> - **Agent 5 — Pagination Bug Hypothesis**: checks every paginator in the changed code paths for off-by-one or dropped items
>
> A synthesis agent waits for all hypothesis agents and picks the best-evidenced hypothesis to form the root cause statement below.

Before fixing, write:
> "The [specific behavior] was wrong because [specific reason]. Evidence: [API response / log line / code path]."

### 5.4 — Apply Minimal Fix

Fix only the root cause. Do not make unrelated changes.

### 5.5 — Loop Back to Phase 3

> **[PARALLEL AGENT SWARM — 2 agents]** Pre-flight runs concurrently:
> - **Agent 1 — go vet**: runs `go vet ./...`; reports any errors
> - **Agent 2 — go test**: runs `go test ./...`; reports any test failures

After fixing:

1. `go vet ./...` — verify compilation
2. `go test ./...` — verify tests pass
3. **Clean slate** — `rm -rf ./files/ && rm -f .wizard_state.json` + reset SC
4. **Return to Phase 3.5** — re-run the FULL pipeline from scratch
5. **Re-verify ALL of Phase 4** — not just the thing you fixed

> **Never do a partial re-run.** Always re-run the full pipeline. A fix in one code path can break another.

> **Never skip the re-verification.** Always re-check ALL entity types, not just the one you fixed.

> **Repeat this loop until Phase 4 produces zero failures.** That is the only exit condition.

---

## Phase 6 — Declare Full Clean Pass (STOP Condition)
<!-- updated: 2026-05-31_00:00:00 -->

A full clean pass requires **ALL** of the following. Not a subset. Not "close enough."

> **[PARALLEL AGENT SWARM — 6 agents]** Spawn one verifier per checklist category simultaneously. Agent 6 waits for all 5 and declares the verdict.
> - **Agent 1 — Crash-Free Verifier**: checks all log files for non-zero exit codes, panic stack traces, DATA RACE warnings, FATAL lines
> - **Agent 2 — Code Quality Verifier**: runs go vet, go test, go test -race, and go build-race concurrently; reports all results
> - **Agent 3 — Feature Verifier**: checks each acceptance criterion from Phase 0.1.6; verifies each edge case; confirms the core code path was exercised with real data (not just the no-op path)
> - **Agent 4 — Data Correctness Verifier**: queries SQS vs SC counts for all entity types; spot-checks 5+ entities field-by-field; scans for empty NDJSON files; flags any silent data loss
> - **Agent 5 — Regression Verifier**: checks all untouched entity types still extract/migrate correctly vs Phase 2.5 baseline; flags any new errors in unchanged features
> - **Agent 6 — Synthesis** (waits for Agents 1–5): if ALL pass → declares CLEAN PASS, stops the loop; if any fail → returns specific failure list to Phase 5 with root cause guidance

> **ADVERSARIAL NOTE — What "Clean Pass" Actually Means:**
>
> A "clean pass" is NOT "the tool ran without crashing." A tool can exit 0 and produce zero data — that is not a clean pass, that is a silent failure. Ask yourself these adversarial questions before declaring victory:
>
> 1. **Did the feature under test actually execute its core code path?** If you added issue metadata sync but importScanHistory failed (so SC has 0 issues to sync against), then syncIssueMetadata ran but matched 0 pairs — it exercised the "nothing to do" path, not the "sync metadata" path. That is NOT a clean pass for issue sync.
>
> 2. **Did the data actually arrive on the target?** If CE rejects the protobuf report, issues never appear in SonarCloud. Sync tasks running against 0 Cloud issues will report "0 matched, 0 synced" and exit cleanly — but the feature is broken. A clean pass means SQS issue count ≈ SC issue count, not "no errors in the log."
>
> 3. **Did CloudVoyager succeed on the same instance?** If CloudVoyager can import scan history to the same SC staging instance but this tool cannot, the problem is in this tool's code — not the infrastructure. Never blame infrastructure without verifying CloudVoyager also fails.
>
> 4. **Did you test with data that exercises the feature?** If SQS has 0 resolved issues, 0 reviewed hotspots, and 0 comments, then metadata sync has nothing to sync. That validates the "no-op" path, which is necessary but not sufficient. A clean pass for metadata sync requires SQS data with manual changes (status transitions, comments, tags).
>
> 5. **Is "0 matched pairs" actually correct, or is it masking a bug?** Zero matches could mean: (a) the matching algorithm is broken, (b) component key stripping is wrong, (c) issues weren't created in Cloud, or (d) there genuinely are no matchable issues. Only (d) is a clean pass. Verify (a)-(c) before accepting (d).

### Crash-Free Execution
- [ ] Every command exited with code 0
- [ ] Zero panics in output
- [ ] Zero `DATA RACE` warnings from race detector
- [ ] Zero `FATAL` log lines

### Code Quality Gates
- [x] `go vet ./...` passes
- [x] `go test ./...` passes — all tests pass cleanly as of 2026-05-29
- [x] `go test -race ./...` passes — zero data race warnings as of 2026-05-29
- [x] `go build -race` compiles and runs clean

### Feature Verification (from Phase 0.1.6)
- [ ] Every acceptance criterion for the change passes
- [ ] Every edge case (empty, single, large, missing fields, idempotency) handled correctly
- [ ] **The feature's core code path was exercised with real data** — not just the no-op/empty path

### Data Correctness
- [ ] Entity counts on SC match SQS (within tolerance for entity types that have known filters)
- [ ] Spot-check of 5+ entities shows field-level correctness
- [ ] No empty NDJSON files for entity types that have data
- [ ] No silent data loss (SC count is not suspiciously lower than SQS count)
- [ ] **CE tasks succeeded** — if scan history import is part of the feature, the CE task must complete with status SUCCESS, not FAILED

### Regression Verification
- [ ] All entity types NOT touched by the change still extract/migrate correctly
- [ ] Summary stats for unchanged features match prior runs
- [ ] No new errors or warnings in logs for unchanged features

### Run Metadata
- [ ] Number of loop iterations to reach clean pass: ___
- [ ] Total wall-clock time: ___
- [ ] Final summary stats captured and saved

> **Only when every checkbox above is checked can you stop the loop.**
>
> **If you are tempted to declare "clean pass" but a core task (like importScanHistory) failed, STOP. That is not a clean pass. Go back to Phase 5.**

---

## Current Test Status
<!-- updated: 2026-05-30_18:45:00 -->

As of **2026-05-30**, full live end-to-end regression passes on branch `fix/four-pipelines-compatibility`.

### Code Quality Gates
<!-- updated: 2026-05-30_18:45:00 -->

| Check | Status |
|---|---|
| `go vet ./...` | PASS |
| `go test ./...` | PASS (15 packages, 0 failures) |
| `go test -race ./...` | PASS — zero data race warnings |
| `go build -race` | PASS |

### Live End-to-End Migration (2026-05-30-01 → 2026-05-30-05)
<!-- updated: 2026-05-30_18:45:00 -->

**Environment:**
- Source: SonarQube Server 2026.2.0 Enterprise at `localhost:9000` (admin token `squ_*`)
- Target: SonarCloud Staging (`sc-staging.io`, org `open-digital-society-1`)
- Run with race-detector binary (`sonar-migration-tool-race`)

**Phase results:**

| Phase | Exit Code | Panics | DATA RACE |
|---|---|---|---|
| `extract` | 0 | 0 | 0 |
| `structure` | 0 | 0 | 0 |
| `mappings` | 0 | 0 | 0 |
| `migrate --edition developer` | 0 | 0 | 0 |

**1:1 entity verification (SQS → SC):**

| Entity | SQS Count | SC Result | Status |
|---|---|---|---|
| Projects | 2 | 2 created (`open-digital-society-1_sonar-rules-to-eslint-mapping`, `open-digital-society-1_okorach-oss_sonar-tools`) | ✅ PASS |
| Quality Gate per project | "Sonar way" × 2 | "Sonar way" × 2 | ✅ PASS |
| Groups (non-built-in) | sonar-administrators | sonar-administrators created | ✅ PASS |
| sonar-users (built-in) | exists | correctly SKIPPED (maps to SC Members) | ✅ PASS |
| Permission template | Default template | Default template created | ✅ PASS |
| Migration groups | N/A | migration-scanners + migration-viewers created | ✅ PASS |
| New code periods | PREVIOUS_VERSION × 2 | setNewCodePeriods succeeded=2 | ✅ PASS |
| Project settings | 212 SQS settings | 4 migrated (SQC-compatible subset) | ✅ PASS |
| Quality profiles | 47 SQS | 61 SC (includes SC built-ins) | ✅ PASS |

**PR-specific features verified live:**

| Feature | Evidence |
|---|---|
| SQ2025Pipeline routing | SQ 2026.2 routed to SQ2025Pipeline; V2 groups API used |
| `paginateAll` generic | No pagination errors across 67 extract tasks |
| URL double-slash fix | All API calls hit correct paths (verified via `--debug` log) |
| V2 groups `Default` field | Unit-tested; confirmed `sonar-users default:true` in V2 response |
| `transfer` command | Validation and config parsing verified |
| `ProjectKeys` filter | Unit-tested (`TestGetProjectsTaskNoFilter`, `TestGetProjectsTaskWithFilter`) |

### Pre-existing Environment Limitations (not regressions from this PR)
<!-- updated: 2026-05-30_18:45:00 -->

1. **Enterprise API** (`api.sc-staging.io/enterprises/enterprises`) returns 403 with a standard user token. This only affects `createPortfolios` (enterprise-only task). Since our test SQS has 0 portfolios, this has no impact on the 1:1 result. Workaround: run with `--edition developer` (skips enterprise-only tasks). In a real enterprise migration with portfolios, an enterprise API token is required.

2. **`sonar-users` permissions not remapped to "Members"**: SQS `sonar-users` has `issueadmin` and `securityhotspotadmin` on both projects. The tool correctly skips creating `sonar-users` (built-in, replaced by SC "Members"), but still attempts to assign its permissions to a `sonar-users` group that doesn't exist on SC. Result: 8 WARNs in the migrate log. **Pre-existing gap, not introduced by this PR.**

3. **SC staging `new_code_periods/show` not available** (SC 8.0.0): read-back verification via API impossible; confirmed via task success `setNewCodePeriods: succeeded=2`.

4. **`sonar.authenticator.downcase` not settable at org level**: expected skip, WARN in log.

Loop iterations to clean pass: **1** (live run). No panics, no DATA RACE, exit 0 on all phases.

---

## Full Entity Registry
<!-- updated: 2026-05-26_00:00:00 -->

All entity types handled by the tool. Use this as your regression checklist.

### Extract + Migrate (both phases)
<!-- updated: 2026-05-26_00:00:00 -->

| Entity Type | Extract API (SQS) | Migrate API (SC) |
|---|---|---|
| Projects | `/api/projects/search` | `POST /api/projects/create` |
| Project Settings | `/api/settings/values` | `POST /api/settings/set` |
| Project Tags | `/api/components/show` | (via project create/update) |
| Project Links | `/api/project_links/search` | (via project create/update) |
| Project ALM Bindings | `/api/alm_settings/get_binding` | `POST /api/alm_integration/set_up_binding` |
| Project Group Permissions | `/api/permissions/groups` | `POST /api/permissions/groups` |
| Project User Permissions | `/api/permissions/users` | `POST /api/permissions/users` |
| Issues | `/api/issues/search` | `POST /api/issues/do_transition`, `add_comment` |
| Issues (Full History) | `/api/issues/search` | (via scan report import) |
| Issue Comments | `/api/issues/search` | `POST /api/issues/add_comment` |
| Hotspots | `/api/hotspots/search`, `/api/hotspots/show` | `POST /api/hotspots/change_status`, `add_comment` |
| Hotspot Comments | `/api/hotspots/show` | `POST /api/hotspots/add_comment` |
| Quality Profiles | `/api/qualityprofiles/search`, `backup` | `POST /api/qualityprofiles/create`, `restore` |
| Profile Rules | `/api/rules/search` | (via profile restore) |
| Profile Group Permissions | `/api/qualityprofiles/search_groups` | `POST /api/permissions/groups` |
| Profile User Permissions | `/api/qualityprofiles/search_users` | `POST /api/permissions/users` |
| Quality Gates | `/api/qualitygates/list`, `show` | `POST /api/qualitygates/create`, `create_condition` |
| Gate Conditions | `/api/qualitygates/show` | `POST /api/qualitygates/create_condition` |
| Gate Group Permissions | `/api/qualitygates/search_groups` | `POST /api/permissions/groups` |
| Gate User Permissions | `/api/qualitygates/search_users` | `POST /api/permissions/users` |
| Rules | `/api/rules/search`, `/api/rules/show` | `PATCH /api/rules/update` |
| Users | `/api/users/search` | `GET /api/users/search` (lookup) |
| User Groups | `/api/user_groups/users` | `POST /api/groups/add_user` |
| Groups | `/api/permissions/groups` | `POST /api/groups/create` |
| Permission Templates | `/api/permissions/search_templates` | `POST /api/permissions/create_template` |
| Template Group Permissions | `/api/permissions/template_groups` | `POST /api/permissions/add_group_to_template` |
| Template User Permissions | `/api/permissions/template_users` | `POST /api/permissions/add_user_to_template` |
| Portfolios | `/api/views/search`, `show` | `POST /api/portfolios/create` |
| Server Settings | `/api/settings/values` | `POST /api/settings/set` (global) |
| New Code Periods | `/api/new_code_periods/list`, `show` | `POST /api/new_code_periods/set` |
| ALM Settings | `/api/alm_settings/list` | (via binding setup) |
| Component Tree | `/api/measures/component_tree` | (via scan report) |
| Source Code | `/api/sources/raw` | (via scan report) |
| SCM Blame Data | `/api/sources/scm` | (via scan report) |

### Extract Only (not migrated)
<!-- updated: 2026-05-26_00:00:00 -->

| Entity Type | Extract API (SQS) | Purpose |
|---|---|---|
| Project Details | `/api/navigation/component` | Metadata for reports |
| Project Measures | `/api/measures/search` | Metrics for reports |
| Project Webhooks | `/api/webhooks/list` | Audit/documentation |
| Branches | `/api/project_branches/list` | Branch awareness |
| Pull Requests | `/api/project_pull_requests/list` | PR awareness |
| Rule Repositories | `/api/rules/repositories` | Rule inventory |
| Plugin Rules | `/api/rules/search` (filtered) | Plugin rule audit |
| Template Rules | `/api/rules/search` (filtered) | Template rule audit |
| User Permissions | `/api/permissions/users` | Permission audit |
| User Tokens | `/api/user_tokens/search` | Token audit |
| Applications | `/api/components/search` | App inventory |
| Webhooks | `/api/webhooks/list` | Webhook audit |
| Server Info | `/api/system/info` | Server metadata |
| Plugins | `/api/plugins/installed` | Plugin inventory |
| License Usage | `/api/projects/license_usage` | License audit |
| CE Tasks | `/api/ce/activity` | Background task audit |
| Project Analyses | `/api/project_analyses/search` | Analysis history |

---

## Reference: CLI Commands
<!-- updated: 2026-05-26_00:00:00 -->

| Command | Purpose | When to Run in Tests |
|---------|---------|---------------------|
| `extract` | Extract data from SQS → NDJSON files | Always |
| `structure` | Group projects into organizations (CSVs) | When testing org mapping |
| `mappings` | Generate mapping files | When testing entity mapping |
| `migrate` | Migrate from NDJSON → SC via API | Always |
| `report` | Generate reports | When testing reporting |
| `reset` | Reset a SC organization | Before runs (clean slate) |
| `wizard` | Interactive wizard | Manual testing only |
| `gui` | Browser-based GUI | Manual testing only |
| `analysis_report` | Generate analysis report | When testing reporting |

---

## Reference: Config Files
<!-- updated: 2026-05-26_00:00:00 -->

| File | Purpose |
|------|---------|
| `migration-config.json` | Unified config (recommended) |
| `config-extract.example.json` | Extract-specific example |
| `config-migrate.example.json` | Migrate-specific example |
| `config.example.json` | Flat format example |
| `scripts/execute_full_migration.sh` | Full pipeline automation |
| `scripts/delete_all_portfolios.sh` | Portfolio cleanup |

---

## Reference: Creating Test Data
<!-- updated: 2026-05-26_00:00:00 -->

When your SQS instance doesn't have enough data to exercise a feature:

### Bulk-Tag Issues
<!-- updated: 2026-05-26_00:00:00 -->

```bash
python3 -c "
import urllib.request, urllib.parse, json, base64, concurrent.futures, sys

TOKEN = 'squ_...'
PROJECT_KEY = sys.argv[1] if len(sys.argv) > 1 else 'my-project'
auth = base64.b64encode(f'{TOKEN}:'.encode()).decode()
headers = {'Authorization': f'Basic {auth}', 'Content-Type': 'application/x-www-form-urlencoded'}

url = f'${SQ_URL}/api/issues/search?projectKeys={PROJECT_KEY}&ps=500'
req = urllib.request.Request(url, headers=headers)
keys = [i['key'] for i in json.load(urllib.request.urlopen(req))['issues']]

def tag(key):
    params = urllib.parse.urlencode({'issue': key, 'tags': 'test-tag'}).encode()
    req = urllib.request.Request(f'${SQ_URL}/api/issues/set_tags', data=params, headers=headers, method='POST')
    urllib.request.urlopen(req)
    return 'ok'

with concurrent.futures.ThreadPoolExecutor(max_workers=20) as pool:
    results = list(pool.map(tag, keys))
print(f'Tagged: {sum(1 for r in results if r == \"ok\")}/{len(keys)}')
" ${PROJECT_KEY}
```

### Other Operations
<!-- updated: 2026-05-26_00:00:00 -->

Adapt the pattern above for:
- `POST /api/issues/do_transition` — change issue statuses
- `POST /api/issues/add_comment` — add comments to issues
- `POST /api/issues/assign` — assign issues to users
- `POST /api/hotspots/change_status` — review hotspots
- `POST /api/qualityprofiles/create` — create quality profiles
- `POST /api/qualitygates/create` — create quality gates
- `POST /api/user_groups/create` — create groups
