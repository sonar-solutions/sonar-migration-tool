package migrate

import (
	"encoding/json"
	"testing"
)

func rawJSON(t *testing.T, v map[string]any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}

func TestAnalyzeProfile_CustomSeverity(t *testing.T) {
	// Custom severity is per-activation: lives in getProfileRules'
	// actives map, decorated with the synthetic "key" rule key.
	in := ProfileAnalysisInput{
		CloudProfileKey: "cp1", ProfileName: "Java way", Language: "java",
		Activations: []json.RawMessage{
			rawJSON(t, map[string]any{"key": "java:S1", "qProfile": "qp1", "severity": "CRITICAL"}),
			rawJSON(t, map[string]any{"key": "java:S2", "qProfile": "qp1", "severity": "MAJOR"}),
		},
		BaseRulesByKey: map[string]json.RawMessage{
			"java:S1": rawJSON(t, map[string]any{"key": "java:S1", "severity": "MAJOR"}),
			"java:S2": rawJSON(t, map[string]any{"key": "java:S2", "severity": "MAJOR"}),
		},
	}
	out := AnalyzeProfile(in)
	gotCustom := 0
	for _, f := range out {
		if f.Kind == FindingKindCustomSeverity {
			gotCustom++
			if f.RuleKey != "java:S1" {
				t.Errorf("expected custom severity on java:S1, got %s", f.RuleKey)
			}
			if f.Detail != "MAJOR → CRITICAL" {
				t.Errorf("Detail: got %q want %q", f.Detail, "MAJOR → CRITICAL")
			}
		}
	}
	if gotCustom != 1 {
		t.Errorf("expected exactly 1 custom-severity finding (java:S1 only), got %d (%+v)", gotCustom, out)
	}
}

func TestAnalyzeProfile_Prioritized(t *testing.T) {
	// Prioritized data lives on the per-activation record (sourced
	// from getProfileRules' "actives" map), NOT on the rule catalog
	// rows. The migrate task decorates each activation with a
	// synthetic "key" field carrying the rule key.
	in := ProfileAnalysisInput{
		CloudProfileKey: "cp1", ProfileName: "x", Language: "java",
		Activations: []json.RawMessage{
			rawJSON(t, map[string]any{"key": "java:S1", "qProfile": "qp1", "prioritizedRule": true}),
			rawJSON(t, map[string]any{"key": "java:S2", "qProfile": "qp1", "prioritizedRule": false}),
			rawJSON(t, map[string]any{"key": "java:S3", "qProfile": "qp1"}),
		},
	}
	out := AnalyzeProfile(in)
	gotPrio := 0
	for _, f := range out {
		if f.Kind == FindingKindPrioritized {
			gotPrio++
			if f.RuleKey != "java:S1" {
				t.Errorf("expected prioritized on java:S1 only, got %s", f.RuleKey)
			}
		}
	}
	if gotPrio != 1 {
		t.Errorf("expected exactly 1 prioritized finding, got %d", gotPrio)
	}
}

func TestAnalyzeProfile_ThirdParty(t *testing.T) {
	in := ProfileAnalysisInput{
		CloudProfileKey: "cp1", ProfileName: "x", Language: "java",
		ActiveRules: []json.RawMessage{
			rawJSON(t, map[string]any{"key": "java:S1", "repo": "java"}),                    // standard
			rawJSON(t, map[string]any{"key": "vendor:R1", "repo": "vendor-plugin"}),         // 3rd-party
			rawJSON(t, map[string]any{"key": "javasecurity:X", "repo": "javasecurity"}),     // standard (in list)
			rawJSON(t, map[string]any{"key": "custom-checks:C1", "repo": "custom-checks"}),  // 3rd-party
		},
	}
	out := AnalyzeProfile(in)
	gotThirdParty := map[string]bool{}
	for _, f := range out {
		if f.Kind == FindingKindThirdParty {
			gotThirdParty[f.RuleKey] = true
		}
	}
	want := map[string]bool{"vendor:R1": true, "custom-checks:C1": true}
	for k := range want {
		if !gotThirdParty[k] {
			t.Errorf("missing third-party finding for %s", k)
		}
	}
	for k := range gotThirdParty {
		if !want[k] {
			t.Errorf("unexpected third-party finding for %s", k)
		}
	}
}

func TestAnalyzeProfile_CustomParams(t *testing.T) {
	// Per-activation params with values come from getProfileRules'
	// actives map (decorated with the synthetic rule "key" field).
	in := ProfileAnalysisInput{
		CloudProfileKey: "cp1", ProfileName: "x", Language: "java",
		Activations: []json.RawMessage{
			rawJSON(t, map[string]any{
				"key":      "java:S1",
				"qProfile": "qp1",
				"params": []map[string]string{
					{"key": "maxLines", "value": "50"},        // custom
					{"key": "exemptFromMain", "value": "true"}, // matches default
					{"key": "noValueGiven", "value": ""},       // empty → ignored
				},
			}),
		},
		BaseRulesByKey: map[string]json.RawMessage{
			"java:S1": rawJSON(t, map[string]any{
				"key": "java:S1",
				"params": []map[string]any{
					{"key": "maxLines", "defaultValue": "100"},
					{"key": "exemptFromMain", "defaultValue": "true"},
					{"key": "noValueGiven", "defaultValue": "X"},
				},
			}),
		},
	}
	out := AnalyzeProfile(in)
	var custom []ProfileFinding
	for _, f := range out {
		if f.Kind == FindingKindCustomParams {
			custom = append(custom, f)
		}
	}
	if len(custom) != 1 {
		t.Fatalf("expected exactly 1 custom-params finding (only maxLines differs; exemptFromMain matches default; noValueGiven is empty), got %d (%+v)", len(custom), custom)
	}
	if custom[0].RuleKey != "java:S1" || custom[0].Detail != "maxLines=50 (default 100)" {
		t.Errorf("unexpected Detail: got %+v", custom[0])
	}
}

func TestAnalyzeProfile_TemplateInstance(t *testing.T) {
	in := ProfileAnalysisInput{
		CloudProfileKey: "cp1", ProfileName: "x", Language: "java",
		ActiveRules: []json.RawMessage{
			rawJSON(t, map[string]any{"key": "java:S1", "repo": "java"}),
			rawJSON(t, map[string]any{"key": "java:custom1", "repo": "java"}),
		},
		BaseRulesByKey: map[string]json.RawMessage{
			"java:S1":       rawJSON(t, map[string]any{"key": "java:S1"}),
			"java:custom1":  rawJSON(t, map[string]any{"key": "java:custom1", "templateKey": "java:T1"}),
		},
	}
	out := AnalyzeProfile(in)
	got := 0
	for _, f := range out {
		if f.Kind == FindingKindTemplateInstance {
			got++
			if f.RuleKey != "java:custom1" {
				t.Errorf("expected template-instance on java:custom1, got %s", f.RuleKey)
			}
		}
	}
	if got != 1 {
		t.Errorf("expected exactly 1 template-instance finding, got %d", got)
	}
}

func TestAnalyzeProfile_DisabledInherited(t *testing.T) {
	in := ProfileAnalysisInput{
		CloudProfileKey: "cp1", ProfileName: "child", Language: "java",
		DeactivatedInheritedRules: []json.RawMessage{
			rawJSON(t, map[string]any{"key": "java:S1"}),
			rawJSON(t, map[string]any{"key": "java:S2"}),
		},
	}
	out := AnalyzeProfile(in)
	got := map[string]bool{}
	for _, f := range out {
		if f.Kind == FindingKindDisabledInherited {
			got[f.RuleKey] = true
		}
	}
	if len(got) != 2 || !got["java:S1"] || !got["java:S2"] {
		t.Errorf("expected disabled-inherited on java:S1,java:S2, got %+v", got)
	}
}
