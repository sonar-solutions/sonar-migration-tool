# Live Regression Testing Protocol
<!-- updated: 2026-06-04_15:30:00 -->

After any fix or feature, run the full pipeline against real SonarQube Server and SonarCloud instances, then verify **everything** migrated correctly. "Everything" means: projects, issues (all statuses/severities/types/resolutions), hotspots, quality profiles, quality gates, groups, permission templates, settings, new code periods, custom rules, project permissions, ALM bindings, portfolios, measures, and extract file integrity.

## Steps

### 1. Build
```bash
cd go && go build -o ../sonar-migration-tool ./main.go && cd ..
```

### 2. Health check
```bash
SQ_URL=$(jq -r '.source.url' transfer-config.json)
SQ_TOKEN=$(jq -r '.source.token' transfer-config.json)
SC_URL=$(jq -r '.target.url' transfer-config.json)
SC_TOKEN=$(jq -r '.target.token' transfer-config.json)
curl -sf -u "${SQ_TOKEN}:" "${SQ_URL}/api/system/status" | jq .status
curl -sf -H "Authorization: Bearer ${SC_TOKEN}" "${SC_URL}/api/system/status" | jq .status
```
Both must return `"UP"`. If not, stop — servers are down.

### 3. Clean slate (delete prior SC projects to avoid stale state)
```bash
./sonar-migration-tool reset --config transfer-config.json
```

### 4. Run transfer (single-command extract+structure+mappings+migrate)
```bash
rm -rf ./migration-files/
./sonar-migration-tool transfer --config transfer-config.json --include-scan-history 2>&1 | tee /tmp/smt-transfer.log
```
Must exit 0. If not, inspect the log and fix.

### 5. Run regtest (automated verification of ALL entities)
```bash
./sonar-migration-tool regtest --config config.json --format table --verbose 2>&1 | tee /tmp/smt-regtest.log
```
The `regtest` command compares SonarQube Server vs SonarCloud across **27 check functions / 44+ checks**: project count and identity, issue totals and distributions (5 statuses, 5 severities, 3 types, 3 resolutions per project), hotspot totals and status distribution, quality profile count/rules/defaults/inheritance, quality gate count/conditions/default/associations, group count/membership, permission template count/permissions, global+project settings, new code periods, custom rules, project group permissions (6 types), ALM bindings, portfolios, measures (12 metrics per project), and extract file completeness.

Exit code 0 = **ALL PASS**. Exit code 1 = failures exist — read the table, fix, loop back to step 4.

### 6. Scan logs for silent failures
```bash
grep -Ei "panic|fatal" /tmp/smt-transfer.log
grep -Ei "error|fail" /tmp/smt-transfer.log | grep -v "INFO"
```

### 7. Loop
If any check failed or logs show errors: fix the code, rebuild (step 1), re-run (steps 4-6). Repeat until regtest exits 0 with zero log errors. That is the only stop condition.
