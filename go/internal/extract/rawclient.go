package extract

import "github.com/sonar-solutions/sonar-migration-tool/internal/common"

// RawClient is an alias for common.RawClient.
type RawClient = common.RawClient

// NewRawClient wraps an sqapi.Client's HTTP infrastructure.
var NewRawClient = common.NewRawClient

// PaginatedOpts is an alias for common.PaginatedOpts.
type PaginatedOpts = common.PaginatedOpts
