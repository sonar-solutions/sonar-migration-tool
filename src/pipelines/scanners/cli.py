from parser import extract_path_value
from pipelines.scanners.base import get_mappings


def get_config_file_name():
    return 'sonar-project.properties',

def _find_mapped_value(key, path, projects, project_mappings):
    for project in projects:
        if project in project_mappings:
            return format_value(key=key, path=path, project=project_mappings[project])
    return None


def update_content(content, projects:set, project_mappings):
    mapping = get_mappings()
    updated_keys = set()
    updated_content = list()
    for i in content.splitlines():
        key = i.split('=')[0]
        if key in mapping:
            mapped = _find_mapped_value(key, mapping[key], projects, project_mappings)
            if mapped is not None:
                updated_content.append(mapped)
                updated_keys.add(key)
                continue
        updated_content.append(i)
    for key, path in mapping.items():
        if key not in updated_keys:
            updated_content.append(_find_mapped_value(key, path, projects, project_mappings) or "")
    is_updated = any(p in project_mappings for p in projects)
    return {'updated_content': "\n".join(updated_content), 'is_updated': is_updated}

def format_value(key, path, project):
    return f"{key}={extract_path_value(obj=project, path=path, default='')}"