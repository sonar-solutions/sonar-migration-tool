import io
import xml.etree.ElementTree as ET

from parser import extract_path_value
from pipelines.scanners.base import get_mappings


def get_config_file_name():
    return 'pom.xml',


def update_content(content, projects: set, project_mappings):
    mappings = get_mappings()
    updated_keys = set()

    namespaces = {}
    for _, (prefix, uri) in ET.iterparse(io.StringIO(content), events=['start-ns']):
        namespaces[prefix] = uri
        ET.register_namespace(prefix, uri)

    elements = ET.fromstring(content)
    first_line = ""
    if 'encoding' in content.split('\n')[0]:
        first_line = content.split('\n')[0] + '\n'

    default_ns = namespaces.get('', '')

    for element in elements:
        local_tag = element.tag.split('}')[-1] if '}' in element.tag else element.tag
        if local_tag == 'properties':
            for prop in element:
                prop_local = prop.tag.split('}')[-1] if '}' in prop.tag else prop.tag
                if prop_local in mappings:
                    updated_keys.add(prop_local)
                    for project in projects:
                        if project in project_mappings:
                            prop.text = extract_path_value(
                                obj=project_mappings[project], path=mappings[prop_local], default=''
                            )
            for key, path in mappings.items():
                if key not in updated_keys:
                    tag = ('{' + default_ns + '}' + key) if default_ns else key
                    new_element = ET.Element(tag)
                    for project in projects:
                        if project in project_mappings:
                            new_element.text = extract_path_value(
                                obj=project_mappings[project], path=path, default=''
                            )
                    element.append(new_element)
            break

    ET.indent(elements)
    return {'updated_content': first_line + ET.tostring(elements, encoding='unicode'), 'is_updated': True}
