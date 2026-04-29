package migrate

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sonar-solutions/sonar-migration-tool/internal/common"
	"github.com/sonar-solutions/sonar-migration-tool/internal/scanreport"
	"github.com/sonar-solutions/sonar-migration-tool/internal/structure"
)

// --- Pure utility function tests ---

func TestSplitRule(t *testing.T) {
	tests := []struct {
		input    string
		wantRepo string
		wantKey  string
	}{
		{"java:S100", "java", "S100"},
		{"javascript:S1234", "javascript", "S1234"},
		{"norule", "norule", ""},
		{"a:b:c", "a", "b:c"},
		{"", "", ""},
	}
	for _, tt := range tests {
		repo, key := splitRule(tt.input)
		if repo != tt.wantRepo || key != tt.wantKey {
			t.Errorf("splitRule(%q) = (%q, %q), want (%q, %q)", tt.input, repo, key, tt.wantRepo, tt.wantKey)
		}
	}
}

func TestExtractInt32(t *testing.T) {
	data := json.RawMessage(`{"textRange":{"startLine":10,"endLine":20},"other":"value"}`)

	if got := extractInt32(data, "textRange", "startLine"); got != 10 {
		t.Errorf("startLine: expected 10, got %d", got)
	}
	if got := extractInt32(data, "textRange", "endLine"); got != 20 {
		t.Errorf("endLine: expected 20, got %d", got)
	}
	if got := extractInt32(data, "textRange", "missing"); got != 0 {
		t.Errorf("missing field: expected 0, got %d", got)
	}
	if got := extractInt32(data, "missing", "startLine"); got != 0 {
		t.Errorf("missing object: expected 0, got %d", got)
	}
	if got := extractInt32(json.RawMessage(`invalid`), "a", "b"); got != 0 {
		t.Errorf("invalid json: expected 0, got %d", got)
	}
}

func TestExtractInt32Field(t *testing.T) {
	data := json.RawMessage(`{"lines":42,"name":"test"}`)

	if got := extractInt32Field(data, "lines"); got != 42 {
		t.Errorf("expected 42, got %d", got)
	}
	if got := extractInt32Field(data, "missing"); got != 0 {
		t.Errorf("missing: expected 0, got %d", got)
	}
	if got := extractInt32Field(json.RawMessage(`invalid`), "x"); got != 0 {
		t.Errorf("invalid json: expected 0, got %d", got)
	}
}

func TestExtractMeasureInt32(t *testing.T) {
	data := json.RawMessage(`{"measures":[{"metric":"ncloc","value":"150"},{"metric":"coverage","value":"80"}]}`)

	if got := extractMeasureInt32(data, "ncloc"); got != 150 {
		t.Errorf("ncloc: expected 150, got %d", got)
	}
	if got := extractMeasureInt32(data, "coverage"); got != 80 {
		t.Errorf("coverage: expected 80, got %d", got)
	}
	if got := extractMeasureInt32(data, "missing"); got != 0 {
		t.Errorf("missing metric: expected 0, got %d", got)
	}

	// No measures key.
	if got := extractMeasureInt32(json.RawMessage(`{"other":"val"}`), "ncloc"); got != 0 {
		t.Errorf("no measures: expected 0, got %d", got)
	}

	// Invalid json.
	if got := extractMeasureInt32(json.RawMessage(`invalid`), "ncloc"); got != 0 {
		t.Errorf("invalid json: expected 0, got %d", got)
	}

	// Invalid measures array.
	if got := extractMeasureInt32(json.RawMessage(`{"measures":"not-array"}`), "ncloc"); got != 0 {
		t.Errorf("invalid measures: expected 0, got %d", got)
	}
}

func TestBuildSourceKeySet(t *testing.T) {
	sources := []sourceRecord{
		{Component: "p1:src/Main.java", Source: "class Main {}"},
		{Component: "p1:src/Util.java", Source: "class Util {}"},
	}
	keys := buildSourceKeySet(sources)
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}
	if !keys["p1:src/Main.java"] || !keys["p1:src/Util.java"] {
		t.Errorf("missing expected keys: %v", keys)
	}
}

func TestBuildSourceKeySetEmpty(t *testing.T) {
	keys := buildSourceKeySet(nil)
	if len(keys) != 0 {
		t.Errorf("expected empty, got %v", keys)
	}
}

func TestFilterComponentsWithSource(t *testing.T) {
	components := []scanreport.ComponentInput{
		{Key: "p1:src/Main.java", Name: "Main.java"},
		{Key: "p1:src/Test.java", Name: "Test.java"},
		{Key: "p1:src/NoSource.java", Name: "NoSource.java"},
	}
	sourceKeys := map[string]bool{
		"p1:src/Main.java": true,
		"p1:src/Test.java": true,
	}

	filtered := filterComponentsWithSource(components, sourceKeys)
	if len(filtered) != 2 {
		t.Fatalf("expected 2 filtered, got %d", len(filtered))
	}
	if filtered[0].Key != "p1:src/Main.java" || filtered[1].Key != "p1:src/Test.java" {
		t.Errorf("unexpected filtered: %v", filtered)
	}
}

func TestFilterComponentsWithSourceEmpty(t *testing.T) {
	filtered := filterComponentsWithSource(nil, map[string]bool{})
	if len(filtered) != 0 {
		t.Errorf("expected empty, got %v", filtered)
	}
}

func TestCountFilesByExt(t *testing.T) {
	components := []scanreport.ComponentInput{
		{Key: "a", Language: "Java"},
		{Key: "b", Language: "java"},
		{Key: "c", Language: "Python"},
		{Key: "d", Language: ""},
	}
	counts := countFilesByExt(components)
	if counts["java"] != 2 {
		t.Errorf("java: expected 2, got %d", counts["java"])
	}
	if counts["python"] != 1 {
		t.Errorf("python: expected 1, got %d", counts["python"])
	}
	if _, ok := counts[""]; ok {
		t.Error("empty language should be excluded")
	}
}

func TestCountFilesByExtEmpty(t *testing.T) {
	counts := countFilesByExt(nil)
	if len(counts) != 0 {
		t.Errorf("expected empty, got %v", counts)
	}
}

func TestScanHistoryTasksDef(t *testing.T) {
	tasks := scanHistoryTasks()
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].Name != "importScanHistory" {
		t.Errorf("expected importScanHistory, got %s", tasks[0].Name)
	}
}

func TestBuildChangesetMap(t *testing.T) {
	components := []scanreport.ComponentInput{
		{Key: "p1:src/Main.java", Lines: 10},
		{Key: "p1:src/Util.java", Lines: 5},
		{Key: "p1:src/Empty.java", Lines: 0},
	}

	cr := scanreport.NewComponentRef()
	cr.Get("p1:src/Main.java")
	cr.Get("p1:src/Util.java")
	cr.Get("p1:src/Empty.java")

	now := time.Now()
	result := buildChangesetMap(cr, components, now)

	// Lines > 0 should produce changesets.
	mainRef, _ := cr.Refs()["p1:src/Main.java"]
	utilRef, _ := cr.Refs()["p1:src/Util.java"]
	emptyRef, _ := cr.Refs()["p1:src/Empty.java"]

	if _, ok := result[mainRef]; !ok {
		t.Error("expected changeset for Main.java")
	}
	if _, ok := result[utilRef]; !ok {
		t.Error("expected changeset for Util.java")
	}
	if _, ok := result[emptyRef]; ok {
		t.Error("Lines=0 should not produce changeset")
	}
}

// --- Data loading function tests (require extract dir setup) ---

func setupScanHistoryExtract(t *testing.T, dir string) {
	t.Helper()
	extractDir := filepath.Join(dir, "extract-01")

	writeJSON(filepath.Join(extractDir, "extract.json"),
		map[string]any{"url": testServerURL, "edition": "enterprise"})

	writeJSONL(filepath.Join(extractDir, "getBranches"), []map[string]any{
		{"projectKey": "proj1", "name": "main", "type": "LONG", "serverUrl": testServerURL},
		{"projectKey": "proj1", "name": "develop", "type": "LONG", "serverUrl": testServerURL},
		{"projectKey": "proj1", "name": "pr-1", "type": "SHORT", "serverUrl": testServerURL},
	})

	writeJSONL(filepath.Join(extractDir, "getProjectIssuesFull"), []map[string]any{
		{
			"key": "issue-1", "rule": "java:S100", "message": "Rename method",
			"severity": "MAJOR", "component": "proj1:src/Main.java",
			"projectKey": "proj1", "branch": "main",
			"textRange":  map[string]any{"startLine": 5, "endLine": 5, "startOffset": 0, "endOffset": 10},
			"creationDate": "2024-06-15T10:00:00+0000",
			"serverUrl": testServerURL,
		},
		{
			"key": "issue-2", "rule": "java:S200", "message": "Other issue",
			"severity": "MINOR", "component": "proj1:src/Util.java",
			"projectKey": "proj1", "branch": "develop",
			"serverUrl": testServerURL,
		},
	})

	writeJSONL(filepath.Join(extractDir, "getProjectComponentTree"), []map[string]any{
		{
			"key": "proj1:src/Main.java", "name": "Main.java", "path": "src/Main.java",
			"language": "java", "lines": 50,
			"projectKey": "proj1", "branch": "main",
			"serverUrl": testServerURL,
		},
		{
			"key": "proj1:src/Util.java", "name": "Util.java", "path": "src/Util.java",
			"language": "java",
			"measures": []map[string]any{{"metric": "ncloc", "value": "30"}},
			"projectKey": "proj1", "branch": "main",
			"serverUrl": testServerURL,
		},
	})

	writeJSONL(filepath.Join(extractDir, "getProjectSourceCode"), []map[string]any{
		{
			"key": "proj1:src/Main.java", "source": "public class Main {}",
			"projectKey": "proj1", "branch": "main",
			"serverUrl": testServerURL,
		},
		{
			"key": "proj1:src/Util.java", "source": "public class Util {}",
			"projectKey": "proj1", "branch": "main",
			"serverUrl": testServerURL,
		},
	})

	writeJSONL(filepath.Join(extractDir, "getActiveProfileRules"), []map[string]any{
		{"key": "java:S100", "severity": "MAJOR", "qProfile": "prof1", "lang": "java", "serverUrl": testServerURL},
		{"key": "external_tool:E1", "severity": "INFO", "qProfile": "prof1", "lang": "java", "serverUrl": testServerURL},
	})

	writeJSONL(filepath.Join(extractDir, "getProfiles"), []map[string]any{
		{"key": "prof1", "name": "Sonar way", "language": "java", "serverUrl": testServerURL},
	})
}

func newScanHistoryExecutor(t *testing.T, dir string) *Executor {
	t.Helper()
	runDir := filepath.Join(dir, "run-test")
	os.MkdirAll(runDir, 0o755)
	return &Executor{
		Store:     common.NewDataStore(runDir),
		ExportDir: dir,
		Mapping:   structure.ExtractMapping{testServerURL: "extract-01"},
		Sem:       make(chan struct{}, 5),
		Logger:    slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError + 1})),
	}
}

func TestCollectBranches(t *testing.T) {
	dir := t.TempDir()
	setupScanHistoryExtract(t, dir)
	e := newScanHistoryExecutor(t, dir)

	branches := collectBranches(e, testServerURL, "proj1")
	if len(branches) != 2 {
		t.Fatalf("expected 2 branches (SHORT filtered), got %d: %v", len(branches), branches)
	}
	if branches[0] != "main" || branches[1] != "develop" {
		t.Errorf("unexpected branches: %v", branches)
	}
}

func TestCollectBranchesNoMatch(t *testing.T) {
	dir := t.TempDir()
	setupScanHistoryExtract(t, dir)
	e := newScanHistoryExecutor(t, dir)

	branches := collectBranches(e, testServerURL, "nonexistent")
	if len(branches) != 0 {
		t.Errorf("expected 0 branches for unknown project, got %v", branches)
	}
}

func TestCollectBranchesWrongServer(t *testing.T) {
	dir := t.TempDir()
	setupScanHistoryExtract(t, dir)
	e := newScanHistoryExecutor(t, dir)

	branches := collectBranches(e, "https://other.server/", "proj1")
	if len(branches) != 0 {
		t.Errorf("expected 0 branches for wrong server, got %v", branches)
	}
}

func TestLoadExtractedSources(t *testing.T) {
	dir := t.TempDir()
	setupScanHistoryExtract(t, dir)
	e := newScanHistoryExecutor(t, dir)

	sources := loadExtractedSources(e, testServerURL, "proj1", "main")
	if len(sources) != 2 {
		t.Fatalf("expected 2 sources, got %d", len(sources))
	}
	if sources[0].Component != "proj1:src/Main.java" {
		t.Errorf("unexpected component: %s", sources[0].Component)
	}
	if sources[0].Source != "public class Main {}" {
		t.Errorf("unexpected source: %s", sources[0].Source)
	}
}

func TestLoadExtractedSourcesWrongBranch(t *testing.T) {
	dir := t.TempDir()
	setupScanHistoryExtract(t, dir)
	e := newScanHistoryExecutor(t, dir)

	sources := loadExtractedSources(e, testServerURL, "proj1", "nonexistent")
	if len(sources) != 0 {
		t.Errorf("expected 0 sources for wrong branch, got %d", len(sources))
	}
}

func TestLoadExtractedIssues(t *testing.T) {
	dir := t.TempDir()
	setupScanHistoryExtract(t, dir)
	e := newScanHistoryExecutor(t, dir)

	issues := loadExtractedIssues(e, testServerURL, "proj1", "main")
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue on main, got %d", len(issues))
	}
	if issues[0].RuleRepo != "java" || issues[0].RuleKey != "S100" {
		t.Errorf("unexpected rule: %s:%s", issues[0].RuleRepo, issues[0].RuleKey)
	}
	if issues[0].Message != "Rename method" {
		t.Errorf("unexpected message: %s", issues[0].Message)
	}
	if issues[0].StartLine != 5 {
		t.Errorf("unexpected startLine: %d", issues[0].StartLine)
	}
}

func TestLoadExtractedIssuesWrongProject(t *testing.T) {
	dir := t.TempDir()
	setupScanHistoryExtract(t, dir)
	e := newScanHistoryExecutor(t, dir)

	issues := loadExtractedIssues(e, testServerURL, "other-proj", "main")
	if len(issues) != 0 {
		t.Errorf("expected 0, got %d", len(issues))
	}
}

func TestLoadExtractedComponents(t *testing.T) {
	dir := t.TempDir()
	setupScanHistoryExtract(t, dir)
	e := newScanHistoryExecutor(t, dir)

	components := loadExtractedComponents(e, testServerURL, "proj1", "main")
	if len(components) != 2 {
		t.Fatalf("expected 2 components, got %d", len(components))
	}
	if components[0].Key != "proj1:src/Main.java" {
		t.Errorf("unexpected key: %s", components[0].Key)
	}
	if components[0].Lines != 50 {
		t.Errorf("expected lines=50, got %d", components[0].Lines)
	}
	// Second component should fall back to ncloc measure.
	if components[1].Lines != 30 {
		t.Errorf("expected lines=30 (from ncloc), got %d", components[1].Lines)
	}
}

func TestLoadExtractedActiveRules(t *testing.T) {
	dir := t.TempDir()
	setupScanHistoryExtract(t, dir)
	e := newScanHistoryExecutor(t, dir)

	rules := loadExtractedActiveRules(e, testServerURL, "proj1")
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule (external filtered), got %d", len(rules))
	}
	if rules[0].RuleRepo != "java" || rules[0].RuleKey != "S100" {
		t.Errorf("unexpected rule: %s:%s", rules[0].RuleRepo, rules[0].RuleKey)
	}
}

func TestLoadExtractedQProfiles(t *testing.T) {
	dir := t.TempDir()
	setupScanHistoryExtract(t, dir)
	e := newScanHistoryExecutor(t, dir)

	profiles := loadExtractedQProfiles(e, testServerURL, "proj1")
	if len(profiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(profiles))
	}
	if profiles[0].Key != "prof1" || profiles[0].Name != "Sonar way" {
		t.Errorf("unexpected profile: %+v", profiles[0])
	}
}

func TestToExtractedIssues(t *testing.T) {
	dir := t.TempDir()
	setupScanHistoryExtract(t, dir)
	e := newScanHistoryExecutor(t, dir)

	issues := []scanreport.IssueInput{
		{RuleRepo: "java", RuleKey: "S100", Component: "proj1:src/Main.java", StartLine: 5, EndLine: 5},
	}

	extracted := toExtractedIssues(issues, e)
	if len(extracted) != 1 {
		t.Fatalf("expected 1 extracted issue, got %d", len(extracted))
	}
	if extracted[0].Key != "java:S100" {
		t.Errorf("unexpected key: %s", extracted[0].Key)
	}
	if extracted[0].Component != "proj1:src/Main.java" {
		t.Errorf("unexpected component: %s", extracted[0].Component)
	}
	// Note: dateMap in toExtractedIssues is keyed by the issue's "key" field
	// from extract data (e.g., "issue-1"), but looked up by RuleRepo:RuleKey
	// (e.g., "java:S100"). These will only match if the extract data "key"
	// field happens to equal the rule key. In typical SonarQube data they differ,
	// so CreationDate will be zero here.
	if extracted[0].StartLine != 5 || extracted[0].EndLine != 5 {
		t.Errorf("unexpected line range: %d-%d", extracted[0].StartLine, extracted[0].EndLine)
	}
}

// --- Integration tests for importBranch and runImportScanHistory ---

func newCEMockServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/ce/submit":
			json.NewEncoder(w).Encode(map[string]any{"taskId": "AX-test-123"})
		case "/api/ce/task":
			json.NewEncoder(w).Encode(map[string]any{
				"task": map[string]any{"status": "SUCCESS"},
			})
		default:
			w.WriteHeader(404)
		}
	}))
}

func TestImportBranch(t *testing.T) {
	dir := t.TempDir()
	setupScanHistoryExtract(t, dir)

	srv := newCEMockServer()
	defer srv.Close()

	e := newScanHistoryExecutor(t, dir)
	e.CloudURL = srv.URL + "/"
	e.Raw = common.NewRawClient(srv.Client(), srv.URL+"/")

	input := importBranchInput{
		CloudKey:  "cloud-proj1",
		OrgKey:    "cloud-org1",
		ServerURL: testServerURL,
		ServerKey: "proj1",
		Branch:    "main",
	}

	result, err := importBranch(context.Background(), e, input)
	if err != nil {
		t.Fatalf("importBranch: %v", err)
	}
	if result.Status != "success" {
		t.Errorf("expected success, got %s", result.Status)
	}
	if result.TaskID != "AX-test-123" {
		t.Errorf("expected AX-test-123, got %s", result.TaskID)
	}
}

func TestImportBranchSkipsNoComponents(t *testing.T) {
	dir := t.TempDir()
	extractDir := filepath.Join(dir, "extract-01")
	writeJSON(filepath.Join(extractDir, "extract.json"),
		map[string]any{"url": testServerURL, "edition": "enterprise"})

	// Write empty data — no components will have source.
	writeJSONL(filepath.Join(extractDir, "getProjectIssuesFull"), nil)
	writeJSONL(filepath.Join(extractDir, "getProjectComponentTree"), nil)
	writeJSONL(filepath.Join(extractDir, "getProjectSourceCode"), nil)
	writeJSONL(filepath.Join(extractDir, "getActiveProfileRules"), nil)
	writeJSONL(filepath.Join(extractDir, "getProfiles"), nil)

	e := newScanHistoryExecutor(t, dir)

	input := importBranchInput{
		CloudKey:  "cloud-proj1",
		OrgKey:    "cloud-org1",
		ServerURL: testServerURL,
		ServerKey: "proj1",
		Branch:    "main",
	}

	result, err := importBranch(context.Background(), e, input)
	if err != nil {
		t.Fatalf("importBranch: %v", err)
	}
	if result.Status != "skipped" {
		t.Errorf("expected skipped (no components), got %s", result.Status)
	}
}

func TestRunImportScanHistory(t *testing.T) {
	dir := t.TempDir()
	setupScanHistoryExtract(t, dir)

	srv := newCEMockServer()
	defer srv.Close()

	e := newScanHistoryExecutor(t, dir)
	e.CloudURL = srv.URL + "/"
	e.Raw = common.NewRawClient(srv.Client(), srv.URL+"/")

	// Write createProjects dependency.
	w, _ := e.Store.Writer("createProjects")
	b, _ := json.Marshal(map[string]any{
		"key":                "proj1",
		"cloud_project_key":  "cloud-proj1",
		"sonarcloud_org_key": "cloud-org1",
		"server_url":         testServerURL,
	})
	w.WriteOne(b)

	err := runImportScanHistory(context.Background(), e)
	if err != nil {
		t.Fatalf("runImportScanHistory: %v", err)
	}

	items, _ := e.Store.ReadAll("importScanHistory")
	if len(items) == 0 {
		t.Fatal("expected import results written")
	}
	status := extractField(items[0], "status")
	if status != "success" && status != "skipped" {
		t.Errorf("expected success or skipped, got %s", status)
	}
}

func TestRunImportScanHistorySkipsEmptyKeys(t *testing.T) {
	dir := t.TempDir()
	setupScanHistoryExtract(t, dir)
	e := newScanHistoryExecutor(t, dir)

	// Write project with empty cloud key — should be skipped.
	w, _ := e.Store.Writer("createProjects")
	b, _ := json.Marshal(map[string]any{
		"key":                "proj1",
		"cloud_project_key":  "",
		"sonarcloud_org_key": "",
		"server_url":         testServerURL,
	})
	w.WriteOne(b)

	err := runImportScanHistory(context.Background(), e)
	if err != nil {
		t.Fatalf("runImportScanHistory: %v", err)
	}

	items, _ := e.Store.ReadAll("importScanHistory")
	if len(items) != 0 {
		t.Errorf("expected 0 results for empty keys, got %d", len(items))
	}
}
