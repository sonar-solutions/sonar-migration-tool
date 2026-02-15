"""Extract components and source code"""
import json
import os

from protobufs.scanner_report_pb2 import Component
from utils import multi_extract_object_reader
from history.utilities import write_protobuf_to_file


def extract_project_components(extract_directory, mapping, project_key):
    """Extract real components from getProjectComponents"""
    components = {}
    for url, component in multi_extract_object_reader(directory=extract_directory, mapping=mapping,
                                                      key='getProjectComponents'):
        if component['projectKey'] != project_key:
            continue
        if component['qualifier'] not in ['UTS', "FIL"]:
            continue
        components[component['path']] = {
            'ref': len(components) + 2,
            'projectRelativePath': component['path'],
            'is_test': component['qualifier'] == "UTS",
            'type': component['qualifier'],
            'language': component.get('language'),
        }
    return components


def write_source_file(source, scan_dir, ref_id):
    """Write source content to file"""
    with open(os.path.join(scan_dir, f'source-{ref_id}.txt'), 'wt') as f:
        try:
            f.write(source['content'])
            return len(source['content'].splitlines())
        except Exception:
            f.write(json.dumps(
                {k: v for k, v in source.items() 
                 if k not in ['projectComponentKey', 'projectKey', 'serverUrl']}))
            return 1


def extract_source(extract_directory, mapping, components, project_key, scan_dir):
    """Extract real source code from getProjectComponentSourceRaw"""
    for url, source in multi_extract_object_reader(directory=extract_directory, mapping=mapping,
                                                   key='getProjectComponentSourceRaw'):
        if source['projectKey'] != project_key:
            continue
        path = source['projectComponentKey'].replace(source['projectKey'] + ':', '')
        if not components.get(path):
            continue
        ref_id = components[path]['ref']
        components[path]['type'] = 'FILE'
        components[path]['status'] = 'ADDED'
        components[path]['lines'] = write_source_file(source, scan_dir, ref_id)
        write_protobuf_to_file(
            filepath=os.path.join(scan_dir, f'component-{ref_id}.pb'),
            payload=components[path],
            proto_class=Component
        )

