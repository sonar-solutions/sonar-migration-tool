# SCM Blame Migration (Code-view author/date/revision per line)

<!-- updated: 2026-06-19_18:05:00 -->

## Goal

In SonarQube Server's Code view each line shows who changed it, when, and at which revision. Before this change the migrated SonarCloud project showed none of that: every line carried a synthetic stub author and the analysis date. This change ships the **real** per-line SCM blame so the SonarCloud Code view matches the source.

## How blame reaches the report

<!-- updated: 2026-06-19_18:05:00 -->

The scanner report conveys blame through `changesets-{ref}.pb` (`Changesets` proto): a list of `Changeset{revision, author, date}` entries plus `changesetIndexByLine[]` mapping each source line (0-indexed) to one entry. SonarCloud's Code view renders author/date/revision from this, and — because `pb.Issue` carries **no date field** — the CE also derives each issue's creation date from the changeset date of the issue's line.

## What was wrong

<!-- updated: 2026-06-19_18:05:00 -->

The extract already pulled real blame via `getProjectSCMData` (`/api/sources/scm`), but the migrate side **never read it**. `buildChangesetMap` always called `BuildDefaultChangesets`, emitting one synthetic changeset per file (`revision="migration-initial"`, `author="sonar-migration-tool@sonarcloud.io"`, date=now). `BackdateChangesets` then rebuilt those entries entirely, keyed on issue creation dates, with the stub author. Net result: no real author, no real revision, dates only meaningful on issue lines.

## What changed

<!-- updated: 2026-06-19_18:05:00 -->

1. **`/api/sources/scm` is run-length encoded.** Confirmed against the live source: the default response lists a line only when its blame differs from the previous line (968 sparse entries vs 1571 with `commits_by_line=true` for one file; sparse line starts `[1,3,4,20,21,23…]`). Each listed line begins a run that continues until the next listed line. Extract keeps the default (smaller payload); migrate expands the runs.

2. **`loadExtractedSCM` + `parseBlameLines`** (`internal/migrate/tasks_projectdata.go`): read `getProjectSCMData` into `[]blameLine{Line, Author, Date, Revision}` per component, sorted by line.

3. **`scanreport.BuildChangesetsFromBlame`** (`internal/scanreport/builder.go`): builds a `Changesets` from the runs — dedups `(revision, author, date)` into entries and expands runs forward across every one of the file's lines. Lines before the first run inherit it; a run with a zero/unparseable date falls back to the analysis date.

4. **`buildChangesetMap`** now prefers real blame and falls back to the synthetic changeset only when blame is **not meaningful** — i.e. every line has an empty author *and* empty revision (a project analyzed without git history). `blameRunsFor` enforces that guard so we never ship all-empty blame.

5. **Blame-preserving backdating** (`internal/scanreport/backdate.go`): `rebuildChangesetForFile` was rewritten. Instead of discarding entries and rebuilding from issue dates, it now overrides only the **date** on issue lines (so issues keep their original creation date) while **preserving the real author and revision** already on that line. Non-issue lines keep their real blame untouched. Multi-line issues get the same (oldest-wins) date on every line of their range, so the CE's MAX-over-lines collapses to that date — no inflation. All existing `backdate_test.go` cases still pass (they assert on dates and the line-1 index, both preserved).

## Source-data caveat (important for verification)

<!-- updated: 2026-06-19_18:05:00 -->

Blame can only be migrated if the **source** has it. Verified directly against `localhost:9000`:

- `okorach-oss_sonar-tools-test` — **no blame**: every line of all 369 files across all 3 branches has empty author and empty revision (uniform analysis-date only). This is the signature of a project analyzed without git history. For this project the tool correctly falls back to the synthetic changeset; the Code view will show no real authors because there is nothing to migrate.
- `okorach-oss_sonar-tools` (v3.21) — **real blame**: real authors (e.g. `olivier.korach@gmail.com`), real ISO dates, real 40-char revisions. This is the project to use when verifying the blame fix end-to-end.

## Tests

<!-- updated: 2026-06-19_18:05:00 -->

`TestBuildChangesetsFromBlame` covers run-length expansion (lines 1-3=rA, 4-5=rB, 6-8=rA), `(rev,author,date)` dedup, zero-date fallback, and the nil-on-empty case. `go vet` clean; `internal/scanreport`, `internal/migrate`, `internal/extract` all green.
