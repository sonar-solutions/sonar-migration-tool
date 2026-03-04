from collections import defaultdict

from utils import object_reader
import os
def load_dependencies(task, inputs, task_config, concurrency, output_directory, run_ids) -> list:
    from functools import partial
    from itertools import product
    # required_keys = plan_dependency_values(config=task_config, task=task)
    dependencies = [i for i in task_config.get('dependencies', []) if i.get('strategy', 'each') != 'none']
    load_funcs = [
        partial(
            load_dependency,
            dependency=dependency,
            directory=output_directory,
            run_ids=run_ids
            # required_keys=required_keys[dependency['key']]
        )
        for dependency in dependencies
    ]
    chunk = list()
    if len(load_funcs) == 0:
        yield [dict(obj=dict(), inputs=inputs)]
    else:
        if len(load_funcs) > 1:
            loader = partial(product, *[i() for i in load_funcs])
        else:
            loader = load_funcs[0]
        for loaded_dependencies in loader():
            if any(len(dependency) == 0 for dependency in loaded_dependencies):
                continue
            dependency_map = dict(obj=dict(), inputs=inputs)
            if len(load_funcs) == 1:
                loaded_dependencies = [loaded_dependencies]
            for idx, dependency in enumerate(dependencies):
                dependency_map[dependency['key']] = loaded_dependencies[idx]
            chunk.append(dependency_map)
            if len(chunk) >= concurrency:
                yield chunk
                chunk = list()
        if chunk:
            yield chunk

def load_dependency(dependency, directory, run_ids):
    strategy_mapper = {
        'each': load_each,
        'chunk': load_chunk,
        'all': load_all,
        'map': load_map,
        "none": lambda **kwargs: []
    }
    load_strategy = dependency.get('strategy', 'each')
    for results in strategy_mapper[load_strategy](dependency=dependency, directory=directory, run_ids=run_ids):
        yield results


def load_chunk(dependency, directory, run_ids: set[str]):
    objs = list()
    for run_id in run_ids:
        for obj in object_reader(directory=os.path.join(directory, run_id), key=dependency['key']):
            # results = clean_entity(entity=obj, required_keys=required_keys)
            objs.append(obj)
            if len(objs) >= dependency['chunkSize']:
                yield objs
                objs = list()
    if objs:
        yield objs


def load_all(dependency, directory, run_ids: set[str]):
    objs = list()
    for run_id in run_ids:
        for obj in object_reader(directory=os.path.join(directory, run_id), key=dependency['key']):
            objs.append(obj)
    yield objs


def load_each(dependency, directory, run_ids: set[str]):
    for run_id in run_ids:
        for obj in object_reader(directory=os.path.join(directory, run_id), key=dependency['key']):
            yield obj


def load_map(dependency, directory, run_ids: set[str]):
    objs = defaultdict(list)
    for run_id in run_ids:
        for obj in object_reader(directory=os.path.join(directory, run_id),  key=dependency['key']):
            objs[obj[dependency['groupKey']]].append(obj)
    yield objs


def clean_entity(entity, required_keys):
    return {k: v for k, v in entity.items() if k in required_keys}


def find_required_keys(task, field):
    required_keys = defaultdict(set)
    if isinstance(field, dict):
        for k, v in field.items():
            if isinstance(v, dict):
                sub_keys = find_required_keys(field=v, task=task)
                for sub_key, keys in sub_keys.items():
                    required_keys[sub_key] = required_keys[sub_key].union(keys)
            elif k == 'path':
                if 'source' in field.keys():
                    required_keys[field['source']].add(v.replace('$.', '').split('.')[0])
                else:
                    required_keys[task].add(v.replace('$.', '').split('.')[0])
    elif isinstance(field, list):
        for i in field:
            sub_keys = find_required_keys(field=i, task=task)
            for sub_key, keys in sub_keys.items():
                required_keys[sub_key] = required_keys[sub_key].union(keys)
    return required_keys


def plan_dependency_values(config, task):
    field_keys = ['operations', 'prefilters', 'parameters', 'body', 'json']
    required_keys = defaultdict(set)
    for key in field_keys:
        if key in config.keys():
            requirements = find_required_keys(task=task, field=config[key])
            for k, v in requirements.items():
                required_keys[k] = required_keys[k].union(v)
    return required_keys
