import os
import json
import csv
import tempfile
from collections import defaultdict
from datetime import datetime, UTC


def _is_safe_subdir(base_dir, name):
    path = os.path.realpath(os.path.join(base_dir, name))
    return path.startswith(base_dir + os.sep) and os.path.isdir(path)


def generate_run_id(directory: str) -> str:
    directory = os.path.realpath(directory)
    allowed_bases = [os.path.realpath(os.getcwd()), os.path.realpath(tempfile.gettempdir())]
    if not any(directory.startswith(b + os.sep) or directory == b for b in allowed_bases):
        raise ValueError(f"Directory must be within the working directory: {directory}")
    today = datetime.now(UTC).strftime('%m-%d-%Y')
    existing = [
        d for d in os.listdir(directory)
        if _is_safe_subdir(directory, d) and d.startswith(today + '-')
    ]
    return f"{today}-{len(existing) + 1:02d}"


def object_reader(directory: str, key: str):
    folder = os.path.join(directory, key)
    if not os.path.exists(folder):
        return []
    for file in os.listdir(folder):
        if not file.endswith('.jsonl'):
            continue
        with open(os.path.join(folder, file), 'rt') as f:
            for row in f.readlines():
                if row:
                    yield json.loads(row)


def get_latest_extract_id(directory):
    dirs = [d for d in os.listdir(directory)
            if os.path.isdir(os.path.join(directory, d))
            and os.path.exists(os.path.join(directory, d, 'extract.json'))]
    return max(dirs) if dirs else None


def get_unique_extracts(directory, key='extract.json'):
    url_mappings = defaultdict(set)
    for extract_id in os.listdir(directory):
        if not os.path.isdir(os.path.join(directory, extract_id)) or not os.path.exists(
                os.path.join(directory, extract_id, key)):
            continue
        with open(os.path.join(directory, extract_id, key), 'rt') as f:
            plan = json.load(f)
            url_mappings[plan['url']].add(extract_id)
    return {k: max(v) for k, v in url_mappings.items()}


def multi_extract_object_reader(directory: str, mapping: dict[str: str], key):
    for url, extract_id in mapping.items():
        for row in object_reader(directory=os.path.join(directory, extract_id), key=key):
            yield url, row


def export_csv(directory, name, data):
    directory = os.path.realpath(directory)
    cwd_base = os.path.realpath(os.getcwd())
    tmp_base = os.path.realpath(tempfile.gettempdir())
    if not directory.startswith(cwd_base + os.sep) and not directory.startswith(tmp_base + os.sep):
        raise ValueError(f"directory must be within the working directory: {directory}")
    csv_path = os.path.realpath(os.path.join(directory, f'{name}.csv'))
    if not csv_path.startswith(directory + os.sep):
        raise ValueError(f"Invalid CSV filename: {name}")
    with open(csv_path, 'wt') as f:
        if data:
            writer = csv.DictWriter(f, fieldnames=data[0].keys())
            writer.writeheader()
            writer.writerows([{k:json.dumps(v) if any([isinstance(v, i) for i in [dict, list, bool]]) else v for k, v in row.items()} for row in data])


def export_jsonl(directory: str, name: str, data: list, idx=0, key='obj'):
    filename = f"{directory}/{name}/results.{idx + 1}.jsonl"
    with open(filename, 'wt') as f:
        for i in data:
            obj = i[key] if key is not None else i
            f.write(json.dumps(obj) + '\n')
    return filename


def load_csv(directory, filename, coerce_types=True):
    if not os.path.exists(os.path.join(directory, filename)):
        return []
    with open(os.path.join(directory, filename), 'rt') as f:
        reader = csv.DictReader(f)
        data = list(reader)
        if coerce_types:
            for row in data:
                for k, v in row.items():
                    if v:
                        try:
                            row[k] = json.loads(v)
                        except json.JSONDecodeError:
                            pass
        return data

def filter_completed(plan, directory:str):
    filtered = list()
    completed = list()
    for phase in plan:
        for task in phase:
            path = os.path.join(directory, task)
            if os.path.exists(path):
                completed.append(task)
            else:
                break
    completed = completed[:-1]
    for phase in plan:
        filtered_phase = []
        for task in phase:
            if task not in completed:
                filtered_phase.append(task)
        if filtered_phase:
            filtered.append(filtered_phase)
    return filtered

def generate_hash_id(data):
    """Generate a consistent uuid for a given input

    :return: a UUID4 formatted string
    """
    import json
    import uuid
    import hashlib
    hash_id = uuid.UUID(
        hashlib.md5(
            str(json.dumps(data, sort_keys=True)).encode('utf-8')
        ).hexdigest()
    )
    return str(hash_id)