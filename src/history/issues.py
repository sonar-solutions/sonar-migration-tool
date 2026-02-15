"""Extract issues and hotspots"""
import os

from protobufs.scanner_report_pb2 import Issue
from utils import multi_extract_object_reader
from history.utilities import append_protobuf_to_file

SEVERITY_MAPPING = {
    'INFO': 'INFO',
    'LOW': 'INFO',
    'MINOR': 'MINOR',
    'MEDIUM': 'MINOR',
    'HIGH': 'MAJOR',
    'MAJOR': 'MAJOR',
    'CRITICAL': 'CRITICAL',
    'BLOCKER': 'BLOCKER',
}


def build_issue_location(location, issue, components):
    """Build issue location dict"""
    loc = {'textRange': location['textRange']}
    if 'component' in location:
        loc_path = location['component'].replace(issue['projectKey'] + ':', '')
        if components.get(loc_path):
            loc['componentRef'] = components[loc_path]['ref']
    if 'msg' in location:
        loc['msg'] = location['msg']
    return loc


def build_issue_flow(flow_item, issue, components):
    """Build issue flow dict"""
    flow = {'location': []}
    for location in flow_item.get('locations', []):
        flow['location'].append(build_issue_location(location, issue, components))
    if flow_item.get('type'):
        flow['type'] = flow_item['type']
    if flow_item.get('description'):
        flow['description'] = flow_item['description']
    return flow


def build_issue_json(issue, components, ref_id):
    """Build issue JSON payload"""
    rule_key = 'rule' if 'rule' in issue.keys() else 'ruleKey'
    severity = issue['severity'] if 'severity' in issue.keys() else issue.get('vulnerabilityProbability', 'INFO')
    issue_json = {
        'ruleRepository': issue[rule_key].split(':')[0],
        'ruleKey': issue[rule_key].split(':')[1],
        'msg': issue['message'],
        'severity': SEVERITY_MAPPING.get(severity, 'INFO'),
        'gap': 0.0,
        'textRange': issue.get('textRange'),
    }
    if issue.get('gap'):
        issue_json['gap'] = issue['gap']
    if issue.get('flows'):
        issue_json['flow'] = [build_issue_flow(i, issue, components) for i in issue['flows']]
    if issue.get('textRange'):
        issue_json['textRange'] = issue['textRange']
    return issue_json


def extract_issues(extract_directory, mapping, components, project_key, scan_dir, source='getProjectComponentIssues'):
    """Extract real issues from getProjectComponentIssues or getProjectComponentHotspots"""
    for url, issue in multi_extract_object_reader(directory=extract_directory, mapping=mapping,
                                                  key=source):
        if issue['projectKey'] != project_key:
            continue
        path = issue['projectComponentKey'].replace(issue['projectKey'] + ':', '')
        if not components.get(path):
            continue
        ref_id = components[path]['ref']
        issue_json = build_issue_json(issue, components, ref_id)
        append_protobuf_to_file(
            file_path=os.path.join(scan_dir, f'issues-{ref_id}.pb'),
            proto_class=Issue,
            payload=issue_json,
        )

