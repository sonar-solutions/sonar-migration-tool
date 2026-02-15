"""Extract duplication data"""
import os

from protobufs.scanner_report_pb2 import Duplication
from utils import multi_extract_object_reader
from history.utilities import append_protobuf_to_file


def build_ref_mapping(duplications, components):
    """Build reference mapping for duplications"""
    return {
        k: components.get(v['key'].replace(duplications['projectKey'] + ':', ''))['ref']
        for k, v in duplications.get('files', {}).items()
        if components.get(v['key'].replace(duplications['projectKey'] + ':', ''))
    }


def build_duplicate_block(block, ref_mapping, ref_id):
    """Build duplicate block dict"""
    return {
        'other_file_ref': ref_mapping[block['_ref']] if ref_mapping.get(block['_ref']) != ref_id else 0,
        'range': {
            'startLine': block['from'],
            'endLine': block['from'] + block['size'] - 1
        }
    }


def build_duplicate_payload(section, ref_mapping, ref_id):
    """Build duplication payload dict"""
    return {
        'originPosition': {
            'startLine': section['blocks'][0]['from'],
            'endLine': section['blocks'][0]['from'] + section['blocks'][0]['size'] - 1
        },
        'duplicate': [
            build_duplicate_block(block, ref_mapping, ref_id)
            for block in section['blocks'][1:]
        ]
    }


def extract_project_duplications(extract_directory, mapping, components, project_key, scan_dir):
    """Extract real duplications from getProjectComponentSourceDuplications"""
    for url, duplications in multi_extract_object_reader(directory=extract_directory, mapping=mapping,
                                                         key='getProjectComponentSourceDuplications'):
        if duplications['projectKey'] != project_key:
            continue
        ref_mapping = build_ref_mapping(duplications, components)
        path = duplications['projectComponentKey'].replace(duplications['projectKey'] + ':', '')
        if not components.get(path):
            continue
        ref_id = components[path]['ref']
        for section in duplications['duplications']:
            duplicate = build_duplicate_payload(section, ref_mapping, ref_id)
            append_protobuf_to_file(
                file_path=os.path.join(scan_dir, f'duplications-{ref_id}.pb'),
                proto_class=Duplication,
                payload=duplicate,
            )

