package wizard

import "testing"

func TestNewCLIPrompter(t *testing.T) {
	p := NewCLIPrompter()
	if p == nil {
		t.Fatal("expected non-nil CLIPrompter")
	}
}

func TestBuildProgressBar(t *testing.T) {
	tests := []struct {
		current, total int
		want           string
	}{
		{0, 7, "[-------]"},
		{3, 7, "[###----]"},
		{7, 7, "[#######]"},
	}
	for _, tt := range tests {
		got := buildProgressBar(tt.current, tt.total)
		if got != tt.want {
			t.Errorf("buildProgressBar(%d, %d) = %q, want %q", tt.current, tt.total, got, tt.want)
		}
	}
}

// Display methods are output-only (fmt.Printf). These tests verify
// they don't panic and exercise the code paths for coverage.

func TestCLIPrompterDisplayMethods(t *testing.T) {
	p := NewCLIPrompter()

	p.DisplayWelcome()
	p.DisplayPhaseProgress(PhaseExtract)
	p.DisplayPhaseProgress(PhaseMigrate)
	p.DisplayMessage("test message")
	p.DisplayError("test error")
	p.DisplayWarning("test warning")
	p.DisplaySuccess("test success")
	p.DisplaySummary("Summary", []KV{{"Key1", "Val1"}, {"Key2", "Val2"}})
	p.DisplayWizardComplete()
}

func TestCLIPrompterDisplayResumeInfo(t *testing.T) {
	p := NewCLIPrompter()

	// Minimal state — only required field.
	p.DisplayResumeInfo(&WizardState{Phase: PhaseExtract})

	// Full state with all pointer fields set.
	p.DisplayResumeInfo(&WizardState{
		Phase:     PhaseStructure,
		SourceURL: strPtr("https://sq.example.com/"),
		TargetURL: strPtr("https://sonarcloud.io/"),
		ExtractID: strPtr("run-42"),
	})
}

func TestDisplayColorLine(t *testing.T) {
	// Just verify no panic.
	displayColorLine(colorRed, "red message")
	displayColorLine(colorGreen, "green message")
	displayColorLine(colorYellow, "yellow message")
}

func TestDisplayLocalhostNotice(t *testing.T) {
	// Verify no panic.
	displayLocalhostNotice()
}
