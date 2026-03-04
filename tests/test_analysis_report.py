import json
import os
import tempfile

import pytest

from analysis_report import (
    URL_ENTITY_MAP,
    generate_final_analysis_report,
    _classify_entity_type,
    _extract_entity_name,
    _extract_organization,
    _extract_error_message,
    _parse_log_line,
)


def _make_log_entry(method='POST', url='/api/projects/create', status_code=200,
                    log_status='success', data=None, json_body=None, response=None,
                    process_type='request_completed'):
    payload = {
        'method': method,
        'url': url,
        'status': status_code,
        'created_ts': 1234567890.0,
    }
    if data is not None:
        payload['data'] = data
    if json_body is not None:
        payload['json'] = json_body
    if response is not None:
        payload['response'] = response
    return {
        'process_type': process_type,
        'created_ts': 1234567890.0,
        'status': log_status,
        'payload': payload,
    }


def _write_log(tmpdir, entries):
    log_path = os.path.join(tmpdir, 'requests.log')
    with open(log_path, 'wt') as f:
        for entry in entries:
            f.write(json.dumps(entry) + '\n')
    return log_path


class TestParseLogLine:
    def test_valid_json(self):
        result = _parse_log_line('{"key": "value"}')
        assert result == {"key": "value"}

    def test_empty_line(self):
        assert _parse_log_line('') is None
        assert _parse_log_line('   ') is None

    def test_invalid_json(self):
        assert _parse_log_line('not json') is None


class TestClassifyEntityType:
    def test_known_urls(self):
        assert _classify_entity_type('/api/projects/create') == 'Project'
        assert _classify_entity_type('/api/qualitygates/create') == 'Quality Gate'
        assert _classify_entity_type('/api/qualityprofiles/create') == 'Quality Profile'
        assert _classify_entity_type('/api/user_groups/create') == 'Group'
        assert _classify_entity_type('/api/permissions/create_template') == 'Permission Template'
        assert _classify_entity_type('/enterprises/portfolios') == 'Portfolio'
        assert _classify_entity_type('/dop-translation/project-bindings') == 'Project Binding'
        assert _classify_entity_type('/api/rules/update') == 'Rule'

    def test_unknown_url(self):
        assert _classify_entity_type('/api/unknown/endpoint') == 'Unknown'

    def test_none_url(self):
        assert _classify_entity_type(None) == 'Unknown'

    def test_empty_url(self):
        assert _classify_entity_type('') == 'Unknown'


class TestExtractEntityName:
    def test_name_field(self):
        payload = {'data': {'name': 'My Project', 'key': 'proj-1'}}
        assert _extract_entity_name(payload) == 'My Project'

    def test_project_field(self):
        payload = {'data': {'project': 'proj-key'}}
        assert _extract_entity_name(payload) == 'proj-key'

    def test_project_key_field(self):
        payload = {'json': {'projectKey': 'proj-key-2'}}
        assert _extract_entity_name(payload) == 'proj-key-2'

    def test_gate_name_field(self):
        payload = {'data': {'gateName': 'My Gate'}}
        assert _extract_entity_name(payload) == 'My Gate'

    def test_group_name_field(self):
        payload = {'data': {'groupName': 'developers'}}
        assert _extract_entity_name(payload) == 'developers'

    def test_key_field(self):
        payload = {'data': {'key': 'some-key'}}
        assert _extract_entity_name(payload) == 'some-key'

    def test_language_field(self):
        payload = {'data': {'language': 'java'}}
        assert _extract_entity_name(payload) == 'java'

    def test_no_matching_field(self):
        payload = {'data': {'foo': 'bar'}}
        assert _extract_entity_name(payload) == ''

    def test_empty_payload(self):
        assert _extract_entity_name({}) == ''

    def test_params_fallback(self):
        payload = {'params': {'name': 'from-params'}}
        assert _extract_entity_name(payload) == 'from-params'

    def test_priority_order(self):
        payload = {'data': {'key': 'key-val', 'name': 'name-val'}}
        assert _extract_entity_name(payload) == 'name-val'


class TestExtractOrganization:
    def test_from_data(self):
        payload = {'data': {'organization': 'my-org', 'name': 'test'}}
        assert _extract_organization(payload) == 'my-org'

    def test_from_json(self):
        payload = {'json': {'organization': 'json-org'}}
        assert _extract_organization(payload) == 'json-org'

    def test_missing(self):
        payload = {'data': {'name': 'test'}}
        assert _extract_organization(payload) == ''

    def test_empty_payload(self):
        assert _extract_organization({}) == ''


class TestExtractErrorMessage:
    def test_sonarqube_error_format_string(self):
        payload = {
            'response': json.dumps({'errors': [{'msg': 'Project already exists'}]})
        }
        assert _extract_error_message(payload) == 'Project already exists'

    def test_multiple_errors(self):
        payload = {
            'response': json.dumps({'errors': [{'msg': 'Error 1'}, {'msg': 'Error 2'}]})
        }
        assert _extract_error_message(payload) == 'Error 1; Error 2'

    def test_no_response(self):
        assert _extract_error_message({}) == ''

    def test_non_json_response(self):
        payload = {'response': 'Internal Server Error'}
        assert _extract_error_message(payload) == ''

    def test_content_field(self):
        payload = {
            'content': json.dumps({'errors': [{'msg': 'From content field'}]})
        }
        assert _extract_error_message(payload) == 'From content field'

    def test_response_as_dict(self):
        payload = {
            'response': {'errors': [{'msg': 'Already a dict'}]}
        }
        assert _extract_error_message(payload) == 'Already a dict'


class TestGenerateFinalAnalysisReport:
    def test_basic_success_entry(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            entry = _make_log_entry(
                url='/api/projects/create',
                status_code=200,
                data={'name': 'My Project', 'organization': 'my-org'},
            )
            _write_log(tmpdir, [entry])

            rows = generate_final_analysis_report(run_directory=tmpdir)

            assert len(rows) == 1
            assert rows[0]['entity_type'] == 'Project'
            assert rows[0]['entity_name'] == 'My Project'
            assert rows[0]['organization'] == 'my-org'
            assert rows[0]['url'] == '/api/projects/create'
            assert rows[0]['http_status'] == 200
            assert rows[0]['outcome'] == 'success'
            assert rows[0]['error_message'] == ''

    def test_failure_entry(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            entry = _make_log_entry(
                url='/api/projects/create',
                status_code=400,
                log_status='failure',
                data={'name': 'Dup Project', 'organization': 'my-org'},
                response=json.dumps({'errors': [{'msg': 'Project already exists'}]}),
            )
            _write_log(tmpdir, [entry])

            rows = generate_final_analysis_report(run_directory=tmpdir)

            assert len(rows) == 1
            assert rows[0]['outcome'] == 'failure'
            assert rows[0]['http_status'] == 400
            assert rows[0]['error_message'] == 'Project already exists'

    def test_filters_get_requests(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            entries = [
                _make_log_entry(method='GET', url='/api/server/version', status_code=200),
                _make_log_entry(method='POST', url='/api/projects/create', status_code=200,
                                data={'name': 'Test'}),
            ]
            _write_log(tmpdir, entries)

            rows = generate_final_analysis_report(run_directory=tmpdir)
            assert len(rows) == 1
            assert rows[0]['entity_type'] == 'Project'

    def test_filters_request_started(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            entries = [
                _make_log_entry(process_type='request_started'),
                _make_log_entry(url='/api/projects/create', data={'name': 'Test'}),
            ]
            _write_log(tmpdir, entries)

            rows = generate_final_analysis_report(run_directory=tmpdir)
            assert len(rows) == 1

    def test_csv_file_created(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            entry = _make_log_entry(
                url='/api/projects/create',
                status_code=200,
                data={'name': 'Test'},
            )
            _write_log(tmpdir, [entry])

            generate_final_analysis_report(run_directory=tmpdir)

            csv_path = os.path.join(tmpdir, 'final_analysis_report.csv')
            assert os.path.exists(csv_path)

    def test_custom_output_directory(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            run_dir = os.path.join(tmpdir, 'run')
            out_dir = os.path.join(tmpdir, 'output')
            os.makedirs(run_dir)
            os.makedirs(out_dir)

            entry = _make_log_entry(
                url='/api/projects/create',
                status_code=200,
                data={'name': 'Test'},
            )
            _write_log(run_dir, [entry])

            generate_final_analysis_report(run_directory=run_dir, output_directory=out_dir)

            assert os.path.exists(os.path.join(out_dir, 'final_analysis_report.csv'))
            assert not os.path.exists(os.path.join(run_dir, 'final_analysis_report.csv'))

    def test_empty_log(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            _write_log(tmpdir, [])
            rows = generate_final_analysis_report(run_directory=tmpdir)
            assert rows == []
            assert not os.path.exists(os.path.join(tmpdir, 'final_analysis_report.csv'))

    def test_no_log_file(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            rows = generate_final_analysis_report(run_directory=tmpdir)
            assert rows == []

    def test_multiple_entity_types(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            entries = [
                _make_log_entry(url='/api/projects/create', status_code=200,
                                data={'name': 'Proj1', 'organization': 'org1'}),
                _make_log_entry(url='/api/qualitygates/create', status_code=200,
                                data={'name': 'Gate1', 'organization': 'org1'}),
                _make_log_entry(url='/api/user_groups/create', status_code=200,
                                data={'name': 'Group1', 'organization': 'org1'}),
                _make_log_entry(url='/api/qualityprofiles/create', status_code=200,
                                data={'name': 'Profile1', 'organization': 'org1'}),
            ]
            _write_log(tmpdir, entries)

            rows = generate_final_analysis_report(run_directory=tmpdir)
            assert len(rows) == 4
            types = {r['entity_type'] for r in rows}
            assert types == {'Project', 'Quality Gate', 'Group', 'Quality Profile'}

    def test_failure_with_log_status(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            entry = _make_log_entry(
                url='/api/projects/create',
                status_code=None,
                log_status='failure',
                data={'name': 'Failed Project'},
            )
            entry['payload']['status'] = None
            _write_log(tmpdir, [entry])

            rows = generate_final_analysis_report(run_directory=tmpdir)
            assert len(rows) == 1
            assert rows[0]['outcome'] == 'failure'
            assert rows[0]['http_status'] == ''

    def test_csv_columns(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            entry = _make_log_entry(
                url='/api/projects/create',
                status_code=200,
                data={'name': 'Test', 'organization': 'org1'},
            )
            _write_log(tmpdir, [entry])

            rows = generate_final_analysis_report(run_directory=tmpdir)
            expected_keys = {'entity_type', 'entity_name', 'organization', 'url',
                             'http_status', 'outcome', 'error_message'}
            assert set(rows[0].keys()) == expected_keys

    def test_json_body_extraction(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            entry = _make_log_entry(
                url='/enterprises/portfolios',
                status_code=200,
                json_body={'name': 'My Portfolio', 'organization': 'org1'},
            )
            _write_log(tmpdir, [entry])

            rows = generate_final_analysis_report(run_directory=tmpdir)
            assert len(rows) == 1
            assert rows[0]['entity_name'] == 'My Portfolio'
            assert rows[0]['organization'] == 'org1'
            assert rows[0]['entity_type'] == 'Portfolio'


class TestUrlEntityMapCoverage:
    def test_all_migration_post_endpoints_mapped(self):
        expected_urls = [
            '/api/projects/create',
            '/api/qualitygates/create',
            '/api/qualitygates/create_condition',
            '/api/qualitygates/select',
            '/api/qualitygates/set_as_default',
            '/api/qualityprofiles/create',
            '/api/qualityprofiles/restore',
            '/api/qualityprofiles/set_default',
            '/api/qualityprofiles/add_project',
            '/api/qualityprofiles/add_group',
            '/api/qualityprofiles/change_parent',
            '/api/user_groups/create',
            '/api/user_groups/add_user',
            '/api/permissions/create_template',
            '/api/permissions/set_default_template',
            '/api/permissions/add_group_to_template',
            '/api/permissions/add_group',
            '/api/settings/set',
            '/api/rules/update',
            '/api/project_branches/rename',
            '/api/project_tags/set',
            '/dop-translation/project-bindings',
            '/enterprises/portfolios',
        ]
        for url in expected_urls:
            assert url in URL_ENTITY_MAP, f"Missing URL mapping: {url}"
            assert _classify_entity_type(url) != 'Unknown', f"URL classified as Unknown: {url}"
