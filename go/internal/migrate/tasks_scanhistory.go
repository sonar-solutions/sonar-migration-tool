package migrate

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/sonar-solutions/sonar-migration-tool/internal/common"
	"github.com/sonar-solutions/sonar-migration-tool/internal/scanreport"
	pb "github.com/sonar-solutions/sonar-migration-tool/internal/scanreport/proto"
)

func scanHistoryTasks() []TaskDef {
	return []TaskDef{
		{
			Name:         "importScanHistory",
			Editions:     common.AllEditions,
			Dependencies: []string{"createProjects", "setProjectProfiles"},
			Run:          runImportScanHistory,
		},
	}
}

func runImportScanHistory(ctx context.Context, e *Executor) error {
	projects, err := e.Store.ReadAll("createProjects")
	if err != nil {
		return fmt.Errorf("importScanHistory: reading createProjects: %w", err)
	}

	w, err := e.Store.Writer("importScanHistory")
	if err != nil {
		return err
	}

	for i, proj := range projects {
		cloudKey := extractField(proj, "cloud_project_key")
		orgKey := extractField(proj, "sonarcloud_org_key")
		serverURL := extractField(proj, "server_url")
		serverKey := extractField(proj, "key")

		if cloudKey == "" || orgKey == "" {
			continue
		}

		e.Logger.Info("importing scan history", "project", cloudKey, "progress", fmt.Sprintf("%d/%d", i+1, len(projects)))

		branches := collectBranches(e, serverURL, serverKey)
		if len(branches) == 0 {
			branches = []string{"main"}
		}

		for _, branch := range branches {
			result, err := importBranch(ctx, e, importBranchInput{
				CloudKey:  cloudKey,
				OrgKey:    orgKey,
				ServerURL: serverURL,
				ServerKey: serverKey,
				Branch:    branch,
			})
			if err != nil {
				e.Logger.Warn("scan history import failed", "project", cloudKey, "branch", branch, "err", err)
				result = &importResult{Status: "failed", Error: err.Error()}
			}

			record, _ := json.Marshal(map[string]any{
				"cloud_project_key": cloudKey,
				"branch":            branch,
				"status":            result.Status,
				"task_id":           result.TaskID,
				"error":             result.Error,
			})
			w.WriteOne(record)
		}
	}
	return nil
}

type importBranchInput struct {
	CloudKey  string
	OrgKey    string
	ServerURL string
	ServerKey string
	Branch    string
}

type importResult struct {
	Status string
	TaskID string
	Error  string
}

func importBranch(ctx context.Context, e *Executor, input importBranchInput) (*importResult, error) {
	issues := loadExtractedIssues(e, input.ServerURL, input.ServerKey, input.Branch)
	allComponents := loadExtractedComponents(e, input.ServerURL, input.ServerKey, input.Branch)
	sources := loadExtractedSources(e, input.ServerURL, input.ServerKey, input.Branch)
	activeRules := loadExtractedActiveRules(e, input.ServerURL, input.ServerKey)
	qprofiles := loadExtractedQProfiles(e, input.ServerURL, input.ServerKey)

	// Only include components that have matching source code (matches cloudvoyager behavior).
	sourceKeys := buildSourceKeySet(sources)
	components := filterComponentsWithSource(allComponents, sourceKeys)

	if len(components) == 0 {
		return &importResult{Status: "skipped"}, nil
	}

	root, fileComps, cr := scanreport.BuildComponents(input.CloudKey, input.CloudKey, components)
	pbIssues := scanreport.BuildIssues(issues, cr)
	pbActiveRules := scanreport.BuildActiveRules(activeRules)

	now := time.Now()
	changesetsRef := buildChangesetMap(cr, components, now)

	// BackdateChangesets uses component keys (strings), so build a parallel map.
	changesetsKey := make(map[string]*pb.Changesets)
	refToKey := make(map[int32]string)
	for _, comp := range components {
		if ref, ok := cr.Refs()[comp.Key]; ok {
			if cs, ok := changesetsRef[ref]; ok {
				changesetsKey[comp.Key] = cs
				refToKey[ref] = comp.Key
			}
		}
	}

	extractedIssues := toExtractedIssues(issues, e)
	scanreport.BackdateChangesets(extractedIssues, changesetsKey, now)

	// Copy back to ref-keyed map after backdating.
	changesets := make(map[int32]*pb.Changesets)
	for ref, key := range refToKey {
		changesets[ref] = changesetsKey[key]
	}

	fileCounts := countFilesByExt(components)

	metadata := scanreport.BuildMetadata(scanreport.MetadataInput{
		AnalysisDate:   now,
		OrgKey:         input.OrgKey,
		ProjectKey:     input.CloudKey,
		BranchName:     input.Branch,
		BranchType:     pb.Metadata_BRANCH,
		QProfiles:      qprofiles,
		FileCountByExt: fileCounts,
	}, root.Ref)

	pbSources := make(map[int32]string)
	for _, s := range sources {
		if ref, ok := cr.Refs()[s.Component]; ok {
			pbSources[ref] = s.Source
		}
	}

	reportData := &scanreport.ReportData{
		Metadata:       metadata,
		RootComponent:  root,
		FileComponents: fileComps,
		Issues:         pbIssues,
		Measures:       make(map[int32][]*pb.Measure),
		Changesets:     changesets,
		ActiveRules:    pbActiveRules,
		Sources:        pbSources,
	}

	zipBytes, err := scanreport.PackageReport(reportData)
	if err != nil {
		return nil, fmt.Errorf("packaging report: %w", err)
	}

	cfg := scanreport.SubmitConfig{
		CloudURL:   e.CloudURL,
		ProjectKey: input.CloudKey,
		OrgKey:     input.OrgKey,
		BranchName: input.Branch,
	}

	result, err := scanreport.SubmitReport(ctx, e.Raw.HTTPClient(), cfg, zipBytes)
	if err != nil {
		return nil, fmt.Errorf("submitting report: %w", err)
	}

	e.Logger.Info("CE task submitted", "project", input.CloudKey, "branch", input.Branch, "taskId", result.TaskID)

	if err := scanreport.PollCETask(ctx, e.Raw.HTTPClient(), e.CloudURL, result.TaskID, e.Logger); err != nil {
		return nil, fmt.Errorf("CE task failed: %w", err)
	}

	return &importResult{Status: "success", TaskID: result.TaskID}, nil
}

// collectBranches reads extracted branch data for a project.
func collectBranches(e *Executor, serverURL, serverKey string) []string {
	items, err := readExtractItems(e, "getBranches")
	if err != nil {
		return nil
	}
	var branches []string
	for _, item := range items {
		if item.ServerURL != serverURL {
			continue
		}
		projKey := extractField(item.Data, "projectKey")
		if projKey != serverKey {
			continue
		}
		branchType := strings.ToUpper(extractField(item.Data, "type"))
		if branchType == "SHORT" {
			continue
		}
		name := extractField(item.Data, "name")
		if name != "" {
			branches = append(branches, name)
		}
	}
	return branches
}

type sourceRecord struct {
	Component string
	Source    string
}

func loadExtractedSources(e *Executor, serverURL, serverKey, branch string) []sourceRecord {
	items, err := readExtractItems(e, "getProjectSourceCode")
	if err != nil {
		return nil
	}
	var sources []sourceRecord
	for _, item := range items {
		if item.ServerURL != serverURL {
			continue
		}
		if extractField(item.Data, "projectKey") != serverKey {
			continue
		}
		if extractField(item.Data, "branch") != branch {
			continue
		}
		sources = append(sources, sourceRecord{
			Component: extractField(item.Data, "key"),
			Source:    extractField(item.Data, "source"),
		})
	}
	return sources
}

func loadExtractedIssues(e *Executor, serverURL, serverKey, branch string) []scanreport.IssueInput {
	items, err := readExtractItems(e, "getProjectIssuesFull")
	if err != nil {
		return nil
	}
	var issues []scanreport.IssueInput
	for _, item := range items {
		if item.ServerURL != serverURL {
			continue
		}
		if extractField(item.Data, "projectKey") != serverKey {
			continue
		}
		if extractField(item.Data, "branch") != branch {
			continue
		}
		rule := extractField(item.Data, "rule")
		repo, key := splitRule(rule)
		issues = append(issues, scanreport.IssueInput{
			RuleRepo:  repo,
			RuleKey:   key,
			Message:   extractField(item.Data, "message"),
			Severity:  extractField(item.Data, "severity"),
			StartLine: extractInt32(item.Data, "textRange", "startLine"),
			EndLine:   extractInt32(item.Data, "textRange", "endLine"),
			StartOff:  extractInt32(item.Data, "textRange", "startOffset"),
			EndOff:    extractInt32(item.Data, "textRange", "endOffset"),
			Component: extractField(item.Data, "component"),
		})
	}
	return issues
}

func loadExtractedComponents(e *Executor, serverURL, serverKey, branch string) []scanreport.ComponentInput {
	items, err := readExtractItems(e, "getProjectComponentTree")
	if err != nil {
		return nil
	}
	var components []scanreport.ComponentInput
	for _, item := range items {
		if item.ServerURL != serverURL {
			continue
		}
		if extractField(item.Data, "projectKey") != serverKey {
			continue
		}
		if extractField(item.Data, "branch") != branch {
			continue
		}
		lines := extractInt32Field(item.Data, "lines")
		if lines == 0 {
			lines = extractMeasureInt32(item.Data, "ncloc")
		}
		components = append(components, scanreport.ComponentInput{
			Key:      extractField(item.Data, "key"),
			Name:     extractField(item.Data, "name"),
			Path:     extractField(item.Data, "path"),
			Language: extractField(item.Data, "language"),
			Lines:    lines,
		})
	}
	return components
}

// sonarCloudRuleRepos lists rule repositories known to exist in SonarCloud.
// Rules from external/third-party repos are excluded from the report to
// prevent CE processing errors.
var sonarCloudRuleRepos = map[string]bool{
	"common-java": true, "java": true, "squid": true,
	"common-js": true, "javascript": true, "typescript": true,
	"common-ts": true, "css": true, "web": true,
	"common-py": true, "python": true, "pythonbugs": true,
	"common-cs": true, "csharpsquid": true, "roslyn.sonaranalyzer.security.cs": true,
	"common-vbnet": true, "vbnet": true,
	"common-kotlin": true, "kotlin": true,
	"common-ruby": true, "ruby": true,
	"common-scala": true, "scala": true,
	"common-go": true, "go": true,
	"common-php": true, "php": true, "phpsecurity": true,
	"common-swift": true, "swift": true,
	"common-c": true, "c": true, "cpp": true, "common-cpp": true,
	"common-objc": true, "objc": true,
	"common-xml": true, "xml": true,
	"common-html": true, "html": true,
	"common-text": true, "text": true, "secrets": true,
	"plsql": true, "tsql": true, "abap": true, "cobol": true, "rpg": true,
	"flex": true, "pli": true, "apex": true, "cloudformation": true,
	"terraform": true, "docker": true, "kubernetes": true,
	"azureresourcemanager": true, "ipython": true,
	"jssecurity": true, "tssecurity": true, "javasecurity": true,
}

func loadExtractedActiveRules(e *Executor, serverURL, serverKey string) []scanreport.ActiveRuleInput {
	items, err := readExtractItems(e, "getActiveProfileRules")
	if err != nil {
		return nil
	}
	var rules []scanreport.ActiveRuleInput
	for _, item := range items {
		if item.ServerURL != serverURL {
			continue
		}
		rule := extractField(item.Data, "key")
		repo, key := splitRule(rule)
		// Only include rules from known SonarCloud repositories.
		if !sonarCloudRuleRepos[repo] {
			continue
		}
		rules = append(rules, scanreport.ActiveRuleInput{
			RuleRepo:    repo,
			RuleKey:     key,
			Severity:    extractField(item.Data, "severity"),
			QProfileKey: extractField(item.Data, "qProfile"),
			Language:    extractField(item.Data, "lang"),
		})
	}
	return rules
}

func loadExtractedQProfiles(e *Executor, serverURL, serverKey string) []scanreport.QProfileInfo {
	items, err := readExtractItems(e, "getProfiles")
	if err != nil {
		return nil
	}
	var profiles []scanreport.QProfileInfo
	for _, item := range items {
		if item.ServerURL != serverURL {
			continue
		}
		profiles = append(profiles, scanreport.QProfileInfo{
			Key:      extractField(item.Data, "key"),
			Name:     extractField(item.Data, "name"),
			Language: extractField(item.Data, "language"),
		})
	}
	return profiles
}

func buildChangesetMap(cr *scanreport.ComponentRef, components []scanreport.ComponentInput, date time.Time) map[int32]*pb.Changesets {
	changesets := make(map[int32]*pb.Changesets)
	for _, comp := range components {
		if ref, ok := cr.Refs()[comp.Key]; ok && comp.Lines > 0 {
			changesets[ref] = scanreport.BuildDefaultChangesets(ref, int(comp.Lines), date)
		}
	}
	return changesets
}

func toExtractedIssues(issues []scanreport.IssueInput, e *Executor) []scanreport.ExtractedIssue {
	fullItems, _ := readExtractItems(e, "getProjectIssuesFull")
	dateMap := make(map[string]time.Time)
	for _, item := range fullItems {
		key := extractField(item.Data, "key")
		dateStr := extractField(item.Data, "creationDate")
		if key != "" && dateStr != "" {
			t, err := time.Parse(time.RFC3339, dateStr)
			if err != nil {
				// SonarQube often returns -0500 instead of -05:00
				t, err = time.Parse("2006-01-02T15:04:05-0700", dateStr)
			}
			if err == nil {
				dateMap[key] = t
			}
		}
	}

	result := make([]scanreport.ExtractedIssue, 0, len(issues))
	for _, iss := range issues {
		result = append(result, scanreport.ExtractedIssue{
			Key:          iss.RuleRepo + ":" + iss.RuleKey,
			Component:    iss.Component,
			CreationDate: dateMap[iss.RuleRepo+":"+iss.RuleKey],
			StartLine:    iss.StartLine,
			EndLine:      iss.EndLine,
		})
	}
	return result
}

// buildSourceKeySet returns a set of component keys that have extracted source code.
func buildSourceKeySet(sources []sourceRecord) map[string]bool {
	keys := make(map[string]bool, len(sources))
	for _, s := range sources {
		keys[s.Component] = true
	}
	return keys
}

// filterComponentsWithSource returns only components that have matching source code.
func filterComponentsWithSource(components []scanreport.ComponentInput, sourceKeys map[string]bool) []scanreport.ComponentInput {
	var filtered []scanreport.ComponentInput
	for _, c := range components {
		if sourceKeys[c.Key] {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

func countFilesByExt(components []scanreport.ComponentInput) map[string]int32 {
	counts := make(map[string]int32)
	for _, c := range components {
		if c.Language != "" {
			counts[strings.ToLower(c.Language)]++
		}
	}
	return counts
}

func splitRule(rule string) (string, string) {
	idx := strings.Index(rule, ":")
	if idx < 0 {
		return rule, ""
	}
	return rule[:idx], rule[idx+1:]
}

func extractInt32(data json.RawMessage, objectKey, fieldKey string) int32 {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil {
		return 0
	}
	nested, ok := obj[objectKey]
	if !ok {
		return 0
	}
	return extractInt32Field(nested, fieldKey)
}

func extractInt32Field(data json.RawMessage, key string) int32 {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil {
		return 0
	}
	raw, ok := obj[key]
	if !ok {
		return 0
	}
	var v int32
	json.Unmarshal(raw, &v)
	return v
}

// extractMeasureInt32 reads a numeric value from the "measures" array by metric key.
// The measures array format is: [{"metric":"ncloc","value":"50"}, ...]
func extractMeasureInt32(data json.RawMessage, metricKey string) int32 {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil {
		return 0
	}
	raw, ok := obj["measures"]
	if !ok {
		return 0
	}
	var measures []struct {
		Metric string `json:"metric"`
		Value  string `json:"value"`
	}
	if err := json.Unmarshal(raw, &measures); err != nil {
		return 0
	}
	for _, m := range measures {
		if m.Metric == metricKey {
			v, _ := strconv.ParseInt(m.Value, 10, 32)
			return int32(v)
		}
	}
	return 0
}

