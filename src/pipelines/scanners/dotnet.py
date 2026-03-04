def get_config_file_name():
    return ('sonar-project.properties',)


def update_content(content, _projects, _project_mappings):
    return {'updated_content': content, 'is_updated': False}
