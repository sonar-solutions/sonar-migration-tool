# Empty Main Branch — Scanner Report Diff (our tool vs `@sonar/scan`)
<!-- updated: 2026-06-19_17:30:00 -->

## Symptom

After `transfer`, the SonarCloud project list shows **"The main branch of this project is empty"**, even though the project, quality gate, issues, and hotspots all migrate. A native `npx @sonar/scan` of the same project renders correctly (non-empty).

## How this was verified
<!-- updated: 2026-06-19_17:30:00 -->

A literal, evidence-backed comparison of the two scanner reports for `okorach-oss_sonar-tools-test`:

1. **Real report** — captured by re-running `npx @sonar/scan` against the source server (`localhost:9000`) with `-Dsonar.scanner.keepReport=true`, which retains `.scannerwork/scanner-report/` instead of deleting it post-upload. (The flag IS honored by `@sonar/scan` despite being undocumented.)
2. **Our report** — reproduced **offline and faithfully** by calling the production `buildBranchReport` for the main branch with the already-extracted data (white-box test `internal/migrate/zz_dumpreport_test.go`, gated on `SMT_DUMP_OUT`). The main-branch path makes no network calls (`buildSCProfileMap` is nil-safe; no create-analysis handshake for main), so the packaged bytes equal what the tool would submit — except `metadata.pb` qprofiles are empty when `e.Cloud == nil`.

## File-inventory diff
<!-- updated: 2026-06-19_17:30:00 -->

| File type | Our report | Real `@sonar/scan` |
|---|---:|---:|
| component-N.pb | 124 (1 PROJECT + 123 FILE) | 124 (1 PROJECT + 123 FILE) |
| source-N.txt | 123 | 123 |
| issues-N.pb | 46 | 46 |
| external-issues-N.pb | 82 | 82 |
| metadata.pb | 1 | 1 |
| context-props.pb | 1 (empty) | 1 |
| activerules.pb | 1 | 1 |
| adhocrules.pb | 1 | 1 |
| **measures-N.pb** | **0** | **123** |
| syntax-highlightings-N.pb | 0 | 123 |
| symbols-N.pb | 0 | 108 |
| duplications-N.pb | 0 | 95 |
| coverages-N.pb | 0 | 93 |
| telemetry-entries.pb / analysis-cache2.pb / analysis-warnings.pb / sca / analysis.log | 0 | present |
| changesets-N.pb | 123 | 0 (SCM disabled in this scan) |
| **Total entries** | **502** | **926** |

Component structure matches exactly (flat PROJECT → 123 FILE, no DIRECTORY tree — the modern scanner does not emit DIRECTORY components, so our PROJECT+FILE shape is correct). Our FILE components correctly carry `lines` and `language` (e.g. `ref=2 type=FILE lines=33 lang="py"`).

No file is byte-identical, and none can be: timestamps, `analysis_uuid`, random `scm_revision_id`, Deflate framing, and entry ordering all differ by construction. Even where the *type* matches, content volume differs (real `metadata.pb` 3608 B vs ours 184 B; real `activerules.pb` 425 KB activating the full ruleset vs ours 17 KB activating only the extracted subset). A raw byte diff is therefore neither achievable nor meaningful — the structural + semantic diff above is the correct comparison.

## Root cause
<!-- updated: 2026-06-19_17:30:00 -->

`go/internal/migrate/tasks_projectdata.go:449` builds the report with `Measures: make(map[int32][]*pb.Measure)` — an empty map that is never populated. `PackageReport` → `addMeasures` (`go/internal/scanreport/packager.go:119`) therefore writes **zero** `measures-N.pb` files, so the report carries no `ncloc`.

A real file-level `measures-N.pb` contains (decoded from the real report):

```
ncloc                 int=481
ncloc_data            string="23=1;24=1;26=1;..."   (per-line bitmap)
comment_lines         int=73
complexity            int=103
cognitive_complexity  int=129
functions             int=21
statements            int=398
classes               int=0
executable_lines_data string="512=1;513=1;..."
```

There is **no `measures-1.pb`** (the project root) — SonarCloud's Compute Engine **aggregates project `ncloc` from the per-file measures**. We send none, so project `ncloc` computes to null → the "main branch is empty" overlay fires.

The data is already on hand: `loadExtractedComponents` reads each file's `ncloc` via `extractMeasureInt32(item.Data, "ncloc")` but uses it only for the component `Lines` field. The extract phase requests only `ncloc` (`internal/extract/tasks_projectdata.go:249` → `"metricKeys": {"ncloc"}`); other metrics are not extracted.

## Why this is the right fix (CloudVoyager precedent)
<!-- updated: 2026-06-19_17:30:00 -->

CloudVoyager (the predecessor this tool harvests from) **did** populate measures and shipped working, non-empty branches:

- `src/pipelines/sq-2025/sonarcloud/report-packager/helpers/add-core-files.js` and `.../uploader/helpers/add-protobuf-files.js` write `measures-${ref}.pb` from a per-ref measures map.
- It sourced measures from `/api/measures/component` and computed `linesOfCode` from the scalar `ncloc` measure.

This strongly implies a **scalar `ncloc` measure per file is sufficient** to make the branch non-empty; `ncloc_data` (the per-line bitmap) is needed for line-level decoration and coverage-on-new-code but not for the branch to register lines of code.

## Fix (APPLIED 2026-06-19)
<!-- updated: 2026-06-19_18:05:00 -->

Implemented in three parts:

1. **Extract** (`internal/extract/tasks_projectdata.go`, `getProjectComponentTree`): the `metricKeys` request was widened from `ncloc` alone to `ncloc,comment_lines,complexity,cognitive_complexity,functions,statements,classes`. These are the scalar measures `api/measures/component_tree` allows. The per-line *data* metrics `ncloc_data` and `executable_lines_data` are **rejected** by `component_tree` (`HTTP 400: "Metrics ncloc_data can't be requested in this web service. Please use api/measures/component"`), so they are intentionally omitted — they would each require one `api/measures/component` call per file.

2. **Migrate load** (`internal/migrate/tasks_projectdata.go`): new `loadComponentMeasures` reads the per-file measures array from the extracted component tree and returns `[]scanreport.MeasureInput`. Helper `extractMeasurePairs` parses `"measures":[{"metric":..,"value":..}]`.

3. **Migrate wire-up** (`buildBranchReport`): `Measures: make(map[int32][]*pb.Measure)` (the empty map at the old line 449) was replaced with `Measures: scanreport.BuildMeasures(componentMeasures, cr)`.

**Offline verification** (white-box `zz_dumpreport_test.go`, regenerating the actual main-branch zip from extracted data): `measures-N.pb` went from **0 → 123**; **116/123** files carry `ncloc`; project total = **17,379 LOC** (non-zero → overlay cleared). No `measures-1.pb` root file (correct — the CE aggregates the project total from per-file measures). Total report entries 502 → 625.

Open follow-up (not needed for the overlay): `ncloc_data`/`executable_lines_data` per-line bitmaps for line-level LOC/coverage decoration would require a separate `api/measures/component` extract pass.

## Diagnostic artifacts (temporary — safe to delete)
<!-- updated: 2026-06-19_17:30:00 -->

- `go/tmp_pbdump/main.go` — decodes report `.pb` files using the repo's proto bindings (`go run ./tmp_pbdump measures <file>` / `component <file>`).
- `go/internal/migrate/zz_dumpreport_test.go` — dumps our actual main-branch zip offline; gated on `SMT_DUMP_OUT`, skips in CI.
