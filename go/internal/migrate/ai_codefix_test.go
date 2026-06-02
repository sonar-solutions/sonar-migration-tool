package migrate

import (
	"strings"
	"testing"
)

// Each test mirrors one bullet of the issue #251 strategy. We assert
// (1) the chosen outcome status for both setting rows,
// (2) the NearPerfect marker on the rows that require it, and
// (3) the SQC PATCH payload — enablement, provider key, model key,
// and (when applicable) the SQS-keyed enabledProjectKeys hint.

func TestEvaluateAiCodeFix_HiddenOnSQS(t *testing.T) {
	d := EvaluateAiCodeFix(AiCodeFixSourceState{
		Hidden:    true,
		HasConfig: true,
	})
	if d.Hidden.Status != outcomeApplied || !d.Hidden.NearPerfect {
		t.Errorf("hidden row: want applied+nearPerfect, got %+v", d.Hidden)
	}
	if d.Suggestions.Status != outcomeApplied || !d.Suggestions.NearPerfect {
		t.Errorf("suggestions row: want applied+nearPerfect, got %+v", d.Suggestions)
	}
	if d.PatchPayload == nil || d.PatchPayload.AiCodeFix == nil || d.PatchPayload.AiCodeFix.Enablement != sqcEnablementDisabled {
		t.Errorf("expected DISABLED patch, got %+v", d.PatchPayload)
	}
	if d.PatchPayload.Provider != nil {
		t.Errorf("hidden case should not send provider, got %+v", d.PatchPayload.Provider)
	}
}

func TestEvaluateAiCodeFix_SelfHosted(t *testing.T) {
	d := EvaluateAiCodeFix(AiCodeFixSourceState{
		HasConfig:          true,
		Enablement:         sqcEnablementAll,
		SelectedProvider:   "AZURE_OPENAI",
		SelectedSelfHosted: true,
	})
	if d.Suggestions.Status != outcomeSkipped {
		t.Errorf("suggestions row: want skipped, got %s", d.Suggestions.Status)
	}
	if d.Suggestions.Reason != "private-llm" {
		t.Errorf("expected reason=private-llm, got %q", d.Suggestions.Reason)
	}
	if d.PatchPayload == nil || d.PatchPayload.AiCodeFix == nil || d.PatchPayload.AiCodeFix.Enablement != sqcEnablementDisabled {
		t.Errorf("expected DISABLED patch, got %+v", d.PatchPayload)
	}
	if d.Hidden.Status != "" {
		t.Errorf("hidden row should be suppressed when hidden=false, got %+v", d.Hidden)
	}
}

func TestEvaluateAiCodeFix_PublicOpenAIGpt51(t *testing.T) {
	d := EvaluateAiCodeFix(AiCodeFixSourceState{
		HasConfig:        true,
		Enablement:       sqcEnablementAll,
		SelectedProvider: sqcProviderOpenAI,
		SelectedModel:    sqsModelOpenAIGPT51,
	})
	if d.Suggestions.Status != outcomeApplied || d.Suggestions.NearPerfect {
		t.Errorf("gpt-5.1 should be Perfect, got %+v", d.Suggestions)
	}
	if strings.Contains(d.Suggestions.Detail, "Anthropic") {
		t.Errorf("Detail should not mention Anthropic, got %q", d.Suggestions.Detail)
	}
	if d.PatchPayload.Provider.Key != sqcProviderOpenAI || d.PatchPayload.Provider.ModelKey != sqcModelOpenAIGPT51 {
		t.Errorf("unexpected provider/model: %+v", d.PatchPayload.Provider)
	}
}

func TestEvaluateAiCodeFix_PublicOpenAIGpt4o(t *testing.T) {
	d := EvaluateAiCodeFix(AiCodeFixSourceState{
		HasConfig:        true,
		Enablement:       sqcEnablementAll,
		SelectedProvider: sqcProviderOpenAI,
		SelectedModel:    sqsModelOpenAIGPT4O,
	})
	if d.Suggestions.Status != outcomeApplied || !d.Suggestions.NearPerfect {
		t.Errorf("gpt-4o should be Near Perfect (downgraded), got %+v", d.Suggestions)
	}
	if !strings.Contains(d.Suggestions.Detail, "GPT-5.1") {
		t.Errorf("expected Detail to mention GPT-5.1, got %q", d.Suggestions.Detail)
	}
	if strings.Contains(d.Suggestions.Detail, "Anthropic") {
		t.Errorf("Detail should not mention Anthropic, got %q", d.Suggestions.Detail)
	}
	if d.PatchPayload.Provider.ModelKey != sqcModelOpenAIGPT51 {
		t.Errorf("expected GPT-5.1 substitution, got %q", d.PatchPayload.Provider.ModelKey)
	}
}

func TestEvaluateAiCodeFix_PerProject(t *testing.T) {
	d := EvaluateAiCodeFix(AiCodeFixSourceState{
		HasConfig:          true,
		Enablement:         sqcEnablementSome,
		SelectedProvider:   sqcProviderOpenAI,
		SelectedModel:      sqsModelOpenAIGPT51,
		EnabledProjectKeys: []string{"proj-a", "proj-b", "proj-c"},
	})
	if d.Suggestions.Status != outcomeApplied {
		t.Fatalf("expected applied, got %s", d.Suggestions.Status)
	}
	if !strings.Contains(d.Suggestions.Detail, "3 project") {
		t.Errorf("expected project count in Detail, got %q", d.Suggestions.Detail)
	}
	if d.PatchPayload.AiCodeFix.Enablement != sqcEnablementSome {
		t.Errorf("expected SOME enablement, got %s", d.PatchPayload.AiCodeFix.Enablement)
	}
	if got := d.PatchPayload.AiCodeFix.EnabledProjectKeys; len(got) != 3 || got[0] != "proj-a" {
		t.Errorf("expected source keys carried on payload (apply step remaps), got %v", got)
	}
}

func TestEvaluateAiCodeFix_DisabledOnSQS(t *testing.T) {
	d := EvaluateAiCodeFix(AiCodeFixSourceState{
		HasConfig:  true,
		Enablement: sqcEnablementDisabled,
	})
	if d.Suggestions.Status != outcomeApplied || d.Suggestions.NearPerfect {
		t.Errorf("DISABLED should mirror as Perfect, got %+v", d.Suggestions)
	}
	if d.PatchPayload.AiCodeFix.Enablement != sqcEnablementDisabled {
		t.Errorf("expected DISABLED on SQC, got %s", d.PatchPayload.AiCodeFix.Enablement)
	}
}

func TestEvaluateAiCodeFix_MissingExtract(t *testing.T) {
	// Older SQS versions don't expose the fix-suggestions endpoint;
	// without it we can't pick a provider/model. Stay silent rather
	// than emit a noisy Skipped row — operators who actually used
	// AI Code Fix would have hidden=true (covered separately) or a
	// customised sonar.ai.suggestions.enabled.
	d := EvaluateAiCodeFix(AiCodeFixSourceState{HasConfig: false})
	if d.Suggestions.Status != "" {
		t.Errorf("missing extract should suppress the row, got %q", d.Suggestions.Status)
	}
	if d.Hidden.Status != "" {
		t.Errorf("missing extract should suppress the hidden row too, got %q", d.Hidden.Status)
	}
	if d.PatchPayload != nil {
		t.Errorf("missing extract should not produce a PATCH, got %+v", d.PatchPayload)
	}
}

func TestEvaluateAiCodeFix_PublicAnthropicPassthrough(t *testing.T) {
	d := EvaluateAiCodeFix(AiCodeFixSourceState{
		HasConfig:        true,
		Enablement:       sqcEnablementAll,
		SelectedProvider: sqcProviderAnthropic,
		SelectedModel:    sqcModelClaudeSonnet4,
	})
	if d.Suggestions.NearPerfect {
		t.Errorf("Anthropic passthrough should be Perfect, got NearPerfect")
	}
	if d.PatchPayload.Provider.Key != sqcProviderAnthropic ||
		d.PatchPayload.Provider.ModelKey != sqcModelClaudeSonnet4 {
		t.Errorf("Anthropic passthrough: %+v", d.PatchPayload.Provider)
	}
}

func TestMapSourceProjectsToOrg(t *testing.T) {
	pm := map[string]projectMapping{
		"https://sqs.example.com/proj-a": {CloudKey: "org_proj-a", OrgKey: "org"},
		"https://sqs.example.com/proj-b": {CloudKey: "org_proj-b", OrgKey: "org"},
		// proj-c is mapped but to a different org — should be dropped.
		"https://sqs.example.com/proj-c": {CloudKey: "other_proj-c", OrgKey: "other"},
	}
	mapped, dropped := mapSourceProjectsToOrg(
		[]string{"proj-a", "proj-b", "proj-c", "proj-missing"},
		"https://sqs.example.com/", "org", pm)
	if len(mapped) != 2 || mapped[0] != "org_proj-a" || mapped[1] != "org_proj-b" {
		t.Errorf("unexpected mapped: %v", mapped)
	}
	if dropped != 2 {
		t.Errorf("expected 2 dropped (other-org + missing), got %d", dropped)
	}
}
