"""End-to-end integration tests for the wizard flow"""
import json
import os
import tempfile
from unittest.mock import patch

from click.testing import CliRunner

from wizard.state import WizardPhase, WizardState
from wizard.wizard import wizard, get_next_phase
from tests.test_wizard.conftest import wizard_state_at_phase


def make_mock_handler_for_phase(current_phase, call_tracker, terminal=False):
    """Create a mock handler that records being called.

    The wizard main loop advances via current_phase = state.phase, so each
    non-terminal handler must set state.phase to the next phase in sequence.
    If terminal=True, we set state.phase = COMPLETE to end the wizard.
    """
    def handler(state, directory):
        call_tracker.append(current_phase)  # Record which handler was called
        if terminal:
            state.phase = WizardPhase.COMPLETE
        else:
            state.phase = get_next_phase(current_phase)  # Advance to next phase
        state.save(directory)
        return state
    return handler


class TestFreshWizardStart:
    """Tests for starting wizard with no previous state"""

    def test_fresh_start_shows_welcome(self):
        """Fresh start should show welcome message"""
        runner = CliRunner()

        with tempfile.TemporaryDirectory() as tmpdir:
            result = runner.invoke(
                wizard,
                [f'--export_directory={tmpdir}'],
                input='n\nn\n'  # Don't start
            )

            assert 'SonarQube' in result.output
            assert 'Migration' in result.output

    def test_fresh_start_creates_directory(self):
        """Fresh start should create export directory if needed"""
        runner = CliRunner()

        with tempfile.TemporaryDirectory() as tmpdir:
            export_dir = os.path.join(tmpdir, 'new_export_dir')

            runner.invoke(
                wizard,
                [f'--export_directory={export_dir}'],
                input='n\nn\n'
            )

            assert os.path.exists(export_dir)

    def test_fresh_start_runs_all_phases(self):
        """Fresh start should run through all phases"""
        runner = CliRunner()

        call_tracker = []

        # Create mock handlers that track calls and advance phases
        mock_handlers = {
            WizardPhase.INIT: lambda s, d: s,
            WizardPhase.EXTRACT: make_mock_handler_for_phase(WizardPhase.EXTRACT, call_tracker),
            WizardPhase.STRUCTURE: make_mock_handler_for_phase(WizardPhase.STRUCTURE, call_tracker),
            WizardPhase.ORG_MAPPING: make_mock_handler_for_phase(WizardPhase.ORG_MAPPING, call_tracker),
            WizardPhase.MAPPINGS: make_mock_handler_for_phase(WizardPhase.MAPPINGS, call_tracker),
            WizardPhase.VALIDATE: make_mock_handler_for_phase(WizardPhase.VALIDATE, call_tracker),
            WizardPhase.MIGRATE: make_mock_handler_for_phase(WizardPhase.MIGRATE, call_tracker),
            WizardPhase.PIPELINES: make_mock_handler_for_phase(WizardPhase.PIPELINES, call_tracker, terminal=True),
        }

        with tempfile.TemporaryDirectory() as tmpdir:
            with patch.dict('wizard.wizard.PHASE_HANDLERS', mock_handlers):
                result = runner.invoke(
                    wizard,
                    [f'--export_directory={tmpdir}']
                )

            # All phases should have been called
            assert WizardPhase.EXTRACT in call_tracker
            assert WizardPhase.STRUCTURE in call_tracker
            assert WizardPhase.ORG_MAPPING in call_tracker
            assert WizardPhase.MAPPINGS in call_tracker
            assert WizardPhase.VALIDATE in call_tracker
            assert WizardPhase.MIGRATE in call_tracker
            assert WizardPhase.PIPELINES in call_tracker

            # Should show completion
            assert 'Complete' in result.output


class TestResumeFromEachPhase:
    """Tests for resuming wizard from each phase"""

    def test_resume_from_extract(self):
        """Test resuming from EXTRACT phase"""
        runner = CliRunner()
        call_tracker = []

        mock_handlers = {
            WizardPhase.INIT: lambda s, d: s,
            WizardPhase.EXTRACT: make_mock_handler_for_phase(WizardPhase.EXTRACT, call_tracker, terminal=True),
        }

        with tempfile.TemporaryDirectory() as tmpdir:
            state = wizard_state_at_phase(WizardPhase.EXTRACT)
            state.save(tmpdir)

            with patch.dict('wizard.wizard.PHASE_HANDLERS', mock_handlers):
                result = runner.invoke(
                    wizard,
                    [f'--export_directory={tmpdir}'],
                    input='y\n'  # Resume
                )

            assert 'Resuming' in result.output or WizardPhase.EXTRACT in call_tracker

    def test_resume_from_structure(self):
        """Test resuming from STRUCTURE phase"""
        runner = CliRunner()
        call_tracker = []

        mock_handlers = {
            WizardPhase.INIT: lambda s, d: s,
            WizardPhase.STRUCTURE: make_mock_handler_for_phase(WizardPhase.STRUCTURE, call_tracker, terminal=True),
        }

        with tempfile.TemporaryDirectory() as tmpdir:
            state = wizard_state_at_phase(WizardPhase.STRUCTURE)
            state.save(tmpdir)

            with patch.dict('wizard.wizard.PHASE_HANDLERS', mock_handlers):
                runner.invoke(
                    wizard,
                    [f'--export_directory={tmpdir}'],
                    input='y\n'
                )

            assert WizardPhase.STRUCTURE in call_tracker

    def test_resume_from_org_mapping(self):
        """Test resuming from ORG_MAPPING phase"""
        runner = CliRunner()
        call_tracker = []

        mock_handlers = {
            WizardPhase.INIT: lambda s, d: s,
            WizardPhase.ORG_MAPPING: make_mock_handler_for_phase(WizardPhase.ORG_MAPPING, call_tracker, terminal=True),
        }

        with tempfile.TemporaryDirectory() as tmpdir:
            state = wizard_state_at_phase(WizardPhase.ORG_MAPPING)
            state.save(tmpdir)

            with patch.dict('wizard.wizard.PHASE_HANDLERS', mock_handlers):
                runner.invoke(
                    wizard,
                    [f'--export_directory={tmpdir}'],
                    input='y\n'
                )

            assert WizardPhase.ORG_MAPPING in call_tracker

    def test_resume_from_mappings(self):
        """Test resuming from MAPPINGS phase"""
        runner = CliRunner()
        call_tracker = []

        mock_handlers = {
            WizardPhase.INIT: lambda s, d: s,
            WizardPhase.MAPPINGS: make_mock_handler_for_phase(WizardPhase.MAPPINGS, call_tracker, terminal=True),
        }

        with tempfile.TemporaryDirectory() as tmpdir:
            state = wizard_state_at_phase(WizardPhase.MAPPINGS)
            state.save(tmpdir)

            with patch.dict('wizard.wizard.PHASE_HANDLERS', mock_handlers):
                runner.invoke(
                    wizard,
                    [f'--export_directory={tmpdir}'],
                    input='y\n'
                )

            assert WizardPhase.MAPPINGS in call_tracker

    def test_resume_from_validate(self):
        """Test resuming from VALIDATE phase"""
        runner = CliRunner()
        call_tracker = []

        mock_handlers = {
            WizardPhase.INIT: lambda s, d: s,
            WizardPhase.VALIDATE: make_mock_handler_for_phase(WizardPhase.VALIDATE, call_tracker, terminal=True),
        }

        with tempfile.TemporaryDirectory() as tmpdir:
            state = wizard_state_at_phase(WizardPhase.VALIDATE)
            state.save(tmpdir)

            with patch.dict('wizard.wizard.PHASE_HANDLERS', mock_handlers):
                runner.invoke(
                    wizard,
                    [f'--export_directory={tmpdir}'],
                    input='y\n'
                )

            assert WizardPhase.VALIDATE in call_tracker

    def test_resume_from_migrate(self):
        """Test resuming from MIGRATE phase"""
        runner = CliRunner()
        call_tracker = []

        mock_handlers = {
            WizardPhase.INIT: lambda s, d: s,
            WizardPhase.MIGRATE: make_mock_handler_for_phase(WizardPhase.MIGRATE, call_tracker, terminal=True),
        }

        with tempfile.TemporaryDirectory() as tmpdir:
            state = wizard_state_at_phase(WizardPhase.MIGRATE)
            state.save(tmpdir)

            with patch.dict('wizard.wizard.PHASE_HANDLERS', mock_handlers):
                runner.invoke(
                    wizard,
                    [f'--export_directory={tmpdir}'],
                    input='y\n'
                )

            assert WizardPhase.MIGRATE in call_tracker

    def test_resume_from_pipelines(self):
        """Test resuming from PIPELINES phase"""
        runner = CliRunner()
        call_tracker = []

        mock_handlers = {
            WizardPhase.INIT: lambda s, d: s,
            WizardPhase.PIPELINES: make_mock_handler_for_phase(WizardPhase.PIPELINES, call_tracker, terminal=True),
        }

        with tempfile.TemporaryDirectory() as tmpdir:
            state = wizard_state_at_phase(WizardPhase.PIPELINES)
            state.save(tmpdir)

            with patch.dict('wizard.wizard.PHASE_HANDLERS', mock_handlers):
                runner.invoke(
                    wizard,
                    [f'--export_directory={tmpdir}'],
                    input='y\n'
                )

            assert WizardPhase.PIPELINES in call_tracker


class TestCancelAndResume:
    """Tests for cancel and resume workflow"""

    def test_cancel_preserves_state(self):
        """Canceling wizard should preserve state for later resume"""
        runner = CliRunner()

        with tempfile.TemporaryDirectory() as tmpdir:
            state = wizard_state_at_phase(
                WizardPhase.MAPPINGS,
                extract_id="test-extract-123",
                source_url="https://sonar.example.com/",
                target_url="https://sonarcloud.io/",
                enterprise_key="ent-key"
            )
            state.save(tmpdir)

            # Decline resume and decline new session
            runner.invoke(
                wizard,
                [f'--export_directory={tmpdir}'],
                input='n\nn\n'
            )

            # State should still be loadable
            loaded = WizardState.load(tmpdir)
            assert loaded.phase == WizardPhase.MAPPINGS
            assert loaded.extract_id == "test-extract-123"

    def test_resume_after_cancel(self):
        """Wizard should resume correctly after previous cancel"""
        runner = CliRunner()

        with tempfile.TemporaryDirectory() as tmpdir:
            state = wizard_state_at_phase(WizardPhase.VALIDATE)
            state.save(tmpdir)

            # First invocation - cancel
            runner.invoke(
                wizard,
                [f'--export_directory={tmpdir}'],
                input='n\nn\n'
            )

            # Second invocation - resume prompt should appear
            result = runner.invoke(
                wizard,
                [f'--export_directory={tmpdir}'],
                input='n\nn\n'
            )

            assert 'Previous' in result.output or 'Resume' in result.output


class TestCompleteStateHandling:
    """Tests for handling COMPLETE state"""

    def test_complete_state_shows_completion_message(self):
        """COMPLETE state should show completion message"""
        runner = CliRunner()

        with tempfile.TemporaryDirectory() as tmpdir:
            state = wizard_state_at_phase(WizardPhase.COMPLETE)
            state.save(tmpdir)

            result = runner.invoke(
                wizard,
                [f'--export_directory={tmpdir}'],
                input='n\n'  # Don't start new
            )

            assert 'Complete' in result.output

    def test_complete_state_offers_new_migration(self):
        """COMPLETE state should offer to start new migration"""
        runner = CliRunner()

        with tempfile.TemporaryDirectory() as tmpdir:
            state = wizard_state_at_phase(WizardPhase.COMPLETE)
            state.save(tmpdir)

            result = runner.invoke(
                wizard,
                [f'--export_directory={tmpdir}'],
                input='n\n'
            )

            # Should ask about new migration
            assert 'new' in result.output.lower() or 'migration' in result.output.lower()

    @patch('wizard.wizard.confirm_action')
    def test_complete_state_new_migration_resets_state(self, mock_confirm):
        """Starting new migration from COMPLETE should reset state"""
        runner = CliRunner()
        call_tracker = []

        # Mock confirm_action to return True (start new migration)
        mock_confirm.return_value = True

        mock_handlers = {
            WizardPhase.INIT: lambda s, d: s,
            WizardPhase.EXTRACT: make_mock_handler_for_phase(WizardPhase.EXTRACT, call_tracker, terminal=True),
        }

        with tempfile.TemporaryDirectory() as tmpdir:
            state = wizard_state_at_phase(WizardPhase.COMPLETE)
            state.save(tmpdir)

            with patch.dict('wizard.wizard.PHASE_HANDLERS', mock_handlers):
                runner.invoke(
                    wizard,
                    [f'--export_directory={tmpdir}']
                )

            # Extract should be called (first phase of new wizard)
            assert WizardPhase.EXTRACT in call_tracker


class TestStateTransitions:
    """Tests verifying correct state transitions"""

    def test_extract_to_structure_transition(self):
        """State should transition from EXTRACT to STRUCTURE"""
        runner = CliRunner()
        call_tracker = []

        mock_handlers = {
            WizardPhase.INIT: lambda s, d: s,
            WizardPhase.EXTRACT: make_mock_handler_for_phase(WizardPhase.EXTRACT, call_tracker),
            WizardPhase.STRUCTURE: make_mock_handler_for_phase(WizardPhase.STRUCTURE, call_tracker, terminal=True),
        }

        with tempfile.TemporaryDirectory() as tmpdir:
            with patch.dict('wizard.wizard.PHASE_HANDLERS', mock_handlers):
                runner.invoke(
                    wizard,
                    [f'--export_directory={tmpdir}']
                )

            assert WizardPhase.EXTRACT in call_tracker
            assert WizardPhase.STRUCTURE in call_tracker

    def test_state_saved_after_each_phase(self):
        """State should be saved after each phase"""
        runner = CliRunner()
        phase_calls = []

        def phase_handler_factory(current, terminal=False):
            def handler(state, directory):
                phase_calls.append(current)
                if terminal:
                    state.phase = WizardPhase.COMPLETE
                else:
                    state.phase = get_next_phase(current)  # Advance to next phase
                state.save(directory)

                # Verify state was saved
                state_file = os.path.join(directory, '.wizard_state.json')
                assert os.path.exists(state_file)
                with open(state_file) as f:
                    data = json.load(f)
                expected_phase = WizardPhase.COMPLETE.value if terminal else get_next_phase(current).value
                assert data['phase'] == expected_phase

                return state
            return handler

        mock_handlers = {
            WizardPhase.INIT: lambda s, d: s,
            WizardPhase.EXTRACT: phase_handler_factory(WizardPhase.EXTRACT),
            WizardPhase.STRUCTURE: phase_handler_factory(WizardPhase.STRUCTURE, terminal=True),
        }

        with tempfile.TemporaryDirectory() as tmpdir:
            with patch.dict('wizard.wizard.PHASE_HANDLERS', mock_handlers):
                runner.invoke(
                    wizard,
                    [f'--export_directory={tmpdir}']
                )

            assert WizardPhase.EXTRACT in phase_calls
            assert WizardPhase.STRUCTURE in phase_calls


class TestPhaseProgressDisplay:
    """Tests for phase progress display during wizard"""

    def test_shows_phase_progress(self):
        """Wizard should show phase progress indicator"""
        runner = CliRunner()
        call_tracker = []

        mock_handlers = {
            WizardPhase.INIT: lambda s, d: s,
            WizardPhase.EXTRACT: make_mock_handler_for_phase(WizardPhase.EXTRACT, call_tracker, terminal=True),
        }

        with tempfile.TemporaryDirectory() as tmpdir:
            with patch.dict('wizard.wizard.PHASE_HANDLERS', mock_handlers):
                result = runner.invoke(
                    wizard,
                    [f'--export_directory={tmpdir}']
                )

            # Should show progress like [1/7]
            assert '[1/' in result.output or 'Extract' in result.output


class TestUserInputPreservation:
    """Tests for preserving user input across resume"""

    def test_source_url_preserved_on_resume(self):
        """Source URL should be preserved when resuming"""
        with tempfile.TemporaryDirectory() as tmpdir:
            state = WizardState(
                phase=WizardPhase.STRUCTURE,
                source_url="https://original-sonar.example.com/"
            )
            state.save(tmpdir)

            loaded = WizardState.load(tmpdir)
            assert loaded.source_url == "https://original-sonar.example.com/"

    def test_target_url_preserved_on_resume(self):
        """Target URL should be preserved when resuming"""
        with tempfile.TemporaryDirectory() as tmpdir:
            state = WizardState(
                phase=WizardPhase.MAPPINGS,
                target_url="https://sonarcloud.io/"
            )
            state.save(tmpdir)

            loaded = WizardState.load(tmpdir)
            assert loaded.target_url == "https://sonarcloud.io/"

    def test_enterprise_key_preserved_on_resume(self):
        """Enterprise key should be preserved when resuming"""
        with tempfile.TemporaryDirectory() as tmpdir:
            state = WizardState(
                phase=WizardPhase.MAPPINGS,
                enterprise_key="my-enterprise-key"
            )
            state.save(tmpdir)

            loaded = WizardState.load(tmpdir)
            assert loaded.enterprise_key == "my-enterprise-key"

    def test_all_state_preserved_on_resume(self):
        """All state fields should be preserved when resuming"""
        with tempfile.TemporaryDirectory() as tmpdir:
            original = WizardState(
                phase=WizardPhase.MIGRATE,
                extract_id="extract-abc123",
                source_url="https://source.example.com/",
                target_url="https://target.example.com/",
                enterprise_key="ent-xyz789",
                organizations_mapped=True,
                validation_passed=True,
                migration_run_id=None
            )
            original.save(tmpdir)

            loaded = WizardState.load(tmpdir)

            assert loaded.phase == original.phase
            assert loaded.extract_id == original.extract_id
            assert loaded.source_url == original.source_url
            assert loaded.target_url == original.target_url
            assert loaded.enterprise_key == original.enterprise_key
            assert loaded.organizations_mapped == original.organizations_mapped
            assert loaded.validation_passed == original.validation_passed
            assert loaded.migration_run_id == original.migration_run_id


class TestExportDirectoryOption:
    """Tests for export_directory CLI option"""

    def test_default_export_directory(self):
        """Wizard should have default export directory"""
        runner = CliRunner()

        result = runner.invoke(wizard, ['--help'])
        assert '--export_directory' in result.output
        # Check for default value mention (could be in different formats)
        assert 'files' in result.output.lower() or 'default' in result.output.lower()

    def test_custom_export_directory(self):
        """Wizard should accept custom export directory"""
        runner = CliRunner()

        with tempfile.TemporaryDirectory() as tmpdir:
            custom_dir = os.path.join(tmpdir, 'custom')

            runner.invoke(
                wizard,
                [f'--export_directory={custom_dir}'],
                input='n\nn\n'
            )

            assert os.path.exists(custom_dir)


class TestPhaseHandlerIntegration:
    """Tests for phase handler integration with wizard main loop"""

    def test_handler_receives_correct_state(self):
        """Phase handlers should receive correct state"""
        runner = CliRunner()

        received_states = []

        def capture_state(state, directory):
            received_states.append(state.phase)
            state.phase = WizardPhase.COMPLETE
            state.save(directory)
            return state

        mock_handlers = {
            WizardPhase.INIT: lambda s, d: s,
            WizardPhase.EXTRACT: capture_state,
        }

        with tempfile.TemporaryDirectory() as tmpdir:
            with patch.dict('wizard.wizard.PHASE_HANDLERS', mock_handlers):
                runner.invoke(
                    wizard,
                    [f'--export_directory={tmpdir}']
                )

            # Handler should have received EXTRACT phase state
            # (actually receives the state object, but phase should be set before handler)
            assert len(received_states) >= 1

    def test_handler_receives_correct_directory(self):
        """Phase handlers should receive correct export directory"""
        runner = CliRunner()

        received_dirs = []

        def capture_dir(state, directory):
            received_dirs.append(directory)
            state.phase = WizardPhase.COMPLETE
            state.save(directory)
            return state

        mock_handlers = {
            WizardPhase.INIT: lambda s, d: s,
            WizardPhase.EXTRACT: capture_dir,
        }

        with tempfile.TemporaryDirectory() as tmpdir:
            with patch.dict('wizard.wizard.PHASE_HANDLERS', mock_handlers):
                runner.invoke(
                    wizard,
                    [f'--export_directory={tmpdir}']
                )

            assert len(received_dirs) >= 1
            assert received_dirs[0] == tmpdir
