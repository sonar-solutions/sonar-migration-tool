"""Extract component measures"""
import os

from protobufs.scanner_report_pb2 import Measure
from utils import multi_extract_object_reader
from history.utilities import append_protobuf_to_file

COMPONENT_MEASURES = ['classes', 'functions', 'complexity', 'comment_lines', 'statements', 'ncloc',
                      'cognitive_complexity', 'executable_lines_data', 'ncloc_data']


def build_measure_payload(measure):
    """Build measure payload dict"""
    m = {'metricKey': measure['metric']}
    try:
        m['intValue'] = {'value': int(measure['value'])}
    except Exception:
        m['stringValue'] = {'value': measure['value']}
    return m


def extract_component_measures(extract_directory, mapping, components, project_key, scan_dir):
    """Extract real measures from getProjectComponentMeasures"""
    for url, measure in multi_extract_object_reader(directory=extract_directory, mapping=mapping,
                                                    key='getProjectComponentMeasures'):
        if measure['projectKey'] != project_key:
            continue
        path = measure['projectComponentKey'].replace(measure['projectKey'] + ':', '')
        if not components.get(path):
            continue
        ref_id = components[path]['ref']
        if measure['metric'] not in COMPONENT_MEASURES or not measure.get('value'):
            continue
        m = build_measure_payload(measure)
        append_protobuf_to_file(
            file_path=os.path.join(scan_dir, f'measures-{ref_id}.pb'),
            proto_class=Measure,
            payload=m,
        )

