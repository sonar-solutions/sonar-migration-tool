package extract

import (
	"context"
	"testing"
)

func TestBuildRegistry(t *testing.T) {
	defs := []TaskDef{
		{Name: "a"}, {Name: "b"}, {Name: "c"},
	}
	reg := BuildRegistry(defs)
	if len(reg) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(reg))
	}
	if reg["a"].Name != "a" {
		t.Fatalf("expected 'a', got %q", reg["a"].Name)
	}
}

func TestFilterByEdition(t *testing.T) {
	noop := func(ctx context.Context, e *Executor) error { return nil }
	defs := []TaskDef{
		{Name: "all", Editions: AllEditions, Run: noop},
		{Name: "entOnly", Editions: []Edition{EditionEnterprise, EditionDatacenter}, Run: noop},
		{Name: "noEditions", Run: noop}, // no editions = all
	}
	reg := BuildRegistry(defs)
	filtered := FilterByEdition(reg, EditionCommunity)
	if _, ok := filtered["all"]; !ok {
		t.Error("expected 'all' in community filter")
	}
	if _, ok := filtered["entOnly"]; ok {
		t.Error("expected 'entOnly' excluded from community filter")
	}
	if _, ok := filtered["noEditions"]; !ok {
		t.Error("expected 'noEditions' in community filter (empty editions = all)")
	}
}

func TestPlanPhasesSimple(t *testing.T) {
	noop := func(ctx context.Context, e *Executor) error { return nil }
	defs := []TaskDef{
		{Name: "a", Run: noop},
		{Name: "b", Dependencies: []string{"a"}, Run: noop},
		{Name: "c", Dependencies: []string{"a"}, Run: noop},
		{Name: "d", Dependencies: []string{"b", "c"}, Run: noop},
	}
	reg := BuildRegistry(defs)
	tasks := map[string]bool{"a": true, "b": true, "c": true, "d": true}

	plan, err := PlanPhases(tasks, reg)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan) != 3 {
		t.Fatalf("expected 3 phases, got %d: %v", len(plan), plan)
	}
	// Phase 0: a
	if len(plan[0]) != 1 || plan[0][0] != "a" {
		t.Errorf("phase 0: expected [a], got %v", plan[0])
	}
	// Phase 1: b, c (sorted)
	if len(plan[1]) != 2 {
		t.Errorf("phase 1: expected 2 tasks, got %v", plan[1])
	}
	// Phase 2: d
	if len(plan[2]) != 1 || plan[2][0] != "d" {
		t.Errorf("phase 2: expected [d], got %v", plan[2])
	}
}

func TestPlanPhasesCycle(t *testing.T) {
	noop := func(ctx context.Context, e *Executor) error { return nil }
	defs := []TaskDef{
		{Name: "a", Dependencies: []string{"b"}, Run: noop},
		{Name: "b", Dependencies: []string{"a"}, Run: noop},
	}
	reg := BuildRegistry(defs)
	tasks := map[string]bool{"a": true, "b": true}

	_, err := PlanPhases(tasks, reg)
	if err == nil {
		t.Error("expected cycle detection error")
	}
}

func TestResolveDependencies(t *testing.T) {
	noop := func(ctx context.Context, e *Executor) error { return nil }
	defs := []TaskDef{
		{Name: "a", Run: noop},
		{Name: "b", Dependencies: []string{"a"}, Run: noop},
		{Name: "c", Dependencies: []string{"b"}, Run: noop},
	}
	reg := BuildRegistry(defs)
	result := ResolveDependencies([]string{"c"}, reg)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(result))
	}
	for _, name := range []string{"a", "b", "c"} {
		if !result[name] {
			t.Errorf("expected %q in resolved deps", name)
		}
	}
}

func TestResolveDependenciesMissingDep(t *testing.T) {
	noop := func(ctx context.Context, e *Executor) error { return nil }
	defs := []TaskDef{
		{Name: "a", Dependencies: []string{"missing"}, Run: noop},
	}
	reg := BuildRegistry(defs)
	result := ResolveDependencies([]string{"a"}, reg)
	if result != nil {
		t.Error("expected nil for unresolvable dependency")
	}
}

func TestTargetTasks(t *testing.T) {
	noop := func(ctx context.Context, e *Executor) error { return nil }
	reg := BuildRegistry([]TaskDef{
		{Name: "getProjects", Run: noop},
		{Name: "getUsers", Run: noop},
		{Name: "migrate", Run: noop},
	})
	targets := TargetTasks(reg, "", "all")
	if len(targets) != 2 {
		t.Fatalf("expected 2 'get*' tasks, got %d: %v", len(targets), targets)
	}

	targets = TargetTasks(reg, "getProjects", "all")
	if len(targets) != 1 || targets[0] != "getProjects" {
		t.Errorf("expected single target, got %v", targets)
	}
}

func TestRegisterAllCountsAndDependencies(t *testing.T) {
	all := RegisterAll()
	if len(all) < 60 {
		t.Errorf("expected at least 60 tasks, got %d", len(all))
	}

	// Verify all dependencies reference tasks that exist.
	reg := BuildRegistry(all)
	for _, def := range all {
		for _, dep := range def.Dependencies {
			if _, ok := reg[dep]; !ok {
				t.Errorf("task %q depends on %q which does not exist", def.Name, dep)
			}
		}
	}
}

func TestParseEdition(t *testing.T) {
	tests := []struct {
		input    string
		expected Edition
	}{
		{`{"edition":"enterprise"}`, EditionEnterprise},
		{`{"edition":"community"}`, EditionCommunity},
		{`{"edition":"developer"}`, EditionDeveloper},
		{`{"edition":"datacenter"}`, EditionDatacenter},
		{`{"edition":"unknown"}`, EditionCommunity},
		{`{}`, EditionCommunity},
		{`invalid`, EditionCommunity},
	}
	for _, tt := range tests {
		got := ParseEdition([]byte(tt.input))
		if got != tt.expected {
			t.Errorf("ParseEdition(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}
