"""Generate latest scan data using real source code, issues, and measures"""
import os
import uuid
from datetime import datetime, UTC

from constants import PLUGINS
from utils import multi_extract_object_reader
from protobufs.scanner_report_pb2 import Metadata, Component
from history.projects import get_new_projects, get_profiles, get_language_rules
from history.rules import generate_active_rules
from history.components import extract_project_components, extract_source
from history.issues import extract_issues
from history.measures import extract_component_measures
from history.coverage import extract_project_coverage
from history.duplications import extract_project_duplications
from history.changesets import extract_component_changesets
from history.utilities import write_protobuf_to_file


def get_latest_analysis(extract_directory, extract_mapping, project_key):
    """Get latest analysis for a project"""
    latest_analysis = None
    latest_date = None
    for url, analysis in multi_extract_object_reader(directory=extract_directory, mapping=extract_mapping,
                                                     key='getProjectAnalyses'):
        if analysis['projectKey'] != project_key:
            continue
        analysis_date = datetime.fromisoformat(analysis['date'])
        if latest_date is None or analysis_date > latest_date:
            latest_date = analysis_date
            latest_analysis = analysis
    return latest_analysis, latest_date


def build_profile_keys(new_project, org_profiles):
    """Build profile keys dict"""
    profile_keys = {}
    for profile in new_project['profiles']:
        if profile['deleted']:
            continue
        new_profile = org_profiles[new_project['sonarCloudOrgKey']][profile['language']].get(profile['name'])
        if not new_profile:
            new_profile = org_profiles[new_project['sonarCloudOrgKey']][profile['language']]['Sonar Way']
        profile_keys[new_profile['language']] = {
            "key": new_profile['key'],
            "name": new_profile['name'],
            "language": new_profile['language'],
        }
    return profile_keys


def build_metadata_json(new_project, latest_analysis, latest_date, profile_keys):
    """Build metadata JSON payload"""
    metadata_json = {
        'analysisDate': int(latest_date.timestamp() * 1000),
        'organizationKey': new_project['sonarCloudOrgKey'],
        'projectKey': new_project['key'],
        'rootComponentRef': 1,
        'modulesProjectRelativePathByKey': {
            new_project['key']: ''
        },
        'qprofilesPerLanguage': profile_keys,
        'pluginsByKey': {i['key']: i for i in PLUGINS},
        'analysisUuid': str(uuid.uuid4()),
        'analysisStartedTimestamp': int(latest_date.timestamp() * 1000) - 1000
    }
    if latest_analysis.get('revision'):
        metadata_json['scmRevisionId'] = latest_analysis['revision']
    if latest_analysis.get('projectVersion') and latest_analysis.get('projectVersion') != 'not provided':
        metadata_json['projectVersion'] = latest_analysis['projectVersion']
    return metadata_json


def create_root_component(new_project, components):
    """Create root component protobuf"""
    root_component = {
        'ref': 1,
        'type': 'PROJECT',
        'name': new_project['name'],
        'childRef': [comp['ref'] for comp in components.values()],
        'key': new_project['key']
    }
    return root_component


def extract_all_data(extract_directory, extract_mapping, components, project_key, scan_dir):
    """Extract all scan data (issues, measures, coverage, duplications, changesets)"""
    extract_issues(extract_directory, extract_mapping, components, project_key, scan_dir)
    extract_issues(extract_directory, extract_mapping, components, project_key, scan_dir, 
                   source='getProjectComponentHotspots')
    extract_component_measures(extract_directory, extract_mapping, components, project_key, scan_dir)
    extract_project_coverage(extract_directory, extract_mapping, components, project_key, scan_dir)
    extract_project_duplications(extract_directory, extract_mapping, components, project_key, scan_dir)
    extract_component_changesets(extract_directory, extract_mapping, components, project_key, scan_dir)


def setup_project_data(extract_directory, extract_mapping, migration_mapping, project_key):
    """Setup project, profiles, and rules"""
    projects = get_new_projects(extract_directory, migration_mapping)
    if project_key not in projects:
        raise ValueError(f"Project {project_key} not found in migrated projects")
    new_project = projects[project_key]
    org_profiles = get_profiles(extract_directory, migration_mapping)
    language_rules = get_language_rules(extract_directory, extract_mapping)
    return new_project, org_profiles, language_rules


def create_scan_directory(output_dir, new_project, latest_date):
    """Create scan directory and return path"""
    scan_timestamp = int(latest_date.timestamp())
    scan_dir = os.path.join(output_dir, new_project['key'], str(scan_timestamp))
    os.makedirs(scan_dir, exist_ok=True)
    return scan_dir


def write_metadata_and_components(scan_dir, new_project, latest_analysis, latest_date, profile_keys, components):
    """Write metadata and root component protobufs"""
    metadata_json = build_metadata_json(new_project, latest_analysis, latest_date, profile_keys)
    write_protobuf_to_file(
        filepath=os.path.join(scan_dir, 'metadata.pb'),
        payload=metadata_json,
        proto_class=Metadata
    )
    root_component = create_root_component(new_project, components)
    write_protobuf_to_file(
        filepath=os.path.join(scan_dir, 'component-1.pb'),
        payload=root_component,
        proto_class=Component
    )


def extract_components_and_source(extract_directory, extract_mapping, project_key, scan_dir):
    """Extract components and source code"""
    components = extract_project_components(extract_directory, extract_mapping, project_key)
    extract_source(extract_directory, extract_mapping, components, project_key, scan_dir)
    return components


def generate_latest_scan(extract_directory, extract_mapping, migration_mapping, project_key, output_dir):
    """Generate latest scan data for a migrated project using real source code, issues, and measures."""
    new_project, org_profiles, language_rules = setup_project_data(
        extract_directory, extract_mapping, migration_mapping, project_key)
    latest_analysis, latest_date = get_latest_analysis(extract_directory, extract_mapping, project_key)
    if not latest_analysis:
        raise ValueError(f"No analysis found for project {project_key}")
    profile_keys = build_profile_keys(new_project, org_profiles)
    scan_dir = create_scan_directory(output_dir, new_project, latest_date)
    components = extract_components_and_source(extract_directory, extract_mapping, project_key, scan_dir)
    write_metadata_and_components(scan_dir, new_project, latest_analysis, latest_date, profile_keys, components)
    extract_all_data(extract_directory, extract_mapping, components, project_key, scan_dir)
    generate_active_rules(scan_dir, profile_keys, language_rules)
    return scan_dir
