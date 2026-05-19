package migrate

import (
	"context"
	"strings"

	"github.com/sonar-solutions/sq-api-go/cloud"
	"golang.org/x/sync/errgroup"
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

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(cap(e.Sem))
	for _, item := range items {
		g.Go(func() error {
			ruleKey := extractField(item.Data, "key")
			tags := extractStringArray(item.Data, "tags")
			if ruleKey == "" || len(tags) == 0 {
				return nil
			}

			_, err := e.Cloud.Rules.Update(ctx, cloud.UpdateRuleParams{
				Key:  ruleKey,
				Tags: strings.Join(tags, ","),
			})
			if err != nil {
				e.Logger.Warn("updateRuleTags failed", "rule", ruleKey, "err", err)
				return nil
			}
			_ = w.WriteOne(item.Data)
			return nil
		})
	}
	return g.Wait()
}

func runUpdateRuleDescriptions(ctx context.Context, e *Executor) error {
	// Read rule details from extract data.
	items, _ := readExtractItems(e, "getRuleDetails")

	w, err := e.Store.Writer("updateRuleDescriptions")
	if err != nil {
		return err
	}

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(cap(e.Sem))
	for _, item := range items {
		g.Go(func() error {
			ruleKey := extractField(item.Data, "key")
			note := extractField(item.Data, "mdNote")
			if ruleKey == "" || note == "" {
				return nil
			}

			_, err := e.Cloud.Rules.Update(ctx, cloud.UpdateRuleParams{
				Key:          ruleKey,
				MarkdownNote: note,
			})
			if err != nil {
				e.Logger.Warn("updateRuleDescriptions failed", "rule", ruleKey, "err", err)
				return nil
			}
			_ = w.WriteOne(item.Data)
			return nil
		})
	}
	return g.Wait()
}
