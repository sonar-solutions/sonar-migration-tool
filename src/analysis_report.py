import json
import os

from utils import export_csv

URL_ENTITY_MAP = {
    '/api/projects/create': 'Project',
    '/api/projects/search': 'Project',
    '/api/projects/delete': 'Project',
    '/api/navigation/component': 'Project',
    '/api/project_branches/list': 'Branch',
    '/api/project_branches/rename': 'Branch',
    '/api/project_tags/set': 'Project Tag',
    '/api/qualitygates/create': 'Quality Gate',
    '/api/qualitygates/create_condition': 'Quality Gate Condition',
    '/api/qualitygates/select': 'Quality Gate Assignment',
    '/api/qualitygates/set_as_default': 'Quality Gate Default',
    '/api/qualityprofiles/create': 'Quality Profile',
    '/api/qualityprofiles/restore': 'Quality Profile',
    '/api/qualityprofiles/set_default': 'Quality Profile Default',
    '/api/qualityprofiles/add_project': 'Quality Profile Assignment',
    '/api/qualityprofiles/add_group': 'Quality Profile Permission',
    '/api/qualityprofiles/change_parent': 'Quality Profile Inheritance',
    '/api/user_groups/create': 'Group',
    '/api/user_groups/add_user': 'Group Membership',
    '/api/permissions/create_template': 'Permission Template',
    '/api/permissions/set_default_template': 'Permission Template Default',
    '/api/permissions/add_group_to_template': 'Template Permission',
    '/api/permissions/add_group': 'Group Permission',
    '/api/settings/set': 'Setting',
    '/api/settings/values': 'Setting',
    '/api/rules/update': 'Rule',
    '/api/alm_integration/list_repositories': 'ALM Repository',
    '/dop-translation/project-bindings': 'Project Binding',
    '/enterprises/portfolios': 'Portfolio',
    'api/users/current': 'User',
}

ENTITY_NAME_FIELDS = ['name', 'project', 'projectKey', 'gateName', 'groupName', 'key', 'language']


def _parse_log_line(line):
    line = line.strip()
    if not line:
        return None
    try:
        return json.loads(line)
    except json.JSONDecodeError:
        return None


def _classify_entity_type(url):
    if not url:
        return 'Unknown'
    for path, entity_type in URL_ENTITY_MAP.items():
        if url.endswith(path) or url == path:
            return entity_type
    return 'Unknown'


def _get_request_body(payload):
    body = payload.get('data') or payload.get('json') or payload.get('params') or {}
    if isinstance(body, str):
        try:
            body = json.loads(body)
        except (json.JSONDecodeError, TypeError):
            body = {}
    return body


def _extract_entity_name(payload):
    body = _get_request_body(payload)
    if not isinstance(body, dict):
        return ''
    for field in ENTITY_NAME_FIELDS:
        value = body.get(field)
        if value:
            return str(value)
    return ''


def _extract_organization(payload):
    body = _get_request_body(payload)
    if isinstance(body, dict):
        return body.get('organization', '')
    return ''


def _extract_error_message(payload):
    response_text = payload.get('response') or payload.get('content', '')
    if not response_text:
        return ''
    if isinstance(response_text, dict):
        errors = response_text.get('errors', [])
        if errors:
            return '; '.join(e.get('msg', '') for e in errors if e.get('msg'))
        return ''
    if isinstance(response_text, str):
        try:
            response_json = json.loads(response_text)
            errors = response_json.get('errors', [])
            if errors:
                return '; '.join(e.get('msg', '') for e in errors if e.get('msg'))
        except (json.JSONDecodeError, TypeError):
            pass
    return ''


def _determine_outcome(http_status, log_status):
    if http_status is not None and http_status < 400:
        return 'success'
    if log_status == 'failure' or (http_status is not None and http_status >= 400):
        return 'failure'
    return 'success'


def _process_entry(entry):
    if entry.get('process_type') != 'request_completed':
        return None
    payload = entry.get('payload', {})
    if payload.get('method') != 'POST':
        return None

    url = payload.get('url', '')
    http_status = payload.get('status')
    log_status = entry.get('status', '')
    outcome = _determine_outcome(http_status, log_status)
    error_message = _extract_error_message(payload) if outcome == 'failure' else ''

    return {
        'entity_type': _classify_entity_type(url),
        'entity_name': _extract_entity_name(payload),
        'organization': _extract_organization(payload),
        'url': url,
        'http_status': http_status if http_status is not None else '',
        'outcome': outcome,
        'error_message': error_message,
    }


def generate_final_analysis_report(run_directory, output_directory=None):
    if output_directory is None:
        output_directory = run_directory

    log_path = os.path.join(run_directory, 'requests.log')
    if not os.path.exists(log_path):
        return []

    rows = []
    with open(log_path, 'rt') as f:
        for line in f:
            entry = _parse_log_line(line)
            if entry is None:
                continue
            row = _process_entry(entry)
            if row is not None:
                rows.append(row)

    if rows:
        export_csv(directory=output_directory, name='final_analysis_report', data=rows)

    return rows
