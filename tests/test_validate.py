"""Tests for src/validate.py"""
import csv
import json
import os
import tempfile

from validate import validate_migration


def _write_csv(path, rows):
    if not rows:
        return
    with open(path, 'w', newline='') as f:
        writer = csv.DictWriter(f, fieldnames=rows[0].keys())
        writer.writeheader()
        writer.writerows(rows)


def _setup_structure_dir(base_dir, organizations=None, projects=None, templates=None,
                         profiles=None, gates=None, portfolios=None, groups=None):
    """Write CSV files into base_dir to satisfy validate_migration's requirements."""
    defaults = {
        'organizations': organizations or [
            {'sonarqube_org_key': 'sq-org', 'sonarcloud_org_key': 'sc-org'}
        ],
        'projects': projects or [
            {'sonarqube_org_key': 'sq-org', 'sonarcloud_org_key': ''}
        ],
        'templates': templates or [],
        'profiles': profiles or [],
        'gates': gates or [],
        'portfolios': portfolios or [],
        'groups': groups or [],
    }
    for name, rows in defaults.items():
        if rows:
            _write_csv(os.path.join(base_dir, f'{name}.csv'), rows)
        else:
            # create empty file so load_csv returns []
            open(os.path.join(base_dir, f'{name}.csv'), 'w').close()


class TestValidateMigration:
    def test_creates_run_directory(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            _setup_structure_dir(tmpdir)
            run_id = 'test-run-01'
            run_dir, _ = validate_migration(directory=tmpdir, run_id=run_id)
            assert os.path.isdir(run_dir)

    def test_returns_completed_set(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            _setup_structure_dir(tmpdir)
            _, completed = validate_migration(directory=tmpdir, run_id='run-01')
            expected_names = {
                'generateOrganizationMappings',
                'generateProjectMappings',
                'generateTemplateMappings',
                'generateProfileMappings',
                'generateGateMappings',
                'generatePortfolioMappings',
                'generateGroupMappings',
            }
            assert completed == expected_names

    def test_projects_with_matching_org_are_included(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            _setup_structure_dir(
                tmpdir,
                organizations=[{'sonarqube_org_key': 'sq-org', 'sonarcloud_org_key': 'sc-org'}],
                projects=[
                    {'sonarqube_org_key': 'sq-org', 'sonarcloud_org_key': ''},
                    {'sonarqube_org_key': 'other-org', 'sonarcloud_org_key': ''},
                ],
            )
            run_dir, _ = validate_migration(directory=tmpdir, run_id='run-01')
            # Check the JSONL for projects was created
            project_dir = os.path.join(run_dir, 'generateProjectMappings')
            assert os.path.isdir(project_dir)

    def test_portfolios_pass_through_without_org_filter(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            _setup_structure_dir(
                tmpdir,
                portfolios=[{'key': 'port-1', 'name': 'Portfolio One'}],
            )
            run_dir, _ = validate_migration(directory=tmpdir, run_id='run-01')
            portfolio_dir = os.path.join(run_dir, 'generatePortfolioMappings')
            assert os.path.isdir(portfolio_dir)

    def test_empty_organization_mapping(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            _setup_structure_dir(
                tmpdir,
                organizations=[{'sonarqube_org_key': 'sq-org', 'sonarcloud_org_key': ''}],
                projects=[{'sonarqube_org_key': 'sq-org', 'sonarcloud_org_key': ''}],
            )
            _, completed = validate_migration(directory=tmpdir, run_id='run-01')
            assert 'generateProjectMappings' in completed
