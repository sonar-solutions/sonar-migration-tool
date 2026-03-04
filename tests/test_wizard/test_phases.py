"""Tests for individual phase handlers with mocked dependencies"""
import json
import os
from unittest.mock import patch, MagicMock, AsyncMock

import pytest

from wizard.state import WizardPhase
from wizard.wizard import (
    run_extract_phase,
    run_structure_phase,
    run_org_mapping_phase,
    run_mappings_phase,
    run_validate_phase,
    run_migrate_phase,
    run_pipelines_phase,
)
from tests.test_wizard.conftest import wizard_state_at_phase


class TestExtractPhase:
    """Tests for the extract phase handler"""

    @patch('wizard.wizard.confirm_review')
    @patch('execute.execute_plan')
    @patch('plan.generate_task_plan')
    @patch('plan.get_available_task_configs')
    @patch('logs.configure_logger')
    @patch('operations.http_request.configure_client')
    @patch('operations.http_request.get_server_details')
    @patch('operations.http_request.configure_client_cert')
    @patch('wizard.wizard.prompt_credentials')
    @patch('wizard.wizard.prompt_url')
    def test_extract_success_without_cert(
        self,
        mock_prompt_url,
        mock_prompt_credentials,
        mock_configure_cert,
        mock_get_server_details,
        mock_configure_client,
        mock_configure_logger,
        mock_get_configs,
        mock_generate_plan,
        mock_execute_plan,
        mock_confirm_review,
        temp_export_dir
    ):
        """Test successful extract — cert prompt no longer appears upfront"""
        # Setup mocks
        mock_confirm_review.return_value = True
        mock_prompt_url.return_value = "https://sonar.example.com/"
        mock_prompt_credentials.return_value = "test-token"
        mock_configure_cert.return_value = None
        mock_get_server_details.return_value = ("9.9.0.1234", "enterprise")
        mock_get_configs.return_value = {
            "getProjects": {},
            "getQualityProfiles": {},
            "createProject": {}
        }
        mock_generate_plan.return_value = [["getProjects"], ["getQualityProfiles"]]

        state = wizard_state_at_phase(WizardPhase.EXTRACT)
        result = run_extract_phase(state, temp_export_dir)

        assert result.phase == WizardPhase.STRUCTURE
        assert result.source_url == "https://sonar.example.com/"
        assert result.extract_id is not None
        mock_execute_plan.assert_called_once()

    @patch('wizard.wizard.confirm_review')
    @patch('execute.execute_plan')
    @patch('plan.generate_task_plan')
    @patch('plan.get_available_task_configs')
    @patch('logs.configure_logger')
    @patch('operations.http_request.configure_client')
    @patch('operations.http_request.get_server_details')
    @patch('operations.http_request.configure_client_cert')
    @patch('wizard.wizard.prompt_text')
    @patch('wizard.wizard.confirm_action')
    @patch('wizard.wizard.prompt_credentials')
    @patch('wizard.wizard.prompt_url')
    def test_extract_with_client_certificate(
        self,
        mock_prompt_url,
        mock_prompt_credentials,
        mock_confirm_action,
        mock_prompt_text,
        mock_configure_cert,
        mock_get_server_details,
        mock_configure_client,
        mock_configure_logger,
        mock_get_configs,
        mock_generate_plan,
        mock_execute_plan,
        mock_confirm_review,
        temp_export_dir
    ):
        """Test extract: SSL error triggers cert prompt, retry with cert succeeds"""
        import ssl

        mock_confirm_review.return_value = True
        mock_prompt_url.return_value = "https://sonar.example.com/"
        # Calls: token (iter 1), cert password (SSL handler), token (iter 2)
        mock_prompt_credentials.side_effect = ["test-token", "cert-password", "test-token"]
        mock_confirm_action.return_value = True  # Provide cert when prompted
        mock_prompt_text.side_effect = ["/path/to/cert.pem", "/path/to/key.pem"]
        mock_cert = MagicMock()
        mock_configure_cert.side_effect = [None, mock_cert]
        # First attempt raises an ssl.SSLError (httpx wraps these in ConnectError);
        # raising directly here exercises the chain-walk detection code
        mock_get_server_details.side_effect = [
            ssl.SSLError("SSL handshake failed"),
            ("9.9.0.1234", "enterprise"),
        ]
        mock_get_configs.return_value = {"getProjects": {}}
        mock_generate_plan.return_value = [["getProjects"]]

        state = wizard_state_at_phase(WizardPhase.EXTRACT)
        result = run_extract_phase(state, temp_export_dir)

        assert result.phase == WizardPhase.STRUCTURE
        mock_configure_cert.assert_called_with(
            "/path/to/key.pem", "/path/to/cert.pem", "cert-password"
        )

    @patch('wizard.wizard.confirm_review')
    @patch('operations.http_request.configure_client_cert')
    @patch('wizard.wizard.confirm_action')
    @patch('wizard.wizard.prompt_credentials')
    @patch('wizard.wizard.prompt_url')
    def test_extract_server_connection_failure(
        self,
        mock_prompt_url,
        mock_prompt_credentials,
        mock_confirm_action,
        mock_configure_cert,
        mock_confirm_review,
        temp_export_dir
    ):
        """Test extract fails gracefully on server connection error"""
        mock_confirm_review.return_value = True
        mock_prompt_url.return_value = "https://sonar.example.com/"
        mock_prompt_credentials.return_value = "test-token"
        mock_confirm_action.return_value = False
        mock_configure_cert.side_effect = Exception("Connection refused")

        state = wizard_state_at_phase(WizardPhase.EXTRACT)

        with pytest.raises(Exception, match="Connection refused"):
            run_extract_phase(state, temp_export_dir)

    @patch('wizard.wizard.confirm_review')
    @patch('operations.http_request.configure_client_cert')
    @patch('operations.http_request.get_server_details')
    @patch('wizard.wizard.confirm_action')
    @patch('wizard.wizard.prompt_credentials')
    @patch('wizard.wizard.prompt_url')
    def test_extract_invalid_token(
        self,
        mock_prompt_url,
        mock_prompt_credentials,
        mock_confirm_action,
        mock_get_server_details,
        mock_configure_cert,
        mock_confirm_review,
        temp_export_dir
    ):
        """Test extract fails on invalid token (401)"""
        mock_confirm_review.return_value = True
        mock_prompt_url.return_value = "https://sonar.example.com/"
        mock_prompt_credentials.return_value = "invalid-token"
        mock_confirm_action.return_value = False
        mock_configure_cert.return_value = None
        mock_get_server_details.side_effect = Exception("HTTP 401 Unauthorized")

        state = wizard_state_at_phase(WizardPhase.EXTRACT)

        with pytest.raises(Exception, match="401"):
            run_extract_phase(state, temp_export_dir)

    @patch('wizard.wizard.confirm_review')
    @patch('execute.execute_plan')
    @patch('plan.generate_task_plan')
    @patch('plan.get_available_task_configs')
    @patch('logs.configure_logger')
    @patch('operations.http_request.configure_client')
    @patch('operations.http_request.get_server_details')
    @patch('operations.http_request.configure_client_cert')
    @patch('wizard.wizard.confirm_action')
    @patch('wizard.wizard.prompt_credentials')
    def test_extract_uses_existing_source_url(
        self,
        mock_prompt_credentials,
        mock_confirm_action,
        mock_configure_cert,
        mock_get_server_details,
        mock_configure_client,
        mock_configure_logger,
        mock_get_configs,
        mock_generate_plan,
        mock_execute_plan,
        mock_confirm_review,
        temp_export_dir
    ):
        """Test that existing source_url in state is used"""
        mock_confirm_review.return_value = True
        mock_prompt_credentials.return_value = "test-token"
        mock_confirm_action.return_value = False
        mock_configure_cert.return_value = None
        mock_get_server_details.return_value = ("9.9.0.1234", "enterprise")
        mock_get_configs.return_value = {"getProjects": {}}
        mock_generate_plan.return_value = [["getProjects"]]

        state = wizard_state_at_phase(
            WizardPhase.EXTRACT,
            source_url="https://existing.example.com/"
        )
        result = run_extract_phase(state, temp_export_dir)

        assert result.source_url == "https://existing.example.com/"


class TestStructurePhase:
    """Tests for the structure phase handler"""

    @patch('utils.export_csv')
    @patch('structure.map_organization_structure')
    @patch('structure.map_project_structure')
    @patch('utils.get_unique_extracts')
    def test_structure_success(
        self,
        mock_get_extracts,
        mock_map_projects,
        mock_map_orgs,
        mock_export_csv,
        temp_export_dir
    ):
        """Test successful structure analysis"""
        mock_get_extracts.return_value = {"https://sonar.example.com/": "1234567890"}
        mock_map_projects.return_value = (
            [{"binding": "test"}],
            [{"key": "project-1", "name": "Project One"}]
        )
        mock_map_orgs.return_value = [{"sonarqube_org_key": "org-1", "project_count": 1}]

        state = wizard_state_at_phase(WizardPhase.STRUCTURE)
        result = run_structure_phase(state, temp_export_dir)

        assert result.phase == WizardPhase.ORG_MAPPING
        assert mock_export_csv.call_count == 2

    @patch('utils.get_unique_extracts')
    def test_structure_no_extracts_found(self, mock_get_extracts, temp_export_dir):
        """Test structure fails when no extracts exist"""
        mock_get_extracts.return_value = {}

        state = wizard_state_at_phase(WizardPhase.STRUCTURE)

        with pytest.raises(ValueError, match="No extracts found"):
            run_structure_phase(state, temp_export_dir)

    @patch('utils.export_csv')
    @patch('structure.map_organization_structure')
    @patch('structure.map_project_structure')
    @patch('utils.get_unique_extracts')
    def test_structure_empty_results(
        self,
        mock_get_extracts,
        mock_map_projects,
        mock_map_orgs,
        mock_export_csv,
        temp_export_dir
    ):
        """Test structure handles empty mapping results"""
        mock_get_extracts.return_value = {"https://sonar.example.com/": "1234567890"}
        mock_map_projects.return_value = ([], [])
        mock_map_orgs.return_value = []

        state = wizard_state_at_phase(WizardPhase.STRUCTURE)
        result = run_structure_phase(state, temp_export_dir)

        assert result.phase == WizardPhase.ORG_MAPPING


class TestOrgMappingPhase:
    """Tests for the org mapping phase handler"""

    @patch('wizard.wizard.confirm_review')
    @patch('utils.export_csv')
    @patch('utils.load_csv')
    @patch('wizard.wizard.confirm_action')
    @patch('wizard.wizard.prompt_text')
    @patch('wizard.wizard.prompt_url')
    def test_org_mapping_success(
        self,
        mock_prompt_url,
        mock_prompt_text,
        mock_confirm_action,
        mock_load_csv,
        mock_export_csv,
        mock_confirm_review,
        temp_export_dir,
        sample_organizations_csv
    ):
        """Test successful organization mapping"""
        mock_confirm_review.return_value = True
        mock_prompt_url.return_value = "https://sonarcloud.io/"
        mock_prompt_text.side_effect = ["enterprise-key", "mapped-org-1"]
        mock_confirm_action.return_value = False  # Don't skip org
        mock_load_csv.return_value = sample_organizations_csv

        # Create state without pre-set target_url and enterprise_key
        # so that the prompts will be called
        state = wizard_state_at_phase(
            WizardPhase.ORG_MAPPING,
            target_url=None,
            enterprise_key=None
        )
        result = run_org_mapping_phase(state, temp_export_dir)

        assert result.phase == WizardPhase.MAPPINGS
        assert result.organizations_mapped is True
        assert result.target_url == "https://sonarcloud.io/"
        assert result.enterprise_key == "enterprise-key"

    @patch('wizard.wizard.confirm_review')
    @patch('utils.load_csv')
    @patch('wizard.wizard.confirm_action')
    @patch('wizard.wizard.prompt_text')
    @patch('wizard.wizard.prompt_url')
    def test_org_mapping_no_organizations(
        self,
        mock_prompt_url,
        mock_prompt_text,
        mock_confirm_action,
        mock_load_csv,
        mock_confirm_review,
        temp_export_dir
    ):
        """Test org mapping fails when no organizations exist"""
        mock_confirm_review.return_value = True
        mock_prompt_url.return_value = "https://sonarcloud.io/"
        mock_prompt_text.return_value = "enterprise-key"
        mock_confirm_action.return_value = False  # Decline retry
        mock_load_csv.return_value = []

        state = wizard_state_at_phase(WizardPhase.ORG_MAPPING)

        with pytest.raises(ValueError, match="No organizations found"):
            run_org_mapping_phase(state, temp_export_dir)

    @patch('wizard.wizard.confirm_review')
    @patch('utils.export_csv')
    @patch('utils.load_csv')
    @patch('wizard.wizard.prompt_text')
    @patch('wizard.wizard.prompt_url')
    def test_org_mapping_already_mapped(
        self,
        mock_prompt_url,
        mock_prompt_text,
        mock_load_csv,
        mock_export_csv,
        mock_confirm_review,
        temp_export_dir,
        sample_organizations_mapped
    ):
        """Test org mapping when all organizations are already mapped"""
        mock_confirm_review.return_value = True
        mock_prompt_url.return_value = "https://sonarcloud.io/"
        mock_prompt_text.return_value = "enterprise-key"
        mock_load_csv.return_value = sample_organizations_mapped

        state = wizard_state_at_phase(WizardPhase.ORG_MAPPING)
        result = run_org_mapping_phase(state, temp_export_dir)

        assert result.phase == WizardPhase.MAPPINGS
        assert result.organizations_mapped is True
        # Should not export if nothing changed
        mock_export_csv.assert_not_called()

    @patch('wizard.wizard.confirm_review')
    @patch('utils.export_csv')
    @patch('utils.load_csv')
    @patch('wizard.wizard.prompt_text')
    def test_org_mapping_uses_existing_target_url(
        self,
        mock_prompt_text,
        mock_load_csv,
        mock_export_csv,
        mock_confirm_review,
        temp_export_dir,
        sample_organizations_mapped
    ):
        """Test that existing target_url in state is used"""
        mock_confirm_review.return_value = True
        mock_prompt_text.return_value = "enterprise-key"
        mock_load_csv.return_value = sample_organizations_mapped

        state = wizard_state_at_phase(
            WizardPhase.ORG_MAPPING,
            target_url="https://existing.sonarcloud.io/"
        )
        result = run_org_mapping_phase(state, temp_export_dir)

        assert result.target_url == "https://existing.sonarcloud.io/"


class TestMappingsPhase:
    """Tests for the mappings phase handler"""

    @patch('utils.export_csv')
    @patch('structure.map_groups')
    @patch('structure.map_portfolios')
    @patch('structure.map_gates')
    @patch('structure.map_profiles')
    @patch('structure.map_templates')
    @patch('utils.load_csv')
    @patch('utils.get_unique_extracts')
    def test_mappings_success(
        self,
        mock_get_extracts,
        mock_load_csv,
        mock_map_templates,
        mock_map_profiles,
        mock_map_gates,
        mock_map_portfolios,
        mock_map_groups,
        mock_export_csv,
        temp_export_dir,
        sample_projects_csv
    ):
        """Test successful mappings generation"""
        mock_get_extracts.return_value = {"https://sonar.example.com/": "1234567890"}
        mock_load_csv.return_value = sample_projects_csv
        mock_map_templates.return_value = [{"name": "template-1"}]
        mock_map_profiles.return_value = [{"name": "profile-1"}]
        mock_map_gates.return_value = [{"name": "gate-1"}]
        mock_map_portfolios.return_value = [{"key": "portfolio-1"}]
        mock_map_groups.return_value = [{"name": "group-1"}]

        state = wizard_state_at_phase(WizardPhase.MAPPINGS)
        result = run_mappings_phase(state, temp_export_dir)

        assert result.phase == WizardPhase.VALIDATE
        assert mock_export_csv.call_count == 5  # One for each mapping type

    @patch('structure.map_templates')
    @patch('utils.load_csv')
    @patch('utils.get_unique_extracts')
    def test_mappings_failure_propagates(
        self,
        mock_get_extracts,
        mock_load_csv,
        mock_map_templates,
        temp_export_dir,
        sample_projects_csv
    ):
        """Test that mapping failures propagate correctly"""
        mock_get_extracts.return_value = {"https://sonar.example.com/": "1234567890"}
        mock_load_csv.return_value = sample_projects_csv
        mock_map_templates.side_effect = Exception("Template mapping failed")

        state = wizard_state_at_phase(WizardPhase.MAPPINGS)

        with pytest.raises(Exception, match="Template mapping failed"):
            run_mappings_phase(state, temp_export_dir)

    @patch('utils.export_csv')
    @patch('structure.map_groups')
    @patch('structure.map_portfolios')
    @patch('structure.map_gates')
    @patch('structure.map_profiles')
    @patch('structure.map_templates')
    @patch('utils.load_csv')
    @patch('utils.get_unique_extracts')
    def test_mappings_empty_results(
        self,
        mock_get_extracts,
        mock_load_csv,
        mock_map_templates,
        mock_map_profiles,
        mock_map_gates,
        mock_map_portfolios,
        mock_map_groups,
        mock_export_csv,
        temp_export_dir,
        sample_projects_csv
    ):
        """Test mappings handles empty results"""
        mock_get_extracts.return_value = {"https://sonar.example.com/": "1234567890"}
        mock_load_csv.return_value = sample_projects_csv
        mock_map_templates.return_value = []
        mock_map_profiles.return_value = []
        mock_map_gates.return_value = []
        mock_map_portfolios.return_value = []
        mock_map_groups.return_value = []

        state = wizard_state_at_phase(WizardPhase.MAPPINGS)
        result = run_mappings_phase(state, temp_export_dir)

        assert result.phase == WizardPhase.VALIDATE


class TestValidatePhase:
    """Tests for the validate phase handler"""

    @patch('utils.load_csv')
    def test_validate_success(
        self,
        mock_load_csv,
        export_dir_with_mappings,
        sample_organizations_mapped,
        sample_projects_csv,
        sample_profiles_csv,
        sample_templates_csv,
        sample_gates_csv,
        sample_groups_csv
    ):
        """Test successful validation"""
        mock_load_csv.side_effect = [
            sample_organizations_mapped,
            sample_projects_csv,
            sample_profiles_csv,
            sample_templates_csv,
            sample_gates_csv,
            sample_groups_csv
        ]

        state = wizard_state_at_phase(WizardPhase.VALIDATE)
        result = run_validate_phase(state, export_dir_with_mappings)

        assert result.phase == WizardPhase.MIGRATE
        assert result.validation_passed is True

    def test_validate_missing_files(self, temp_export_dir):
        """Test validation fails when required files are missing"""
        state = wizard_state_at_phase(WizardPhase.VALIDATE)

        with pytest.raises(ValueError, match="Missing required files"):
            run_validate_phase(state, temp_export_dir)

    @patch('utils.load_csv')
    def test_validate_unmapped_organizations(
        self,
        mock_load_csv,
        export_dir_with_mappings,
        sample_organizations_csv  # Has unmapped orgs
    ):
        """Test validation fails when organizations are not mapped"""
        mock_load_csv.return_value = sample_organizations_csv

        state = wizard_state_at_phase(WizardPhase.VALIDATE)

        with pytest.raises(ValueError, match="Unmapped organizations"):
            run_validate_phase(state, export_dir_with_mappings)

    @patch('utils.load_csv')
    def test_validate_empty_csv_files(
        self,
        mock_load_csv,
        export_dir_with_mappings
    ):
        """Test validation handles empty CSV files"""
        mock_load_csv.side_effect = [
            [],  # Empty organizations
            [], [], [], [], []
        ]

        state = wizard_state_at_phase(WizardPhase.VALIDATE)

        # Empty organizations should not raise unmapped error
        result = run_validate_phase(state, export_dir_with_mappings)
        assert result.phase == WizardPhase.MIGRATE


class TestMigratePhase:
    """Tests for the migrate phase handler"""

    @patch('execute.execute_plan')
    @patch('utils.filter_completed')
    @patch('plan.generate_task_plan')
    @patch('validate.validate_migration')
    @patch('utils.get_unique_extracts')
    @patch('plan.get_available_task_configs')
    @patch('logs.configure_logger')
    @patch('operations.http_request.configure_client')
    @patch('wizard.wizard.prompt_credentials')
    @patch('wizard.wizard.confirm_action')
    def test_migrate_success(
        self,
        mock_confirm,
        mock_prompt_creds,
        mock_configure_client,
        mock_configure_logger,
        mock_get_configs,
        mock_get_extracts,
        mock_validate_migration,
        mock_generate_plan,
        mock_filter_completed,
        mock_execute_plan,
        temp_export_dir
    ):
        """Test successful migration"""
        mock_confirm.return_value = True
        mock_prompt_creds.return_value = "cloud-token"
        mock_get_configs.return_value = {"createProject": {}, "getProjects": {}}
        mock_get_extracts.return_value = {"https://sonar.example.com/": "1234567890"}
        mock_validate_migration.return_value = (temp_export_dir, set())
        mock_generate_plan.return_value = [["createProject"]]
        mock_filter_completed.return_value = [["createProject"]]

        state = wizard_state_at_phase(WizardPhase.MIGRATE)
        result = run_migrate_phase(state, temp_export_dir)

        assert result.phase == WizardPhase.PIPELINES
        assert result.migration_run_id is not None
        mock_execute_plan.assert_called_once()

    @patch('wizard.wizard.confirm_action')
    def test_migrate_user_cancels(self, mock_confirm, temp_export_dir):
        """Test migration cancelled by user exits cleanly"""
        mock_confirm.return_value = False

        state = wizard_state_at_phase(WizardPhase.MIGRATE)
        with pytest.raises(SystemExit) as exc_info:
            run_migrate_phase(state, temp_export_dir)

        assert exc_info.value.code == 0

    @patch('operations.http_request.configure_client')
    @patch('wizard.wizard.prompt_credentials')
    @patch('wizard.wizard.confirm_action')
    def test_migrate_cloud_token_error(
        self,
        mock_confirm,
        mock_prompt_creds,
        mock_configure_client,
        temp_export_dir
    ):
        """Test migration fails on invalid cloud token"""
        mock_confirm.side_effect = [True, False]  # Proceed, then decline retry
        mock_prompt_creds.return_value = "invalid-token"
        mock_configure_client.side_effect = Exception("Authentication failed")

        state = wizard_state_at_phase(WizardPhase.MIGRATE)

        with pytest.raises(Exception, match="Authentication failed"):
            run_migrate_phase(state, temp_export_dir)

    @patch('execute.execute_plan')
    @patch('utils.filter_completed')
    @patch('plan.generate_task_plan')
    @patch('validate.validate_migration')
    @patch('utils.get_unique_extracts')
    @patch('plan.get_available_task_configs')
    @patch('logs.configure_logger')
    @patch('operations.http_request.configure_client')
    @patch('wizard.wizard.prompt_credentials')
    @patch('wizard.wizard.confirm_action')
    def test_migrate_execution_error(
        self,
        mock_confirm,
        mock_prompt_creds,
        mock_configure_client,
        mock_configure_logger,
        mock_get_configs,
        mock_get_extracts,
        mock_validate_migration,
        mock_generate_plan,
        mock_filter_completed,
        mock_execute_plan,
        temp_export_dir
    ):
        """Test migration handles execution errors"""
        mock_confirm.side_effect = [True, False]  # Proceed, then decline retry
        mock_prompt_creds.return_value = "cloud-token"
        mock_get_configs.return_value = {"createProject": {}}
        mock_get_extracts.return_value = {"https://sonar.example.com/": "1234567890"}
        mock_validate_migration.return_value = (temp_export_dir, set())
        mock_generate_plan.return_value = [["createProject"]]
        mock_filter_completed.return_value = [["createProject"]]
        mock_execute_plan.side_effect = Exception("Migration execution failed")

        state = wizard_state_at_phase(WizardPhase.MIGRATE)

        with pytest.raises(Exception, match="Migration execution failed"):
            run_migrate_phase(state, temp_export_dir)


class TestPipelinesPhase:
    """Tests for the pipelines phase handler"""

    @patch('wizard.wizard.confirm_action')
    def test_pipelines_skip_no_secrets(
        self,
        mock_confirm,
        temp_export_dir
    ):
        """Test pipelines phase skips when no secrets.json exists"""
        mock_confirm.return_value = True  # Confirm skip

        state = wizard_state_at_phase(WizardPhase.PIPELINES)
        result = run_pipelines_phase(state, temp_export_dir)

        assert result.phase == WizardPhase.COMPLETE

    @patch('wizard.wizard.confirm_action')
    def test_pipelines_user_declines(
        self,
        mock_confirm,
        temp_export_dir,
        sample_secrets_json
    ):
        """Test pipelines phase when user declines updates"""
        # Create secrets file
        with open(os.path.join(temp_export_dir, "secrets.json"), "wt") as f:
            json.dump(sample_secrets_json, f)

        mock_confirm.return_value = False  # Decline pipeline updates

        state = wizard_state_at_phase(WizardPhase.PIPELINES)
        result = run_pipelines_phase(state, temp_export_dir)

        assert result.phase == WizardPhase.COMPLETE

    @patch('pipelines.process.update_pipelines', new_callable=AsyncMock)
    @patch('logs.configure_logger')
    @patch('utils.get_unique_extracts')
    @patch('wizard.wizard.prompt_credentials')
    @patch('wizard.wizard.confirm_action')
    def test_pipelines_success(
        self,
        mock_confirm,
        mock_prompt_creds,
        mock_get_extracts,
        mock_configure_logger,
        mock_update_pipelines,
        temp_export_dir,
        sample_secrets_json
    ):
        """Test successful pipeline updates"""
        # Create secrets file
        with open(os.path.join(temp_export_dir, "secrets.json"), "wt") as f:
            json.dump(sample_secrets_json, f)

        # Create required directory structure
        os.makedirs(os.path.join(temp_export_dir, "generateOrganizationMappings"), exist_ok=True)

        mock_confirm.return_value = True
        mock_prompt_creds.return_value = "pipeline-token"
        mock_get_extracts.return_value = {}

        # Configure AsyncMock to return the expected result
        mock_update_pipelines.return_value = {"repo1": "success"}

        state = wizard_state_at_phase(WizardPhase.PIPELINES)
        result = run_pipelines_phase(state, temp_export_dir)

        assert result.phase == WizardPhase.COMPLETE

    @patch('pipelines.process.update_pipelines', new_callable=AsyncMock)
    @patch('logs.configure_logger')
    @patch('utils.get_unique_extracts')
    @patch('wizard.wizard.prompt_credentials')
    @patch('wizard.wizard.confirm_action')
    def test_pipelines_error_with_continue(
        self,
        mock_confirm,
        mock_prompt_creds,
        mock_get_extracts,
        mock_configure_logger,
        mock_update_pipelines,
        temp_export_dir,
        sample_secrets_json
    ):
        """Test pipeline error with user choosing to continue"""
        # Create secrets file
        with open(os.path.join(temp_export_dir, "secrets.json"), "wt") as f:
            json.dump(sample_secrets_json, f)

        os.makedirs(os.path.join(temp_export_dir, "generateOrganizationMappings"), exist_ok=True)

        mock_confirm.side_effect = [True, True]  # Confirm update, then continue despite error
        mock_prompt_creds.return_value = "pipeline-token"
        mock_get_extracts.return_value = {}

        # Configure AsyncMock to raise exception
        mock_update_pipelines.side_effect = Exception("Pipeline update failed")

        state = wizard_state_at_phase(WizardPhase.PIPELINES)
        result = run_pipelines_phase(state, temp_export_dir)

        # Should complete despite error since user chose to continue
        assert result.phase == WizardPhase.COMPLETE

    @patch('pipelines.process.update_pipelines', new_callable=AsyncMock)
    @patch('logs.configure_logger')
    @patch('utils.get_unique_extracts')
    @patch('wizard.wizard.prompt_credentials')
    @patch('wizard.wizard.confirm_action')
    def test_pipelines_error_without_continue(
        self,
        mock_confirm,
        mock_prompt_creds,
        mock_get_extracts,
        mock_configure_logger,
        mock_update_pipelines,
        temp_export_dir,
        sample_secrets_json
    ):
        """Test pipeline error when user chooses not to continue"""
        # Create secrets file
        with open(os.path.join(temp_export_dir, "secrets.json"), "wt") as f:
            json.dump(sample_secrets_json, f)

        os.makedirs(os.path.join(temp_export_dir, "generateOrganizationMappings"), exist_ok=True)

        mock_confirm.side_effect = [True, False]  # Confirm update, then don't continue
        mock_prompt_creds.return_value = "pipeline-token"
        mock_get_extracts.return_value = {}

        # Configure AsyncMock to raise exception
        mock_update_pipelines.side_effect = Exception("Pipeline update failed")

        state = wizard_state_at_phase(WizardPhase.PIPELINES)

        with pytest.raises(Exception, match="Pipeline update failed"):
            run_pipelines_phase(state, temp_export_dir)


class TestPhaseStatePersistence:
    """Tests verifying state is saved after each phase"""

    @patch('utils.export_csv')
    @patch('structure.map_organization_structure')
    @patch('structure.map_project_structure')
    @patch('utils.get_unique_extracts')
    def test_structure_saves_state(
        self,
        mock_get_extracts,
        mock_map_projects,
        mock_map_orgs,
        mock_export_csv,
        temp_export_dir
    ):
        """Test that structure phase saves state"""
        mock_get_extracts.return_value = {"https://sonar.example.com/": "1234567890"}
        mock_map_projects.return_value = ([], [])
        mock_map_orgs.return_value = []

        state = wizard_state_at_phase(WizardPhase.STRUCTURE)
        run_structure_phase(state, temp_export_dir)

        # Verify state file was created
        state_file = os.path.join(temp_export_dir, ".wizard_state.json")
        assert os.path.exists(state_file)

        with open(state_file, "r") as f:
            saved_state = json.load(f)
        assert saved_state["phase"] == "org_mapping"

    @patch('utils.load_csv')
    def test_validate_saves_state(
        self,
        mock_load_csv,
        export_dir_with_mappings,
        sample_organizations_mapped
    ):
        """Test that validate phase saves state"""
        mock_load_csv.side_effect = [
            sample_organizations_mapped,
            [], [], [], [], []
        ]

        state = wizard_state_at_phase(WizardPhase.VALIDATE)
        run_validate_phase(state, export_dir_with_mappings)

        state_file = os.path.join(export_dir_with_mappings, ".wizard_state.json")
        assert os.path.exists(state_file)

        with open(state_file, "r") as f:
            saved_state = json.load(f)
        assert saved_state["phase"] == "migrate"
        assert saved_state["validation_passed"] is True
