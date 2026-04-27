package extract

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// scanHistoryTasks returns extract tasks needed for scan history migration.
// These tasks extract full issue data, component trees, source code, and SCM
// blame data — all per-project, per-branch.
func scanHistoryTasks() []TaskDef {
	return []TaskDef{
		{
			Name:         "getProjectIssuesFull",
			Editions:     AllEditions,
			Dependencies: []string{"getProjects", "getBranches"},
			Run:          projectIssuesFullTask(),
		},
		{
			Name:         "getProjectComponentTree",
			Editions:     AllEditions,
			Dependencies: []string{"getProjects", "getBranches"},
			Run:          projectComponentTreeTask(),
		},
		{
			Name:         "getProjectSourceCode",
			Editions:     AllEditions,
			Dependencies: []string{"getProjectComponentTree"},
			Run:          projectSourceCodeTask(),
		},
		{
			Name:         "getProjectSCMData",
			Editions:     AllEditions,
			Dependencies: []string{"getProjectComponentTree"},
			Run:          projectSCMDataTask(),
		},
	}
}

// projectIssuesFullTask extracts all open issues per project per branch with
// full metadata (creation date, text range, flows, etc.).
func projectIssuesFullTask() func(ctx context.Context, e *Executor) error {
	return func(ctx context.Context, e *Executor) error {
		return forEachProjectBranch(ctx, e, "getProjectIssuesFull",
			func(ctx context.Context, projectKey, branch string, w *ChunkWriter) error {
				params := url.Values{
					"components":   {projectKey},
					"branch":       {branch},
					"ps":           {"500"},
					"statuses":     {"OPEN,CONFIRMED,REOPENED"},
					"additionalFields": {"_all"},
				}
				items, err := e.Raw.GetPaginated(ctx, PaginatedOpts{
					Path:      issuesSearchAPI,
					Params:    params,
					ResultKey: "issues",
					PageLimit: 20, // SonarQube caps at 10,000 results
				})
				if err != nil {
					if isNonFatalHTTPErr(err) {
						e.Logger.Warn("getProjectIssuesFull skipped", "project", projectKey, "branch", branch, "err", err)
						return nil
					}
					return err
				}
				meta := map[string]any{
					"projectKey": projectKey,
					"branch":     branch,
					"serverUrl":  e.ServerURL,
				}
				return w.WriteChunk(enrichAll(items, meta))
			})
	}
}

// projectComponentTreeTask extracts the file component tree per project per branch.
func projectComponentTreeTask() func(ctx context.Context, e *Executor) error {
	return func(ctx context.Context, e *Executor) error {
		return forEachProjectBranch(ctx, e, "getProjectComponentTree",
			func(ctx context.Context, projectKey, branch string, w *ChunkWriter) error {
				params := url.Values{
					"component":  {projectKey},
					"branch":     {branch},
					"qualifiers": {"FIL"},
					"metricKeys": {"ncloc"},
					"ps":         {"500"},
				}
				items, err := e.Raw.GetPaginated(ctx, PaginatedOpts{
					Path:      "api/measures/component_tree",
					Params:    params,
					ResultKey: "components",
					PageLimit: 20, // SonarQube caps at 10,000 results
				})
				if err != nil {
					if isNonFatalHTTPErr(err) {
						e.Logger.Warn("getProjectComponentTree skipped", "project", projectKey, "branch", branch, "err", err)
						return nil
					}
					return err
				}
				meta := map[string]any{
					"projectKey": projectKey,
					"branch":     branch,
					"serverUrl":  e.ServerURL,
				}
				return w.WriteChunk(enrichAll(items, meta))
			})
	}
}

// projectSourceCodeTask extracts source code for each file component.
func projectSourceCodeTask() func(ctx context.Context, e *Executor) error {
	return func(ctx context.Context, e *Executor) error {
		return forEachDep(ctx, e, "getProjectSourceCode", "getProjectComponentTree",
			func(ctx context.Context, item json.RawMessage, w *ChunkWriter) error {
				fileKey := extractField(item, "key")
				branch := extractField(item, "branch")
				if fileKey == "" {
					return nil
				}
				params := url.Values{"key": {fileKey}}
				if branch != "" {
					params.Set("branch", branch)
				}
				raw, err := e.Raw.GetRaw(ctx, "api/sources/raw", params)
				if err != nil {
					if isNonFatalHTTPErr(err) {
						e.Logger.Warn("getProjectSourceCode skipped", "file", fileKey, "err", err)
						return nil
					}
					return err
				}
				record := map[string]any{
					"key":        fileKey,
					"branch":     branch,
					"projectKey": extractField(item, "projectKey"),
					"source":     string(raw),
					"serverUrl":  e.ServerURL,
				}
				b, err := json.Marshal(record)
				if err != nil {
					return err
				}
				return w.WriteOne(b)
			})
	}
}

// projectSCMDataTask extracts SCM blame data for each file component.
func projectSCMDataTask() func(ctx context.Context, e *Executor) error {
	return func(ctx context.Context, e *Executor) error {
		return forEachDep(ctx, e, "getProjectSCMData", "getProjectComponentTree",
			func(ctx context.Context, item json.RawMessage, w *ChunkWriter) error {
				fileKey := extractField(item, "key")
				branch := extractField(item, "branch")
				if fileKey == "" {
					return nil
				}
				params := url.Values{"key": {fileKey}}
				if branch != "" {
					params.Set("branch", branch)
				}
				raw, err := e.Raw.Get(ctx, "api/sources/scm", params)
				if err != nil {
					if isNonFatalHTTPErr(err) {
						e.Logger.Warn("getProjectSCMData skipped", "file", fileKey, "err", err)
						return nil
					}
					return err
				}
				return w.WriteOne(EnrichRaw(raw, map[string]any{
					"key":        fileKey,
					"branch":     branch,
					"projectKey": extractField(item, "projectKey"),
					"serverUrl":  e.ServerURL,
				}))
			})
	}
}

// forEachProjectBranch iterates over all projects and their branches,
// calling fn for each project+branch combination.
func forEachProjectBranch(ctx context.Context, e *Executor, taskName string,
	fn func(ctx context.Context, projectKey, branch string, w *ChunkWriter) error) error {

	projects, err := e.Store.ReadAll("getProjects")
	if err != nil {
		return fmt.Errorf("%s: reading projects: %w", taskName, err)
	}
	branches, err := e.Store.ReadAll("getBranches")
	if err != nil {
		return fmt.Errorf("%s: reading branches: %w", taskName, err)
	}

	// Build a lookup of project -> branch names.
	branchMap := buildBranchMap(branches)

	w, err := e.Store.Writer(taskName)
	if err != nil {
		return err
	}

	for _, proj := range projects {
		projectKey := extractField(proj, "key")
		if projectKey == "" || e.IsSkipped(projectKey) {
			continue
		}
		projBranches := branchMap[projectKey]
		if len(projBranches) == 0 {
			projBranches = []string{"main"}
		}
		for _, branch := range projBranches {
			if err := acquireSem(ctx, e.Sem); err != nil {
				return err
			}
			err := fn(ctx, projectKey, branch, w)
			<-e.Sem
			if err != nil {
				return fmt.Errorf("%s [%s/%s]: %w", taskName, projectKey, branch, err)
			}
		}
	}
	return nil
}

// buildBranchMap builds a map of projectKey -> []branchName from extracted branch data.
func buildBranchMap(branches []json.RawMessage) map[string][]string {
	result := make(map[string][]string)
	for _, item := range branches {
		projectKey := extractField(item, "projectKey")
		name := extractField(item, "name")
		if projectKey == "" || name == "" {
			continue
		}
		// Only include long-lived branches (not short-lived/PR branches).
		branchType := strings.ToUpper(extractField(item, "type"))
		if branchType == "SHORT" {
			continue
		}
		result[projectKey] = append(result[projectKey], name)
	}
	return result
}
