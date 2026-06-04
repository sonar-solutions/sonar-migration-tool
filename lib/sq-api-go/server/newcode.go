// Copyright (C) SonarSource Sàrl
// For more information, see https://sonarsource.com/legal/
// mailto:info AT sonarsource DOT com

package server

import (
	"context"
	"net/url"

	"github.com/sonar-solutions/sq-api-go/types"
)

// NewCodeClient provides methods for /api/new_code_periods endpoints.
type NewCodeClient struct{ baseClient }

// List returns all new code period definitions for a project from
// /api/new_code_periods/list.
func (n *NewCodeClient) List(ctx context.Context, projectKey string) ([]types.NewCodePeriod, error) {
	params := url.Values{}
	params.Set("project", projectKey)

	var result types.NewCodePeriodsResponse
	if err := n.get(ctx, "api/new_code_periods/list", params, &result); err != nil {
		return nil, err
	}
	return result.NewCodePeriods, nil
}

// Show returns the new code period definition for a (project, branch).
// When both projectKey and branchKey are empty, SonarQube Server
// returns the platform-global default — that's the form the migration
// tool uses to discover the global NCD it should propagate to every
// SonarQube Cloud organization (issue #136).
func (n *NewCodeClient) Show(ctx context.Context, projectKey, branchKey string) (*types.NewCodePeriod, error) {
	params := url.Values{}
	if projectKey != "" {
		params.Set("project", projectKey)
	}
	if branchKey != "" {
		params.Set("branch", branchKey)
	}
	var ncd types.NewCodePeriod
	if err := n.get(ctx, "api/new_code_periods/show", params, &ncd); err != nil {
		return nil, err
	}
	return &ncd, nil
}
