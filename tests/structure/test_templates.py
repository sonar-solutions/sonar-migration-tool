"""Tests for src/structure/templates.py"""
import json
import os
import tempfile

from structure.templates import map_templates, add_template


def _write_jsonl(directory, key, objects):
    folder = os.path.join(directory, key)
    os.makedirs(folder, exist_ok=True)
    path = os.path.join(folder, 'results.1.jsonl')
    with open(path, 'w') as f:
        for obj in objects:
            f.write(json.dumps(obj) + '\n')


def _make_extract(export_dir, extract_id, default_templates=None, templates=None):
    run_dir = os.path.join(export_dir, extract_id)
    os.makedirs(run_dir, exist_ok=True)
    with open(os.path.join(run_dir, 'extract.json'), 'w') as f:
        json.dump({'url': 'https://sonar.example.com/'}, f)
    if default_templates:
        _write_jsonl(run_dir, 'getDefaultTemplates', default_templates)
    if templates:
        _write_jsonl(run_dir, 'getTemplates', templates)


class TestAddTemplate:
    def test_creates_template_entry(self):
        results = {}
        add_template(
            results=results,
            org_key='my-org',
            template={'id': 'tmpl-1', 'name': 'My Template', 'description': 'desc', 'projectKeyPattern': ''},
            server_url='https://sonar.example.com/',
            is_default=False,
        )
        assert 'my-orgtmpl-1' in results
        entry = results['my-orgtmpl-1']
        assert entry['name'] == 'My Template'
        assert entry['is_default'] is False
        assert entry['sonarqube_org_key'] == 'my-org'

    def test_default_template_flagged(self):
        results = {}
        add_template(
            results=results,
            org_key='org',
            template={'id': 't1', 'name': 'Default', 'description': None, 'projectKeyPattern': ''},
            server_url='https://sonar.example.com/',
            is_default=True,
        )
        assert results['orgt1']['is_default'] is True


class TestMapTemplates:
    def test_default_templates_mapped_to_all_orgs(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            project_org_mapping = {
                'https://sonar.example.com/proj1': 'org-a',
                'https://sonar.example.com/proj2': 'org-b',
            }
            _make_extract(
                tmpdir,
                extract_id='run-01',
                default_templates=[{'templateId': 'tmpl-1'}],
                templates=[{'id': 'tmpl-1', 'name': 'T1', 'projectKeyPattern': ''}],
            )
            extract_mapping = {'https://sonar.example.com/': 'run-01'}
            result = map_templates(
                project_org_mapping=project_org_mapping,
                extract_mapping=extract_mapping,
                export_directory=tmpdir,
            )
            # Default template should be applied to both orgs
            assert len(result) == 2
            names = {r['name'] for r in result}
            assert names == {'T1'}

    def test_regex_template_matches_by_project_key(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            project_org_mapping = {
                'https://sonar.example.com/myapp': 'org-a',
                'https://sonar.example.com/other': 'org-b',
            }
            _make_extract(
                tmpdir,
                extract_id='run-01',
                default_templates=[],
                templates=[{
                    'id': 'tmpl-2',
                    'name': 'Regex Template',
                    'projectKeyPattern': 'myapp.*',
                }],
            )
            extract_mapping = {'https://sonar.example.com/': 'run-01'}
            result = map_templates(
                project_org_mapping=project_org_mapping,
                extract_mapping=extract_mapping,
                export_directory=tmpdir,
            )
            # Only matches 'myapp', not 'other'
            assert len(result) == 1
            assert result[0]['sonarqube_org_key'] == 'org-a'

    def test_invalid_regex_is_skipped(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            project_org_mapping = {'https://sonar.example.com/proj1': 'org-a'}
            _make_extract(
                tmpdir,
                extract_id='run-01',
                default_templates=[],
                templates=[{
                    'id': 'tmpl-3',
                    'name': 'Bad Regex',
                    'projectKeyPattern': '[invalid(regex',
                }],
            )
            extract_mapping = {'https://sonar.example.com/': 'run-01'}
            # Should not raise, just skip the bad regex template
            result = map_templates(
                project_org_mapping=project_org_mapping,
                extract_mapping=extract_mapping,
                export_directory=tmpdir,
            )
            assert result == []
