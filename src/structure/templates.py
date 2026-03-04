from utils import multi_extract_object_reader
import re

def add_template(results, org_key, template, server_url, is_default):
    unique_key = org_key + template['id']
    results[unique_key] = dict(
        unique_key=unique_key,
        source_template_key=template['id'],
        name=template['name'],
        description=template.get('description'),
        project_key_pattern=template.get('projectKeyPattern', ''),
        server_url=server_url,
        is_default=is_default,
        sonarqube_org_key=org_key,
    )
    return results

def map_templates(project_org_mapping, extract_mapping, export_directory):
    results = dict()
    org_keys = set(project_org_mapping.values())
    default_templates = set()
    for server_url, template in multi_extract_object_reader(directory=export_directory, mapping=extract_mapping,
                                                            key='getDefaultTemplates'):
        template_key = server_url + template['templateId']
        default_templates.add(template_key)

    for server_url, template in multi_extract_object_reader(directory=export_directory, mapping=extract_mapping,
                                                            key='getTemplates'):
        template_key = server_url + template['id']
        if template_key in default_templates or not template.get('projectKeyPattern', ''):
            for org_key in org_keys:
                results = add_template(results=results, org_key=org_key, template=template, server_url=server_url,
                                       is_default=template_key in default_templates)
        else:
            try:
                regex = re.compile(template['projectKeyPattern'])

                for project_key in project_org_mapping.keys():
                    if regex.match(project_key.replace(server_url, '')):
                        org_key = project_org_mapping[project_key]
                        results = add_template(results=results, org_key=org_key, template=template, server_url=server_url,
                                               is_default=False)
            except Exception:
                continue
    return list(results.values())