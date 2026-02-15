"""Extract changeset data"""
import os
from datetime import datetime

from protobufs.scanner_report_pb2 import Changesets
from utils import multi_extract_object_reader
from history.utilities import write_protobuf_to_file


def build_revision_dict(line):
    """Build revision dict from SCM line"""
    return {
        'revision': line[3],
        'author': line[1],
        'date': int(datetime.fromisoformat(line[2]).timestamp()*1000)
    }


def process_scm_lines(scm_lines):
    """Process SCM lines into revisions and line indices"""
    lines = []
    revisions = {}
    for line in scm_lines:
        if len(line) < 4:
            continue
        if line[3] not in revisions:
            revisions[line[3]] = build_revision_dict(line)
        lines.append(list(revisions.keys()).index(line[3]))
    return list(revisions.values()), lines


def build_changeset_payload(ref_id, scm_lines):
    """Build changeset payload dict"""
    changeset_list, line_indices = process_scm_lines(scm_lines)
    return {
        'component_ref': ref_id,
        'changeset': changeset_list,
        'changesetIndexByLine': line_indices
    }


def extract_component_changesets(extract_directory, mapping, components, project_key, scan_dir):
    """Extract real changesets from getProjectComponentSourceControl"""
    for url, component in multi_extract_object_reader(directory=extract_directory, mapping=mapping,
                                                      key='getProjectComponentSourceControl'):
        if component['projectKey'] != project_key:
            continue
        path = component['projectComponentKey'].replace(component['projectKey'] + ':', '')
        if not components.get(path):
            continue
        ref_id = components[path]['ref']
        change = build_changeset_payload(ref_id, component['scm'])
        write_protobuf_to_file(
            filepath=os.path.join(scan_dir, f'changesets-{ref_id}.pb'),
            payload=change,
            proto_class=Changesets
        )

