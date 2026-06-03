package migrate

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sonar-solutions/sonar-migration-tool/internal/common"
)

func writeOrgCSV(t *testing.T, dir, contents string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "organizations.csv"), []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}

// Issue #279: when every sonarcloud_org_key column is empty, migrate
// should fail fast with the specified message and exit code 2 instead
// of "succeeding" while doing nothing.
func TestValidateOrgMapping_AllEmptyReturnsExit2(t *testing.T) {
	dir := t.TempDir()
	writeOrgCSV(t, dir, `sonarqube_org_key,sonarcloud_org_key
org-a,
org-b,
org-c,
`)
	err := validateOrgMapping(dir)
	if err == nil {
		t.Fatal("expected an error when no mapping is defined")
	}
	var ec *common.ExitCodeError
	if !errors.As(err, &ec) {
		t.Fatalf("expected *ExitCodeError, got %T", err)
	}
	if ec.Code != 2 {
		t.Errorf("exit code: got %d, want 2", ec.Code)
	}
	expected := "No organization mapping has been defined, please review the"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("error message: got %q, want it to contain %q", err.Error(), expected)
	}
	if !strings.Contains(err.Error(), filepath.Join(dir, "organizations.csv")) {
		t.Errorf("error should include the CSV path, got %q", err.Error())
	}
}

// At least one mapped row → no error.
func TestValidateOrgMapping_OneMappedPasses(t *testing.T) {
	dir := t.TempDir()
	writeOrgCSV(t, dir, `sonarqube_org_key,sonarcloud_org_key
org-a,
org-b,my-cloud-org
org-c,
`)
	if err := validateOrgMapping(dir); err != nil {
		t.Errorf("expected nil for partial mapping, got %v", err)
	}
}

// SKIPPED rows count as deliberately mapped — they do not trigger the
// "no mapping defined" error even if every other row is empty. The
// sentinel is the user's explicit choice; an empty cell is the
// forgotten-to-fill case.
func TestValidateOrgMapping_OnlySkippedRowsPass(t *testing.T) {
	dir := t.TempDir()
	writeOrgCSV(t, dir, `sonarqube_org_key,sonarcloud_org_key
org-a,SKIPPED
org-b,SKIPPED
`)
	if err := validateOrgMapping(dir); err != nil {
		t.Errorf("expected nil when all rows are SKIPPED, got %v", err)
	}
}

// Whitespace-only cells are still "empty" — the user did not type a
// real key, so the error must fire.
func TestValidateOrgMapping_WhitespaceCountsAsEmpty(t *testing.T) {
	dir := t.TempDir()
	writeOrgCSV(t, dir, `sonarqube_org_key,sonarcloud_org_key
org-a,
org-b,
`)
	if err := validateOrgMapping(dir); err == nil {
		t.Error("expected error when all sonarcloud_org_key values are whitespace")
	}
}

// Missing file → also surfaces the same error so the operator hears
// about a misconfigured working directory immediately.
func TestValidateOrgMapping_MissingFileReturnsExit2(t *testing.T) {
	dir := t.TempDir()
	err := validateOrgMapping(dir)
	if err == nil {
		t.Fatal("expected error when organizations.csv is missing")
	}
	var ec *common.ExitCodeError
	if !errors.As(err, &ec) || ec.Code != 2 {
		t.Errorf("expected exit code 2 error, got %v (%T)", err, err)
	}
}
