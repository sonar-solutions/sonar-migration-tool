"""Load migrated projects, profiles, and rules"""
from collections import defaultdict

from utils import multi_extract_object_reader


def get_new_projects(extract_directory, mapping):
    """Get migrated projects from createProjects"""
    projects = {}
    for url, project in multi_extract_object_reader(directory=extract_directory, mapping=mapping,
                                                    key='createProjects'):
        projects[project['sourceProjectKey']] = project
    return projects


def get_profiles(extract_directory, mapping):
    """Get profiles from pullAllProfiles"""
    org_profiles = defaultdict(lambda: defaultdict(dict))
    for url, profile in multi_extract_object_reader(directory=extract_directory, mapping=mapping,
                                                    key='pullAllProfiles'):
        org_profiles[profile['sonarCloudOrgKey']][profile['language']][profile['name']] = profile
    return org_profiles


def get_language_rules(extract_directory, mapping):
    """Get rules from getRuleDetails"""
    language_rules = defaultdict(lambda: {
        'bug': None,
        'vulnerability': None,
        'code_smell': None,
        'security_hotspot': None
    })
    for url, rule in multi_extract_object_reader(directory=extract_directory, mapping=mapping,
                                                 key='getRuleDetails'):
        if rule.get('templateKey') or rule.get('isTemplate'):
            continue
        language_rules[rule['lang']][rule['type'].lower()] = {
            'key': rule['key'].split(':')[-1],
            'rule_repository': rule['repo'],
            'severity': rule['severity'],
        }
    return language_rules

