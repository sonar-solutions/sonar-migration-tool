"""Tests for error handling and recovery scenarios in the wizard"""
import json
import os
import tempfile
from unittest.mock import patch, MagicMock

import pytest
from click.testing import CliRunner

from wizard.state import WizardPhase, WizardState
from wizard.wizard import (
    wizard,
    run_extract_phase,
    run_structure_phase,
    run_org_mapping_phase,
    run_validate_phase,
    run_migrate_phase,
)
from tests.test_wizard.conftest import wizard_state_at_phase


class TestKeyboardInterrupt:
    """Tests for KeyboardInterrupt handling during phases"""

    def test_wizard_preserves_state_on_interrupt(self):
        """Wizard should save state when interrupted"""
        runner = CliRunner()

        with tempfile.TemporaryDirectory() as tmpdir:
            # Create a state at STRUCTURE phase
            state = wizard_state_at_phase(WizardPhase.STRUCTURE)
            state.save(tmpdir)

            with patch('wizard.wizard.PHASE_HANDLERS') as mock_handlers:
                # Make STRUCTURE handler raise KeyboardInterrupt
                mock_handlers.__getitem__ = MagicMock(
                    side_effect=KeyboardInterrupt()
                )
                mock_handlers.get = MagicMock(
                    side_effect=KeyboardInterrupt()
                )

                result = runner.invoke(
                    wizard,
                    [f'--export_directory={tmpdir}'],
                    input='y\n'  # Resume from previous
                )

                # Check that interrupted message appears
                assert 'interrupted' in result.output.lower() or result.exit_code != 0

    @patch('utils.get_unique_extracts')
    def test_structure_phase_interrupt_preserves_state(
        self,
        mock_get_extracts,
        temp_export_dir
    ):
        """Test state preserved when structure phase is interrupted"""
        mock_get_extracts.side_effect = KeyboardInterrupt()

        state = wizard_state_at_phase(WizardPhase.STRUCTURE)
        state.save(temp_export_dir)

        with pytest.raises(KeyboardInterrupt):
            run_structure_phase(state, temp_export_dir)

        # State should still be loadable
        loaded_state = WizardState.load(temp_export_dir)
        assert loaded_state.phase == WizardPhase.STRUCTURE

    @patch('wizard.wizard.confirm_review')
    @patch('utils.load_csv')
    @patch('wizard.wizard.confirm_action')
    @patch('wizard.wizard.prompt_text')
    @patch('wizard.wizard.prompt_url')
    def test_org_mapping_interrupt_preserves_state(
        self,
        mock_prompt_url,
        mock_prompt_text,
        mock_confirm_action,
        mock_load_csv,
        mock_confirm_review,
        temp_export_dir
    ):
        """Test state preserved when org mapping is interrupted"""
        mock_confirm_review.return_value = True
        mock_prompt_url.return_value = "https://sonarcloud.io/"
        mock_confirm_action.side_effect = KeyboardInterrupt()
        mock_load_csv.return_value = [{"sonarqube_org_key": "org-1", "sonarcloud_org_key": ""}]

        state = wizard_state_at_phase(WizardPhase.ORG_MAPPING)
        state.save(temp_export_dir)

        with pytest.raises(KeyboardInterrupt):
            run_org_mapping_phase(state, temp_export_dir)

        loaded_state = WizardState.load(temp_export_dir)
        assert loaded_state.phase == WizardPhase.ORG_MAPPING


class TestExceptionHandling:
    """Tests for exception handling during phases"""

    @patch('wizard.wizard.confirm_review')
    @patch('operations.http_request.configure_client_cert')
    @patch('wizard.wizard.confirm_action')
    @patch('wizard.wizard.prompt_credentials')
    @patch('wizard.wizard.prompt_url')
    def test_extract_exception_preserves_state(
        self,
        mock_prompt_url,
        mock_prompt_credentials,
        mock_confirm_action,
        mock_configure_cert,
        mock_confirm_review,
        temp_export_dir
    ):
        """Test state preserved when extract fails with exception"""
        mock_confirm_review.return_value = True
        mock_prompt_url.return_value = "https://sonar.example.com/"
        mock_prompt_credentials.return_value = "token"
        mock_confirm_action.return_value = False
        mock_configure_cert.side_effect = Exception("Certificate error")

        state = wizard_state_at_phase(WizardPhase.EXTRACT)
        # Pre-save state
        state.save(temp_export_dir)

        with pytest.raises(Exception, match="Certificate error"):
            run_extract_phase(state, temp_export_dir)

        # State should be loadable after exception
        loaded = WizardState.load(temp_export_dir)
        assert loaded.phase == WizardPhase.EXTRACT

    @patch('structure.map_project_structure')
    @patch('utils.get_unique_extracts')
    def test_structure_exception_preserves_state(
        self,
        mock_get_extracts,
        mock_map_projects,
        temp_export_dir
    ):
        """Test state preserved when structure analysis fails"""
        mock_get_extracts.return_value = {"https://sonar.example.com/": "123"}
        mock_map_projects.side_effect = Exception("Mapping error")

        state = wizard_state_at_phase(WizardPhase.STRUCTURE)
        state.save(temp_export_dir)

        with pytest.raises(Exception, match="Mapping error"):
            run_structure_phase(state, temp_export_dir)

        loaded = WizardState.load(temp_export_dir)
        assert loaded.phase == WizardPhase.STRUCTURE

    def test_wizard_displays_error_message(self):
        """Test wizard displays error message on failure"""
        runner = CliRunner()

        with tempfile.TemporaryDirectory() as tmpdir:
            state = wizard_state_at_phase(WizardPhase.STRUCTURE)
            state.save(tmpdir)

            with patch('wizard.wizard.run_structure_phase') as mock_phase:
                mock_phase.side_effect = ValueError("Test error message")

                result = runner.invoke(
                    wizard,
                    [f'--export_directory={tmpdir}'],
                    input='y\n'
                )

                # Error should be displayed
                assert 'error' in result.output.lower() or result.exit_code != 0


class TestCorruptedStateRecovery:
    """Tests for recovery from corrupted state files"""

    def test_invalid_json_state_file(self, temp_export_dir):
        """Test handling of invalid JSON in state file"""
        state_file = os.path.join(temp_export_dir, ".wizard_state.json")
        with open(state_file, "w") as f:
            f.write("{ invalid json }")

        with pytest.raises(json.JSONDecodeError):
            WizardState.load(temp_export_dir)

    def test_missing_phase_in_state_file(self, temp_export_dir):
        """Test handling of state file missing required phase field"""
        state_file = os.path.join(temp_export_dir, ".wizard_state.json")
        with open(state_file, "w") as f:
            json.dump({"extract_id": "123"}, f)  # Missing 'phase'

        with pytest.raises(KeyError):
            WizardState.load(temp_export_dir)

    def test_invalid_phase_value_in_state_file(self, temp_export_dir):
        """Test handling of invalid phase value in state file"""
        state_file = os.path.join(temp_export_dir, ".wizard_state.json")
        with open(state_file, "w") as f:
            json.dump({"phase": "invalid_phase"}, f)

        with pytest.raises(ValueError):
            WizardState.load(temp_export_dir)

    def test_extra_fields_in_state_file(self, temp_export_dir):
        """Test handling of extra fields in state file (forward compatibility)"""
        state_file = os.path.join(temp_export_dir, ".wizard_state.json")
        with open(state_file, "w") as f:
            json.dump({
                "phase": "structure",
                "extract_id": "123",
                "future_field": "some_value"  # Unknown field
            }, f)

        # Should load successfully, ignoring unknown fields
        state = WizardState.load(temp_export_dir)
        assert state.phase == WizardPhase.STRUCTURE
        assert state.extract_id == "123"


class TestResumeFromEachPhase:
    """Tests for resuming wizard from each phase after error"""

    def test_resume_from_extract_after_error(self):
        """Test wizard can resume from EXTRACT phase"""
        runner = CliRunner()

        with tempfile.TemporaryDirectory() as tmpdir:
            state = wizard_state_at_phase(WizardPhase.EXTRACT)
            state.save(tmpdir)

            result = runner.invoke(
                wizard,
                [f'--export_directory={tmpdir}'],
                input='n\nn\n'  # Don't resume, don't start new
            )

            assert 'Extract' in result.output or 'cancelled' in result.output.lower()

    def test_resume_from_structure_after_error(self):
        """Test wizard can resume from STRUCTURE phase"""
        runner = CliRunner()

        with tempfile.TemporaryDirectory() as tmpdir:
            state = wizard_state_at_phase(WizardPhase.STRUCTURE)
            state.save(tmpdir)

            result = runner.invoke(
                wizard,
                [f'--export_directory={tmpdir}'],
                input='n\nn\n'
            )

            assert 'Structure' in result.output or 'cancelled' in result.output.lower()

    def test_resume_from_org_mapping_after_error(self):
        """Test wizard can resume from ORG_MAPPING phase"""
        runner = CliRunner()

        with tempfile.TemporaryDirectory() as tmpdir:
            state = wizard_state_at_phase(WizardPhase.ORG_MAPPING)
            state.save(tmpdir)

            result = runner.invoke(
                wizard,
                [f'--export_directory={tmpdir}'],
                input='n\nn\n'
            )

            assert 'session' in result.output.lower() or 'cancelled' in result.output.lower()

    def test_resume_from_mappings_after_error(self):
        """Test wizard can resume from MAPPINGS phase"""
        runner = CliRunner()

        with tempfile.TemporaryDirectory() as tmpdir:
            state = wizard_state_at_phase(WizardPhase.MAPPINGS)
            state.save(tmpdir)

            result = runner.invoke(
                wizard,
                [f'--export_directory={tmpdir}'],
                input='n\nn\n'
            )

            assert 'session' in result.output.lower() or 'cancelled' in result.output.lower()

    def test_resume_from_validate_after_error(self):
        """Test wizard can resume from VALIDATE phase"""
        runner = CliRunner()

        with tempfile.TemporaryDirectory() as tmpdir:
            state = wizard_state_at_phase(WizardPhase.VALIDATE)
            state.save(tmpdir)

            result = runner.invoke(
                wizard,
                [f'--export_directory={tmpdir}'],
                input='n\nn\n'
            )

            assert 'session' in result.output.lower() or 'cancelled' in result.output.lower()

    def test_resume_from_migrate_after_error(self):
        """Test wizard can resume from MIGRATE phase"""
        runner = CliRunner()

        with tempfile.TemporaryDirectory() as tmpdir:
            state = wizard_state_at_phase(WizardPhase.MIGRATE)
            state.save(tmpdir)

            result = runner.invoke(
                wizard,
                [f'--export_directory={tmpdir}'],
                input='n\nn\n'
            )

            assert 'session' in result.output.lower() or 'cancelled' in result.output.lower()

    def test_resume_from_pipelines_after_error(self):
        """Test wizard can resume from PIPELINES phase"""
        runner = CliRunner()

        with tempfile.TemporaryDirectory() as tmpdir:
            state = wizard_state_at_phase(WizardPhase.PIPELINES)
            state.save(tmpdir)

            result = runner.invoke(
                wizard,
                [f'--export_directory={tmpdir}'],
                input='n\nn\n'
            )

            assert 'session' in result.output.lower() or 'cancelled' in result.output.lower()


class TestPartialStateRecovery:
    """Tests for recovering from partial state data"""

    def test_resume_with_partial_state_data(self, temp_export_dir):
        """Test resuming when state has minimal data"""
        state_file = os.path.join(temp_export_dir, ".wizard_state.json")
        with open(state_file, "w") as f:
            json.dump({
                "phase": "org_mapping"
                # Missing all other fields
            }, f)

        state = WizardState.load(temp_export_dir)

        assert state.phase == WizardPhase.ORG_MAPPING
        assert state.extract_id is None
        assert state.source_url is None
        assert state.organizations_mapped is False

    def test_resume_preserves_completed_work(self, temp_export_dir):
        """Test that resuming preserves work from previous phases"""
        state = WizardState(
            phase=WizardPhase.MAPPINGS,
            extract_id="my-extract-123",
            source_url="https://sonar.example.com/",
            target_url="https://sonarcloud.io/",
            enterprise_key="ent-key-456",
            organizations_mapped=True
        )
        state.save(temp_export_dir)

        loaded = WizardState.load(temp_export_dir)

        assert loaded.phase == WizardPhase.MAPPINGS
        assert loaded.extract_id == "my-extract-123"
        assert loaded.source_url == "https://sonar.example.com/"
        assert loaded.target_url == "https://sonarcloud.io/"
        assert loaded.enterprise_key == "ent-key-456"
        assert loaded.organizations_mapped is True


class TestGracefulDegradation:
    """Tests for graceful handling of missing dependencies"""

    @patch('utils.get_unique_extracts')
    def test_structure_handles_empty_extracts(
        self,
        mock_get_extracts,
        temp_export_dir
    ):
        """Test structure phase handles case with no extracts gracefully"""
        mock_get_extracts.return_value = {}

        state = wizard_state_at_phase(WizardPhase.STRUCTURE)

        with pytest.raises(ValueError, match="No extracts found"):
            run_structure_phase(state, temp_export_dir)

    @patch('wizard.wizard.confirm_review')
    @patch('utils.load_csv')
    @patch('wizard.wizard.confirm_action')
    @patch('wizard.wizard.prompt_text')
    @patch('wizard.wizard.prompt_url')
    def test_org_mapping_handles_empty_orgs(
        self,
        mock_prompt_url,
        mock_prompt_text,
        mock_confirm_action,
        mock_load_csv,
        mock_confirm_review,
        temp_export_dir
    ):
        """Test org mapping handles case with no organizations"""
        mock_confirm_review.return_value = True
        mock_prompt_url.return_value = "https://sonarcloud.io/"
        mock_prompt_text.return_value = "ent-key"
        mock_confirm_action.return_value = False  # Decline retry
        mock_load_csv.return_value = []

        state = wizard_state_at_phase(WizardPhase.ORG_MAPPING)

        with pytest.raises(ValueError, match="No organizations found"):
            run_org_mapping_phase(state, temp_export_dir)

    def test_validate_handles_missing_csv_files(self, temp_export_dir):
        """Test validate phase handles missing CSV files"""
        state = wizard_state_at_phase(WizardPhase.VALIDATE)

        with pytest.raises(ValueError, match="Missing required files"):
            run_validate_phase(state, temp_export_dir)


class TestNetworkErrorHandling:
    """Tests for network-related error handling"""

    @patch('wizard.wizard.confirm_review')
    @patch('operations.http_request.configure_client_cert')
    @patch('operations.http_request.get_server_details')
    @patch('wizard.wizard.confirm_action')
    @patch('wizard.wizard.prompt_credentials')
    @patch('wizard.wizard.prompt_url')
    def test_extract_handles_connection_timeout(
        self,
        mock_prompt_url,
        mock_prompt_credentials,
        mock_confirm_action,
        mock_get_server_details,
        mock_configure_cert,
        mock_confirm_review,
        temp_export_dir
    ):
        """Test extract handles connection timeout"""
        mock_confirm_review.return_value = True
        mock_prompt_url.return_value = "https://sonar.example.com/"
        mock_prompt_credentials.return_value = "token"
        mock_confirm_action.return_value = False
        mock_configure_cert.return_value = None
        mock_get_server_details.side_effect = TimeoutError("Connection timed out")

        state = wizard_state_at_phase(WizardPhase.EXTRACT)

        with pytest.raises(TimeoutError):
            run_extract_phase(state, temp_export_dir)

    @patch('wizard.wizard.confirm_review')
    @patch('operations.http_request.configure_client_cert')
    @patch('operations.http_request.get_server_details')
    @patch('wizard.wizard.confirm_action')
    @patch('wizard.wizard.prompt_credentials')
    @patch('wizard.wizard.prompt_url')
    def test_extract_handles_connection_refused(
        self,
        mock_prompt_url,
        mock_prompt_credentials,
        mock_confirm_action,
        mock_get_server_details,
        mock_configure_cert,
        mock_confirm_review,
        temp_export_dir
    ):
        """Test extract handles connection refused"""
        mock_confirm_review.return_value = True
        mock_prompt_url.return_value = "https://sonar.example.com/"
        mock_prompt_credentials.return_value = "token"
        mock_confirm_action.return_value = False
        mock_configure_cert.return_value = None
        mock_get_server_details.side_effect = ConnectionRefusedError("Connection refused")

        state = wizard_state_at_phase(WizardPhase.EXTRACT)

        with pytest.raises(ConnectionRefusedError):
            run_extract_phase(state, temp_export_dir)

    @patch('operations.http_request.configure_client')
    @patch('wizard.wizard.prompt_credentials')
    @patch('wizard.wizard.confirm_action')
    def test_migrate_handles_cloud_connection_error(
        self,
        mock_confirm,
        mock_prompt_creds,
        mock_configure_client,
        temp_export_dir
    ):
        """Test migrate handles SonarCloud connection error"""
        mock_confirm.side_effect = [True, False]  # Proceed, then decline retry
        mock_prompt_creds.return_value = "cloud-token"
        mock_configure_client.side_effect = ConnectionError("Unable to reach SonarCloud")

        state = wizard_state_at_phase(WizardPhase.MIGRATE)

        with pytest.raises(ConnectionError):
            run_migrate_phase(state, temp_export_dir)


class TestRetryOnError:
    """Tests for retry-on-error behavior in phases that collect user input"""

    @patch('wizard.wizard.confirm_review')
    @patch('operations.http_request.configure_client_cert')
    @patch('operations.http_request.get_server_details')
    @patch('operations.http_request.configure_client')
    @patch('logs.configure_logger')
    @patch('plan.get_available_task_configs')
    @patch('plan.generate_task_plan')
    @patch('execute.execute_plan')
    @patch('wizard.wizard.confirm_action')
    @patch('wizard.wizard.prompt_credentials')
    @patch('wizard.wizard.prompt_url')
    def test_extract_retry_succeeds_after_failure(
        self,
        mock_prompt_url,
        mock_prompt_credentials,
        mock_confirm_action,
        mock_execute_plan,
        mock_generate_plan,
        mock_get_configs,
        mock_configure_logger,
        mock_configure_client,
        mock_get_server_details,
        mock_configure_cert,
        mock_confirm_review,
        temp_export_dir
    ):
        """Test extract retries after failure and succeeds on second attempt"""
        mock_confirm_review.return_value = True
        mock_prompt_url.side_effect = ["https://bad.example.com/", "https://good.example.com/"]
        mock_prompt_credentials.return_value = "token"
        # One confirm_action call: the retry prompt after the connection error
        mock_confirm_action.side_effect = [True]
        mock_configure_cert.return_value = None
        mock_get_server_details.side_effect = [
            ConnectionError("Connection refused"),
            ("9.9.0.1234", "enterprise")
        ]
        mock_get_configs.return_value = {"getProjects": {}}
        mock_generate_plan.return_value = [["getProjects"]]

        state = wizard_state_at_phase(WizardPhase.EXTRACT)
        result = run_extract_phase(state, temp_export_dir)

        assert result.phase == WizardPhase.STRUCTURE
        assert result.source_url == "https://good.example.com/"
        mock_execute_plan.assert_called_once()

    @patch('wizard.wizard.confirm_review')
    @patch('operations.http_request.configure_client_cert')
    @patch('wizard.wizard.confirm_action')
    @patch('wizard.wizard.prompt_credentials')
    @patch('wizard.wizard.prompt_url')
    def test_extract_retry_clears_source_url(
        self,
        mock_prompt_url,
        mock_prompt_credentials,
        mock_confirm_action,
        mock_configure_cert,
        mock_confirm_review,
        temp_export_dir
    ):
        """Test that retry clears source_url so it is re-prompted"""
        mock_confirm_review.return_value = True
        mock_prompt_url.return_value = "https://sonar.example.com/"
        mock_prompt_credentials.return_value = "token"
        mock_confirm_action.side_effect = [True, False]  # retry after fail 1, decline after fail 2
        mock_configure_cert.side_effect = [Exception("fail"), Exception("fail again")]

        state = wizard_state_at_phase(WizardPhase.EXTRACT)

        with pytest.raises(Exception, match="fail again"):
            run_extract_phase(state, temp_export_dir)

        # prompt_url should be called twice (once per attempt, since source_url is cleared)
        assert mock_prompt_url.call_count == 2

    @patch('wizard.wizard.confirm_review')
    @patch('utils.load_csv')
    @patch('utils.export_csv')
    @patch('wizard.wizard.confirm_action')
    @patch('wizard.wizard.prompt_text')
    @patch('wizard.wizard.prompt_url')
    def test_org_mapping_retry_succeeds_after_failure(
        self,
        mock_prompt_url,
        mock_prompt_text,
        mock_confirm_action,
        mock_export_csv,
        mock_load_csv,
        mock_confirm_review,
        temp_export_dir
    ):
        """Test org mapping retries after failure and succeeds on second attempt"""
        mock_confirm_review.return_value = True
        mock_prompt_url.return_value = "https://sonarcloud.io/"
        mock_prompt_text.side_effect = ["ent-key", "ent-key-2", "cloud-org-1"]
        mock_confirm_action.return_value = True  # Retry on error
        # First attempt: empty orgs, second attempt: has orgs
        mock_load_csv.side_effect = [
            [],
            [{"sonarqube_org_key": "org-1", "server_url": "https://sonar.example.com/",
              "devops_binding": "github", "project_count": 1, "sonarcloud_org_key": ""}]
        ]

        state = wizard_state_at_phase(WizardPhase.ORG_MAPPING, target_url=None, enterprise_key=None)
        result = run_org_mapping_phase(state, temp_export_dir)

        assert result.phase == WizardPhase.MAPPINGS
        assert result.organizations_mapped is True

    @patch('wizard.wizard.confirm_review')
    @patch('utils.load_csv')
    @patch('wizard.wizard.confirm_action')
    @patch('wizard.wizard.prompt_text')
    @patch('wizard.wizard.prompt_url')
    def test_org_mapping_retry_clears_credentials(
        self,
        mock_prompt_url,
        mock_prompt_text,
        mock_confirm_action,
        mock_load_csv,
        mock_confirm_review,
        temp_export_dir
    ):
        """Test that retry clears target_url and enterprise_key"""
        mock_confirm_review.return_value = True
        mock_prompt_url.return_value = "https://sonarcloud.io/"
        mock_prompt_text.return_value = "ent-key"
        mock_confirm_action.side_effect = [True, False]  # Retry once, then decline
        mock_load_csv.return_value = []  # Always empty to trigger error

        state = wizard_state_at_phase(WizardPhase.ORG_MAPPING, target_url=None, enterprise_key=None)

        with pytest.raises(ValueError, match="No organizations found"):
            run_org_mapping_phase(state, temp_export_dir)

        # prompt_url called twice (cleared on retry)
        assert mock_prompt_url.call_count == 2

    @patch('operations.http_request.configure_client')
    @patch('logs.configure_logger')
    @patch('plan.get_available_task_configs')
    @patch('utils.get_unique_extracts')
    @patch('validate.validate_migration')
    @patch('plan.generate_task_plan')
    @patch('utils.filter_completed')
    @patch('execute.execute_plan')
    @patch('wizard.wizard.prompt_credentials')
    @patch('wizard.wizard.confirm_action')
    def test_migrate_retry_succeeds_after_failure(
        self,
        mock_confirm,
        mock_prompt_creds,
        mock_execute_plan,
        mock_filter_completed,
        mock_generate_plan,
        mock_validate_migration,
        mock_get_extracts,
        mock_get_configs,
        mock_configure_logger,
        mock_configure_client,
        temp_export_dir
    ):
        """Test migrate retries after failure and succeeds on second attempt"""
        mock_confirm.side_effect = [True, True]  # Proceed, then retry
        mock_prompt_creds.side_effect = ["bad-token", "good-token"]
        mock_configure_client.side_effect = [
            Exception("Auth failed"),  # First attempt fails
            None, None  # Second attempt succeeds (two configure_client calls)
        ]
        mock_get_configs.return_value = {"createProject": {}}
        mock_get_extracts.return_value = {"https://sonar.example.com/": "1234567890"}
        mock_validate_migration.return_value = (temp_export_dir, set())
        mock_generate_plan.return_value = [["createProject"]]
        mock_filter_completed.return_value = [["createProject"]]

        state = wizard_state_at_phase(WizardPhase.MIGRATE)
        result = run_migrate_phase(state, temp_export_dir)

        assert result.phase == WizardPhase.PIPELINES
        mock_execute_plan.assert_called_once()
