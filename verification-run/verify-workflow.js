export const meta = {
  name: 'verify-transfer',
  description: 'Exhaustively verify a sonar-migration-tool transfer by comparing source (SonarQube Server) vs target (SonarQube Cloud) across every migrated dimension, adversarially verifying each verdict, then a completeness critic.',
  phases: [
    { title: 'Assess', detail: 'one agent per migrated dimension compares source vs target' },
    { title: 'Verify', detail: 'an independent skeptic re-checks each verdict with its own queries' },
    { title: 'Synthesize', detail: 'completeness critic + overall verdict' },
  ],
}

// args = { sourceUrl, targetUrl, org, sourceKey, targetKey, runDir, baselinePath, enterpriseKey }
const A = args || {}
const SRC = A.sourceUrl
const TGT = A.targetUrl
const ORG = A.org
const SKEY = A.sourceKey
const TKEY = A.targetKey
const RUNDIR = A.runDir
const BASELINE = A.baselinePath

// Shared connection cheat-sheet handed to every agent.
const CONN = `
CONNECTION CHEAT-SHEET (do NOT print tokens anywhere in your output):
- SOURCE (SonarQube Server): ${SRC}
    token: SRC_TOKEN=$(jq -r '.sonarqube.token' config.json)   ; auth: curl -s -u "$SRC_TOKEN:" "${SRC}/api/..."
    project key: ${SKEY}
- TARGET (SonarQube Cloud / staging): ${TGT}   organization: ${ORG}
    token: TGT_TOKEN=$(jq -r '.target.token' transfer-config.json) ; auth: curl -s -H "Authorization: Bearer $TGT_TOKEN" "${TGT}/api/..."
    project key: ${TKEY}   (org-prefixed)
    NOTE: SonarCloud endpoints usually require &organization=${ORG}. If a call returns empty/insufficient-privileges, retry WITH the organization param.
- Pre-transfer source baseline JSON is at: ${BASELINE}  (read it with: cat ${BASELINE})
- The migration run directory (logs/instrumentation) is at: ${RUNDIR}
Use curl + jq. Be precise. Always report the exact numbers you observed on BOTH sides.
`

const VERDICT_SCHEMA = {
  type: 'object',
  additionalProperties: false,
  properties: {
    dimension: { type: 'string' },
    status: { type: 'string', enum: ['PASS', 'PARTIAL', 'FAIL', 'NA'] },
    expected: { type: 'string', description: 'what the source side has / what migration should produce' },
    actual: { type: 'string', description: 'what was actually observed on the target' },
    evidence: { type: 'string', description: 'concrete numbers and the API calls/commands used' },
    discrepancies: { type: 'array', items: { type: 'string' } },
    confidence: { type: 'string', enum: ['high', 'medium', 'low'] },
  },
  required: ['dimension', 'status', 'expected', 'actual', 'evidence', 'confidence'],
}

const VERIFY_SCHEMA = {
  type: 'object',
  additionalProperties: false,
  properties: {
    dimension: { type: 'string' },
    agree: { type: 'boolean', description: 'do you agree with the original verdict after your own independent check?' },
    adjusted_status: { type: 'string', enum: ['PASS', 'PARTIAL', 'FAIL', 'NA'] },
    refutation: { type: 'string', description: 'if you disagree, the concrete evidence that refutes the original verdict' },
    notes: { type: 'string' },
  },
  required: ['dimension', 'agree', 'adjusted_status'],
}

const CRITIC_SCHEMA = {
  type: 'object',
  additionalProperties: false,
  properties: {
    overall_verdict: { type: 'string', enum: ['WORKS', 'WORKS_WITH_CAVEATS', 'BROKEN'] },
    headline: { type: 'string' },
    confirmed_working: { type: 'array', items: { type: 'string' } },
    issues_found: { type: 'array', items: { type: 'string' } },
    not_checked: { type: 'array', items: { type: 'string' }, description: 'dimensions or risks no agent actually verified' },
    suspicious_passes: { type: 'array', items: { type: 'string' } },
    recommended_followups: { type: 'array', items: { type: 'string' } },
  },
  required: ['overall_verdict', 'headline', 'confirmed_working', 'issues_found'],
}

const DIMENSIONS = [
  {
    key: 'project',
    prompt: `Verify the TARGET project exists after migration. Check that project key "${TKEY}" exists in organization ${ORG} on the target, its name matches the source project "${SKEY}" name, and it is visible via /api/projects/search and /api/components/show. Report source name vs target name.`,
  },
  {
    key: 'branches',
    prompt: `THIS IS THE HIGHEST-PRIORITY CHECK. Verify branch migration. The SOURCE project ${SKEY} has 6 branches: master(main), develop, release-3.x, my-test, feat/add-ruff-linting, reduce-tech-debt. Source per-branch issue totals: master=1731, release-3.x=1511, reduce-tech-debt=1291, my-test=831, develop=164, feat/add-ruff-linting=0. Use ${TGT}/api/project_branches/list?project=${TKEY}&organization=${ORG} to list TARGET branches. For each branch report: present on target? isMain? branchType (should be LONG/long-lived for non-main, NOT SHORT).
IMPORTANT CONTEXT — the live run log MUST be consulted: grep the run events for branch handshakes and skips:  grep -E 'analysis pre-created|skipping branch' ${RUNDIR}/run_events.jsonl ${RUNDIR}/transfer.log 2>/dev/null  (also try: cat verification-run/transfer.log | grep -E 'branch=|skipping branch|analysis pre-created'). A non-main branch is legitimately and CORRECTLY SKIPPED when the source server no longer has its source code (purged by housekeeping for an inactive branch) — the tool logs 'skipping branch: source code not retrievable'. In THIS run, 'develop' is expected to be skipped for exactly that documented reason. To CONFIRM that is correct (not a bug), independently verify develop's source is unavailable on the SOURCE: try ${SRC}/api/sources/raw or ${SRC}/api/measures/component_tree?component=${SKEY}&branch=develop&qualifiers=FIL — if source files/text are gone, the skip is correct. feat/add-ruff-linting has 0 issues/0 hotspots. Explicitly state which of the 6 source branches landed on target as long-lived branches, which were skipped, and WHY (with the log line). PASS if every branch whose source code is still present on the source server is migrated as a long-lived branch, AND any skipped branch has a documented purged-source reason that you independently confirmed. Treat a confirmed documented skip as PASS, not FAIL.`,
  },
  {
    key: 'issues-main',
    prompt: `Verify MAIN-branch issue migration parity. Source main branch is "master". Compute the SOURCE migratable issue count on master = issues NOT in status CLOSED and NOT resolved as FIXED (these have no Cloud counterpart and are intentionally skipped). Use ${SRC}/api/issues/search?componentKeys=${SKEY}&branch=master with facets/params to derive: total, count with statuses=CLOSED, count with resolutions=FIXED. migratable = total - CLOSED - FIXED (approx; CLOSED issues are also FIXED/REMOVED — reason carefully, prefer counting statuses in OPEN,CONFIRMED,REOPENED plus resolutions in FALSE-POSITIVE,WONTFIX,ACCEPTED). Then count TARGET issues: ${TGT}/api/issues/search?componentKeys=${TKEY}&organization=${ORG} (main branch). Compare. Report both numbers and the computed expected range. Small residual differences are acceptable (rules not present in any Cloud profile, etc.) — explain them. PASS if target is within ~5% of expected migratable, PARTIAL if notably off, FAIL if target is near 0 or wildly off.`,
  },
  {
    key: 'issues-branches',
    prompt: `Verify NON-MAIN branch issue counts on the target. For each non-main branch that exists on the target (develop, release-3.x, my-test, reduce-tech-debt, feat/add-ruff-linting), query ${TGT}/api/issues/search?componentKeys=${TKEY}&organization=${ORG}&branch=BRANCH and report the target issue count. Compare against the source migratable count for that same branch (source totals: release-3.x=1511, reduce-tech-debt=1291, my-test=831, develop=164, feat=0; remember CLOSED/FIXED are skipped so target will be lower than source total). Report a per-branch table source-total vs target-count. PASS if each present branch carries a plausible (non-zero where source>0) issue count.`,
  },
  {
    key: 'issue-triage',
    prompt: `Verify issue TRIAGE state was synced. On the SOURCE master branch, find issues that are triaged: resolution=FALSE-POSITIVE, resolution=WONTFIX, resolution=ACCEPTED, and issues with comments or assignees. Count each. Then on the TARGET (${TGT}/api/issues/search?componentKeys=${TKEY}&organization=${ORG} with resolutions/statuses filters) verify those triage states exist on target (counts of false-positive / accepted / confirmed, presence of comments and assignees on a sample). Pull a sample issue from source that is WONTFIX/ACCEPTED and confirm a corresponding target issue carries the same resolution. PASS if triage states are present on target in comparable counts.`,
  },
  {
    key: 'issue-dates',
    prompt: `Verify issue CREATION DATES were backdated (not stamped with the migration date). On the TARGET, fetch a sample of issues: ${TGT}/api/issues/search?componentKeys=${TKEY}&organization=${ORG}&ps=100&s=CREATION_DATE&asc=true . Inspect the creationDate field of the oldest issues — they should reflect historical dates (e.g. 2023/2024/2025), NOT all clustered at today's migration timestamp. Compare the min/max creationDate on target against source master issues' creationDate range (${SRC}/api/issues/search?componentKeys=${SKEY}&branch=master&s=CREATION_DATE&asc=true&ps=1). PASS if target oldest issue dates match historical source dates; FAIL if all target issues are dated at/near the migration time.`,
  },
  {
    key: 'hotspots',
    prompt: `Verify SECURITY HOTSPOT migration. Source master has 31 hotspots. Count TARGET hotspots: ${TGT}/api/hotspots/search?projectKey=${TKEY}&organization=${ORG}. Compare totals (hotspots transfer in FULL, so expect ~31 on main). Also check review status breakdown (TO_REVIEW / REVIEWED with SAFE/FIXED/ACKNOWLEDGED) is synced: compare source ${SRC}/api/hotspots/search?projectKey=${SKEY}&branch=master status breakdown vs target. PASS if target hotspot count ~matches source and statuses are synced.`,
  },
  {
    key: 'quality-gate',
    prompt: `Verify QUALITY GATE migration. Source project ${SKEY} uses gate "0 - Corp Platinum". On target, ${TGT}/api/qualitygates/get_by_project?project=${TKEY}&organization=${ORG} should show the same gate name is associated, and ${TGT}/api/qualitygates/show?name=...&organization=${ORG} should show its conditions. Compare condition count/metrics against source ${SRC}/api/qualitygates/get_by_project + show. PASS if the gate exists on target with matching conditions and is assigned to the project.`,
  },
  {
    key: 'quality-profiles',
    prompt: `Verify QUALITY PROFILE migration. The source project associates these NON-DEFAULT profiles worth checking: Python="Olivier Way", Java="Security Max", JavaScript="security-max", PHP="All rules", C++="Sonar MISRA C++:2023 Compliance", Dart="Critical projects", XML="Xpath instantiated". On target list org profiles ${TGT}/api/qualityprofiles/search?organization=${ORG} and the project's associated profiles ${TGT}/api/qualityprofiles/search?project=${TKEY}&organization=${ORG}. Confirm the non-default profiles above were recreated and associated, and spot-check that at least one (e.g. Python "Olivier Way") has a comparable active-rule count to source (${SRC}/api/qualityprofiles/search?project=${SKEY} gives activeRuleCount). PASS if the non-default profiles are present on target with restored rules and associated to the project.`,
  },
  {
    key: 'permissions',
    prompt: `Verify PROJECT PERMISSIONS migration. Source project group permissions (from baseline ${BASELINE} .permissions): developers(codeviewer,user), project-admins(admin,codeviewer,user), security-auditors(codeviewer,issueadmin,securityhotspotadmin,user), sonar-administrators, sonar-users(user), tech-leads(codeviewer,issueadmin,user). On target ${TGT}/api/permissions/groups?projectKey=${TKEY}&organization=${ORG}, verify the groups with non-empty permissions were recreated and granted comparable project permissions. Some Server-only groups/permissions may not map 1:1 to Cloud — note any documented drops. PASS if the meaningful group permissions are present on target.`,
  },
  {
    key: 'settings-tags-links-ncp',
    prompt: `Verify PROJECT SETTINGS, TAGS, LINKS, and NEW CODE PERIOD. Baseline (${BASELINE}): tags=["python"]; links=[scm: https://github.com/okorach/sonar-tools.git, bug-tracker: https://github.com/okorach/sonar-tools/issues]; new_code_period=PREVIOUS_VERSION. On target check: tags via ${TGT}/api/components/show?component=${TKEY}&organization=${ORG} (.component.tags); links via ${TGT}/api/project_links/search?projectKey=${TKEY}&organization=${ORG}; new code period via ${TGT}/api/new_code_periods/show?project=${TKEY}&organization=${ORG}. Report each. PASS if tags, links, and NCP match source; PARTIAL if some present.`,
  },
  {
    key: 'source-code-measures',
    prompt: `Verify SOURCE CODE & MEASURES landed. Source master: ncloc=17379, files=143. On target ${TGT}/api/measures/component?component=${TKEY}&organization=${ORG}&metricKeys=ncloc,files,coverage,duplicated_lines_density,security_hotspots,violations . Confirm ncloc and files are present and within a reasonable range of source (exact ncloc may differ slightly due to Cloud analyzers). Also spot-check that source code is browsable: ${TGT}/api/components/tree?component=${TKEY}&organization=${ORG}&qualifiers=FIL&ps=10 should return source files. PASS if ncloc/files are present (non-zero, comparable) and files are listed.`,
  },
  {
    key: 'run-logs',
    prompt: `Verify the MIGRATION RUN INSTRUMENTATION (no API calls needed — read files). The run directory is ${RUNDIR}. (a) Read ${RUNDIR}/run_meta.json — report overall_status (success|partial|failed) and any per-phase/per-task statuses. (b) Read ${RUNDIR}/run_events.jsonl — count events by level (grep -c 'level=ERROR' style on JSON), list any ERROR/failed events, and confirm the key tasks ran: createProjects, restoreProfiles, createGates, setProjectGroupPermissions, importProjectData, syncIssueMetadata, syncHotspotMetadata. (c) Look for any requests.log files under ${RUNDIR} (find ${RUNDIR} -name 'requests.log') and grep for HTTP 4xx/5xx responses; summarize error counts. (d) Check the importProjectData subdir for per-branch artifacts and any 'create analysis'/analysisUuid handshake evidence for non-main branches. PASS if overall_status is success and no blocking errors; PARTIAL if overall_status=partial with explained failures; FAIL if failed.`,
  },
  {
    key: 'summary-report',
    prompt: `Verify the MIGRATION SUMMARY report. Find the latest migration_summary.md (check ./migration-files/migration_summary.md and ${RUNDIR}/.. ). Read it and extract: the overall status it claims, the per-entity counts it reports (projects, issues, hotspots, branches, profiles, gates), and any failures/warnings it surfaces. Cross-check that its claimed counts are internally consistent with what a transfer of ${SKEY} should produce. Also confirm a migration_summary.pdf was generated (ls -la). PASS if the summary exists, reports success, and its counts are plausible.`,
  },
]

phase('Assess')
log(`Verifying transfer of ${SKEY} -> ${TKEY} across ${DIMENSIONS.length} dimensions (source ${SRC} vs target ${TGT})`)

const results = await pipeline(
  DIMENSIONS,
  (d) => agent(`${CONN}\n\nYOUR DIMENSION: ${d.key}\n${d.prompt}\n\nReturn a precise verdict. Set dimension="${d.key}".`,
    { label: `assess:${d.key}`, phase: 'Assess', schema: VERDICT_SCHEMA }),
  (verdict, d) => agent(`${CONN}\n\nYou are an adversarial verifier for dimension "${d.key}". Another agent produced this verdict:\n${JSON.stringify(verdict, null, 2)}\n\nDo NOT trust it. Run your OWN independent queries against source and target to confirm or REFUTE it. If the original said PASS, actively try to find a reason it is wrong (e.g. counts that don't actually match, branch missing, dates stamped at migration time, target project empty). Default to skepticism. Set dimension="${d.key}". If you cannot refute and the evidence holds, agree.`,
    { label: `verify:${d.key}`, phase: 'Verify', schema: VERIFY_SCHEMA })
    .then((v) => ({ key: d.key, verdict, verify: v }))
    .catch(() => ({ key: d.key, verdict, verify: null }))
)

const clean = results.filter(Boolean)

phase('Synthesize')
const critic = await agent(
  `${CONN}\n\nYou are the COMPLETENESS CRITIC and final judge for a verification of the sonar-migration-tool 'transfer' command (migrating project ${SKEY} from SonarQube Server to SonarQube Cloud project ${TKEY}).\n\nHere are all per-dimension verdicts and their adversarial re-checks:\n${JSON.stringify(clean, null, 2)}\n\nProduce the OVERALL verdict on whether the tool works for a real end user. Reconcile each dimension using the adversarial check's adjusted_status when it disagrees. Call out: what is confirmed working, what issues were found (especially any branch that failed to migrate, e.g. develop), what was NOT actually checked, and any PASS that looks suspicious or under-evidenced. Be rigorous and honest — this determines whether we tell the user the tool is verified.`,
  { label: 'completeness-critic', phase: 'Synthesize', schema: CRITIC_SCHEMA }
)

return { target: TKEY, source: SKEY, dimensions: clean, critic }
