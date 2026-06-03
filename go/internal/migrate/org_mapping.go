package migrate

import (
	"encoding/csv"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/sonar-solutions/sonar-migration-tool/internal/common"
	"github.com/sonar-solutions/sonar-migration-tool/internal/structure"
)

// applyOrgMapping checks organizations.csv has at least one row with a
// non-empty sonarcloud_org_key. A migrate run with all rows unmapped
// used to exit 0 with nothing migrated, which silently looked like a
// successful no-op (issue #279).
//
// When defaultOrg is non-empty, behaviour follows issue #281:
//
//   - Mapping defined + defaultOrg set: defaultOrg is ignored, a WARN is
//     logged, the CSV is left untouched.
//   - No mapping + defaultOrg set: organizations.csv is rewritten with
//     defaultOrg in every sonarcloud_org_key cell, no error.
//   - No mapping + defaultOrg empty: returns the #279 exit-code-2 error.
//
// SKIPPED rows count as "mapped" — the user has deliberately chosen to
// exclude them, distinct from forgetting to fill in the file at all.
func applyOrgMapping(exportDir, defaultOrg string, logger *slog.Logger) error {
	rows, err := structure.LoadCSV(exportDir, "organizations.csv")
	if err != nil {
		return fmt.Errorf("loading organizations.csv: %w", err)
	}
	csvPath := filepath.Join(exportDir, "organizations.csv")

	if len(rows) == 0 {
		// No file at all → cannot synthesize even with defaultOrg
		// because the columns/rows aren't there yet.
		return missingMappingError(csvPath)
	}

	hasMapping := false
	for _, row := range rows {
		val, _ := row["sonarcloud_org_key"].(string)
		if strings.TrimSpace(val) != "" {
			hasMapping = true
			break
		}
	}

	if hasMapping {
		if defaultOrg != "" {
			logger.Warn("Since organizations.csv mapping is defined, the provided default organization parameter is ignored",
				"default_organization", defaultOrg, "file", csvPath)
		}
		return nil
	}

	// No mapping defined.
	if defaultOrg == "" {
		return missingMappingError(csvPath)
	}

	// Apply defaultOrg to every row and write back.
	if err := writeOrgCSVWithDefault(csvPath, rows, defaultOrg); err != nil {
		return fmt.Errorf("applying default_organization to organizations.csv: %w", err)
	}
	logger.Info("organizations.csv was empty — every project will migrate to the provided default organization",
		"default_organization", defaultOrg, "rows", len(rows))
	return nil
}

func missingMappingError(csvPath string) error {
	return common.NewExitError(2, fmt.Errorf(
		`No organization mapping has been defined, please review the %q file`, csvPath))
}

// writeOrgCSVWithDefault rewrites the file so every row's
// sonarcloud_org_key cell carries defaultOrg, preserving all other
// columns and their order. The file is read once to recover the
// canonical header order — map iteration would otherwise scramble it.
func writeOrgCSVWithDefault(path string, rows []map[string]any, defaultOrg string) error {
	headers, err := readOrgCSVHeaders(path)
	if err != nil {
		return err
	}
	tmpPath := path + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return err
	}
	w := csv.NewWriter(f)
	if err := w.Write(headers); err != nil {
		_ = f.Close()
		return err
	}
	for _, row := range rows {
		out := make([]string, len(headers))
		for i, h := range headers {
			if h == "sonarcloud_org_key" {
				out[i] = defaultOrg
				continue
			}
			out[i] = fmt.Sprintf("%v", row[h])
			if row[h] == nil {
				out[i] = ""
			}
		}
		if err := w.Write(out); err != nil {
			_ = f.Close()
			return err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func readOrgCSVHeaders(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	headers, err := r.Read()
	if err != nil {
		return nil, err
	}
	return headers, nil
}
