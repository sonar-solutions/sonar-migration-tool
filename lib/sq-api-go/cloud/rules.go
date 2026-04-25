package cloud

import (
	"context"
	"net/url"
	"strings"

	"github.com/sonar-solutions/sq-api-go/types"
)

// RulesClient provides write-path methods for SonarQube Cloud rules.
type RulesClient struct{ baseClient }

// UpdateRuleParams holds the parameters for updating a rule.
// All fields except Key are optional — only non-empty values are sent.
type UpdateRuleParams struct {
	// Key is the rule key, e.g. "java:S1234". Required.
	Key string
	// Tags is a comma-separated list of tags to set. Empty string removes all tags.
	Tags string
	// MarkdownNote is a custom note to attach to the rule.
	MarkdownNote string
}

// Update updates a rule's metadata (tags, note) via /api/rules/update.
func (r *RulesClient) Update(ctx context.Context, params UpdateRuleParams) (*types.Rule, error) {
	form := url.Values{}
	form.Set("key", params.Key)
	if strings.TrimSpace(params.Tags) != "" {
		form.Set("tags", params.Tags)
	}
	if params.MarkdownNote != "" {
		form.Set("markdown_note", params.MarkdownNote)
	}

	var result types.RuleShowResponse
	if err := r.postForm(ctx, "api/rules/update", form, &result); err != nil {
		return nil, err
	}
	return &result.Rule, nil
}
