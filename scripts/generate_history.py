import json
import os
import uuid
from constants import PLUGINS, SEVERITY_MAPPING
from collections import defaultdict
from utils import multi_extract_object_reader, get_unique_extracts
from parser import extract_path_value
from google.protobuf.internal.encoder import _VarintBytes
from datetime import datetime, UTC
from google.protobuf import json_format
from protobufs.scanner_report_pb2 import Metadata, Component, Measure, ActiveRule, Changesets, LineCoverage, Issue, \
    Duplication, LineCoverage
import itertools
from history.utilities import append_protobuf_to_file, write_protobuf_to_file

LINE_CONTENT = "test_def{idx}();\n"
component_measures = ['classes', 'functions', 'complexity', 'comment_lines', 'statements', 'ncloc',
                      'cognitive_complexity', 'executable_lines_data', 'ncloc_data']
directory = '../files/'

extract_mapping = get_unique_extracts(directory=directory)
migration_mapping = get_unique_extracts(directory=directory, key='migrate.json')
server_projects = defaultdict(lambda: defaultdict(lambda: defaultdict(list)))
run_id = int(datetime.now(UTC).timestamp())
result_dir = os.path.join(directory, str(run_id))
os.makedirs(result_dir, exist_ok=True)

profile_rules = dict()


def get_new_projects(extract_directory, mapping):
    projects = dict()
    for url, project in multi_extract_object_reader(directory=extract_directory, mapping=mapping,
                                                    key='createProjects'):
        if project['key'] != 'do-more-thing_demo:java-security':
            continue
        projects[project['sourceProjectKey']] = project
    return projects


def get_profiles(extract_directory, mapping):
    org_profiles = defaultdict(lambda: defaultdict(dict))
    for url, profile in multi_extract_object_reader(directory=extract_directory, mapping=mapping,
                                                    key='pullAllProfiles'):
        org_profiles[profile['sonarCloudOrgKey']][profile['language']][profile['name']] = profile
    return org_profiles


def get_language_rules(extract_directory, mapping):
    language_rules = defaultdict(lambda: dict(
        bug=None,
        vulnerability=None,
        code_smell=None,
        security_hotspot=None
    ))
    for url, rule in multi_extract_object_reader(directory=extract_directory, mapping=mapping,
                                                 key='getRuleDetails'):
        if rule.get('templateKey') or rule.get('isTemplate'):
            continue
        language_rules[rule['lang']][rule['type'].lower()] = dict(
            key=rule['key'].split(':')[-1],
            rule_repository=rule['repo'],
            severity=rule['severity'],
        )
    return language_rules


def extract_project_scans(*, created_projects, org_profiles, result_dir, extract_directory, mapping, ):
    projects = defaultdict(list)
    for url, analysis in multi_extract_object_reader(directory=extract_directory, mapping=mapping,
                                                     key='getProjectAnalyses'):
        key = analysis['projectKey']
        if key not in created_projects:
            continue
        analysis_date = datetime.fromisoformat(analysis['date'])
        new_project = created_projects[key]
        new_key = new_project['key']
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
        project_dir = str(os.path.join(result_dir, new_key))
        os.makedirs(project_dir, exist_ok=True)
        new_project['profile_keys'] = profile_keys
        os.makedirs(os.path.join(project_dir, str(int(analysis_date.timestamp()))), exist_ok=True)
        metadata_json = {
            "analysisDate": int(analysis_date.timestamp() * 1000),
            "organizationKey": new_project['sonarCloudOrgKey'],
            "projectKey": new_key,
            "rootComponentRef": 1,
            "modulesProjectRelativePathByKey": {
                new_key: ""
            },
            "qprofilesPerLanguage": profile_keys,
            "pluginsByKey": {i['key']: i for i in PLUGINS},
            "analysisUuid": str(uuid.uuid4()),
            "analysisStartedTimestamp": int(analysis_date.timestamp() * 1000) - 1000
        }
        if analysis.get('revision'):
            metadata_json['scmRevisionId'] = analysis['revision']
        if analysis.get('projectVersion') and analysis.get('projectVersion') != 'not provided':
            metadata_json['projectVersion'] = analysis['projectVersion']
        file_path = os.path.join(project_dir, str(int(analysis_date.timestamp())), 'metadata.pb')
        write_protobuf_to_file(filepath=file_path, payload=metadata_json, proto_class=Metadata)
        analysis['profiles'] = profile_keys
        projects[new_key].append(analysis)
    return projects


RELEVANT_MEASURES = [
    "files",
    "classes",
    "functions",
    "statements",

    "lines",
    "generated_lines",
    "generated_ncloc",
    "comment_lines",
    "ncloc",

    "vulnerabilities",
    "code_smells",
    "bugs",
    "security_hotspots",

    "complexity",
    "cognitive_complexity",

    "tests",
    "coverage",
    "lines_to_cover",
    "conditions",
    "conditions_to_cover",

    "maintainability_issues",
    "reliability_issues",
    "security_issues",

    "duplicated_lines",
    "duplicated_blocks",
]

COMPONENT_MEASURES = [
    "complexity",
    "cognitive_complexity",
    "generated_lines",
    "generated_ncloc",
    "comment_lines",
    "ncloc",
    "classes",
    "functions",
    "statements",
]

COMPONENT_ISSUES = [
    "vulnerabilities",
    "code_smells",
    "bugs",
    "security_hotspots",
]

SEVERITIES = [
    "blocker_violations",
    "critical_violations",
    "major_violations",
    "minor_violations",
    "info_violations"
]


def generate_historical_structure(extract_directory, mapping, created_projects):
    historical_projects = defaultdict(
        lambda: dict(
            name=None,
            languages=[],
            profiles=dict(),
            scans=defaultdict(
                lambda: dict.fromkeys(RELEVANT_MEASURES, 0)
            )
        )
    )
    for url, measures in multi_extract_object_reader(directory=extract_directory, mapping=mapping,
                                                     key='getProjectMeasureHistory'):
        key = measures['projectKey']
        if key not in created_projects or measures['metric'] not in RELEVANT_MEASURES:
            continue
        new_project = created_projects[key]
        new_key = new_project['key']
        historical_projects[new_key]['name'] = new_project['name']
        historical_projects[new_key]['languages'] = [i['language'] for i in new_project['profiles']]
        historical_projects[new_key]['profiles'] = new_project['profile_keys']
        for measure in measures['history']:
            if 'value' not in measure:
                continue
            date = datetime.fromisoformat(measure['date'])
            historical_projects[new_key]['scans'][int(date.timestamp())][measures['metric']] = json.loads(
                measure['value'])
    return historical_projects


"""
{
  string rule_repository = 1;
  string rule_key = 2;
  Severity severity = 3;
  map<string, string> params_by_key = 4;
  int64 createdAt = 5;
  int64 updatedAt = 6;
  string q_profile_key = 7;
}

"""


def generate_active_rules(output_dir, profile_keys, rules):
    for profile in profile_keys.values():
        for rule in rules[profile['language']].values():
            if not rule:
                continue
            active_rule = {
                "rule_repository": rule['rule_repository'],
                "rule_key": rule['key'].split(':')[-1],
                "severity": rule['severity'],
                "createdAt": int(datetime.now(UTC).timestamp()) * 1000,
                "updatedAt": int(datetime.now(UTC).timestamp()) * 1000,
                "q_profile_key": profile['key']
            }
            append_protobuf_to_file(
                file_path=os.path.join(output_dir, f'activerules.pb'),
                proto_class=ActiveRule,
                payload=active_rule,
            )


def generate_historical_analysis(output_dir, historical_analysis, rules):
    for project_key, project in historical_analysis.items():
        for date, measures in project['scans'].items():
            if date == max(project['scans'].keys()):
                continue
            analysis_dir = os.path.join(output_dir, project_key, str(date))
            files = generate_project_files(output_dir=analysis_dir, file_count=measures['files'],
                                           lines=measures['lines'], languages=project['languages'], rules=rules)
            generate_project_measures(output_dir=analysis_dir, files=files,
                                      **{k: v for k, v in measures.items() if k in COMPONENT_MEASURES})
            generate_active_rules(output_dir=analysis_dir, rules=rules, profile_keys=project['profiles'])
            generate_issues(
                output_dir=analysis_dir,
                files=files,
                rules=rules,
                bugs=measures.get('bugs', 0),
                vulnerabilities=measures.get('vulnerabilities'),
                code_smells=measures.get('code_smells', 0),
                severities={k: v for k, v in measures.items() if k in SEVERITIES},
                hotspots=measures.get('security_hotspots', 0),
                quality_counts={
                    "RELIABILITY": measures.get('reliability_issues', {}),
                    "SECURITY": measures.get('security_issues', {}),
                    "MAINTAINABILITY": measures.get('maintainability_issues', {}),
                }
            )
            root_component = dict(
                ref=1,
                type='PROJECT',
                name=project['name'],
                childRef=[
                    i['component_id'] for i in files
                ],
                key=project_key
            )
            write_protobuf_to_file(filepath=os.path.join(analysis_dir, 'component-1.pb'), payload=root_component,
                                   proto_class=Component)


def generate_project_measures(output_dir, files, **measures):
    measures = {i: dict(total=measures.get(i, 0), per_file=int(measures.get(i, 0) / len(files) if files else 0),
                        additional=int(measures.get(i, 0) % len(files) if files else 0)) for i in RELEVANT_MEASURES if
                measures.get(i, 0)}
    for idx, file in enumerate(files):
        for key, details in measures.items():
            value = details['per_file'] + details['additional'] if idx == 0 else details['per_file']
            measure = dict(
                metricKey=key,
                intValue=dict(value=value)
            )
            append_protobuf_to_file(
                file_path=os.path.join(output_dir, f'measures-{file['component_id']}.pb'),
                proto_class=Measure,
                payload=measure,
            )


def generate_project_files(output_dir, file_count, languages, lines, rules):
    files = list()
    if file_count == 0:
        return files
    lines_per_file = int(lines / file_count)
    additional_lines = lines % file_count
    language = [i for i in languages if all(list(rules[i].values()))]
    for component_id, idx in enumerate(range(int(file_count)), start=2):
        path = 'src/{component_id}.txt'.format(component_id=component_id)
        with open(os.path.join(output_dir, f'source-{component_id}.txt'), 'wt') as f:
            total_lines = lines_per_file
            if component_id == 2:
                total_lines += additional_lines
            for idx in range(int(total_lines) - 1):
                f.write(LINE_CONTENT.format(idx=idx))
        component = dict(
            ref=component_id,
            projectRelativePath=path,
            is_test=False,
            type='FILE',
            language=languages[0],
            status='ADDED',
            lines=total_lines,
        )
        write_protobuf_to_file(filepath=os.path.join(output_dir, f'component-{component_id}.pb'), payload=component,
                               proto_class=Component)
        files.append(dict(path=path, component_id=component_id, lines=total_lines, language=languages[0]))
    return files


def generate_issues(output_dir, severities, bugs, code_smells, vulnerabilities, hotspots, quality_counts, files, rules):
    distribution = dict(
        bug=dict(
            per_file=bugs // len(files),
            remainder=bugs % len(files)
        ),
        code_smell=dict(
            per_file=code_smells// len(files),
            remainder=code_smells % len(files)
        ),
        vulnerability=dict(
            per_file=vulnerabilities// len(files),
            remainder=vulnerabilities % len(files)
        ),
        security_hotspot=dict(
            per_file=hotspots // len(files),
            remainder=hotspots % len(files)
        )
    )
    print(distribution)
    for idx, file in enumerate(files):
        for issue_type, counts in distribution.items():
            if issue_type == 'security_hotspot':
                continue
            total = counts['per_file'] + counts['remainder'] if idx == 0 else counts['per_file']
            for issue_idx in range(int(total)):
                issue = {
                    "rule_repository": file['language'],
                    "rule_key": rules[file['language']][issue_type]['key'],
                    "msg": "bad thing",
                    "severity": "INFO",
                    "textRange": {
                        "startLine": issue_idx + 1,
                        "endLine": issue_idx + 1,
                        "startOffset": 1,
                        "endOffset": len(LINE_CONTENT.format(idx=issue_idx))
                    },
                    "flow": [
                        {
                            "location": [
                                {
                                    "textRange": {
                                        "startLine": issue_idx + 1,
                                        "endLine": issue_idx + 1,
                                        "startOffset": 1,
                                        "endOffset": len(LINE_CONTENT.format(idx=issue_idx))
                                    },
                                    "msg": "original implementation"
                                }
                            ]
                        }
                    ],
                    "overriddenImpacts": []
                }
                for severity in sorted(severities.keys()):
                    if severities[severity] == 0:
                        continue
                    issue['severity'] = SEVERITY_MAPPING[severity.replace('_violations', '').upper()]
                    severities[severity] -= 1
                    break
                if issue_type != "security_hotspot":
                    for quality in sorted(quality_counts.keys()):
                        for severity in sorted(quality_counts[quality].keys()):
                            if quality_counts[quality][severity] == 0 or severity == 'total':
                                continue
                            issue['overriddenImpacts'].append({
                                "software_quality": quality.upper(),
                                "severity": "ImpactSeverity_" + severity.upper(),
                            })
                            quality_counts[quality][severity] -= 1
                            break
                append_protobuf_to_file(
                    file_path=os.path.join(output_dir, f'issues-{file['component_id']}.pb'),
                    proto_class=Issue,
                    payload=issue,
                )



def generate_coverage(output_dir, lines_to_cover, covered_lines, conditions, covered_conditions, files):
    pass


def generate_duplications(output_dir, files, duplicated_lines, duplicated_blocks, duplicated_files):
    pass


projects = get_new_projects(extract_directory=directory, mapping=migration_mapping)
profiles = get_profiles(extract_directory=directory, mapping=migration_mapping)
language_rules = get_language_rules(extract_directory=directory, mapping=extract_mapping)
analysis = extract_project_scans(result_dir=result_dir, extract_directory=directory, mapping=extract_mapping,
                                 created_projects=projects, org_profiles=profiles)
history = generate_historical_structure(extract_directory=directory, mapping=extract_mapping, created_projects=projects)
generate_historical_analysis(output_dir=result_dir, historical_analysis=history, rules=language_rules)
