"""Tests for the wizard module"""
import json
import os
import tempfile

from click.testing import CliRunner

from wizard.state import WizardPhase, WizardState
from wizard.wizard import (
    wizard,
    get_next_phase,
    PHASE_SEQUENCE,
    PHASE_HANDLERS,
)
from wizard.prompts import (
    display_welcome,
    display_summary,
    PHASE_DISPLAY_NAMES,
    PHASE_ORDER,
)


class TestPhaseSequence:
    def test_phase_sequence_completeness(self):
        """Verify all phases except INIT and COMPLETE are in sequence"""
        for phase in WizardPhase:
            if phase not in (WizardPhase.INIT, WizardPhase.COMPLETE):
                assert phase in PHASE_SEQUENCE, f"Missing phase: {phase}"

    def test_phase_sequence_order(self):
        """Verify phase sequence is in correct order"""
        expected_order = [
            WizardPhase.EXTRACT,
            WizardPhase.STRUCTURE,
            WizardPhase.ORG_MAPPING,
            WizardPhase.MAPPINGS,
            WizardPhase.VALIDATE,
            WizardPhase.MIGRATE,
            WizardPhase.PIPELINES,
            WizardPhase.COMPLETE,
        ]
        assert PHASE_SEQUENCE == expected_order

    def test_all_phases_have_handlers(self):
        """Verify all phases have handlers"""
        for phase in PHASE_SEQUENCE:
            if phase != WizardPhase.COMPLETE:
                assert phase in PHASE_HANDLERS, f"Missing handler for: {phase}"


class TestGetNextPhase:
    def test_init_returns_extract(self):
        """From INIT, next phase should be EXTRACT"""
        assert get_next_phase(WizardPhase.INIT) == WizardPhase.EXTRACT

    def test_extract_returns_structure(self):
        """From EXTRACT, next phase should be STRUCTURE"""
        assert get_next_phase(WizardPhase.EXTRACT) == WizardPhase.STRUCTURE

    def test_structure_returns_org_mapping(self):
        """From STRUCTURE, next phase should be ORG_MAPPING"""
        assert get_next_phase(WizardPhase.STRUCTURE) == WizardPhase.ORG_MAPPING

    def test_org_mapping_returns_mappings(self):
        """From ORG_MAPPING, next phase should be MAPPINGS"""
        assert get_next_phase(WizardPhase.ORG_MAPPING) == WizardPhase.MAPPINGS

    def test_mappings_returns_validate(self):
        """From MAPPINGS, next phase should be VALIDATE"""
        assert get_next_phase(WizardPhase.MAPPINGS) == WizardPhase.VALIDATE

    def test_validate_returns_migrate(self):
        """From VALIDATE, next phase should be MIGRATE"""
        assert get_next_phase(WizardPhase.VALIDATE) == WizardPhase.MIGRATE

    def test_migrate_returns_pipelines(self):
        """From MIGRATE, next phase should be PIPELINES"""
        assert get_next_phase(WizardPhase.MIGRATE) == WizardPhase.PIPELINES

    def test_pipelines_returns_complete(self):
        """From PIPELINES, next phase should be COMPLETE"""
        assert get_next_phase(WizardPhase.PIPELINES) == WizardPhase.COMPLETE

    def test_complete_returns_none(self):
        """From COMPLETE, next phase should be None"""
        assert get_next_phase(WizardPhase.COMPLETE) is None


class TestWizardCommand:
    def test_wizard_command_exists(self):
        """Verify wizard command is properly defined"""
        assert wizard is not None
        assert hasattr(wizard, 'callback')

    def test_wizard_has_export_directory_option(self):
        """Verify wizard has export_directory option"""
        runner = CliRunner()
        result = runner.invoke(wizard, ['--help'])
        assert '--export_directory' in result.output

    def test_wizard_shows_welcome(self):
        """Verify wizard shows welcome message on start"""
        runner = CliRunner()
        with tempfile.TemporaryDirectory() as tmpdir:
            # Exit immediately by not confirming new session
            result = runner.invoke(wizard, [f'--export_directory={tmpdir}'],
                                   input='n\nn\n')
            assert 'Migration Wizard' in result.output or 'cancelled' in result.output.lower()


class TestResumeCapability:
    def test_detects_existing_state(self):
        """Wizard should detect and offer to resume from existing state"""
        runner = CliRunner()
        with tempfile.TemporaryDirectory() as tmpdir:
            # Create existing state
            state = WizardState(
                phase=WizardPhase.STRUCTURE,
                extract_id="test-123",
                source_url="https://sonar.example.com/"
            )
            state.save(tmpdir)

            # Run wizard and decline resume
            result = runner.invoke(wizard, [f'--export_directory={tmpdir}'],
                                   input='n\nn\n')

            # Should show resume prompt
            assert 'Previous wizard session found' in result.output or \
                   'Resume' in result.output or 'session' in result.output.lower()

    def test_resume_preserves_state(self):
        """Resuming should preserve existing state values"""
        with tempfile.TemporaryDirectory() as tmpdir:
            # Create existing state
            original_state = WizardState(
                phase=WizardPhase.ORG_MAPPING,
                extract_id="extract-456",
                source_url="https://sonar.example.com/",
                target_url="https://sonarcloud.io/"
            )
            original_state.save(tmpdir)

            # Load state (as wizard would do)
            loaded_state = WizardState.load(tmpdir)

            assert loaded_state.phase == WizardPhase.ORG_MAPPING
            assert loaded_state.extract_id == "extract-456"
            assert loaded_state.source_url == "https://sonar.example.com/"
            assert loaded_state.target_url == "https://sonarcloud.io/"


class TestStatePersistence:
    def test_state_saved_after_phase(self):
        """State should be saved after each phase completion"""
        with tempfile.TemporaryDirectory() as tmpdir:
            state = WizardState(phase=WizardPhase.EXTRACT)
            state.extract_id = "test-extract"
            state.source_url = "https://source.example.com/"
            state.phase = WizardPhase.STRUCTURE
            state.save(tmpdir)

            # Verify state file exists
            state_file = os.path.join(tmpdir, '.wizard_state.json')
            assert os.path.exists(state_file)

            # Verify content
            with open(state_file, 'r') as f:
                data = json.load(f)
            assert data['phase'] == 'structure'
            assert data['extract_id'] == 'test-extract'

    def test_state_survives_interruption(self):
        """State should survive process interruption"""
        with tempfile.TemporaryDirectory() as tmpdir:
            # Simulate mid-process state
            state = WizardState(
                phase=WizardPhase.MAPPINGS,
                extract_id="mid-process-123",
                source_url="https://mid.example.com/",
                target_url="https://sonarcloud.io/",
                enterprise_key="test-enterprise",
                organizations_mapped=True
            )
            state.save(tmpdir)

            # Simulate "restart" by loading
            loaded = WizardState.load(tmpdir)

            assert loaded.phase == WizardPhase.MAPPINGS
            assert loaded.extract_id == "mid-process-123"
            assert loaded.organizations_mapped is True


class TestPrompts:
    def test_phase_display_names_complete(self):
        """Verify all phases have display names"""
        for phase in WizardPhase:
            assert phase in PHASE_DISPLAY_NAMES, f"Missing display name for: {phase}"

    def test_phase_order_matches_sequence(self):
        """Verify PHASE_ORDER in prompts matches wizard sequence"""
        # PHASE_ORDER should contain all workflow phases (not INIT or COMPLETE)
        for phase in PHASE_ORDER:
            assert phase in PHASE_SEQUENCE

    def test_display_welcome_output(self, capsys):
        """Verify welcome message contains expected content"""
        display_welcome()
        captured = capsys.readouterr()
        assert 'SonarQube' in captured.out
        assert 'Migration' in captured.out or 'migration' in captured.out

    def test_display_summary_output(self, capsys):
        """Verify summary displays key-value pairs"""
        display_summary("Test Summary", {"Key1": "Value1", "Key2": 42})
        captured = capsys.readouterr()
        assert 'Test Summary' in captured.out
        assert 'Key1' in captured.out
        assert 'Value1' in captured.out


class TestPhaseHandlers:
    def test_extract_handler_exists(self):
        """Verify extract handler is in PHASE_HANDLERS"""
        from wizard.wizard import run_extract_phase
        assert WizardPhase.EXTRACT in PHASE_HANDLERS
        assert PHASE_HANDLERS[WizardPhase.EXTRACT] == run_extract_phase

    def test_structure_handler_exists(self):
        """Verify structure handler is in PHASE_HANDLERS"""
        from wizard.wizard import run_structure_phase
        assert WizardPhase.STRUCTURE in PHASE_HANDLERS
        assert PHASE_HANDLERS[WizardPhase.STRUCTURE] == run_structure_phase

    def test_all_workflow_phases_have_handlers(self):
        """Verify all workflow phases have handlers in PHASE_HANDLERS"""
        workflow_phases = [
            WizardPhase.EXTRACT,
            WizardPhase.STRUCTURE,
            WizardPhase.ORG_MAPPING,
            WizardPhase.MAPPINGS,
            WizardPhase.VALIDATE,
            WizardPhase.MIGRATE,
            WizardPhase.PIPELINES,
        ]
        for phase in workflow_phases:
            assert phase in PHASE_HANDLERS, f"Missing handler for {phase}"
            assert callable(PHASE_HANDLERS[phase]), f"Handler for {phase} is not callable"


class TestWizardIntegration:
    def test_wizard_cli_integration(self):
        """Test wizard is properly integrated into CLI"""
        from main import cli
        runner = CliRunner()
        result = runner.invoke(cli, ['--help'])
        assert 'wizard' in result.output

    def test_wizard_help_text(self):
        """Test wizard help text is informative"""
        runner = CliRunner()
        result = runner.invoke(wizard, ['--help'])
        assert 'Interactive wizard' in result.output or 'wizard' in result.output.lower()
        assert result.exit_code == 0


class TestCompleteWizardState:
    def test_complete_state_handling(self):
        """Test wizard handles COMPLETE state correctly"""
        runner = CliRunner()
        with tempfile.TemporaryDirectory() as tmpdir:
            # Create completed state
            state = WizardState(
                phase=WizardPhase.COMPLETE,
                extract_id="completed-123",
                source_url="https://source.example.com/",
                target_url="https://sonarcloud.io/",
                enterprise_key="ent-key",
                organizations_mapped=True,
                validation_passed=True,
                migration_run_id="migration-456"
            )
            state.save(tmpdir)

            # Run wizard - should show completion message
            result = runner.invoke(wizard, [f'--export_directory={tmpdir}'],
                                   input='n\n')  # Don't start new migration

            assert 'Complete' in result.output or 'complete' in result.output.lower()
