package migrate

import (
	"context"
	"slices"
	"strings"

	"github.com/sonar-solutions/sonar-migration-tool/internal/common"
)

// TaskDef defines a single migrate task with a typed Run function.
type TaskDef struct {
	Name         string
	Editions     []common.Edition
	Dependencies []string
	Run          func(ctx context.Context, e *Executor) error
}

// TaskName implements common.TaskMeta.
func (t *TaskDef) TaskName() string { return t.Name }

// TaskEditions implements common.TaskMeta.
func (t *TaskDef) TaskEditions() []common.Edition { return t.Editions }

// TaskDeps implements common.TaskMeta.
func (t *TaskDef) TaskDeps() []string { return t.Dependencies }

// BuildMigrateRegistry returns a name-keyed lookup map.
func BuildMigrateRegistry(defs []TaskDef) map[string]*TaskDef {
	reg := make(map[string]*TaskDef, len(defs))
	for i := range defs {
		reg[defs[i].Name] = &defs[i]
	}
	return reg
}

// FilterByEdition returns tasks supporting the given edition.
func FilterByEdition(reg map[string]*TaskDef, edition common.Edition) map[string]*TaskDef {
	return common.FilterByEditionGeneric(reg, edition)
}

// ResolveDependencies recursively resolves transitive dependencies.
func ResolveDependencies(targets []string, reg map[string]*TaskDef) map[string]bool {
	return common.ResolveDependenciesGeneric(targets, reg)
}

// PlanPhases computes topologically sorted execution phases.
func PlanPhases(tasks map[string]bool, reg map[string]*TaskDef) ([][]string, error) {
	return common.PlanPhasesGeneric(tasks, reg)
}

// RegisterAll returns every migrate task definition.
func RegisterAll() []TaskDef {
	var all []TaskDef
	all = append(all, setupTasks()...)
	all = append(all, readTasks()...)
	all = append(all, createTasks()...)
	all = append(all, configureTasks()...)
	all = append(all, associateTasks()...)
	all = append(all, permissionTasks()...)
	all = append(all, almTasks()...)
	all = append(all, portfolioTasks()...)
	all = append(all, ruleTasks()...)
	all = append(all, deleteTasks()...)
	all = append(all, scanHistoryTasks()...)
	return all
}

// migrateScanHistoryTasks lists task names that require the --include-scan-history flag.
var migrateScanHistoryTasks = map[string]bool{
	"importScanHistory": true,
}

// MigrateTargetTasks determines which tasks to run.
// Default: all tasks NOT starting with "get", "delete", or "reset".
func MigrateTargetTasks(reg map[string]*TaskDef, targetTask string, skipProfiles, includeScanHistory bool) []string {
	if targetTask != "" {
		return []string{targetTask}
	}
	excludePrefixes := []string{"get", "delete", "reset"}
	var tasks []string
	for name := range reg {
		skip := false
		for _, prefix := range excludePrefixes {
			if strings.HasPrefix(name, prefix) {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		if skipProfiles && (strings.Contains(name, "Profile") || strings.Contains(name, "profile")) {
			continue
		}
		if migrateScanHistoryTasks[name] && !includeScanHistory {
			continue
		}
		tasks = append(tasks, name)
	}
	slices.Sort(tasks)
	return tasks
}
