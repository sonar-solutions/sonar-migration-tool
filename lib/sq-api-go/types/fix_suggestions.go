package types

// FixSuggestionsOrgConfig is the request body for SonarQube Cloud's
// PATCH /fix-suggestions/organization-configs/{orgId} endpoint
// (api.sonarcloud.io). Pointer fields keep "unset" distinct from "set
// to zero value" so partial PATCH semantics work: only fields the
// caller populates are sent in the JSON body.
type FixSuggestionsOrgConfig struct {
	AiCodeFix *FixSuggestionsAiCodeFix `json:"aiCodeFix,omitempty"`
	Provider  *FixSuggestionsProvider  `json:"provider,omitempty"`
}

// FixSuggestionsAiCodeFix carries the enablement state and (when the
// enablement is ENABLED_FOR_SOME_PROJECTS) the explicit list of SQC
// cloud project keys that have AI Code Fix turned on.
type FixSuggestionsAiCodeFix struct {
	Enablement         string   `json:"enablement,omitempty"`
	EnabledProjectKeys []string `json:"enabledProjectKeys,omitempty"`
}

// FixSuggestionsProvider names the LLM SQC will use for AI Code Fix.
// On SQC the public providers are OPENAI and ANTHROPIC; ModelKey
// values include OPENAI_GPT_5_1 and CLAUDE_SONNET_4.
type FixSuggestionsProvider struct {
	Key      string `json:"key,omitempty"`
	ModelKey string `json:"modelKey,omitempty"`
}
