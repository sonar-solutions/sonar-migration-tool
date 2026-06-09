# Transfer Verification — 2026-06-09
<!-- updated: 2026-06-09_19:20:00 -->

End-to-end verification that `sonar-migration-tool transfer` works the way a real end user runs it. A live, clean-slate `transfer` of a real multi-branch project was executed against staging, then verified with a 29-agent parallel comparison swarm (each migrated dimension assessed + adversarially re-checked), the built-in `regtest`, and direct API spot-checks.

**Verdict: WORKS — with documented caveats and a short list of real, minor, actionable gaps.** No errors, exit 0, `overall_status: success`. Core migration (project, branches, issues, history, triage, config) is correct and faithful.

## What was run
<!-- updated: 2026-06-09_19:20:00 -->

- **Command:** `./sonar-migration-tool transfer -c sonartools-transfer-config.json` (the documented end-user path).
- **Project:** `okorach-oss_sonar-tools` — 6 branches, 1731 issues on master, 31 hotspots, 17,379 ncloc, non-default quality gate + profiles. Chosen because it exercises the highest-risk recent code (multi-branch import via the create-analysis handshake).
- **Source:** SonarQube Server `localhost:9000` (v2026.3.1). **Target:** SonarQube Cloud staging `sc-staging.io`, org `open-digital-society-1`.
- **Clean slate:** the prior staging copy `open-digital-society-1_okorach-oss_sonar-tools` was deleted first (HTTP 204, confirmed gone) so every create path ran fresh.
- **Duration:** 3m10s. **Run dir:** `migration-files/2026-06-09-02` (extract artifacts in `2026-06-09-01`).

## Confirmed working (independently verified on both instances)
<!-- updated: 2026-06-09_19:20:00 -->

- **Project create** — target key `open-digital-society-1_okorach-oss_sonar-tools`, name "Sonar Tools", org-prefix mapping correct.
- **Branches** — master + 3 long-lived non-main branches (`release-3.x`, `reduce-tech-debt`, `my-test`), all `type=LONG`, anchored via the **create-analysis handshake** (`POST .../analysis/analyses` → `analysisUuid`, `referenceBranch=master`). This is the headline recent fix and it works.
- **Main-branch issue parity — EXACT: 1292 source-migratable == 1292 target** (0.00% delta). CLOSED + FIXED-resolution issues correctly excluded.
- **Issue creation-date backdating** — target dates span `2020-07-25 … 2026-05-11`, min/max identical to source; **zero** issues stamped at the migration date.
- **Triage** — false-positive/accepted resolutions, tags (18 custom), and comments (verbatim, with original author+timestamp) synced.
- **Security hotspots** — all 31 migrated.
- **Permissions** — 48/48 grants; source groups recreated with matching permission sets (`sonar-users` → org Members per #269).
- **Quality profiles** — 7 custom profiles recreated and restored (php/All rules + js/security-max Perfect; others Near-Perfect due to documented SonarCloud limits).
- **Settings / tags / links / new-code-period** — tags `[python]`, both SCM+bug-tracker links (URLs byte-identical), NCP `PREVIOUS_VERSION`.
- **Source code** — 192 files browsable on target.
- **Run instrumentation & report** — `overall_status=success`, 28/28 tasks ok, 0 ERROR events; `migration_summary.md` + valid 7-page PDF generated with honest, internally-consistent counts.

## Expected / documented behavior (not defects)
<!-- updated: 2026-06-09_19:20:00 -->

- **`develop` branch skipped (164 issues + 10 hotspots lost).** ROOT-CAUSED and CONFIRMED correct: develop's **source text is purged on the source server** (last analysis 2025-05-01; `api/sources/raw?branch=develop` returns HTTP 200 with a **0-byte body** for every file, `api/sources/lines`=0, vs full source on master). The tool reads extract-phase sources at [go/internal/migrate/tasks_projectdata.go:333](../go/internal/migrate/tasks_projectdata.go#L333) (`totalSourceLen==0 && findings>0` → skip) and logs an actionable message: *re-analyze the branch on the source server to migrate it*. **Caution:** judging source availability by HTTP status code alone is a false read — must measure body bytes.
- **`feat/add-ruff-linting` absent** — 0 issues / 0 hotspots; nothing to migrate.
- **1 quality-gate condition dropped** — `new_software_quality_reliability_remediation_effort` has no SonarQube Cloud equivalent (#143).
- **1 user permission dropped** on py/Olivier Way profile (surfaced per #353).
- **Assignees not migrated** — 1267 assigned issues on source → 0 on target. SonarQube logins don't map to SonarCloud accounts; expected cross-instance limitation. Worth documenting prominently / offering an assignee-mapping option.

## Real, minor, actionable gaps found
<!-- updated: 2026-06-09_19:20:00 -->

1. **Code-size measures not seeded** — `ncloc` and `coverage` are **entirely absent** on the target (every branch); only issue-derived measures (code_smells, violations, sqale_index) are present. Source code is imported as text/structure but not analyzed as code, so dashboards/quality reporting are empty until the first real CI re-scan. README documents the re-scan step, but the total absence of ncloc/coverage is the weakest area and worth root-causing.
2. **6 of 31 hotspots stuck `TO_REVIEW`** — source main = 31 REVIEWED/SAFE; target = 25 REVIEWED/SAFE + 6 TO_REVIEW (all `python:S4823` in three `conf/*2sonar.py` files). Count migrated; review status didn't sync for 6. (`syncHotspotMetadata` logged `line_mismatch=14`, `not_found=32`.)
3. **Quality gate weakened** — target "0 - Corp Platinum" has **8 of 10** source conditions; 2 MQR conditions dropped with **no fallback**, even though viable legacy equivalents exist on target (`software_quality_blocker_issues`→`blocker_violations`, `new_software_quality_reliability_remediation_effort`→`new_reliability_remediation_effort`).
4. **C++ MISRA profile not bound** — `Sonar MISRA C++:2023 Compliance` (509 rules) recreated at org level but the project's C++ binding fell back to default `Sonar way` (456 rules). (Academic for this Python project, but a real association gap.)
5. **Minor resolution drift** — a handful of won't-fix/accepted states didn't replay (regtest: WONTFIX 218→202). Lower confidence; worth a targeted check on non-main branches.

## Tooling note
<!-- updated: 2026-06-09_19:20:00 -->

- **`regtest` is the wrong tool to verify a single-project `transfer`.** It compares the *entire* source instance (78 projects, ~113k issues) against the target org, so it reports massive false failures (1410 failed / 1002 errors) — nearly all "project not on target" for the 77 untransferred projects. Scoped to our project, the meaningful checks pass; the apparent "failures" are a scoping artifact. Use `regtest` only after a full `migrate`.
- **`regtest` requires the legacy config shape** (`sonarqube`/`sonarcloud`), not the unified `source`/`target` shape that `transfer`/`extract`/`migrate` accept.

## Artifacts
<!-- updated: 2026-06-09_19:20:00 -->

- `verification-run/source_baseline.json` — pre-transfer source snapshot.
- `verification-run/transfer.log` — full transfer output (exit 0).
- `verification-run/regtest.json` — regtest results (interpret per tooling note).
- `verification-run/verify-workflow.js` — the 14-dimension verification swarm script (re-runnable).
- `migration-files/2026-06-09-02/` — run instrumentation (`run_meta.json`, `run_events.jsonl`), `migration_summary.{md,pdf}`.
