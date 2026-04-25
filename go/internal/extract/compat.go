package extract

// Forwarded symbols from common for backward compatibility within the extract package.
// These allow existing extract tests and task files to compile without changes.

import "github.com/sonar-solutions/sonar-migration-tool/internal/common"

var extractArray = common.ExtractArray
var extractTotal = common.ExtractTotal
var totalPages = common.TotalPages
var truncate = common.Truncate
