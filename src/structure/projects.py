
from utils import multi_extract_object_reader
from parser import extract_path_value
NEW_CODE_MAPPINGS = dict(
    NUMBER_OF_DAYS='days',
    PREVIOUS_VERSION='previous_version',
)


def is_cloud_binding(binding):
    is_cloud = False
    cloud_endpoints = ['dev.azure.com', 'gitlab.com', 'api.github.com', 'bitbucket.org']
    if any([endpoint in binding.get('url', '') for endpoint in cloud_endpoints]):
        is_cloud = True
    return is_cloud


def generate_unique_project_key(server_url, key, alm=None, repository=None, monorepo=False):
    if repository and alm and not monorepo:
        return f"{alm}_{repository}"
    else:
        return server_url + key


def generate_unique_binding_key(server_url, key, alm, url, repository):
    if alm is None or url is None:
        return server_url
    base_url = url.replace('https://', '').replace('http://', '').split('/')[0]
    if alm in ['gitlab']:
        org_key = f"{key} - {server_url}"
    elif alm in ['github']:
        org_key = repository.split("/")[0]
    else:
        url = url.strip('/')
        org_key = url.split("/")[-1]
    return base_url + '/' + org_key


def map_new_code_definitions(export_directory, extract_mapping):
    new_code_definitions = dict()
    for server_url, new_code_definition in multi_extract_object_reader(directory=export_directory,
                                                                       key='getNewCodePeriods',
                                                                       mapping=extract_mapping):
        ncd_type = extract_path_value(obj=new_code_definition, path='$.type')
        ncd_value = extract_path_value(obj=new_code_definition, path='$.value', default=30)
        project_key = extract_path_value(obj=new_code_definition, path='$.projectKey')
        branch_key = extract_path_value(obj=new_code_definition, path='$.branchKey')
        if ncd_type not in NEW_CODE_MAPPINGS.keys():
            continue

        if server_url not in new_code_definitions.keys():
            new_code_definitions[server_url] = dict()

        if project_key not in new_code_definitions[server_url].keys():
            new_code_definitions[server_url][project_key] = dict()
        new_code_definitions[server_url][project_key][branch_key] = dict(
            type=NEW_CODE_MAPPINGS[ncd_type],
            value=ncd_value if ncd_type == 'NUMBER_OF_DAYS' else 'previous_version'
        )
    return new_code_definitions


def map_project_structure(export_directory, extract_mapping):
    projects = dict()
    new_code_definitions = map_new_code_definitions(export_directory=export_directory, extract_mapping=extract_mapping)
    binding_mapping = {server_url + binding['key']: binding for server_url, binding in
                       multi_extract_object_reader(directory=export_directory, key='getBindings',
                                                   mapping=extract_mapping)}
    unique_bindings = dict()
    project_bindings = {
        server_url + project_binding['projectKey']: dict(
            binding=binding_mapping.get(server_url + project_binding['key']), project_binding=project_binding) for
        server_url, project_binding in
        multi_extract_object_reader(directory=export_directory, key='getProjectBindings', mapping=extract_mapping)
    }
    for server_url, project in multi_extract_object_reader(directory=export_directory, key='getProjectDetails',
                                                           mapping=extract_mapping):
        project_binding = project_bindings.get(server_url + project['key'], dict(binding=dict(key=server_url)))
        unique_binding_key = generate_unique_binding_key(
            server_url=server_url,
            key=project_binding['binding']['key'],
            alm=project_binding['binding'].get('alm'),
            url=project_binding['binding'].get('url'),
            repository=project_binding.get('project_binding', dict()).get('repository')
        )
        if unique_binding_key not in unique_bindings.keys():
            unique_bindings[unique_binding_key] = dict(
                key=unique_binding_key,
                alm=project_binding['binding'].get('alm'),
                url=project_binding['binding'].get('url'),
                server_url=server_url,
                is_cloud=is_cloud_binding(binding=project_binding['binding']),
                project_count=0,
            )
        unique_bindings[unique_binding_key]['project_count'] += 1
        unique_project_key = generate_unique_project_key(
            server_url=server_url,
            key=project['key'],
            alm=project_binding.get('alm'),
            repository=project_binding.get('project_binding', dict()).get('repository'),
            monorepo=project_binding.get('project_binding', dict()).get('monorepo', False),
        )
        branch_name = project.get('branch', 'master')
        new_code_definition = new_code_definitions.get(
            server_url, dict()
        ).get(project['key'], dict()).get(branch_name, dict(
                                                                                                               type='days',
                                                                                                               value=30
                                                                                                           ))
        projects[unique_project_key] = dict(
            key=project['key'],
            name=project['name'],
            gate_name=project.get('qualityGate', dict()).get('name'),
            profiles=project.get('qualityProfiles', []),
            server_url=server_url,
            sonarqube_org_key=unique_binding_key,
            main_branch=branch_name,
            is_cloud_binding=is_cloud_binding(binding=project_binding['binding']),
            new_code_definition_type=new_code_definition['type'],
            new_code_definition_value=new_code_definition['value'],
            alm=project_binding['binding'].get('alm'),
            repository=project_binding.get('project_binding', dict()).get('repository'),
            slug=project_binding.get('project_binding', dict()).get('slug'),
            monorepo=project_binding.get('project_binding', dict()).get('monorepo', False),
            summary_comment_enabled=project_binding.get('project_binding', dict()).get('summaryCommentEnabled', False),
        )
    return list(unique_bindings.values()), list(projects.values())
