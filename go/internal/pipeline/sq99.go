package pipeline

import sqapi "github.com/sonar-solutions/sq-api-go"

// SQ99Pipeline handles SonarQube Server 9.9 LTS.
//   - Uses legacy "statuses" parameter (OPEN, CONFIRMED, REOPENED, RESOLVED, CLOSED)
//   - Batches metricKeys at 15 per request
//   - Clean Code enrichment from SonarQube Cloud (SPEC-012, not yet implemented)
//
// All extraction methods, query parameters, and batching configuration are
// promoted from standardPipeline.
type SQ99Pipeline struct {
	standardPipeline
}

func newSQ99(client *sqapi.Client) *SQ99Pipeline {
	return &SQ99Pipeline{standardPipeline: standardPipeline{
		client:            client,
		issueSearchParam:  "statuses",
		issueStatusValues: []string{"OPEN", "CONFIRMED", "REOPENED", "RESOLVED", "CLOSED"},
		metricBatchSize:   15,
	}}
}

// Compile-time interface check.
var _ Pipeline = (*SQ99Pipeline)(nil)

func (p *SQ99Pipeline) Version() string { return "sq-9.9" }
