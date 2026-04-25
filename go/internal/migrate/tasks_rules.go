package migrate

import (
	"context"
	"strings"

	"github.com/sonar-solutions/sq-api-go/cloud"
)

// ruleTasks returns tasks for updating rules in Cloud.
func ruleTasks() []TaskDef {
	return []TaskDef{
		{
			Name:         "updateRuleTags",
			Dependencies: []string{"generateOrganizationMappings"},
			Run:          runUpdateRuleTags,
		},
		{
			Name:         "updateRuleDescriptions",
			Dependencies: []string{"generateOrganizationMappings"},
			Run:          runUpdateRuleDescriptions,
		},
	}
}

func runUpdateRuleTags(ctx context.Context, e *Executor) error {
	// Read rule details from extract data.
	items, _ := readExtractItems(e, "getRuleDetails")

	w, err := e.Store.Writer("updateRuleTags")
	if err != nil {
		return err
	}

	for _, item := range items {
		ruleKey := extractField(item.Data, "key")
		tags := extractStringArray(item.Data, "tags")
		if ruleKey == "" || len(tags) == 0 {
			continue
		}

		_, err := e.Cloud.Rules.Update(ctx, cloud.UpdateRuleParams{
			Key:  ruleKey,
			Tags: strings.Join(tags, ","),
		})
		if err != nil {
			e.Logger.Warn("updateRuleTags failed", "rule", ruleKey, "err", err)
			continue
		}
		_ = w.WriteOne(item.Data)
	}
	return nil
}

func runUpdateRuleDescriptions(ctx context.Context, e *Executor) error {
	// Read rule details from extract data.
	items, _ := readExtractItems(e, "getRuleDetails")

	w, err := e.Store.Writer("updateRuleDescriptions")
	if err != nil {
		return err
	}

	for _, item := range items {
		ruleKey := extractField(item.Data, "key")
		note := extractField(item.Data, "mdNote")
		if ruleKey == "" || note == "" {
			continue
		}

		_, err := e.Cloud.Rules.Update(ctx, cloud.UpdateRuleParams{
			Key:          ruleKey,
			MarkdownNote: note,
		})
		if err != nil {
			e.Logger.Warn("updateRuleDescriptions failed", "rule", ruleKey, "err", err)
			continue
		}
		_ = w.WriteOne(item.Data)
	}
	return nil
}
