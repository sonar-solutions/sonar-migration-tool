import os
from utils import load_csv, export_jsonl
def validate_migration(directory, run_id):
    run_dir = os.path.realpath(os.path.join(directory, run_id)) + '/'
    os.makedirs(run_dir, exist_ok=True)
    completed = set()
    mappings = [
        "organizations",
        'projects',
        'templates',
        'profiles',
        'gates',
        'portfolios',
        'groups'
    ]
    org_mapping = {p['sonarqube_org_key']: p['sonarcloud_org_key'] for p in load_csv(directory=directory, filename='organizations.csv') if p['sonarcloud_org_key']}
    for mapping in mappings:
        validated = []
        name = f'generate{mapping[:-1].capitalize()}Mappings'
        data = load_csv(directory=directory, filename=f'{mapping}.csv')
        if mapping != 'portfolios':
            for i in data:
                if i['sonarqube_org_key'] in org_mapping:
                    i['sonarcloud_org_key'] = org_mapping[i['sonarqube_org_key']]
                    validated.append(i)
        else:
            validated = data
        os.makedirs(f"{run_dir}/{name}", exist_ok=True)
        export_jsonl(directory=run_dir, name=name, data=validated, key=None)
        completed.add(name)
    return run_dir, completed