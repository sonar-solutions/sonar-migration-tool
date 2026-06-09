# `transfer` Command — Internal Flow (ASCII Chart)
<!-- updated: 2026-06-09_21:35:54 -->

How `sonar-migration-tool transfer` works end-to-end, traced from
[cmd/transfer.go](../go/cmd/transfer.go) down through `extract` → `structure` →
`mappings` → `migrate` → report emission. Verified by a 4-agent parallel trace
of the real call graph (workflow run `wf_471cf8cf-f9c`, 2026-06-09).

**Reading the chart:** `=>` data-on-disk handoff (JSONL/CSV); `->` in-process
call; `[N/4]` is the user-facing phase banner; boxes are package boundaries.
The whole command is bookended by a single deferred
`common.LogCommandDuration(slog, "transfer", t0)` registered as the first
statement of `runTransfer`.

```
================================================================================

   sonar-migration-tool transfer            cmd/transfer.go : runTransfer (L344)

   defer LogCommandDuration("transfer", t0)   <-- fires LAST, bookends everything

================================================================================
        |
        v

   resolveTransferConfig(cmd)          merge --config file + CLI flags (flags win)
   |
   |    loadTransferFileDefaults --> extract.LoadExtractConfigFile  (source.*)
   |                                 migrate.LoadMigrateConfigFile   (target.*)
   |                                 loadTransferOverlay             (project_key)
   |
   |    one-way skip flags:  --skip_issue_sync  /  --skip_project_data_migration
   |
   |    enterprise_key  defaults to  default_organization
   |    exportDir       default      ./migration-files/
        |
        v

   validateTransferConfig(cfg)         source  url + token   required
                                       target  token + org   required
        |
        v

+==============================================================================+
|                                                                              |
|   [1/4]  EXTRACT     extract.RunExtract  (internal/extract/extract.go)       |
|                                                                              |
|   pull SonarQube Server  ->  JSONL files on disk                             |
|                                                                              |
+==============================================================================+
        |
        v

   projectKey != ""  ?  ProjectKeys=[key]  :  nil (= all)
   IncludeProjectData = !skipProjectDataMigration
        |
        +-- applyDefaults:  concurrency=25, timeout=60, exportType="all"
        |
        +-- initClient
        |       ->  GET api/server/version          (detectVersion -> Version)
        |       ->  GET api/system/info  --|403|-->  api/navigation/global
        |                                            (detectEdition)
        |
        +-- prepareExtractDir  ->  mkdir exportDir/<runID>/   (runID = date-NN, max+1)
        |
        +-- buildPlan:  RegisterAll -> Registry -> FilterByEdition(edition)
        |       targets = ALL "get*" tasks   (minus 6 projectData tasks if skip)
        |       ResolveDependencies -> PlanPhases (Kahn topo-sort -> [][]string)
        |
        +-- writeMetadataFile                                   => extract.json
        |
        +-- NewDataStore + filterCompleted   (skip task dirs that already exist)
        |
        +-- newExecutor  (Sem = chan struct{}, cap=concurrency; .ProjectKeys set)
        |
        v

   executePhases  (strictly ordered)  ->  runPhase (errgroup, limit = cap Sem)
        |
        v
  +----------------------------------------------------------------------------+
  |                                                                            |
  | P1   getProjects   GET api/projects/search?projects=<keys>   <== SCOPING   |
  |                                                 => getProjects/*.jsonl     |
  |                                                                            |
  |       every later per-project task ReadAll "getProjects" from disk         |
  |                                                                            |
  | per-project   (forEachDep, 1 goroutine / proj, sem-bounded):               |
  |                                                                            |
  |       getProjectSettings / Links / Measures / Webhooks / Bindings /        |
  |       Details / Tags / GroupsPermissions / UsersScanners                   |
  |                                          => <task>/*.jsonl                 |
  |                                                                            |
  |       403 / 404  ->  RecordSkipped(key)  ->  dropped from all later tasks  |
  |                                                                            |
  | project-data   (IF IncludeProjectData)   forEachProjectBranch:             |
  |                                                                            |
  |       getProjects x getBranches  (drop type==SHORT, branches sequential)   |
  |       getProjectIssuesFull / HotspotsFull / ComponentTree / Versions       |
  |       ComponentTree => getProjectSourceCode + getProjectSCMData (per file) |
  |       hotspots enriched via api/hotspots/show  (rule.key -> ruleKey)       |
  |       version-gate: statuses vs issueStatuses @ SQ 10.4                    |
  |                                          => <task>/*.jsonl                 |
  |                                                                            |
  +----------------------------------------------------------------------------+
        |
        | MarkComplete each task after its phase  (any task error aborts extract)
        v

   return SkippedProjectKeys()  ->  warnSkippedProjects (stderr, NON-fatal)
        |
        v

+==============================================================================+
|                                                                              |
|   [2/4]  STRUCTURE   structure.RunStructure  (internal/structure)            |
|                                                                              |
|   pure local JSON  ->  CSV     ;     NO API calls                            |
|                                                                              |
+==============================================================================+
        |
        v

   GetUniqueExtracts(exportDir)        <= reads <id>/extract.json (url field)
   |                                      serverURL -> newest extractID
        |
        v

   MapProjectStructure   <= getNewCodePeriods, getBindings, getProjectBindings,
   |                         getProjectDetails (*.jsonl)  -> []Binding, []Project
   |
   |    blank branch -> master ;  REFERENCE_BRANCH / SPECIFIC_ANALYSIS dropped
        |
        v

   MapOrganizationStructure(bindings, cfg.defaultOrganization)   <== KEY STEP
   |
   |    sonarcloud_org_key = defaultOrganization  for EVERY row (pre-filled)
        |
        v

   ExportCSV       => organizations.csv   (sc_org_key filled)
                   => projects.csv
        |
        v   (sequential — mappings needs projects.csv)

+==============================================================================+
|                                                                              |
|   [3/4]  MAPPINGS    structure.RunMappings  (no defaultOrganization)         |
|                                                                              |
+==============================================================================+
        |
        v

   GetUniqueExtracts(exportDir)
   LoadCSV(projects.csv) -> projectOrgMapping  (server_url+key -> sonarqube_org)
        |
        v   (Go data-dependency order, NOT randomized map order)

   MapTemplates ---+
   MapProfiles  ---+--> feed --> MapGroups   (consumes profiles[] + templates[])
   MapGates        |
   MapPortfolios   |   (deduped by SHA-256 of sorted project composition)
        |
        v

   ExportCSV  x5   => templates.csv  /  profiles.csv  /  gates.csv  /
                      portfolios.csv /  groups.csv
        |
        v

+==============================================================================+
|                                                                              |
|   [4/4]  MIGRATE     migrate.RunMigrate  (internal/migrate/migrate.go)       |
|                                                                              |
|   TargetTasks = transferTargetTasks  (14 project-scoped leaves)              |
|   DefaultOrganization left UNSET  (structure already stamped sc_org_key)     |
|                                                                              |
+==============================================================================+
        |
        v

   applyDefaults  (concurrency=25, url=sonarcloud.io, edition=enterprise)
   applyOrgMapping  ->  validateOrgsExist
        |
        |    build 2 Cloud clients:
        |        cloudClient @ sonarcloud.io
        |        apiClient   @ api.sonarcloud.io
        |        (+ WithRetryLogger + WithRateLimitObserver -> RateLimitTracker)
        |
        |    runID = generateRunID(dir) = "YYYY-MM-DD-NN", NN = maxN+1 (gap-safe #361)
        |    RegisterAll -> BuildMigrateRegistry -> FilterByEdition
        |    if SkipProjectDataMigration: force SkipIssueSync=true
        |
        v

   MigrateTargetTasks:  TargetTasks non-empty -> return the 14 leaves AS-IS,
   |                    minus skip-gated tasks (importPD / sync* if skipped)
        |
        v

   ResolveDependencies  (DFS transitive closure)            ->  ~25-task set
   PlanPhases  (Kahn topo-sort, alphabetical within phase)  ->  6 phases
        |
        v

   writeMigrateMeta  => plan.json
   filterCompleted   (resume: skip existing dirs)
        |
        v   for each phase: runPhase (errgroup, 1 goroutine PER TASK, no limit)
  +----------------------------- THE 6-PHASE DAG ------------------------------+
  |                                                                            |
  | P1   generate{Project,Profile,Gate,Group,Organization}Mappings   (no deps) |
  |         |                                                                  |
  | P2   createProjects, createProfiles, createGates, createGroups,            |
  |       getMigrationUser                                                     |
  |         |                                                                  |
  | P3   analyzeProfileRules, getGateConditions, getProfileBackups,            |
  |       grantMigrationUserProjectPermissions, setProfileParent               |
  |         |                                                                  |
  | P4   restoreProfiles, addGateConditions     <== profiles + gate configured |
  |       setProjectProfiles / Gates / GroupPermissions / Settings /           |
  |         Tags / Links / Webhooks, setNewCodePeriods  BEFORE scan replay (P5)|
  |         |                                                                  |
  | P5   importProjectData                                                     |
  |         |                                                                  |
  |       per-project fan-out  (g.SetLimit = cap Sem):                         |
  |         |                                                                  |
  |         collectBranchInfo (LONG only) -> sortBranchesMainFirst             |
  |           -> filterBranches(ExcludeBranches globs; main never excluded)    |
  |         |                                                                  |
  |         MAIN branch first = BLOCKING GATE  (fail -> non-main "skipped")    |
  |         |                                                                  |
  |         non-main branches SEQUENTIAL:                                      |
  |           buildBranchReport: load issues / hotspots / components /         |
  |             sources / activeRules ;  skip empty/purged ;                   |
  |             remap SQ->SC profile keys ;  dedup active rules ;              |
  |             drop issues on inactive rules ;                                |
  |             BackdateChangesets -> original creation dates                  |
  |         |                                                                  |
  |           non-main: PreCreateAnalysis POST {api}/analysis/analyses         |
  |             -> analysisUuid stamped into metadata field 19,                |
  |                BranchType="long", reference=MAIN (field 11)                |
  |         |                                                                  |
  |           scanreport.PackageReport (ZIP/Deflate) -> SubmitReport           |
  |             -> /api/ce/submit -> PollCETask        => importProjectData/*  |
  |         |                                                                  |
  | P6   syncIssueMetadata || syncHotspotMetadata  (concurrent; dep importPD)  |
  |         |                                                                  |
  |       forEachMigrateItem(createProjects):                                  |
  |         loadMatchable* (issues: exclude CLOSED/FIXED; hotspots: REVIEWED)  |
  |         actionable filter (manual changes / comments)                      |
  |         waitForCloudIndexing (exp backoff, non-fatal)                      |
  |         runProjectSyncLoop (cap Sem):                                      |
  |           ISSUES:   search componentKeys=key:file & rules=rule &           |
  |             issueStatuses=...  -> classify by line                         |
  |             (1=sync, 0=miss, >1=ambig)                                     |
  |             -> transition -> comments -> tags (+metadataSyncTag idem)      |
  |           HOTSPOTS: search projectKey & files  (NO rules param)            |
  |             -> classify by (ruleKey, line) -> ChangeStatus -> comments     |
  |                                              => sync*Metadata/*.jsonl      |
  |                                                                            |
  +----------------------------------------------------------------------------+
        |
        | deferred (every exit):  run_meta.json, run_events.jsonl,
        |                         rate_limit_events.json
        v

   return runID   (returned even on phase error, so reports still emit)
        |
        v

+==============================================================================+
|                                                                              |
|   REPORTS            emitReports  ->  summary.GenerateReports                |
|                                                                              |
|   fire-and-forget: report failure NEVER fails the transfer                   |
|                                                                              |
+==============================================================================+
        |
        v

   runDir = exportDir/<runID>
   summary.GenerateReports(runDir, outputDir=exportDir, exportDir)
        |
        +-- CollectSummary(runDir, exportDir)   <= task JSONL + requests.log +
        |       run_meta.json + extract JSONL  -> *MigrationSummary (1 pass)
        |
        +-- RenderPDF       -> os.WriteFile  => <exportDir>/migration_summary.pdf
        |
        +-- RenderMarkdown  -> os.WriteFile  => <exportDir>/migration_summary.md
        |
        v

   println "Transfer complete."
        |
        v

   (defer fires)  LogCommandDuration  ->  "Command transfer" total wall-time line
================================================================================
```

## Key facts the chart encodes
<!-- updated: 2026-06-09_21:16:18 -->

- **Project scoping is carried by exactly one parameter.** `getProjects` is the
  sole consumer of `ProjectKeys`; it sets `projects=<key>` on
  `api/projects/search` so the server returns only the scoped project. Every
  later per-project task re-reads `getProjects/*.jsonl` from disk, so scoping
  propagates transitively without any other task knowing the project key.
- **Phases 2 & 3 are pure local JSON→CSV transforms** — no API calls. Structure
  pre-stamps `sonarcloud_org_key` from `--default_organization`, which is why
  phase 4 deliberately leaves migrate's `DefaultOrganization` unset (passing it
  again would trigger the "mapping defined, default ignored" warning).
- **The "profiles/gate before scan replay" guarantee is emergent, not
  hardcoded.** `restoreProfiles`/`addGateConditions` land in P4 and
  `importProjectData` in P5 purely because of the dependency topo-sort
  (`importProjectData ← setProjectProfiles ← createProfiles`).
- **Two levels of parallelism:** task-level = full phase width (one goroutine
  per task, no cap); item-level = `cap(e.Sem)` = `--concurrency` (default 25),
  applied inside `forEachMigrateItem` / `importProjectData` / `runProjectSyncLoop`.
  For a single-project transfer the per-project fan-out is 1, so concurrency
  manifests inside the per-issue/per-hotspot sync loops.
- **Non-main branches need the analysis handshake.** `PreCreateAnalysis`
  (`POST {api}/analysis/analyses`) returns an `analysisUuid` stamped into
  protobuf metadata field 19 with `BranchType="long"` and a reference to the
  **main** branch — without it the CE accepts the report but never persists the
  branch (the BUG-17 fix).
- **Reports are fire-and-forget.** `emitReports` has no error return; if
  `GenerateReports` fails the transfer still prints "Transfer complete." and
  exits 0. Both report files are written to the **top-level** export dir, not
  inside `runDir`.
