"""Tests for the skip organization functionality in the wizard"""
from unittest.mock import patch, call

import pytest

from wizard.state import WizardPhase
from wizard.wizard import (
    SKIPPED_ORG_SENTINEL,
    run_org_mapping_phase,
    run_mappings_phase,
    run_validate_phase,
)
from tests.test_wizard.conftest import wizard_state_at_phase, SONARQUBE_URL


@pytest.fixture
def orgs_both_unmapped():
    """Two organizations, both unmapped"""
    return [
        {
            "sonarqube_org_key": "org-1",
            "server_url": SONARQUBE_URL,
            "devops_binding": "github",
            "project_count": 5,
            "sonarcloud_org_key": ""
        },
        {
            "sonarqube_org_key": "org-2",
            "server_url": SONARQUBE_URL,
            "devops_binding": "gitlab",
            "project_count": 3,
            "sonarcloud_org_key": ""
        },
    ]


@pytest.fixture
def orgs_one_skipped_one_mapped():
    """One org skipped, one mapped"""
    return [
        {
            "sonarqube_org_key": "org-1",
            "server_url": SONARQUBE_URL,
            "devops_binding": "github",
            "project_count": 5,
            "sonarcloud_org_key": "SKIPPED"
        },
        {
            "sonarqube_org_key": "org-2",
            "server_url": SONARQUBE_URL,
            "devops_binding": "gitlab",
            "project_count": 3,
            "sonarcloud_org_key": "cloud-org-2"
        },
    ]


@pytest.fixture
def projects_for_two_orgs():
    """Projects belonging to org-1 and org-2"""
    return [
        {"key": "p1", "name": "P1", "server_url": SONARQUBE_URL, "sonarqube_org_key": "org-1", "visibility": "private"},
        {"key": "p2", "name": "P2", "server_url": SONARQUBE_URL, "sonarqube_org_key": "org-1", "visibility": "public"},
        {"key": "p3", "name": "P3", "server_url": SONARQUBE_URL, "sonarqube_org_key": "org-2", "visibility": "private"},
    ]


class TestOrgMappingSkip:
    """Tests for skipping organizations during org mapping phase"""

    @patch('wizard.wizard.confirm_review')
    @patch('utils.export_csv')
    @patch('utils.load_csv')
    @patch('wizard.wizard.confirm_action')
    @patch('wizard.wizard.prompt_text')
    @patch('wizard.wizard.prompt_url')
    def test_skip_org_sets_sentinel(
        self,
        mock_prompt_url,
        mock_prompt_text,
        mock_confirm_action,
        mock_load_csv,
        mock_export_csv,
        mock_confirm_review,
        temp_export_dir,
        orgs_both_unmapped,
    ):
        """Skipping an org sets the SKIPPED sentinel value"""
        mock_confirm_review.return_value = True
        mock_prompt_url.return_value = "https://sonarcloud.io/"
        mock_prompt_text.side_effect = ["enterprise-key", "cloud-org-2"]
        # First confirm_action: migrate org-1? No. Second: migrate org-2? Yes.
        mock_confirm_action.side_effect = [False, True]
        mock_load_csv.return_value = orgs_both_unmapped

        state = wizard_state_at_phase(
            WizardPhase.ORG_MAPPING, target_url=None, enterprise_key=None
        )
        result = run_org_mapping_phase(state, temp_export_dir)

        assert result.phase == WizardPhase.MAPPINGS
        # Check the data passed to export_csv
        saved_orgs = mock_export_csv.call_args[1].get('data') or mock_export_csv.call_args[0][2] if len(mock_export_csv.call_args[0]) > 2 else mock_export_csv.call_args[1]['data']
        org1 = next(o for o in saved_orgs if o['sonarqube_org_key'] == 'org-1')
        org2 = next(o for o in saved_orgs if o['sonarqube_org_key'] == 'org-2')
        assert org1['sonarcloud_org_key'] == SKIPPED_ORG_SENTINEL
        assert org2['sonarcloud_org_key'] == 'cloud-org-2'

    @patch('wizard.wizard.confirm_review')
    @patch('utils.export_csv')
    @patch('utils.load_csv')
    @patch('wizard.wizard.confirm_action')
    @patch('wizard.wizard.prompt_text')
    @patch('wizard.wizard.prompt_url')
    def test_mixed_skip_and_map(
        self,
        mock_prompt_url,
        mock_prompt_text,
        mock_confirm_action,
        mock_load_csv,
        mock_export_csv,
        mock_confirm_review,
        temp_export_dir,
        orgs_both_unmapped,
    ):
        """User skips one org and maps another"""
        mock_confirm_review.return_value = True
        mock_prompt_url.return_value = "https://sonarcloud.io/"
        # prompt_text: enterprise key, then cloud key for org-2
        mock_prompt_text.side_effect = ["enterprise-key", "cloud-org-2"]
        # Migrate org-1? No; Migrate org-2? Yes
        mock_confirm_action.side_effect = [False, True]
        mock_load_csv.return_value = orgs_both_unmapped

        state = wizard_state_at_phase(
            WizardPhase.ORG_MAPPING, target_url=None, enterprise_key=None
        )
        result = run_org_mapping_phase(state, temp_export_dir)

        assert result.organizations_mapped is True
        saved_orgs = mock_export_csv.call_args.kwargs['data']
        skipped = [o for o in saved_orgs if o['sonarcloud_org_key'] == SKIPPED_ORG_SENTINEL]
        mapped = [o for o in saved_orgs if o['sonarcloud_org_key'] not in ('', SKIPPED_ORG_SENTINEL)]
        assert len(skipped) == 1
        assert len(mapped) == 1

    @patch('wizard.wizard.confirm_review')
    @patch('utils.export_csv')
    @patch('utils.load_csv')
    @patch('wizard.wizard.display_message')
    @patch('wizard.wizard.prompt_text')
    @patch('wizard.wizard.prompt_url')
    def test_already_skipped_org_shown_on_resume(
        self,
        mock_prompt_url,
        mock_prompt_text,
        mock_display_message,
        mock_load_csv,
        mock_export_csv,
        mock_confirm_review,
        temp_export_dir,
        orgs_one_skipped_one_mapped,
    ):
        """On resume, already-skipped orgs display as SKIPPED without re-prompting"""
        mock_confirm_review.return_value = True
        mock_prompt_url.return_value = "https://sonarcloud.io/"
        mock_prompt_text.return_value = "enterprise-key"
        mock_load_csv.return_value = orgs_one_skipped_one_mapped

        state = wizard_state_at_phase(WizardPhase.ORG_MAPPING)
        result = run_org_mapping_phase(state, temp_export_dir)

        assert result.phase == WizardPhase.MAPPINGS
        # All orgs are mapped/skipped, so "All organizations are already mapped." is shown
        mock_display_message.assert_any_call("All organizations are already mapped.")
        # export_csv should not be called since nothing changed
        mock_export_csv.assert_not_called()


class TestMappingsPhaseSkip:
    """Tests for mappings phase filtering of skipped orgs"""

    @patch('utils.export_csv')
    @patch('structure.map_groups')
    @patch('structure.map_portfolios')
    @patch('structure.map_gates')
    @patch('structure.map_profiles')
    @patch('structure.map_templates')
    @patch('utils.load_csv')
    @patch('utils.get_unique_extracts')
    def test_mappings_excludes_skipped_org_projects(
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
        orgs_one_skipped_one_mapped,
        projects_for_two_orgs,
    ):
        """Projects from skipped orgs are excluded from entity mappings"""
        mock_get_extracts.return_value = {SONARQUBE_URL: "1234567890"}
        # load_csv is called twice: first for organizations, then for projects
        mock_load_csv.side_effect = [orgs_one_skipped_one_mapped, projects_for_two_orgs]
        mock_map_templates.return_value = []
        mock_map_profiles.return_value = []
        mock_map_gates.return_value = []
        mock_map_portfolios.return_value = []
        mock_map_groups.return_value = []

        state = wizard_state_at_phase(WizardPhase.MAPPINGS)
        run_mappings_phase(state, temp_export_dir)

        # Check that project_org_mapping passed to map_templates only has org-2 projects
        templates_call_kwargs = mock_map_templates.call_args.kwargs
        project_org_mapping = templates_call_kwargs['project_org_mapping']
        # org-1 projects (p1, p2) should be excluded; only p3 from org-2 remains
        assert len(project_org_mapping) == 1
        assert SONARQUBE_URL + "p3" in project_org_mapping
        assert SONARQUBE_URL + "p1" not in project_org_mapping

    @patch('utils.export_csv')
    @patch('structure.map_groups')
    @patch('structure.map_portfolios')
    @patch('structure.map_gates')
    @patch('structure.map_profiles')
    @patch('structure.map_templates')
    @patch('utils.load_csv')
    @patch('utils.get_unique_extracts')
    def test_mappings_no_skipped_orgs_passes_all_projects(
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
        projects_for_two_orgs,
    ):
        """When no orgs are skipped, all projects are included"""
        all_mapped_orgs = [
            {"sonarqube_org_key": "org-1", "sonarcloud_org_key": "cloud-org-1"},
            {"sonarqube_org_key": "org-2", "sonarcloud_org_key": "cloud-org-2"},
        ]
        mock_get_extracts.return_value = {SONARQUBE_URL: "1234567890"}
        mock_load_csv.side_effect = [all_mapped_orgs, projects_for_two_orgs]
        mock_map_templates.return_value = []
        mock_map_profiles.return_value = []
        mock_map_gates.return_value = []
        mock_map_portfolios.return_value = []
        mock_map_groups.return_value = []

        state = wizard_state_at_phase(WizardPhase.MAPPINGS)
        run_mappings_phase(state, temp_export_dir)

        templates_call_kwargs = mock_map_templates.call_args.kwargs
        project_org_mapping = templates_call_kwargs['project_org_mapping']
        assert len(project_org_mapping) == 3


class TestValidatePhaseSkip:
    """Tests for validate phase with skipped orgs"""

    @patch('utils.load_csv')
    def test_validate_passes_with_skipped_orgs(
        self,
        mock_load_csv,
        export_dir_with_mappings,
        orgs_one_skipped_one_mapped,
        projects_for_two_orgs,
    ):
        """Validation passes when some orgs are skipped (sentinel is truthy)"""
        mock_load_csv.side_effect = [
            orgs_one_skipped_one_mapped,
            projects_for_two_orgs,
            [],  # profiles
            [],  # templates
            [],  # gates
            [],  # groups
        ]

        state = wizard_state_at_phase(WizardPhase.VALIDATE)
        result = run_validate_phase(state, export_dir_with_mappings)

        assert result.phase == WizardPhase.MIGRATE
        assert result.validation_passed is True

    @patch('wizard.wizard.display_summary')
    @patch('utils.load_csv')
    def test_validate_summary_shows_active_and_skipped_counts(
        self,
        mock_load_csv,
        mock_display_summary,
        export_dir_with_mappings,
        orgs_one_skipped_one_mapped,
        projects_for_two_orgs,
    ):
        """Migration summary distinguishes active vs skipped orgs and projects"""
        mock_load_csv.side_effect = [
            orgs_one_skipped_one_mapped,
            projects_for_two_orgs,
            [],  # profiles
            [],  # templates
            [],  # gates
            [],  # groups
        ]

        state = wizard_state_at_phase(WizardPhase.VALIDATE)
        run_validate_phase(state, export_dir_with_mappings)

        summary_call = mock_display_summary.call_args
        summary_data = summary_call[0][1]
        assert "1 active" in summary_data["Organizations"]
        assert "1 skipped" in summary_data["Organizations"]
        assert "1 active" in summary_data["Projects"]
        assert "2 skipped" in summary_data["Projects"]
