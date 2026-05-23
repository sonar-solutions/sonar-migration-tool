package migrate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/sonar-solutions/sq-api-go/types"
	"golang.org/x/sync/errgroup"
)

// runSetGlobalSettings migrates customized SQS-side global settings to
// every SonarQube Cloud organization in scope (issue #186).
//
// Pipeline:
//
//  1. Read getServerSettingsDefinitions to learn each SQS setting's
//     defaultValue (and shape) — that's how we detect "customized".
//  2. Read getServerSettings (raw values) and filter out any setting
//     whose value equals the SQS default — uncustomized settings are
//     skipped entirely.
//  3. Read generateOrganizationMappings and collect every target
//     sonarcloud_org_key that isn't empty / SKIPPED.
//  4. For each org, fetch SQC's list_definitions once (cached) so we know
//     which keys actually exist on the target and what shape (single /
//     multi / property-set) they expect.
//  5. For each (customized SQS setting × target SQC org):
//     – not in SQC's defs → log Warn, record skipped(reason=not-on-sqc).
//     – in SQC's defs → dispatch via applySettingByDef (the same helper
//     that drives setProjectSettings, but with empty projectKey so the
//     SDK scopes the request to the organization).
//  6. Emit one JSONL record per setting key, with applied / failed /
//     skipped org lists plus a pre-built "detail" string that the
//     summary report renders verbatim.
func runSetGlobalSettings(ctx context.Context, e *Executor) error {
	// SQS-side definitions — keyed by setting key. Drives the
	// "customized?" check below.
	sqsDefRecords, _ := e.Store.ReadAll("getServerSettingsDefinitions")
	sqsDefaultByKey := make(map[string]string, len(sqsDefRecords))
	for _, d := range sqsDefRecords {
		k := extractField(d, "key")
		if k == "" {
			continue
		}
		sqsDefaultByKey[k] = extractField(d, "defaultValue")
	}

	// Raw SQS global settings — kept only when customized.
	sqsValues, _ := e.Store.ReadAll("getServerSettings")
	customized := make([]json.RawMessage, 0, len(sqsValues))
	for _, raw := range sqsValues {
		key := extractField(raw, "key")
		if key == "" {
			continue
		}
		if !isSettingCustomized(raw, sqsDefaultByKey[key]) {
			continue
		}
		customized = append(customized, raw)
	}

	// Target SQC orgs.
	orgItems, _ := e.Store.ReadAll("generateOrganizationMappings")
	orgs := make(map[string]struct{})
	orgList := make([]string, 0, len(orgItems))
	for _, o := range orgItems {
		orgKey := extractField(o, "sonarcloud_org_key")
		if shouldSkipOrg(orgKey) {
			continue
		}
		if _, dup := orgs[orgKey]; dup {
			continue
		}
		orgs[orgKey] = struct{}{}
		orgList = append(orgList, orgKey)
	}
	sort.Strings(orgList)

	e.Logger.Info("starting task", "task", "setGlobalSettings",
		"customized_settings", len(customized), "target_orgs", len(orgList))

	// One list_definitions fetch per target org.
	defsByOrg := loadSettingDefinitionsForOrgs(ctx, e, orgs, "setGlobalSettings")

	counter := NewTaskCounter("setGlobalSettings")
	w, err := e.Store.Writer("setGlobalSettings")
	if err != nil {
		return err
	}

	var mu sync.Mutex
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(cap(e.Sem))
	for _, raw := range customized {
		g.Go(func() error {
			if gctx.Err() != nil {
				return gctx.Err()
			}
			rec := applyOneGlobalSetting(gctx, e, raw, orgList, defsByOrg, counter)
			b, _ := json.Marshal(rec)
			mu.Lock()
			defer mu.Unlock()
			return w.WriteOne(b)
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}
	counter.LogSummary(e.Logger)
	return nil
}

// applyOneGlobalSetting applies a single customized SQS global setting to
// every target SQC org and returns a result record describing the per-org
// outcomes plus a pre-built detail string for the report.
func applyOneGlobalSetting(ctx context.Context, e *Executor, raw json.RawMessage, orgs []string,
	defsByOrg map[string]map[string]types.SettingDefinition, counter *TaskCounter) globalSettingResult {

	key := extractField(raw, "key")
	rec := globalSettingResult{Key: key}
	rec.Value, rec.Values, rec.FieldValues = readSettingPayload(raw)

	for _, org := range orgs {
		def, hasDef := defsByOrg[org][key]
		if !hasDef {
			e.Logger.Warn("setGlobalSettings: setting key not available on SQC, skipping",
				"key", key, "org", org)
			rec.SkippedOrgs = append(rec.SkippedOrgs, skippedOrg{Org: org, Reason: "not-on-sqc"})
			continue
		}
		err := applySettingByDef(ctx, e, "", org, raw, key, def, true)
		switch {
		case errors.Is(err, errSettingEmpty):
			rec.SkippedOrgs = append(rec.SkippedOrgs, skippedOrg{Org: org, Reason: "empty"})
		case err != nil:
			counter.Fail()
			logAPIWarn(e.Logger, "setGlobalSettings failed", err, "key", key, "org", org)
			rec.FailedOrgs = append(rec.FailedOrgs, failedOrg{Org: org, Reason: err.Error()})
		default:
			counter.Success()
			rec.AppliedOrgs = append(rec.AppliedOrgs, org)
		}
	}
	rec.Detail = renderGlobalSettingDetail(rec)
	return rec
}

// isSettingCustomized reports whether the SQS-side value for a setting
// differs from its declared defaultValue. SQS exposes values in three
// shapes (value / values / fieldValues); for the comparison we collapse
// each into a comparable scalar string — fieldValues collapses to its
// JSON encoding, values to a sorted CSV.
func isSettingCustomized(raw json.RawMessage, defaultValue string) bool {
	if fvs := extractObjectArray(raw, "fieldValues"); len(fvs) > 0 {
		// PROPERTY_SET — defaultValue is unlikely to match a complex
		// JSON payload, so treat any populated fieldValues as
		// customized.
		return true
	}
	if vals := extractStringArray(raw, "values"); len(vals) > 0 {
		sorted := append([]string(nil), vals...)
		sort.Strings(sorted)
		joined := strings.Join(sorted, ",")
		defSorted := strings.Split(defaultValue, ",")
		sort.Strings(defSorted)
		return joined != strings.Join(defSorted, ",")
	}
	return extractField(raw, "value") != defaultValue
}

// readSettingPayload extracts the three possible value shapes from a
// settings record so the result can be serialized back into the output
// JSONL record exactly as it came from SQS.
func readSettingPayload(raw json.RawMessage) (value string, values []string, fieldValues []map[string]any) {
	return extractField(raw, "value"),
		extractStringArray(raw, "values"),
		extractObjectArray(raw, "fieldValues")
}

// renderGlobalSettingDetail produces the string the summary report shows
// in the Detail column for one global-setting row. Matches the format
// requested in issue #186: "value=… — applied to: org1, org2".
func renderGlobalSettingDetail(r globalSettingResult) string {
	var parts []string
	switch {
	case len(r.FieldValues) > 0:
		b, _ := json.Marshal(r.FieldValues)
		parts = append(parts, fmt.Sprintf("fieldValues=%s", string(b)))
	case len(r.Values) > 0:
		parts = append(parts, fmt.Sprintf("values=[%s]", strings.Join(r.Values, ",")))
	default:
		parts = append(parts, fmt.Sprintf("value=%s", r.Value))
	}
	if len(r.AppliedOrgs) > 0 {
		parts = append(parts, "applied to: "+strings.Join(r.AppliedOrgs, ", "))
	}
	if len(r.SkippedOrgs) > 0 {
		skipped := make([]string, 0, len(r.SkippedOrgs))
		for _, s := range r.SkippedOrgs {
			skipped = append(skipped, fmt.Sprintf("%s (%s)", s.Org, s.Reason))
		}
		parts = append(parts, "skipped: "+strings.Join(skipped, ", "))
	}
	if len(r.FailedOrgs) > 0 {
		failed := make([]string, 0, len(r.FailedOrgs))
		for _, f := range r.FailedOrgs {
			failed = append(failed, f.Org)
		}
		parts = append(parts, "failed: "+strings.Join(failed, ", "))
	}
	return strings.Join(parts, " — ")
}

// globalSettingResult is the per-setting record written to the
// setGlobalSettings task output (one JSONL line per setting key) and
// read back by the summary report to populate the Global Settings
// section.
type globalSettingResult struct {
	Key         string           `json:"key"`
	Value       string           `json:"value,omitempty"`
	Values      []string         `json:"values,omitempty"`
	FieldValues []map[string]any `json:"fieldValues,omitempty"`
	AppliedOrgs []string         `json:"applied_orgs,omitempty"`
	SkippedOrgs []skippedOrg     `json:"skipped_orgs,omitempty"`
	FailedOrgs  []failedOrg      `json:"failed_orgs,omitempty"`
	Detail      string           `json:"detail"`
}

type skippedOrg struct {
	Org    string `json:"org"`
	Reason string `json:"reason"`
}

type failedOrg struct {
	Org    string `json:"org"`
	Reason string `json:"reason"`
}
