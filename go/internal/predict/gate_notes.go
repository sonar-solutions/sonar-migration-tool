// Copyright (C) SonarSource Sàrl
// For more information, see https://sonarsource.com/legal/
// mailto:info AT sonarsource DOT com

package predict

import (
	"encoding/json"
	"fmt"

	"github.com/sonar-solutions/sonar-migration-tool/internal/common"
	"github.com/sonar-solutions/sonar-migration-tool/internal/migrate"
	"github.com/sonar-solutions/sonar-migration-tool/internal/structure"
)

// synthesizeAddGateConditionsNotes walks every QG condition from the
// extract's getGateConditions task, runs each one through the migrate
// metric-mapping table, and writes the resulting "remapped" / "dropped"
// decisions to the predictive run's addGateConditions.notes sidecar.
//
// The output JSONL schema is identical to what
// migrate.recordGateConditionNote writes (source + targets carrying
// {metric, op, error}), so summary.collectGateMappingNotes consumes it
// unchanged. Obvious 1:1 software_quality_*_rating → *_rating remaps
// are suppressed via migrate.IsObviousMetricRemap, matching real-migrate
// behaviour from #232.
//
// Conditions whose source metric is unmapped (pure passthrough) do not
// emit a note — same as the real migrate path.
func synthesizeAddGateConditionsNotes(exportDir, runDir string, extractMapping structure.ExtractMapping, orgLookup map[string]string) error {
	// Load every source condition from extract.
	condItems, err := structure.ReadExtractData(exportDir, extractMapping, "getGateConditions")
	if err != nil {
		return fmt.Errorf("reading getGateConditions extract: %w", err)
	}
	if len(condItems) == 0 {
		return nil
	}

	// Group conditions by (serverURL, gateName) — the extract writes
	// one record per source condition, decorated with the parent
	// gate's name and serverUrl, just like the migrate-side
	// getGateConditions task expects.
	type condKey struct{ serverURL, gateName string }
	byGate := make(map[condKey][]map[string]any, len(condItems))
	for _, ei := range condItems {
		var cond map[string]any
		if err := json.Unmarshal(ei.Data, &cond); err != nil {
			continue
		}
		gateName, _ := cond["gateName"].(string)
		if gateName == "" {
			continue
		}
		k := condKey{serverURL: ei.ServerURL, gateName: gateName}
		// Strip bookkeeping fields so what remains is the condition payload.
		delete(cond, "gateName")
		delete(cond, "serverUrl")
		byGate[k] = append(byGate[k], cond)
	}

	// Walk the synthesized createGates output: each row carries the
	// gate's source key, server URL, target org, and the synthetic
	// cloud_gate_id we minted in writeCreateJSONL. We join (serverURL,
	// gateName) → conditions, then write notes against the cloud id.
	store := common.NewDataStore(runDir)
	gateRows, err := store.ReadAll("createGates")
	if err != nil {
		return err
	}

	w, err := store.Writer("addGateConditions.notes")
	if err != nil {
		return err
	}

	for _, raw := range gateRows {
		var row map[string]any
		if err := json.Unmarshal(raw, &row); err != nil {
			continue
		}
		gateName, _ := row["name"].(string)
		serverURL, _ := row["server_url"].(string)
		cloudGateID, _ := row["cloud_gate_id"].(string)
		if gateName == "" || cloudGateID == "" {
			continue
		}
		conds := byGate[condKey{serverURL: serverURL, gateName: gateName}]
		for _, cond := range conds {
			metric, _ := cond["metric"].(string)
			op, _ := cond["op"].(string)
			errorVal, _ := cond["error"].(string)
			if metric == "" || op == "" {
				continue
			}
			if note, ok := classifyGateCondition(gateName, cloudGateID, metric, op, errorVal); ok {
				b, err := json.Marshal(note)
				if err != nil {
					continue
				}
				if err := w.WriteOne(b); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// classifyGateCondition mirrors the per-condition branch of
// migrate.addGateConditions: it looks up the source metric in the
// mapping table and returns the JSON record that would be written to
// addGateConditions.notes, or ok=false when no note is needed
// (passthrough, or obvious remap suppressed per #232).
//
// Returns ok=true with a record for:
//   - "dropped" when the source metric has no SQC equivalent.
//   - "remapped" when the source metric is mapped to one or more
//     non-obvious targets.
func classifyGateCondition(gateName, cloudGateID, metric, op, errorVal string) (map[string]any, bool) {
	targets, mapped := migrate.LookupMetricReplacement(metric)
	if mapped && len(targets) == 0 {
		return map[string]any{
			"cloud_gate_id": cloudGateID,
			"gate_name":     gateName,
			"action":        "dropped",
			"source":        condRef(metric, op, errorVal),
		}, true
	}
	if !mapped {
		// Pure passthrough — no note in real migrate either.
		return nil, false
	}

	// Compute effective op/error per target (inherit source unless overridden).
	targetMetrics := make([]string, 0, len(targets))
	targetObjs := make([]map[string]string, 0, len(targets))
	for _, repl := range targets {
		effOp := repl.Op
		if effOp == "" {
			effOp = op
		}
		effErr := repl.Error
		if effErr == "" {
			effErr = errorVal
		}
		targetMetrics = append(targetMetrics, repl.Metric)
		targetObjs = append(targetObjs, condRef(repl.Metric, effOp, effErr))
	}
	if migrate.IsObviousMetricRemap(metric, targetMetrics) {
		return nil, false
	}
	return map[string]any{
		"cloud_gate_id": cloudGateID,
		"gate_name":     gateName,
		"action":        "remapped",
		"source":        condRef(metric, op, errorVal),
		"targets":       targetObjs,
	}, true
}

func condRef(metric, op, errorVal string) map[string]string {
	return map[string]string{"metric": metric, "op": op, "error": errorVal}
}
