// Copyright (C) SonarSource Sàrl
// For more information, see https://sonarsource.com/legal/
// mailto:info AT sonarsource DOT com

package pipeline

import sqapi "github.com/sonar-solutions/sq-api-go"

// SQ104Pipeline handles SonarQube Server 10.4 through 10.8.
//   - Uses modern "issueStatuses" parameter (OPEN, CONFIRMED, FALSE_POSITIVE, ACCEPTED, FIXED)
//   - Batches metricKeys at 15 per request
//   - Native Clean Code attributes in API responses
//
// Note: "issueStatuses" was introduced in SQ 10.2 but became the recommended
// parameter in 10.4. For simplicity, the SQ 10.0-10.3 pipeline uses the legacy
// "statuses" parameter throughout that range.
//
// All extraction methods, query parameters, and batching configuration are
// promoted from standardPipeline.
type SQ104Pipeline struct {
	standardPipeline
}

func newSQ104(client *sqapi.Client) *SQ104Pipeline {
	return &SQ104Pipeline{standardPipeline: standardPipeline{
		client:            client,
		issueSearchParam:  "issueStatuses",
		issueStatusValues: []string{"OPEN", "CONFIRMED", "FALSE_POSITIVE", "ACCEPTED", "FIXED"},
		metricBatchSize:   15,
	}}
}

var _ Pipeline = (*SQ104Pipeline)(nil)

func (p *SQ104Pipeline) Version() string { return "sq-10.4" }
