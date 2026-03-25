import os
import json

TASK_DIR = os.path.join(os.path.dirname(__file__), './tasks/')


def get_sonarqube_config(client_version, root_dir, edition, files):
    available_versions = [i for i in files if i.endswith('json') and not i.startswith('cloud') and float(
        i.replace('.json', '')) <= client_version]
    if not available_versions:
        return None
    latest_version = max(available_versions, key=lambda x: float(x.replace('.json', '')))
    with open(f'{root_dir}/{latest_version}', 'r') as f:
        config = json.load(f)
    if edition in config.get('editions', ['datacenter', 'developer', 'enterprise', 'community']):
        return config


def get_cloud_config(root_dir):
    cloud_file = os.path.join(root_dir, 'cloud.json')
    config = None
    if os.path.exists(cloud_file):
        with open(cloud_file, 'rt') as f:
            config = json.load(f)
    return config


def get_available_task_configs(client_version, edition):
    available_tasks = dict()
    for root, dirs, files in os.walk(TASK_DIR):
        task = root.split('/')[-1]
        if client_version != 'cloud':
            config = get_sonarqube_config(client_version=client_version, edition=edition, root_dir=root, files=files)
        else:
            config = get_cloud_config(root_dir=root)
        if config is not None:
            available_tasks[task] = config
    return available_tasks


def generate_task_plan(target_tasks, task_configs, completed=None):
    tasks = set()
    if completed is None:
        completed = set()
    for task in target_tasks:
        dependencies = extract_dependencies(task=task, task_configs=task_configs, completed=completed)
        if dependencies is None:
            continue
        tasks.add(task)
        tasks = tasks.union(dependencies)
    return plan_tasks(tasks=tasks, task_configs=task_configs, completed=completed)


def extract_dependencies(task, task_configs, completed):
    dependencies = set()
    entity_config = task_configs.get(task)
    if entity_config is None:
        return None
    for dependency in entity_config.get('dependencies', []):
        if dependency['key'] in completed:
            continue
        nested_dependencies = extract_dependencies(task=dependency['key'], task_configs=task_configs, completed=completed)
        if nested_dependencies is None:
            dependencies = None
            break
        dependencies = dependencies.union(nested_dependencies)
        dependencies.add(dependency['key'])
    return dependencies


def plan_tasks(tasks: set, task_configs: dict, completed: set = None):
    if completed is None:
        completed = set()
    task_plan = list()
    continue_planning = True
    while continue_planning:
        phase = list()
        for extraction in tasks - completed:
            if completed >= set([i['key'] for i in task_configs[extraction].get('dependencies', list())]):
                phase.append(extraction)
        completed = completed.union(set(phase))
        if not phase or len(completed) == len(tasks):
            continue_planning = False
        if phase:
            task_plan.append(phase)
    return task_plan


def filter_plan(plan, output_directory):
    filtered_plan = list()
    for phase in plan:
        filtered_phase = list()
        for task in phase:
            if not os.path.exists(os.path.join(output_directory, task)):
                filtered_phase.append(task)
        if filtered_phase:
            filtered_plan.append(filtered_phase)
    return filtered_plan
