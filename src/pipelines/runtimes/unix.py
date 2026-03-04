import bashlex
import os

SONAR_COMMANDS = {
    'sonar-scanner': "cli",
    'mvn': "maven",
    'gradle': "gradle"
}


def update_script(script, root_dir, dir_project_mapping):
    updated = list()
    try:
        _, dir_project_mapping = process_command_list(
            root_directory=root_dir,
            parsed=bashlex.parse(script),
            dir_project_mapping=dir_project_mapping
        )
    except Exception:
        pass

    if not dir_project_mapping:
        dir_project_mapping = {
            root_dir: dict(
                projects=set(),
                scanners=set()
            )
        }

    try:
        tokens = list(bashlex.split(script))
    except Exception:
        tokens = []
    for line in tokens:
        include = True
        if 'sonar.projectKey' in line:
            include = False
            if root_dir not in dir_project_mapping:
                dir_project_mapping[root_dir] = {'projects': set(), 'scanners': set()}
            dir_project_mapping[root_dir]['projects'].add(line.split('=')[-1])
        if 'sonar.projectName' in line:
            include = False
        if include:
            updated.append(line)

    updated_script = " ".join(updated).replace('\n ', '\n').replace(' \n', '\n')
    return updated_script, dir_project_mapping


def process_command_list(root_directory, parsed: list, current_dir=None, dir_project_mapping=None):
    if dir_project_mapping is None:
        dir_project_mapping = dict()
    if current_dir is None:
        current_dir = root_directory

    for idx, command in enumerate(parsed):
        if command.kind == 'command':
            current_dir, runs_sonar, project, scanners = process_command_parts(
                parts=command.parts,
                current_dir=current_dir,
                root_directory=root_directory
            )
            if runs_sonar:
                if current_dir not in dir_project_mapping.keys():
                    dir_project_mapping[current_dir] = dict(
                        scanners=set(),
                        projects=set()
                    )
                if project:
                    dir_project_mapping[current_dir]['projects'].add(project)
                if scanners:
                    dir_project_mapping[current_dir]['scanners']=dir_project_mapping[current_dir]['scanners'].union(scanners)
        elif command.kind == 'list':
            current_dir, dir_project_mapping = process_command_list(
                parsed=command.parts,
                root_directory=root_directory,
                current_dir=current_dir,
                dir_project_mapping=dir_project_mapping
            )
    return current_dir, dir_project_mapping


def process_command_parts(parts, current_dir, root_directory):
    change_dir = False
    project = None
    runs_sonar = False
    scanners = set()
    for idx, part in enumerate(parts):
        if part.kind != 'word':
            pass
        elif idx == 0 and part.word == 'cd':
            change_dir = True
        elif idx == 0 and part.word in SONAR_COMMANDS.keys():
            runs_sonar = True
            scanners.add(SONAR_COMMANDS[part.word])
        elif change_dir:
            current_dir = os.path.join(current_dir, part.word)
            change_dir = False
        elif 'sonar.projectKey' in part.word:
            project = part.word.split('=')[-1]
            runs_sonar = True
    if change_dir:
        current_dir = root_directory
    return current_dir, runs_sonar, project, scanners
