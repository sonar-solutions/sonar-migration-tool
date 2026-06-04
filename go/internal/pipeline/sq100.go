// Copyright (C) SonarSource Sàrl
// For more information, see https://sonarsource.com/legal/
// mailto:info AT sonarsource DOT com

package pipeline

import sqapi "github.com/sonar-solutions/sq-api-go"

// SQ100Pipeline handles SonarQube Server 10.0 through 10.3.
//   - Uses legacy "statuses" parameter (same values as SQ 9.9; in this range
//     FALSE_POSITIVE and ACCEPTED are resolutions, not statuses)
//   - Batches metricKeys at 15 per request
//   - Native Clean Code attributes in API responses (no Cloud enrichment needed)
//
// All extraction methods, query parameters, and batching configuration are
// promoted from standardPipeline.
type SQ100Pipeline struct {
	standardPipeline
}

func newSQ100(client *sqapi.Client) *SQ100Pipeline {
	return &SQ100Pipeline{standardPipeline: standardPipeline{
		client:            client,
		issueSearchParam:  "statuses",
		issueStatusValues: []string{"OPEN", "CONFIRMED", "REOPENED", "RESOLVED", "CLOSED"},
		metricBatchSize:   15,
	}}
}

var _ Pipeline = (*SQ100Pipeline)(nil)

func (p *SQ100Pipeline) Version() string { return "sq-10.0" }
