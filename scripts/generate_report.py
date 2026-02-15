"""Working version that recreates the existing analysis protobuf files"""
import json
import os
import uuid
from constants import PLUGINS
from collections import defaultdict, OrderedDict
from utils import multi_extract_object_reader, get_unique_extracts
from parser import extract_path_value
from google.protobuf.internal.encoder import _VarintBytes
from datetime import datetime, UTC
from google.protobuf import json_format
from protobufs.scanner_report_pb2 import Metadata, Component, Measure, ActiveRule, Changesets, LineCoverage, Issue, \
    Duplication, LineCoverage
import itertools

component_measures = ['classes', 'functions', 'complexity', 'comment_lines', 'statements', 'ncloc',
                      'cognitive_complexity', 'executable_lines_data', 'ncloc_data']
directory = '../files/'

extract_mapping = get_unique_extracts(directory=directory)

server_projects = defaultdict(lambda: defaultdict(lambda: defaultdict(list)))


def append_protobuf_to_file(file_path, proto):
    with open(file_path, 'ab') as f:
        f.write(_VarintBytes(proto.ByteSize()))
        f.write(proto.SerializeToString())


def extract_project_revisions(extract_directory, mapping):
    project_revisions = dict()
    for url, project in multi_extract_object_reader(directory=extract_directory, mapping=mapping,
                                                    key='getProjects'):
        project_revisions[project['key']] = project.get('revision')
    return project_revisions


def extract_project_details(extract_directory, mapping, project_revisions):
    directories = dict()
    for url, project in multi_extract_object_reader(directory=extract_directory, mapping=mapping,
                                                    key='getProjectDetails'):
        project_dir = os.path.join(extract_directory, 'history/', project['projectKey'])
        if not project.get('analysisDate'):
            continue
        os.makedirs(project_dir, exist_ok=True)
        project['projectKey'] = 'testing-things_' + project['projectKey']
        directories[project_dir] = project
        with open(project_dir + '/' + 'metadata.pb', 'wb') as f:
            m = {
                "analysisDate": int(datetime.fromisoformat(project['analysisDate']).timestamp() * 1000),
                "organizationKey": "testing-things",
                "projectKey": project['projectKey'],
                "rootComponentRef": 1,
                "modulesProjectRelativePathByKey": {
                    project['projectKey']: ""
                },
                "qprofilesPerLanguage": {
                    "js": {
                        "key": "AZg3wtHd0MyPec0bmgNJ",
                        "name": "Sonar way",
                        "language": "js",
                        "rulesUpdatedAt": "1753353909000"
                    },
                    "cs": {
                        "key": "AZg3wtHd0MyPec0bmgNO",
                        "name": "Sonar way",
                        "language": "cs",
                        "rulesUpdatedAt": "1747922547000"
                    },
                    "web": {
                        "key": "AZg3wtHc0MyPec0bmgM6",
                        "name": "Sonar way",
                        "language": "web",
                        "rulesUpdatedAt": "1740590655000"
                    },
                    "jsp": {
                        "key": "AZg3wtHb0MyPec0bmgMn",
                        "name": "Sonar way",
                        "language": "jsp",
                        "rulesUpdatedAt": "1548922322000"
                    },
                    "rust": {
                        "key": "AZg3wtHc0MyPec0bmgNA",
                        "name": "Sonar way",
                        "language": "rust",
                        "rulesUpdatedAt": "1744818210000"
                    },
                    "azureresourcemanager": {
                        "key": "AZg3wtHc0MyPec0bmgMv",
                        "name": "Sonar way",
                        "language": "azureresourcemanager",
                        "rulesUpdatedAt": "1742381308000"
                    },
                    "c": {
                        "key": "AZg3wtHb0MyPec0bmgMr",
                        "name": "Sonar way",
                        "language": "c",
                        "rulesUpdatedAt": "1741088900000"
                    },
                    "flex": {
                        "key": "AZg3wtHb0MyPec0bmgMl",
                        "name": "Sonar way",
                        "language": "flex",
                        "rulesUpdatedAt": "1732614202000"
                    },
                    "json": {
                        "key": "AZg3wtHc0MyPec0bmgM4",
                        "name": "Sonar way",
                        "language": "json",
                        "rulesUpdatedAt": "1753704842000"
                    },
                    "dart": {
                        "key": "AZg3wtHd0MyPec0bmgNI",
                        "name": "Sonar way",
                        "language": "dart",
                        "rulesUpdatedAt": "1751284375000"
                    },
                    "pli": {
                        "key": "AZg3wtHc0MyPec0bmgM_",
                        "name": "Sonar way",
                        "language": "pli",
                        "rulesUpdatedAt": "1734534781000"
                    },
                    "secrets": {
                        "key": "AZg3wtHc0MyPec0bmgM1",
                        "name": "Sonar way",
                        "language": "secrets",
                        "rulesUpdatedAt": "1751986233000"
                    },
                    "ipynb": {
                        "key": "AZg3wtHc0MyPec0bmgMx",
                        "name": "Sonar way",
                        "language": "ipynb",
                        "rulesUpdatedAt": "1757002379000"
                    },
                    "php": {
                        "key": "AZg3wtHc0MyPec0bmgNE",
                        "name": "Sonar way",
                        "language": "php",
                        "rulesUpdatedAt": "1742293210000"
                    },
                    "kotlin": {
                        "key": "AZg3wtHc0MyPec0bmgNF",
                        "name": "Sonar way",
                        "language": "kotlin",
                        "rulesUpdatedAt": "1753961753000"
                    },
                    "ansible": {
                        "key": "AZg3wtHc0MyPec0bmgNC",
                        "name": "Sonar way",
                        "language": "ansible",
                        "rulesUpdatedAt": "1730713223000"
                    },
                    "jcl": {
                        "key": "AZg3wtHc0MyPec0bmgM0",
                        "name": "Sonar way",
                        "language": "jcl",
                        "rulesUpdatedAt": "1741615264000"
                    },
                    "vb": {
                        "key": "AZg3wtHc0MyPec0bmgM7",
                        "name": "Sonar way",
                        "language": "vb",
                        "rulesUpdatedAt": "1748262844000"
                    },
                    "vbnet": {
                        "key": "AZg3wtHc0MyPec0bmgND",
                        "name": "Sonar way",
                        "language": "vbnet",
                        "rulesUpdatedAt": "1753961446000"
                    },
                    "cpp": {
                        "key": "AZg3wtHb0MyPec0bmgMq",
                        "name": "Sonar way",
                        "language": "cpp",
                        "rulesUpdatedAt": "1753361100000"
                    },
                    "plsql": {
                        "key": "AZg3wtHb0MyPec0bmgMp",
                        "name": "Sonar way",
                        "language": "plsql",
                        "rulesUpdatedAt": "1748262768000"
                    },
                    "rpg": {
                        "key": "AZg3wtHc0MyPec0bmgM9",
                        "name": "Sonar way",
                        "language": "rpg",
                        "rulesUpdatedAt": "1701338487000"
                    },
                    "terraform": {
                        "key": "AZg3wtHb0MyPec0bmgMj",
                        "name": "Sonar way",
                        "language": "terraform",
                        "rulesUpdatedAt": "1744210277000"
                    },
                    "py": {
                        "key": "AZg3wtHd0MyPec0bmgNK",
                        "name": "Sonar way",
                        "language": "py",
                        "rulesUpdatedAt": "1757001832000"
                    },
                    "ts": {
                        "key": "AZg3wtHd0MyPec0bmgNM",
                        "name": "Sonar way",
                        "language": "ts",
                        "rulesUpdatedAt": "1756315514000"
                    },
                    "docker": {
                        "key": "AZg3wtHc0MyPec0bmgM8",
                        "name": "Sonar way",
                        "language": "docker",
                        "rulesUpdatedAt": "1742380616000"
                    },
                    "scala": {
                        "key": "AZg3wtHc0MyPec0bmgM3",
                        "name": "Sonar way",
                        "language": "scala",
                        "rulesUpdatedAt": "1740746008000"
                    },
                    "text": {
                        "key": "AZg3wtHb0MyPec0bmgMo",
                        "name": "Sonar way",
                        "language": "text",
                        "rulesUpdatedAt": "1751985825000"
                    },
                    "go": {
                        "key": "AZg3wtHd0MyPec0bmgNH",
                        "name": "Sonar way",
                        "language": "go",
                        "rulesUpdatedAt": "1751524963000"
                    },
                    "tsql": {
                        "key": "AZg3wtHc0MyPec0bmgMy",
                        "name": "Sonar way",
                        "language": "tsql",
                        "rulesUpdatedAt": "1748262850000"
                    },
                    "cobol": {
                        "key": "AZg3wtHc0MyPec0bmgMu",
                        "name": "Sonar way",
                        "language": "cobol",
                        "rulesUpdatedAt": "1746439544000"
                    },
                    "abap": {
                        "key": "AZg3wtHc0MyPec0bmgNB",
                        "name": "Sonar way",
                        "language": "abap",
                        "rulesUpdatedAt": "1734534774000"
                    },
                    "cloudformation": {
                        "key": "AZg3wtHb0MyPec0bmgMi",
                        "name": "Sonar way",
                        "language": "cloudformation",
                        "rulesUpdatedAt": "1744209911000"
                    },
                    "apex": {
                        "key": "AZg3wtHb0MyPec0bmgMm",
                        "name": "Sonar way",
                        "language": "apex",
                        "rulesUpdatedAt": "1741005181000"
                    },
                    "swift": {
                        "key": "AZg3wtHc0MyPec0bmgMz",
                        "name": "Sonar way",
                        "language": "swift",
                        "rulesUpdatedAt": "1740589781000"
                    },
                    "yaml": {
                        "key": "AZg3wtHc0MyPec0bmgM5",
                        "name": "Sonar way",
                        "language": "yaml",
                        "rulesUpdatedAt": "1753705089000"
                    },
                    "css": {
                        "key": "AZg3wtHc0MyPec0bmgMt",
                        "name": "Sonar way",
                        "language": "css",
                        "rulesUpdatedAt": "1756314470000"
                    },
                    "xml": {
                        "key": "AZg3wtHc0MyPec0bmgMw",
                        "name": "Sonar way",
                        "language": "xml",
                        "rulesUpdatedAt": "1744024443000"
                    },
                    "githubactions": {
                        "key": "AZi8VfIkeH2qyaclV_cQ",
                        "name": "Sonar way",
                        "language": "githubactions",
                        "rulesUpdatedAt": "1755506402000"
                    },
                    "java": {
                        "key": "AZg3wtHd0MyPec0bmgNL",
                        "name": "Sonar way",
                        "language": "java",
                        "rulesUpdatedAt": "1753960834000"
                    },
                    "kubernetes": {
                        "key": "AZg3wtHd0MyPec0bmgNG",
                        "name": "Sonar way",
                        "language": "kubernetes",
                        "rulesUpdatedAt": "1755505537000"
                    },
                    "ruby": {
                        "key": "AZg3wtHc0MyPec0bmgM2",
                        "name": "Sonar way",
                        "language": "ruby",
                        "rulesUpdatedAt": "1740731609000"
                    },
                    "objc": {
                        "key": "AZg3wtHc0MyPec0bmgMs",
                        "name": "Sonar way",
                        "language": "objc",
                        "rulesUpdatedAt": "1741089202000"
                    }
                },
                "pluginsByKey": {i['key']: i for i in PLUGINS},
                "projectVersion": project.get('version', '1.0.0'),
                "scmRevisionId": str(uuid.uuid4()),
                "analysisUuid": str(uuid.uuid4()),
                "analysisStartedTimestamp": int(
                    datetime.fromisoformat(project['analysisDate']).timestamp() * 1000) - 1000
            }
            if project_revisions.get(project['projectKey']):
                m['scm_revision_id'] = project_revisions[project['projectKey']]
            metadata = Metadata()
            json_format.Parse(json.dumps(m), metadata)
            f.write(metadata.SerializeToString())

    return directories


def extract_project_components(extract_directory, mapping, directories):
    components = defaultdict(dict)
    for url, component in multi_extract_object_reader(directory=extract_directory, mapping=mapping,
                                                      key='getProjectComponents'):
        if component['qualifier'] not in ['UTS', "FIL"]:
            continue
        components[component['projectKey']][component['path']] = dict(
            ref=len(components[component['projectKey']]) + 2,
            projectRelativePath=component['path'],
            is_test=component['qualifier'] == "UTS",
            type=component['qualifier'],
            language=component.get('language'),
        )

    for k, v in components.items():
        project_dir = os.path.join(extract_directory, 'history/', k)
        with open(os.path.join(project_dir, 'component-1.pb'), 'wb') as f:
            component = Component()
            json_format.Parse(json.dumps(dict(
                ref=1,
                type='PROJECT',
                name=directories[project_dir]['name'],
                childRef=[
                    i['ref'] for i in v.values()
                ],
                key=directories[project_dir]['projectKey']
            )), component)
            f.write(component.SerializeToString())
    return components


def extract_source(extract_directory, mapping, components):
    for url, source in multi_extract_object_reader(directory=extract_directory, mapping=mapping,
                                                   key='getProjectComponentSourceRaw'):
        path = source['projectComponentKey'].replace(source['projectKey'] + ':', '')
        if not components[source['projectKey']].get(path):
            continue
        ref_id = components[source['projectKey']][path]['ref']
        components[source['projectKey']][path]['type'] = 'FILE'
        components[source['projectKey']][path]['status'] = 'ADDED'
        with open(os.path.join(extract_directory, 'history/', source['projectKey'], f'source-{ref_id}.txt'), 'wt') as f:
            try:
                f.write(source['content'])
                components[source['projectKey']][path]['lines'] = len(source['content'].splitlines())
            except Exception as e:
                f.write(json.dumps(
                    {k: v for k, v in source.items() if k not in ['projectComponentKey', 'projectKey', 'serverUrl']}))
                components[source['projectKey']][path]['lines'] = 1
        with open(os.path.join(extract_directory, 'history/', source['projectKey'], f'component-{ref_id}.pb'),
                  'wb') as f:
            component = Component()
            json_format.Parse(json.dumps(components[source['projectKey']][path]), component)
            f.write(component.SerializeToString())


def extract_project_duplications(extract_directory, mapping, components):
    for url, duplications in multi_extract_object_reader(directory=extract_directory, mapping=mapping,
                                                         key='getProjectComponentSourceDuplications'):
        ref_mapping = {
            k: components[duplications['projectKey']].get(v['key'].replace(duplications['projectKey'] + ':', ''))['ref']
            for k, v in
            duplications.get('files', dict()).items()
        }
        path = duplications['projectComponentKey'].replace(duplications['projectKey'] + ':', '')
        if not components[duplications['projectKey']].get(path):
            continue
        ref_id = components[duplications['projectKey']][path]['ref']
        for section in duplications['duplications']:
            duplicate = Duplication()
            d = {
                "originPosition": {
                    "startLine": section['blocks'][0]['from'],
                    "endLine": section['blocks'][0]['from'] + section['blocks'][0]['size'] - 1
                },
                "duplicate": [
                    {
                        "other_file_ref": ref_mapping[block['_ref']] if ref_mapping[block['_ref']] != ref_id else 0,
                        "range": {

                            "startLine": block['from'],
                            "endLine": block['from'] + block['size'] - 1
                        }
                    }
                    for block in section['blocks'][1:]
                ]
            }
            json_format.Parse(json.dumps(d), duplicate)
            append_protobuf_to_file(
                file_path=os.path.join(extract_directory, 'history/', duplications['projectKey'],
                                       f'duplications-{ref_id}.pb'),
                proto=duplicate
            )


def extract_project_coverage(extract_directory, mapping, components):
    for url, coverage in multi_extract_object_reader(directory=extract_directory, mapping=mapping,
                                                     key='getProjectComponentSource'):
        path = coverage['projectComponentKey'].replace(coverage['projectKey'] + ':', '')
        if not components[coverage['projectKey']].get(path):
            continue
        ref_id = components[coverage['projectKey']][path]['ref']
        for line in coverage['sources']:
            if 'lineHits' not in line.keys():
                continue
            c = dict(
                line=line['line'],
                hits=True if line['lineHits'] else False,
            )
            if 'conditions' in line.keys():
                c['conditions'] = line['conditions']
                c['covered_conditions'] = line['coveredConditions']
            cov = LineCoverage()
            json_format.Parse(json.dumps(c), cov)
            append_protobuf_to_file(
                file_path=os.path.join(extract_directory, 'history/', coverage['projectKey'], f'coverages-{ref_id}.pb'),
                proto=cov
            )


def extract_issues(extract_directory, mapping, components, source='getProjectComponentIssues'):
    severity_mapping = {
        "INFO": "INFO",
        "LOW": "INFO",
        "MINOR": "MINOR",
        "MEDIUM": "MINOR",
        "HIGH": "MAJOR",
        "MAJOR": "MAJOR",
        "CRITICAL": "CRITICAL",
        "BLOCKER": "BLOCKER",
    }
    for url, issue in multi_extract_object_reader(directory=extract_directory, mapping=mapping,
                                                  key=source):
        path = issue['projectComponentKey'].replace(issue['projectKey'] + ':', '')
        if not components[issue['projectKey']].get(path):
            continue
        ref_id = components[issue['projectKey']][path]['ref']
        rule_key = 'rule' if 'rule' in issue.keys() else 'ruleKey'
        severity = issue['severity'] if 'severity' in issue.keys() else issue['vulnerabilityProbability']
        issue_json = dict(
            ruleRepository=issue[rule_key].split(':')[0],
            ruleKey=issue[rule_key].split(':')[1],
            msg=issue['message'],
            severity=severity_mapping[severity],
            gap=0.0,
            textRange=issue.get('textRange'),
        )
        if issue.get('gap'):
            issue_json['gap'] = issue['gap']
        if issue.get('flows'):
            issue_json['flow'] = []
            for i in issue['flows']:
                flow = dict(location=[])
                for location in i.get('locations', []):
                    loc = dict(
                        textRange=location['textRange'],
                    )
                    if 'component' in location:
                        path = location['component'].replace(issue['projectKey'] + ':', '')
                        loc['componentRef'] = components[issue['projectKey']][path]['ref']
                    if 'msg' in location:
                        loc['msg'] = location['msg']
                    flow['location'].append(loc)
                if i.get('type'):
                    flow['type'] = i['type']
                if i.get('description'):
                    flow['description'] = i['description']
                issue_json['flow'].append(flow)
        if issue.get('textRange'):
            issue_json['textRange'] = issue['textRange']
        issue_proto = Issue()
        json_format.Parse(json.dumps(issue_json), issue_proto)
        append_protobuf_to_file(
            file_path=os.path.join(extract_directory, 'history/', issue['projectKey'], f'issues-{ref_id}.pb'),
            proto=issue_proto
        )


def extract_component_changesets(extract_directory, mapping, components):
    for url, component in multi_extract_object_reader(directory=extract_directory, mapping=mapping,
                                                      key='getProjectComponentSourceControl'):
        path = component['projectComponentKey'].replace(component['projectKey'] + ':', '')
        if not components[component['projectKey']].get(path):
            continue
        ref_id = components[component['projectKey']][path]['ref']
        lines = []
        revisions = {}
        changeset = Changesets()
        change = dict(
            component_ref=ref_id,
            changeset=[],
            changesetIndexByLine=[]
        )
        for line in component['scm']:
            if len(line) < 4:
                continue
            if line[3] not in revisions:
                revisions[line[3]] = dict(
                    revision=line[3],
                    author=line[1],
                    date=int(datetime.fromisoformat(line[2]).timestamp()*1000)
                )
            lines.append(list(revisions.keys()).index(line[3]))
        change['changeset'] = list(revisions.values())
        change['changesetIndexByLine'] = lines
        with open(os.path.join(extract_directory, 'history/', component['projectKey'], f'changesets-{ref_id}.pb'),
                  'wb') as f:
            json_format.Parse(json.dumps(change), changeset)
            f.write(changeset.SerializeToString())


def extract_component_measures(extract_directory, mapping, components):
    for url, measure in multi_extract_object_reader(directory=extract_directory, mapping=mapping,
                                                    key='getProjectComponentMeasures'):
        path = measure['projectComponentKey'].replace(measure['projectKey'] + ':', '')
        if not components[measure['projectKey']].get(path):
            continue
        ref_id = components[measure['projectKey']][path]['ref']
        if measure['metric'] not in component_measures or not measure.get('value'):
            continue
        m = dict(
            metricKey=measure['metric'],
        )
        try:
            m['intValue'] = dict(value=int(measure['value']))
        except Exception as e:
            m['stringValue'] = dict(value=measure['value'])
        measure_proto = Measure()
        json_format.Parse(json.dumps(m), measure_proto)
        append_protobuf_to_file(
            file_path=os.path.join(extract_directory, 'history/', measure['projectKey'], f'measures-{ref_id}.pb'),
            proto=measure_proto
        )


project_revisions = extract_project_revisions(directory, extract_mapping)
project_directories = extract_project_details(directory, extract_mapping, project_revisions)
project_components = extract_project_components(directory, extract_mapping, project_directories)
extract_source(directory, extract_mapping, project_components)
extract_issues(directory, extract_mapping, project_components)
extract_issues(directory, extract_mapping, project_components, source='getProjectComponentHotspots')
extract_component_measures(directory, extract_mapping, project_components)
extract_project_coverage(directory, extract_mapping, project_components)
extract_project_duplications(directory, extract_mapping, project_components)
extract_component_changesets(directory, extract_mapping, project_components)