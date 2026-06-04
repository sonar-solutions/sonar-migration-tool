// Copyright (C) SonarSource Sàrl
// For more information, see https://sonarsource.com/legal/
// mailto:info AT sonarsource DOT com

// Package cloud — fix_suggestions.go covers the SonarQube Cloud AI
// Code Fix organization-config endpoint, used by the migration tool to
// translate SQS-side AI Code Fix configuration into the equivalent SQC
// org settings (issue #251).
//
// The endpoint lives on the api.sonarcloud.io base
// (PATCH /fix-suggestions/organization-configs/{organizationId}), so
// the owning client must be constructed against that base URL.
package cloud

import (
	"context"

	"github.com/sonar-solutions/sq-api-go/types"
)

// FixSuggestionsClient provides methods for the SonarQube Cloud
// "Fix Suggestions" (AI Code Fix) API.
type FixSuggestionsClient struct{ baseClient }

// PatchOrganizationConfig PATCHes the AI Code Fix configuration for a
// single SonarQube Cloud organization. orgID is the organization's
// UUID (resolve via OrganizationsClient.LookupID when only the human
// key is available). The request body uses merge-patch semantics —
// fields left nil on the payload are unchanged on the server side.
func (c *FixSuggestionsClient) PatchOrganizationConfig(
	ctx context.Context, orgID string, payload types.FixSuggestionsOrgConfig,
) error {
	return c.patchJSON(ctx, "fix-suggestions/organization-configs/"+orgID, payload, nil)
}
