"""Extract coverage data"""
import os

from protobufs.scanner_report_pb2 import LineCoverage
from utils import multi_extract_object_reader
from history.utilities import append_protobuf_to_file


def build_coverage_payload(line):
    """Build coverage payload dict"""
    c = {
        'line': line['line'],
        'hits': bool(line['lineHits']),
    }
    if 'conditions' in line.keys():
        c['conditions'] = line['conditions']
        c['covered_conditions'] = line['coveredConditions']
    return c


def extract_project_coverage(extract_directory, mapping, components, project_key, scan_dir):
    """Extract real coverage from getProjectComponentSource"""
    for url, coverage in multi_extract_object_reader(directory=extract_directory, mapping=mapping,
                                                     key='getProjectComponentSource'):
        if coverage['projectKey'] != project_key:
            continue
        path = coverage['projectComponentKey'].replace(coverage['projectKey'] + ':', '')
        if not components.get(path):
            continue
        ref_id = components[path]['ref']
        for line in coverage['sources']:
            if 'lineHits' not in line.keys():
                continue
            c = build_coverage_payload(line)
            append_protobuf_to_file(
                file_path=os.path.join(scan_dir, f'coverages-{ref_id}.pb'),
                proto_class=LineCoverage,
                payload=c,
            )

