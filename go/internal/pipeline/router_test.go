package pipeline

import "testing"

func TestParseVersion(t *testing.T) {
	tests := []struct {
		input     string
		wantMajor int
		wantMinor int
		wantErr   bool
	}{
		{"9.9.0.65466", 9, 9, false},
		{"10.0.0", 10, 0, false},
		{"10.3.5", 10, 3, false},
		{"10.4.1.87632", 10, 4, false},
		{"10.8.0", 10, 8, false},
		{"2025.1.0", 2025, 1, false},
		{"2025.3.0", 2025, 3, false},
		{"10.5-SNAPSHOT", 10, 5, false},
		{"9.9", 9, 9, false},
		{"invalid", 0, 0, true},
		{"10", 0, 0, true},
		{"abc.def", 0, 0, true},
		{"10.abc", 0, 0, true},
	}

	for _, tt := range tests {
		major, minor, err := parseVersion(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("parseVersion(%q) error = %v, wantErr = %v", tt.input, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && (major != tt.wantMajor || minor != tt.wantMinor) {
			t.Errorf("parseVersion(%q) = (%d, %d), want (%d, %d)", tt.input, major, minor, tt.wantMajor, tt.wantMinor)
		}
	}
}

func TestSelectPipeline(t *testing.T) {
	tests := []struct {
		versionStr   string
		wantPipeline string
		wantErr      bool
	}{
		{"9.9.0.65466", "sq-9.9", false},
		{"10.0.0", "sq-10.0", false},
		{"10.3.5", "sq-10.0", false},
		{"10.4.1.87632", "sq-10.4", false},
		{"10.8.0", "sq-10.4", false},
		{"2025.1.0", "sq-2025", false},
		{"2025.3.0", "sq-2025", false},
		// Forward compatibility: unexpected major falls back to 10.4
		{"11.0.0", "sq-10.4", false},
		// Unsupported versions
		{"9.8.0", "", true},
		{"8.9.0", "", true},
		{"9.7.0", "", true},
	}

	for _, tt := range tests {
		major, minor, err := parseVersion(tt.versionStr)
		if err != nil {
			if !tt.wantErr {
				t.Errorf("parseVersion(%q) unexpected error: %v", tt.versionStr, err)
			}
			continue
		}

		p, err := selectPipeline(major, minor, nil)
		if (err != nil) != tt.wantErr {
			t.Errorf("selectPipeline for %q: error = %v, wantErr = %v", tt.versionStr, err, tt.wantErr)
			continue
		}
		if tt.wantErr {
			continue
		}
		if p.Version() != tt.wantPipeline {
			t.Errorf("selectPipeline for %q: got pipeline %q, want %q", tt.versionStr, p.Version(), tt.wantPipeline)
		}
	}
}

// TestPipelineInterfaceCompliance verifies all four pipelines implement
// the Pipeline interface at compile time (the var _ = checks in each file
// cover this, but a single test makes the intent explicit).
func TestPipelineInterfaceCompliance(t *testing.T) {
	var _ Pipeline = newSQ99(nil)
	var _ Pipeline = newSQ100(nil)
	var _ Pipeline = newSQ104(nil)
	var _ Pipeline = newSQ2025(nil)
}

func TestIssueSearchParams(t *testing.T) {
	tests := []struct {
		pipeline    Pipeline
		wantParam   string
		wantStatus0 string
	}{
		{newSQ99(nil), "statuses", "OPEN"},
		{newSQ100(nil), "statuses", "OPEN"},
		{newSQ104(nil), "issueStatuses", "OPEN"},
		{newSQ2025(nil), "issueStatuses", "OPEN"},
	}
	for _, tt := range tests {
		if got := tt.pipeline.IssueSearchParam(); got != tt.wantParam {
			t.Errorf("%s.IssueSearchParam() = %q, want %q", tt.pipeline.Version(), got, tt.wantParam)
		}
		vals := tt.pipeline.IssueStatusValues()
		if len(vals) == 0 {
			t.Errorf("%s.IssueStatusValues() is empty", tt.pipeline.Version())
			continue
		}
		if vals[0] != tt.wantStatus0 {
			t.Errorf("%s.IssueStatusValues()[0] = %q, want %q", tt.pipeline.Version(), vals[0], tt.wantStatus0)
		}
	}
}

func TestMetricBatching(t *testing.T) {
	type result struct {
		supported bool
		batchSize int
	}
	tests := []struct {
		pipeline Pipeline
		want     result
	}{
		{newSQ99(nil), result{true, 15}},
		{newSQ100(nil), result{true, 15}},
		{newSQ104(nil), result{true, 15}},
		{newSQ2025(nil), result{false, 0}},
	}
	for _, tt := range tests {
		supported, size := tt.pipeline.SupportsMetricBatching()
		if supported != tt.want.supported || size != tt.want.batchSize {
			t.Errorf("%s.SupportsMetricBatching() = (%v, %d), want (%v, %d)",
				tt.pipeline.Version(), supported, size, tt.want.supported, tt.want.batchSize)
		}
	}
}

func TestSQ2025StatusValues(t *testing.T) {
	p := newSQ2025(nil)
	for _, v := range p.IssueStatusValues() {
		if v == "IN_SANDBOX" {
			t.Error("SQ2025 IssueStatusValues must NOT include IN_SANDBOX (it is detected in results, not queried)")
		}
	}
}
