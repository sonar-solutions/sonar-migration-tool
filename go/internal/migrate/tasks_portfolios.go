package migrate

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"

	"github.com/sonar-solutions/sq-api-go/cloud"
	"github.com/sonar-solutions/sonar-migration-tool/internal/common"
)

// portfolioTasks returns tasks for Enterprise portfolio management.
func portfolioTasks() []TaskDef {
	entEditions := []common.Edition{common.EditionEnterprise, common.EditionDatacenter}

	return []TaskDef{
		{
			Name:         "setPortfolioProjects",
			Editions:     entEditions,
			Dependencies: []string{"createPortfolios", "createProjects"},
			Run:          runSetPortfolioProjects,
		},
		{
			Name:         "configurePortfolios",
			Editions:     entEditions,
			Dependencies: []string{"createPortfolios", "createProjects", "setPortfolioProjects"},
			Run:          runConfigurePortfolios,
		},
	}
}

// runSetPortfolioProjects is intentionally a no-op.
//
// The SonarQube Cloud portfolios PATCH endpoint requires `projects` to be an
// array of `{branchId: <UUID>}` objects — there is no way to assign projects
// by key. Looking up branch UUIDs per project would require an extra round
// trip per project, so we instead translate every selection mode (including
// MANUAL) into a `selection: "regex"` PATCH inside configurePortfolios:
// the regex matches the resolved cloud keys exactly. The task survives only
// as a dependency anchor for configurePortfolios.
func runSetPortfolioProjects(_ context.Context, e *Executor) error {
	w, err := e.Store.Writer("setPortfolioProjects")
	if err != nil {
		return err
	}
	return w.WriteChunk(nil)
}

// runConfigurePortfolios sets the SQC selection mode for every source
// portfolio that has a definition we can translate. Three selection modes
// are migrated:
//
//   - REGEXP: rewrite the SQS regex so it matches under the SQC project-key
//     renaming scheme (orgKey_<sqsKey>).
//   - TAGS:   forward the source tag list verbatim.
//   - MANUAL: turn the resolved cloud project keys into a regex that matches
//     exactly those keys (SQC's PATCH endpoint rejects raw project keys —
//     it requires per-project branch UUIDs we do not hold). This keeps the
//     portfolio populated without an extra branches lookup; the trade-off
//     is that new projects with matching keys will be auto-included, which
//     is acceptable because cloud keys are namespaced by org.
//
// Selection modes NONE and REST are left alone — they map to "empty" on
// SQC, which is the default state after createPortfolios.
//
// When a portfolio's resolved projects span no migrated org (typically for
// REGEXP/TAGS portfolios on an SQS instance that has not yet computed the
// match), the task falls back to every migrated org so the portfolio is
// still scoped and the PATCH still goes out.
func runConfigurePortfolios(ctx context.Context, e *Executor) error {
	projectIndex := buildProjectCloudKeyIndex(e)
	portfolioProjects := buildPortfolioProjectList(e, projectIndex)
	portfolioOrgs := buildPortfolioOrgIndex(e, projectIndex)
	allMigratedOrgs := collectAllMigratedOrgs(projectIndex)

	// Side-channel writer that records per-portfolio task-level failures
	// (skipped because of missing data, etc.) so the summary report can mark
	// them as Failed even when no HTTP request was issued.
	failW, _ := e.Store.Writer("configurePortfolios.failures")
	// Cache of cloud_project_key → main branch UUID, populated lazily via
	// /api/project_branches/list. Required by SQC's PATCH endpoint when
	// selection is "projects".
	branchIDByCloudKey := map[string]string{}
	counter := NewTaskCounter("configurePortfolios")
	err := forEachMigrateItem(ctx, e, "configurePortfolios", "createPortfolios",
		func(ctx context.Context, item json.RawMessage, w *common.ChunkWriter) error {
			selectionMode := strings.ToUpper(extractField(item, "selection_mode"))
			if selectionMode != "REGEXP" && selectionMode != "TAGS" && selectionMode != "MANUAL" {
				return nil
			}
			portfolioID := extractField(item, "cloud_portfolio_id")
			sourceKey := extractField(item, "source_portfolio_key")
			name := extractField(item, "name")
			if portfolioID == "" || sourceKey == "" {
				return nil
			}

			orgs := portfolioOrgs[sourceKey]
			if len(orgs) == 0 && len(allMigratedOrgs) > 0 {
				e.Logger.Info("configurePortfolios: no resolved projects on source, falling back to all migrated orgs",
					"portfolio", sourceKey, "orgs", allMigratedOrgs)
				orgs = allMigratedOrgs
			}

			params := cloud.UpdatePortfolioParams{
				PortfolioID: portfolioID,
			}
			switch selectionMode {
			case "REGEXP":
				params.Selection = "regex"
				params.RegularExpression = transformPortfolioRegex(
					extractField(item, "regexp"), orgs)
			case "TAGS":
				params.Selection = "tags"
				if tagsStr := extractField(item, "tags"); tagsStr != "" {
					params.Tags = strings.Split(tagsStr, ",")
				}
			case "MANUAL":
				cloudKeys := portfolioProjects[sourceKey]
				if len(cloudKeys) == 0 {
					msg := "MANUAL portfolio has no resolved cloud projects to migrate"
					e.Logger.Warn("configurePortfolios: "+msg, "portfolio", sourceKey)
					counter.Fail()
					recordPortfolioFailure(failW, portfolioID, name, msg)
					return nil
				}
				refs, lookupErr := resolveProjectBranchRefs(ctx, e, branchIDByCloudKey, cloudKeys)
				if lookupErr != nil || len(refs) == 0 {
					msg := "could not resolve main branch UUID for any MANUAL portfolio project"
					if lookupErr != nil {
						msg = msg + ": " + lookupErr.Error()
					}
					e.Logger.Warn("configurePortfolios: "+msg, "portfolio", sourceKey)
					counter.Fail()
					recordPortfolioFailure(failW, portfolioID, name, msg)
					return nil
				}
				params.Selection = "projects"
				params.Projects = refs
			}

			e.Logger.Info("configurePortfolios: sending PATCH",
				"portfolio", portfolioID,
				"name", name,
				"selection", params.Selection,
				"regex", params.RegularExpression,
				"tags", params.Tags,
				"projects", len(params.Projects),
			)
			if err := e.CloudAPI.Enterprises.UpdatePortfolio(ctx, params); err != nil {
				counter.Fail()
				logAPIWarn(e.Logger, "configurePortfolios failed", err,
					"portfolio", portfolioID, "selection", selectionMode)
				recordPortfolioFailure(failW, portfolioID, name, "PATCH /enterprises/portfolios/<id>: "+err.Error())
			} else {
				counter.Success()
				e.Logger.Info("configurePortfolios: PATCH succeeded",
					"portfolio", portfolioID, "selection", params.Selection)
			}
			return nil
		})
	counter.LogSummary(e.Logger)
	return err
}

// cloudProjectInfo records the per-project data we need when assembling
// portfolio configurations: the cloud key and the SQC organization it belongs to.
type cloudProjectInfo struct {
	CloudKey string
	OrgKey   string
}

// buildProjectCloudKeyIndex returns a map keyed by serverURL+SQS_project_key →
// {cloud key, sonarcloud org key} built from createProjects JSONL.
func buildProjectCloudKeyIndex(e *Executor) map[string]cloudProjectInfo {
	projects, _ := e.Store.ReadAll("createProjects")
	out := make(map[string]cloudProjectInfo, len(projects))
	for _, p := range projects {
		serverURL := extractField(p, "server_url")
		key := extractField(p, "key")
		if serverURL == "" || key == "" {
			continue
		}
		out[serverURL+key] = cloudProjectInfo{
			CloudKey: extractField(p, "cloud_project_key"),
			OrgKey:   extractField(p, "sonarcloud_org_key"),
		}
	}
	return out
}

// buildPortfolioProjectList resolves each portfolio's project membership to
// the cloud project keys we created. Returns source_portfolio_key →
// []cloud_project_key.
func buildPortfolioProjectList(e *Executor, projectIndex map[string]cloudProjectInfo) map[string][]string {
	items, _ := readExtractItems(e, "getPortfolioProjects")
	out := make(map[string][]string)
	seen := make(map[string]map[string]bool)
	for _, item := range items {
		portfolioKey := extractField(item.Data, "portfolioKey")
		refKey := extractField(item.Data, "refKey")
		info, ok := projectIndex[item.ServerURL+refKey]
		if !ok || info.CloudKey == "" {
			continue
		}
		if seen[portfolioKey] == nil {
			seen[portfolioKey] = map[string]bool{}
		}
		if seen[portfolioKey][info.CloudKey] {
			continue
		}
		seen[portfolioKey][info.CloudKey] = true
		out[portfolioKey] = append(out[portfolioKey], info.CloudKey)
	}
	return out
}

// buildPortfolioOrgIndex returns source_portfolio_key → unique list of
// sonarcloud_org_key values across the portfolio's resolved projects.
func buildPortfolioOrgIndex(e *Executor, projectIndex map[string]cloudProjectInfo) map[string][]string {
	items, _ := readExtractItems(e, "getPortfolioProjects")
	tmp := make(map[string]map[string]bool)
	for _, item := range items {
		portfolioKey := extractField(item.Data, "portfolioKey")
		refKey := extractField(item.Data, "refKey")
		info, ok := projectIndex[item.ServerURL+refKey]
		if !ok || info.OrgKey == "" {
			continue
		}
		if tmp[portfolioKey] == nil {
			tmp[portfolioKey] = map[string]bool{}
		}
		tmp[portfolioKey][info.OrgKey] = true
	}
	out := make(map[string][]string, len(tmp))
	for portfolioKey, set := range tmp {
		for orgKey := range set {
			out[portfolioKey] = append(out[portfolioKey], orgKey)
		}
	}
	return out
}

// collectAllMigratedOrgs returns the unique list of SonarQube Cloud
// organization keys that received at least one project during the run.
// Used as the fallback target set for portfolios whose resolved-projects
// list is empty on the source side.
func collectAllMigratedOrgs(projectIndex map[string]cloudProjectInfo) []string {
	seen := map[string]bool{}
	var out []string
	for _, info := range projectIndex {
		if info.OrgKey == "" || seen[info.OrgKey] {
			continue
		}
		seen[info.OrgKey] = true
		out = append(out, info.OrgKey)
	}
	return out
}

// recordPortfolioFailure appends a sidecar JSONL entry describing a
// per-portfolio failure inside configurePortfolios. The summary report
// reads this file to surface task-level failures (e.g. "no resolved
// projects on source") that never produced an HTTP request and therefore
// don't appear in requests.log.
func recordPortfolioFailure(w *common.ChunkWriter, portfolioID, name, reason string) {
	if w == nil {
		return
	}
	rec, _ := json.Marshal(map[string]any{
		"cloud_portfolio_id": portfolioID,
		"name":               name,
		"reason":             reason,
	})
	_ = w.WriteOne(rec)
}

// resolveProjectBranchRefs maps each cloud project key to its main branch
// UUID via /api/project_branches/list and returns the list of
// PortfolioProjectRef objects required by the SQC enterprise portfolios
// PATCH endpoint when selection is "projects". The cache is populated
// lazily so repeated portfolios with overlapping projects share lookups.
//
// If any individual lookup fails, it is logged at Warn level and the
// project is skipped — the rest still get migrated. Returns an error only
// when no UUIDs at all could be resolved (e.g. the endpoint is unreachable).
func resolveProjectBranchRefs(ctx context.Context, e *Executor, cache map[string]string, cloudKeys []string) ([]cloud.PortfolioProjectRef, error) {
	refs := make([]cloud.PortfolioProjectRef, 0, len(cloudKeys))
	var firstErr error
	for _, key := range cloudKeys {
		if key == "" {
			continue
		}
		uuid, ok := cache[key]
		if !ok {
			id, err := e.Cloud.Branches.MainBranchID(ctx, key)
			if err != nil {
				e.Logger.Warn("configurePortfolios: main branch UUID lookup failed, project will be omitted",
					"project", key, "err", err)
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
			cache[key] = id
			uuid = id
		}
		refs = append(refs, cloud.PortfolioProjectRef{BranchID: uuid})
	}
	if len(refs) == 0 {
		if firstErr != nil {
			return nil, firstErr
		}
		return nil, nil
	}
	return refs, nil
}

// transformPortfolioRegex rewrites a source-side regex so that it matches
// projects under the SQC project-key renaming scheme (cloud key =
// <orgKey>_<sqsKey>). When the portfolio's projects span multiple orgs, the
// generated prefix is an alternation of org keys.
//
// Examples (orgKeys = ["org1"]):
//
//	"^backend-"    → "^org1_backend-"
//	"^[A-Z].*"     → "^org1_[A-Z].*"
//	"backend"      → "org1_backend"
//
// With orgKeys = ["org1","org2"]:
//
//	"^foo" → "^(?:org1_|org2_)foo"
func transformPortfolioRegex(regex string, orgKeys []string) string {
	if regex == "" {
		return ""
	}
	seen := map[string]bool{}
	prefixes := make([]string, 0, len(orgKeys))
	for _, o := range orgKeys {
		if o == "" || seen[o] {
			continue
		}
		seen[o] = true
		prefixes = append(prefixes, regexp.QuoteMeta(o+"_"))
	}
	if len(prefixes) == 0 {
		return regex
	}
	var prefix string
	if len(prefixes) == 1 {
		prefix = prefixes[0]
	} else {
		prefix = "(?:" + strings.Join(prefixes, "|") + ")"
	}
	if strings.HasPrefix(regex, "^") {
		return "^" + prefix + regex[1:]
	}
	return prefix + regex
}

