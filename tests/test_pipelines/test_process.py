"""Tests for pipeline processing orchestration"""
import asyncio
import os
import pytest
from unittest.mock import MagicMock, AsyncMock, patch
from ruamel.yaml import YAML


def load_yaml(text):
    return YAML().load(text)


SIMPLE_WORKFLOW_YAML = """
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: SonarSource/sonarcloud-github-action@master
        env:
          SONAR_TOKEN: ${{ secrets.OLD_SONAR_TOKEN }}
          SONAR_HOST_URL: ${{ secrets.OLD_SONAR_HOST_URL }}
"""

NON_SONAR_WORKFLOW_YAML = """
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - run: npm install
"""


class TestUpdatePipelineFile:
    def setup_method(self):
        from pipelines.process import update_pipeline_file
        from pipelines.platforms import github as github_platform
        self.update_pipeline_file = update_pipeline_file
        self.platform = github_platform

    def test_updates_sonar_variables(self):
        file = {
            'file_path': '.github/workflows/sonar.yml',
            'content': SIMPLE_WORKFLOW_YAML,
            'yaml': load_yaml(SIMPLE_WORKFLOW_YAML),
            'sha': 'abc123'
        }
        result_file, _ = self.update_pipeline_file(platform=self.platform, file=file)

        assert result_file['is_updated'] is True
        assert 'SONARQUBE_CLOUD_TOKEN' in result_file['updated_content']
        assert 'SONARQUBE_CLOUD_URL' in result_file['updated_content']

    def test_no_sonar_commands_not_updated(self):
        file = {
            'file_path': '.github/workflows/ci.yml',
            'content': NON_SONAR_WORKFLOW_YAML,
            'yaml': load_yaml(NON_SONAR_WORKFLOW_YAML),
            'sha': 'abc123'
        }
        result_file, _ = self.update_pipeline_file(platform=self.platform, file=file)

        assert result_file['is_updated'] is False

    def test_none_pipeline_type_returns_cleanly(self):
        """Bug fix: None pipeline_type (unresolvable module) should not raise AttributeError"""
        from pipelines.utils import load_module
        from pipelines.pipelines import identify_pipeline_type

        mock_platform = MagicMock()
        # 'nonexistent' causes load_module to return None → identify_pipeline_type returns None
        mock_platform.get_available_pipelines.return_value = ['nonexistent']

        result = identify_pipeline_type(platform=mock_platform, file={})
        assert result is None

        file = {
            'file_path': '.github/workflows/sonar.yml',
            'content': SIMPLE_WORKFLOW_YAML,
            'yaml': load_yaml(SIMPLE_WORKFLOW_YAML),
            'sha': 'abc123'
        }
        result_file, mapping = self.update_pipeline_file(platform=mock_platform, file=file)

        assert result_file['is_updated'] is False
        assert mapping == {}


class TestUpdateConfigFilesEdgeCases:
    def setup_method(self):
        from pipelines.process import update_config_files
        from pipelines.platforms import github as github_platform
        self.update_config_files = update_config_files
        self.platform = github_platform

    def test_empty_project_mappings_returns_empty(self, tmp_path):
        """Bug fix: empty project_mappings should not raise IndexError"""
        async def run():
            return await self.update_config_files(
                platform=self.platform,
                project_mappings={},
                projects=set(),
                root_dir='./',
                scanners={'cli'},
                repository='owner/repo',
                branch={'name': 'test-branch'},
                token='test-token',
                repo_folder=str(tmp_path)
            )

        # With empty project_mappings AND empty projects, early return fires — no HTTP calls
        result = asyncio.run(run())
        assert result == []


class TestCreateOrgSecrets:
    def setup_method(self):
        from pipelines.process import create_org_secrets
        self.create_org_secrets = create_org_secrets

    def test_missing_org_key_raises_key_error(self, tmp_path):
        """Bug fix: missing org key should raise KeyError with useful message"""
        org_data = [{'is_cloud': True, 'sonarcloud_org_key': 'my-org', 'sonarqube_org_key': 'sq-org', 'alm': 'github'}]

        # Write a fake generateOrganizationMappings JSONL file
        import json
        mappings_dir = tmp_path / 'generateOrganizationMappings'
        mappings_dir.mkdir()
        with open(mappings_dir / '0.jsonl', 'wt') as f:
            for item in org_data:
                f.write(json.dumps(item) + '\n')

        async def run():
            return await self.create_org_secrets(
                migration_directory=str(tmp_path),
                org_secret_mapping={},  # missing 'my-org'
                sonar_token='token',
                sonar_url='https://sonarcloud.io'
            )

        with pytest.raises(KeyError, match="my-org"):
            asyncio.run(run())

    def test_non_cloud_orgs_skipped(self, tmp_path):
        """Organizations without is_cloud=True are skipped"""
        import json
        org_data = [{'is_cloud': False, 'sonarcloud_org_key': 'my-org', 'sonarqube_org_key': 'sq-org', 'alm': 'github'}]
        mappings_dir = tmp_path / 'generateOrganizationMappings'
        mappings_dir.mkdir()
        with open(mappings_dir / '0.jsonl', 'wt') as f:
            for item in org_data:
                f.write(json.dumps(item) + '\n')

        async def run():
            return await self.create_org_secrets(
                migration_directory=str(tmp_path),
                org_secret_mapping={},  # would fail if org was processed
                sonar_token='token',
                sonar_url='https://sonarcloud.io'
            )

        orgs, secrets = asyncio.run(run())
        assert orgs == {}
        assert secrets == []
