// Package cloud — enterprises.go covers the /enterprises REST API available on
// SonarQube Cloud Enterprise plans. This API uses a different base URL
// (api.sonarcloud.io rather than sonarcloud.io) and JSON request/response bodies.
package cloud

import (
	"context"
	"fmt"

	"github.com/sonar-solutions/sq-api-go/types"
)

// EnterprisesClient provides methods for the Cloud Enterprises API.
// The base sqapi.Client must be pointed at the enterprise API base URL
// (e.g. "https://api.sonarcloud.io").
type EnterprisesClient struct{ baseClient }

// List returns all enterprises accessible to the authenticated token via
// GET /enterprises/enterprises.
func (e *EnterprisesClient) List(ctx context.Context) ([]types.Enterprise, error) {
	var result types.EnterprisesListResponse
	if err := e.getJSON(ctx, "enterprises/enterprises", &result); err != nil {
		return nil, err
	}
	return result.Enterprises, nil
}

// CreatePortfolioParams holds the parameters for creating a Cloud portfolio.
type CreatePortfolioParams struct {
	EnterpriseID string
	Name         string
	Description  string
	// Selection controls how the portfolio selects projects. Defaults to "projects".
	Selection string
}

// CreatePortfolio creates a portfolio within an enterprise via
// POST /enterprises/portfolios.
func (e *EnterprisesClient) CreatePortfolio(ctx context.Context, params CreatePortfolioParams) (*types.Portfolio, error) {
	selection := params.Selection
	if selection == "" {
		selection = "projects"
	}
	body := map[string]any{
		"enterpriseId": params.EnterpriseID,
		"name":         params.Name,
		"description":  params.Description,
		"selection":    selection,
	}

	var result types.Portfolio
	if err := e.postJSON(ctx, "enterprises/portfolios", body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdatePortfolioParams holds the parameters for updating a portfolio's project list.
type UpdatePortfolioParams struct {
	// PortfolioID is the portfolio's numeric ID.
	PortfolioID string
	// Projects is the list of project keys to include in the portfolio.
	Projects []string
}

// UpdatePortfolio updates the project list for a portfolio via
// PATCH /enterprises/portfolios/{id}.
func (e *EnterprisesClient) UpdatePortfolio(ctx context.Context, params UpdatePortfolioParams) error {
	body := map[string]any{
		"projects": params.Projects,
	}
	path := fmt.Sprintf("enterprises/portfolios/%s", params.PortfolioID)
	return e.patchJSON(ctx, path, body, nil)
}

// DeletePortfolio deletes a portfolio via DELETE /enterprises/portfolios/{id}.
func (e *EnterprisesClient) DeletePortfolio(ctx context.Context, portfolioID string) error {
	path := fmt.Sprintf("enterprises/portfolios/%s", portfolioID)
	return e.deleteReq(ctx, path)
}
