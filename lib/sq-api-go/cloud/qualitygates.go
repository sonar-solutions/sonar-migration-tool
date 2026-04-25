package cloud

import (
	"context"
	"net/url"
	"strconv"

	"github.com/sonar-solutions/sq-api-go/types"
)

// QualityGatesClient provides write-path methods for SonarQube Cloud quality gates.
type QualityGatesClient struct{ baseClient }

// Create creates a new quality gate via /api/qualitygates/create.
func (q *QualityGatesClient) Create(ctx context.Context, name, organization string) (*types.QualityGate, error) {
	form := url.Values{}
	form.Set("name", name)
	form.Set("organization", organization)

	var result types.QualityGate
	if err := q.postForm(ctx, "api/qualitygates/create", form, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateConditionParams holds the parameters for creating a quality gate condition.
type CreateConditionParams struct {
	GateID       int
	Organization string
	Metric       string
	Op           string // "LT" or "GT"
	Error        string // threshold value
}

// CreateCondition adds a condition to a quality gate via /api/qualitygates/create_condition.
func (q *QualityGatesClient) CreateCondition(ctx context.Context, params CreateConditionParams) (*types.QualityGateCondition, error) {
	form := url.Values{}
	form.Set("gateId", strconv.Itoa(params.GateID))
	form.Set("organization", params.Organization)
	form.Set("metric", params.Metric)
	form.Set("op", params.Op)
	form.Set("error", params.Error)

	var result types.QualityGateCondition
	if err := q.postForm(ctx, "api/qualitygates/create_condition", form, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Destroy deletes a quality gate via /api/qualitygates/destroy.
func (q *QualityGatesClient) Destroy(ctx context.Context, gateID int, organization string) error {
	form := url.Values{}
	form.Set("id", strconv.Itoa(gateID))
	form.Set("organization", organization)
	return q.postForm(ctx, "api/qualitygates/destroy", form, nil)
}

// Select associates a quality gate with a project via /api/qualitygates/select.
func (q *QualityGatesClient) Select(ctx context.Context, gateID int, projectKey, organization string) error {
	form := url.Values{}
	form.Set("gateId", strconv.Itoa(gateID))
	form.Set("projectKey", projectKey)
	form.Set("organization", organization)
	return q.postForm(ctx, "api/qualitygates/select", form, nil)
}

// SetDefault sets a quality gate as the default for the organization via
// /api/qualitygates/set_as_default.
func (q *QualityGatesClient) SetDefault(ctx context.Context, gateID int, organization string) error {
	form := url.Values{}
	form.Set("id", strconv.Itoa(gateID))
	form.Set("organization", organization)
	return q.postForm(ctx, "api/qualitygates/set_as_default", form, nil)
}
