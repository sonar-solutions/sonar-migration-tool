package migrate

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/sonar-solutions/sonar-migration-tool/internal/common"
	"github.com/sonar-solutions/sonar-migration-tool/internal/structure"
)

// validateOrgMapping checks organizations.csv has at least one row with a
// non-empty sonarcloud_org_key. A migrate run with all rows unmapped used
// to exit 0 with nothing migrated, which silently looked like a successful
// no-op (issue #279).
//
// Returns an *common.ExitCodeError with code 2 when no mapping is defined,
// matching the contract from #279. SKIPPED rows count as "mapped" — the
// user has deliberately chosen to exclude them, distinct from forgetting
// to fill in the file at all.
func validateOrgMapping(exportDir string) error {
	rows, err := structure.LoadCSV(exportDir, "organizations.csv")
	if err != nil {
		return fmt.Errorf("loading organizations.csv: %w", err)
	}
	csvPath := filepath.Join(exportDir, "organizations.csv")
	if len(rows) == 0 {
		return common.NewExitError(2, fmt.Errorf(
			`No organization mapping has been defined, please review the %q file`, csvPath))
	}
	for _, row := range rows {
		val, _ := row["sonarcloud_org_key"].(string)
		if strings.TrimSpace(val) != "" {
			return nil
		}
	}
	return common.NewExitError(2, fmt.Errorf(
		`No organization mapping has been defined, please review the %q file`, csvPath))
}
