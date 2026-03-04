"""Shared fixtures for wizard tests"""
import json
import os
import tempfile
from typing import Generator, Dict, List, Any

import pytest

from wizard.state import WizardPhase, WizardState

SONARQUBE_URL = "https://sonar.example.com/"


@pytest.fixture
def temp_export_dir() -> Generator[str, None, None]:
    """Create a temporary directory with proper structure for testing"""
    with tempfile.TemporaryDirectory() as tmpdir:
        yield tmpdir


@pytest.fixture
def sample_extract_data() -> Dict[str, Any]:
    """Sample extract.json content"""
    return {
        "plan": [["getProjects", "getQualityProfiles"], ["getQualityGates"]],
        "version": "9.9.0.1234",
        "edition": "enterprise",
        "url": SONARQUBE_URL,
        "target_tasks": ["getProjects", "getQualityProfiles", "getQualityGates"],
        "available_configs": ["getProjects", "getQualityProfiles", "getQualityGates"],
        "run_id": "1234567890"
    }


@pytest.fixture
def sample_organizations_csv() -> List[Dict[str, Any]]:
    """Sample organizations.csv data"""
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
            "sonarcloud_org_key": "mapped-org-2"
        }
    ]


@pytest.fixture
def sample_organizations_mapped() -> List[Dict[str, Any]]:
    """Sample organizations with all mappings complete"""
    return [
        {
            "sonarqube_org_key": "org-1",
            "server_url": SONARQUBE_URL,
            "devops_binding": "github",
            "project_count": 5,
            "sonarcloud_org_key": "cloud-org-1"
        },
        {
            "sonarqube_org_key": "org-2",
            "server_url": SONARQUBE_URL,
            "devops_binding": "gitlab",
            "project_count": 3,
            "sonarcloud_org_key": "cloud-org-2"
        }
    ]


@pytest.fixture
def sample_projects_csv() -> List[Dict[str, Any]]:
    """Sample projects.csv data"""
    return [
        {
            "key": "project-1",
            "name": "Project One",
            "server_url": SONARQUBE_URL,
            "sonarqube_org_key": "org-1",
            "visibility": "private"
        },
        {
            "key": "project-2",
            "name": "Project Two",
            "server_url": SONARQUBE_URL,
            "sonarqube_org_key": "org-1",
            "visibility": "public"
        },
        {
            "key": "project-3",
            "name": "Project Three",
            "server_url": SONARQUBE_URL,
            "sonarqube_org_key": "org-2",
            "visibility": "private"
        }
    ]


@pytest.fixture
def sample_templates_csv() -> List[Dict[str, Any]]:
    """Sample templates.csv data"""
    return [
        {
            "name": "Default Template",
            "key": "template-1",
            "sonarqube_org_key": "org-1",
            "permissions": ["admin", "codeviewer"]
        },
        {
            "name": "CI Template",
            "key": "template-2",
            "sonarqube_org_key": "org-2",
            "permissions": ["scan"]
        }
    ]


@pytest.fixture
def sample_profiles_csv() -> List[Dict[str, Any]]:
    """Sample profiles.csv data"""
    return [
        {
            "name": "Java Profile",
            "language": "java",
            "key": "profile-java-1",
            "sonarqube_org_key": "org-1",
            "isDefault": True
        },
        {
            "name": "Python Profile",
            "language": "py",
            "key": "profile-py-1",
            "sonarqube_org_key": "org-2",
            "isDefault": False
        }
    ]


@pytest.fixture
def sample_gates_csv() -> List[Dict[str, Any]]:
    """Sample gates.csv data"""
    return [
        {
            "name": "Quality Gate 1",
            "key": "gate-1",
            "sonarqube_org_key": "org-1",
            "isDefault": True,
            "conditions": []
        },
        {
            "name": "Quality Gate 2",
            "key": "gate-2",
            "sonarqube_org_key": "org-2",
            "isDefault": False,
            "conditions": [{"metric": "coverage", "op": "LT", "error": "80"}]
        }
    ]


@pytest.fixture
def sample_groups_csv() -> List[Dict[str, Any]]:
    """Sample groups.csv data"""
    return [
        {
            "name": "developers",
            "sonarqube_org_key": "org-1",
            "members": ["user1", "user2"]
        },
        {
            "name": "admins",
            "sonarqube_org_key": "org-2",
            "members": ["admin1"]
        }
    ]


@pytest.fixture
def sample_portfolios_csv() -> List[Dict[str, Any]]:
    """Sample portfolios.csv data"""
    return [
        {
            "key": "portfolio-1",
            "name": "Portfolio One",
            "sonarqube_org_key": "org-1",
            "projects": ["project-1", "project-2"]
        }
    ]


@pytest.fixture
def sample_secrets_json() -> Dict[str, Any]:
    """Sample secrets.json for pipeline updates"""
    return {
        "org-1": {
            "github_token": "ghp_testtoken123",
            "github_org": "test-org"
        }
    }


def write_csv(directory: str, filename: str, data: List[Dict[str, Any]]) -> str:
    """Helper to write CSV file from dict data"""
    import csv
    filepath = os.path.join(directory, filename)
    if data:
        with open(filepath, 'wt', newline='') as f:
            writer = csv.DictWriter(f, fieldnames=data[0].keys())
            writer.writeheader()
            writer.writerows([
                {k: json.dumps(v) if isinstance(v, (dict, list, bool)) else v
                 for k, v in row.items()}
                for row in data
            ])
    return filepath


@pytest.fixture
def export_dir_with_extract(temp_export_dir, sample_extract_data) -> str:
    """Create export directory with extract data"""
    extract_id = sample_extract_data["run_id"]
    extract_dir = os.path.join(temp_export_dir, extract_id)
    os.makedirs(extract_dir, exist_ok=True)

    # Write extract.json
    with open(os.path.join(extract_dir, "extract.json"), "wt") as f:
        json.dump(sample_extract_data, f)

    return temp_export_dir


@pytest.fixture
def export_dir_with_structure(
    export_dir_with_extract,
    sample_organizations_csv,
    sample_projects_csv
) -> str:
    """Create export directory with structure analysis complete"""
    write_csv(export_dir_with_extract, "organizations.csv", sample_organizations_csv)
    write_csv(export_dir_with_extract, "projects.csv", sample_projects_csv)
    return export_dir_with_extract


@pytest.fixture
def export_dir_with_mappings(
    export_dir_with_structure,
    sample_organizations_mapped,
    sample_templates_csv,
    sample_profiles_csv,
    sample_gates_csv,
    sample_groups_csv,
    sample_portfolios_csv
) -> str:
    """Create export directory with all mappings complete"""
    # Overwrite organizations with mapped version
    write_csv(export_dir_with_structure, "organizations.csv", sample_organizations_mapped)
    write_csv(export_dir_with_structure, "templates.csv", sample_templates_csv)
    write_csv(export_dir_with_structure, "profiles.csv", sample_profiles_csv)
    write_csv(export_dir_with_structure, "gates.csv", sample_gates_csv)
    write_csv(export_dir_with_structure, "groups.csv", sample_groups_csv)
    write_csv(export_dir_with_structure, "portfolios.csv", sample_portfolios_csv)
    return export_dir_with_structure


@pytest.fixture
def export_dir_ready_to_migrate(export_dir_with_mappings) -> str:
    """Create export directory fully ready for migration"""
    return export_dir_with_mappings


def wizard_state_at_phase(phase: WizardPhase, **kwargs) -> WizardState:
    """Factory function to create state at specific phase"""
    defaults = {
        "phase": phase,
        "extract_id": None,
        "source_url": None,
        "target_url": None,
        "enterprise_key": None,
        "organizations_mapped": False,
        "validation_passed": False,
        "migration_run_id": None
    }

    # Set sensible defaults based on phase
    if phase in [WizardPhase.STRUCTURE, WizardPhase.ORG_MAPPING, WizardPhase.MAPPINGS,
                 WizardPhase.VALIDATE, WizardPhase.MIGRATE, WizardPhase.PIPELINES,
                 WizardPhase.COMPLETE]:
        defaults["extract_id"] = "1234567890"
        defaults["source_url"] = SONARQUBE_URL

    if phase in [WizardPhase.ORG_MAPPING, WizardPhase.MAPPINGS, WizardPhase.VALIDATE,
                 WizardPhase.MIGRATE, WizardPhase.PIPELINES, WizardPhase.COMPLETE]:
        defaults["target_url"] = "https://sonarcloud.io/"
        defaults["enterprise_key"] = "test-enterprise"

    if phase in [WizardPhase.MAPPINGS, WizardPhase.VALIDATE, WizardPhase.MIGRATE,
                 WizardPhase.PIPELINES, WizardPhase.COMPLETE]:
        defaults["organizations_mapped"] = True

    if phase in [WizardPhase.MIGRATE, WizardPhase.PIPELINES, WizardPhase.COMPLETE]:
        defaults["validation_passed"] = True

    if phase in [WizardPhase.PIPELINES, WizardPhase.COMPLETE]:
        defaults["migration_run_id"] = "9876543210"

    defaults.update(kwargs)
    return WizardState(**defaults)


@pytest.fixture
def state_at_init() -> WizardState:
    """State at INIT phase"""
    return wizard_state_at_phase(WizardPhase.INIT)


@pytest.fixture
def state_at_extract() -> WizardState:
    """State at EXTRACT phase"""
    return wizard_state_at_phase(WizardPhase.EXTRACT)


@pytest.fixture
def state_at_structure() -> WizardState:
    """State at STRUCTURE phase"""
    return wizard_state_at_phase(WizardPhase.STRUCTURE)


@pytest.fixture
def state_at_org_mapping() -> WizardState:
    """State at ORG_MAPPING phase"""
    return wizard_state_at_phase(WizardPhase.ORG_MAPPING)


@pytest.fixture
def state_at_mappings() -> WizardState:
    """State at MAPPINGS phase"""
    return wizard_state_at_phase(WizardPhase.MAPPINGS)


@pytest.fixture
def state_at_validate() -> WizardState:
    """State at VALIDATE phase"""
    return wizard_state_at_phase(WizardPhase.VALIDATE)


@pytest.fixture
def state_at_migrate() -> WizardState:
    """State at MIGRATE phase"""
    return wizard_state_at_phase(WizardPhase.MIGRATE)


@pytest.fixture
def state_at_pipelines() -> WizardState:
    """State at PIPELINES phase"""
    return wizard_state_at_phase(WizardPhase.PIPELINES)


@pytest.fixture
def state_at_complete() -> WizardState:
    """State at COMPLETE phase"""
    return wizard_state_at_phase(WizardPhase.COMPLETE)


@pytest.fixture
def mock_server_version() -> tuple:
    """Mock server version and edition"""
    return ("9.9.0.1234", "enterprise")


@pytest.fixture
def mock_task_configs() -> Dict[str, Any]:
    """Mock task configurations"""
    return {
        "getProjects": {"type": "get", "endpoint": "/api/projects/search"},
        "getQualityProfiles": {"type": "get", "endpoint": "/api/qualityprofiles/search"},
        "getQualityGates": {"type": "get", "endpoint": "/api/qualitygates/list"},
    }


class MockClickContext:
    """Mock Click context for testing prompts"""
    def __init__(self):
        self.invoked_subcommand = None


@pytest.fixture
def mock_click_context() -> MockClickContext:
    """Mock Click context"""
    return MockClickContext()
