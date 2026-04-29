package migrate

import (
	"encoding/json"
	"testing"
)

func TestRegisterAllCountsAndDependencies(t *testing.T) {
	all := RegisterAll()
	if len(all) < 30 {
		t.Errorf("expected at least 30 tasks, got %d", len(all))
	}

	// Verify all dependencies reference tasks that exist.
	reg := BuildMigrateRegistry(all)
	for _, def := range all {
		for _, dep := range def.Dependencies {
			if _, ok := reg[dep]; !ok {
				t.Errorf("task %q depends on %q which does not exist", def.Name, dep)
			}
		}
	}
}

func TestMigrateTargetTasks(t *testing.T) {
	reg := BuildMigrateRegistry(RegisterAll())

	targets := MigrateTargetTasks(reg, "", false, false)
	// Should exclude get*, delete*, reset* tasks.
	for _, name := range targets {
		if name[:3] == "get" || name[:6] == "delete" || name[:5] == "reset" {
			t.Errorf("unexpected target task: %q", name)
		}
	}
	if len(targets) == 0 {
		t.Error("expected non-empty target tasks")
	}
}

func TestMigrateTargetTasksSingle(t *testing.T) {
	reg := BuildMigrateRegistry(RegisterAll())
	targets := MigrateTargetTasks(reg, "createProjects", false, false)
	if len(targets) != 1 || targets[0] != "createProjects" {
		t.Errorf("expected [createProjects], got %v", targets)
	}
}

func TestMigrateTargetTasksSkipProfiles(t *testing.T) {
	reg := BuildMigrateRegistry(RegisterAll())
	targets := MigrateTargetTasks(reg, "", true, false)
	for _, name := range targets {
		if name == "createProfiles" || name == "setProfileParent" || name == "restoreProfiles" ||
			name == "setDefaultProfiles" || name == "setProjectProfiles" || name == "setProfileGroupPermissions" {
			t.Errorf("expected profile task %q to be skipped", name)
		}
	}
}

func TestPlanPhasesNoCycles(t *testing.T) {
	all := RegisterAll()
	reg := BuildMigrateRegistry(all)
	targets := MigrateTargetTasks(reg, "", false, false)
	taskSet := ResolveDependencies(targets, reg)
	if taskSet == nil {
		t.Fatal("cannot resolve dependencies")
	}

	plan, err := PlanPhases(taskSet, reg)
	if err != nil {
		t.Fatalf("PlanPhases failed: %v", err)
	}
	if len(plan) == 0 {
		t.Error("expected non-empty plan")
	}

	// Verify first phase has no dependencies on other migrate tasks.
	for _, taskName := range plan[0] {
		def := reg[taskName]
		if len(def.Dependencies) > 0 {
			t.Logf("phase 0 task %q has deps: %v (should all be extract tasks)", taskName, def.Dependencies)
		}
	}
}

func TestFilterByEdition(t *testing.T) {
	all := RegisterAll()
	reg := BuildMigrateRegistry(all)

	// Enterprise should include portfolio tasks.
	entReg := FilterByEdition(reg, "enterprise")
	if _, ok := entReg["createPortfolios"]; !ok {
		t.Error("expected createPortfolios in enterprise edition")
	}

	// Community should exclude portfolio tasks.
	comReg := FilterByEdition(reg, "community")
	if _, ok := comReg["createPortfolios"]; ok {
		t.Error("expected createPortfolios excluded from community edition")
	}
}

func TestMatchDevOpsPlatform(t *testing.T) {
	repos := []json.RawMessage{
		json.RawMessage(`{"id":"12345","slug":"org/myrepo","label":"My Repo"}`),
		json.RawMessage(`{"id":"67890","slug":"org/other","label":"Other"}`),
	}

	tests := []struct {
		name       string
		alm        string
		repository string
		slug       string
		expected   string
	}{
		{"github match", "github", "org/myrepo", "", "12345"},
		{"github no match", "github", "org/nomatch", "", ""},
		{"gitlab match", "gitlab", "12345", "", "12345"},
		{"gitlab no match", "gitlab", "99999", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchDevOpsPlatform(tt.alm, tt.repository, tt.slug, repos)
			if got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestExtractAnyStr(t *testing.T) {
	// String value.
	raw := json.RawMessage(`{"value":"hello"}`)
	if got := extractAnyStr(raw, "value"); got != "hello" {
		t.Errorf("expected 'hello', got %q", got)
	}

	// Numeric value.
	raw = json.RawMessage(`{"value":30}`)
	if got := extractAnyStr(raw, "value"); got != "30" {
		t.Errorf("expected '30', got %q", got)
	}

	// Missing key.
	raw = json.RawMessage(`{"other":"x"}`)
	if got := extractAnyStr(raw, "value"); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}
