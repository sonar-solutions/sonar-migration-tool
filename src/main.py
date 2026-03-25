import json
import tempfile

from constants import MIGRATION_TASKS
import click
import os
import asyncio
from execute import execute_plan
from logs import configure_logger
from operations.http_request import configure_client, configure_client_cert, get_server_details
from plan import generate_task_plan, get_available_task_configs
from utils import get_unique_extracts, export_csv, load_csv, filter_completed, generate_run_id
from validate import validate_migration
from importlib import import_module
from pipelines.process import update_pipelines
from config import load_config_file, merge_config_with_cli
from wizard import wizard as wizard_command

REQUESTS_LOG = 'requests.log'
DEFAULT_EXPORT_DIR = '/app/files/'


@click.group()
def cli():
    pass


cli.add_command(wizard_command)


def _build_extract_params(config_file, url, token, export_directory, extract_type,
                           pem_file_path, key_file_path, cert_password, target_task,
                           concurrency, timeout, extract_id):
    if config_file:
        config = load_config_file(config_file)
        cli_args = {
            'url': url, 'token': token, 'export_directory': export_directory,
            'extract_type': extract_type, 'pem_file_path': pem_file_path,
            'key_file_path': key_file_path, 'cert_password': cert_password,
            'target_task': target_task, 'concurrency': concurrency,
            'timeout': timeout, 'extract_id': extract_id,
        }
        config = merge_config_with_cli(config, cli_args)
        return {
            'url': config.get('url'), 'token': config.get('token'),
            'export_directory': config.get('export_directory', DEFAULT_EXPORT_DIR),
            'extract_type': config.get('extract_type', 'all'),
            'pem_file_path': config.get('pem_file_path'),
            'key_file_path': config.get('key_file_path'),
            'cert_password': config.get('cert_password'),
            'target_task': config.get('target_task'),
            'concurrency': config.get('concurrency', 25),
            'timeout': config.get('timeout', 60),
            'extract_id': config.get('extract_id'),
        }
    return {
        'url': url, 'token': token,
        'export_directory': export_directory or DEFAULT_EXPORT_DIR,
        'extract_type': extract_type or 'all',
        'pem_file_path': pem_file_path, 'key_file_path': key_file_path,
        'cert_password': cert_password, 'target_task': target_task,
        'concurrency': concurrency or 25, 'timeout': timeout or 60,
        'extract_id': extract_id,
    }


@cli.command()
@click.argument('url', required=False)
@click.argument('token', required=False)
@click.option('--config', 'config_file', help="Path to JSON configuration file")
@click.option('--pem_file_path', help="Path to client certificate pem file")
@click.option('--key_file_path', help="Path to client certificate key file")
@click.option('--cert_password', help="Password for client certificate")
@click.option('--export_directory', default=None, help="Root Directory to output the export")
@click.option('--extract_type', default=None, help='Type of Extract to run')
@click.option('--concurrency', default=None, type=int, help='Maximum number of concurrent requests')
@click.option('--timeout', default=None, type=int, help='Number of seconds before a request will timeout')
@click.option('--extract_id',
              help='ID of an extract to resume in case of failures. Extract will start by retrying last completed task')
@click.option('--target_task', help='Target Task to complete. All dependent tasks will be included')
def extract(url, token, config_file, export_directory: str, extract_type, pem_file_path, key_file_path, cert_password, target_task,
            concurrency, timeout, extract_id):
    """Extracts data from a SonarQube Server instance and stores it in the export directory as new line delimited json files

    URL is the url of the SonarQube instance

    TOKEN is an admin user token to the SonarQube instance

    You can also use --config to specify a JSON configuration file instead of command-line arguments.
    """
    try:
        p = _build_extract_params(config_file, url, token, export_directory, extract_type,
                                   pem_file_path, key_file_path, cert_password, target_task,
                                   concurrency, timeout, extract_id)
    except (FileNotFoundError, ValueError) as e:
        click.echo(f"Error loading config file: {e}")
        return
    url, token = p['url'], p['token']
    export_directory, extract_type = p['export_directory'], p['extract_type']
    pem_file_path, key_file_path, cert_password = p['pem_file_path'], p['key_file_path'], p['cert_password']
    target_task, concurrency, timeout, extract_id = p['target_task'], p['concurrency'], p['timeout'], p['extract_id']

    # Validate required arguments
    if not url or not token:
        click.echo("Error: URL and TOKEN are required (either as arguments or in config file)")
        return

    if not url.endswith('/'):
        url = f"{url}/"
    export_directory = os.path.realpath(export_directory)
    _allowed = [os.path.realpath(os.getcwd()), os.path.realpath(tempfile.gettempdir())]
    if not any(export_directory.startswith(b + os.sep) or export_directory == b for b in _allowed):
        click.echo("Error: export_directory must be within the working directory")
        return
    if extract_id is None:
        extract_id = generate_run_id(export_directory)
    cert = configure_client_cert(key_file_path, pem_file_path, cert_password)
    try:
        server_version, edition = get_server_details(url=url, cert=cert, token=token)
    except PermissionError as e:
        click.echo(f"Error: {e}")
        return
    extract_directory = os.path.join(export_directory, extract_id + '/')
    os.makedirs(extract_directory, exist_ok=True)
    configure_logger(name='http_request', level='INFO', output_file=os.path.join(extract_directory, REQUESTS_LOG))
    configure_client(url=url, cert=cert, server_version=server_version, token=token, concurrency=concurrency,
                     timeout=timeout)
    configs = get_available_task_configs(client_version=server_version, edition=edition)
    if target_task is not None:
        target_tasks = [target_task]
    elif extract_type == 'all':
        target_tasks = list([k for k in configs.keys() if k.startswith('get')])
    else:
        try:
            module = import_module(f'report.{extract_type}')
        except ImportError:
            click.echo(f"Report Type {extract_type} Not Found")
            return
        else:
            target_tasks = module.REQUIRED

    plan = generate_task_plan(target_tasks=target_tasks, task_configs=configs)
    with open(os.path.join(extract_directory, 'extract.json'), 'wt') as f:
        json.dump(
            dict(
                plan=plan,
                version=server_version,
                edition=edition,
                url=url,
                target_tasks=target_tasks,
                available_configs=list(configs.keys()),
                run_id=extract_id,
            ), f
        )
    execute_plan(execution_plan=plan, inputs=dict(url=url), concurrency=concurrency, task_configs=configs,
                 output_directory=export_directory, current_run_id=extract_id, run_ids={extract_id})
    click.echo(f"Extract Complete: {extract_id}")


@cli.command()
@click.option('--export_directory', default='/app/files/',
              help="Root Directory containing all of the SonarQube exports")
@click.option('--report_type', default='migration', help='Type of report to generate')
@click.option('--filename', default=None, help='Filename for the report')
def report(export_directory, report_type, filename):
    """Generates a markdown report based on data extracted from one or more SonarQube Server instances"""
    from importlib import import_module
    try:
        module = import_module(f'report.{report_type}.generate')
    except ImportError:
        click.echo(f"Report Type {report_type} Not Found")
        return
    extract_mapping = get_unique_extracts(directory=export_directory)
    if not extract_mapping:
        click.echo("No Extracts Found")
        return
    md = module.generate_markdown(extract_directory=export_directory, extract_mapping=extract_mapping)
    filename = filename if filename else report_type
    with open(os.path.join(export_directory, f'{filename}.md'), 'wt') as f:
        f.write(md)
    return md


@cli.command()
@click.option('--export_directory', default='/app/files/',
              help="Root Directory containing all of the SonarQube exports")
def structure(export_directory):
    """Groups projects into organizations based on DevOps Bindings and Server Urls. Outputs organizations and projects as CSVs"""
    from structure import map_organization_structure, map_project_structure
    extract_mapping = get_unique_extracts(directory=export_directory)
    bindings, projects = map_project_structure(export_directory=export_directory, extract_mapping=extract_mapping)
    organizations = map_organization_structure(bindings)
    export_csv(directory=export_directory, name='organizations', data=organizations)
    export_csv(directory=export_directory, name='projects', data=projects)


@cli.command()
@click.option('--export_directory', default='/app/files/',
              help="Root Directory containing all of the SonarQube exports")
def mappings(export_directory):
    """Maps groups, permission templates, quality profiles, quality gates, and portfolios to relevant organizations. Outputs CSVs for each entity type"""
    from structure import map_templates, map_groups, map_profiles, map_gates, map_portfolios
    extract_mapping = get_unique_extracts(directory=export_directory)
    projects = load_csv(directory=export_directory, filename='projects.csv')
    project_org_mapping = {p['server_url'] + p['key']: p['sonarqube_org_key'] for p in projects}
    mapping = dict(
        templates=map_templates(project_org_mapping=project_org_mapping, extract_mapping=extract_mapping,
                                export_directory=export_directory),
        profiles=map_profiles(extract_mapping=extract_mapping, project_org_mapping=project_org_mapping,
                              export_directory=export_directory),
        gates=map_gates(project_org_mapping=project_org_mapping, extract_mapping=extract_mapping,
                        export_directory=export_directory),
        portfolios=map_portfolios(export_directory=export_directory, extract_mapping=extract_mapping)
    )
    mapping['groups'] = map_groups(project_org_mapping=project_org_mapping, profiles=mapping['profiles'],
                                   extract_mapping=extract_mapping, export_directory=export_directory,
                                   templates=mapping['templates'])
    for k, v in mapping.items():
        export_csv(directory=export_directory, name=k, data=v)


def _build_migrate_params(config_file, token, enterprise_key, edition, url, run_id,
                           concurrency, export_directory, target_task, skip_profiles):
    if config_file:
        config = load_config_file(config_file)
        cli_args = {
            'token': token, 'enterprise_key': enterprise_key, 'edition': edition,
            'url': url, 'run_id': run_id, 'concurrency': concurrency,
            'export_directory': export_directory, 'target_task': target_task,
            'skip_profiles': skip_profiles if skip_profiles else None,
        }
        config = merge_config_with_cli(config, cli_args)
        return {
            'token': config.get('token'),
            'enterprise_key': config.get('enterprise_key'),
            'edition': config.get('edition', 'enterprise'),
            'url': config.get('url', 'https://sonarcloud.io/'),
            'run_id': config.get('run_id'),
            'concurrency': config.get('concurrency', 25),
            'export_directory': config.get('export_directory', DEFAULT_EXPORT_DIR),
            'target_task': config.get('target_task'),
            'skip_profiles': config.get('skip_profiles', False),
        }
    return {
        'token': token, 'enterprise_key': enterprise_key,
        'edition': edition or 'enterprise',
        'url': url or 'https://sonarcloud.io/',
        'run_id': run_id,
        'concurrency': concurrency or 25,
        'export_directory': export_directory or DEFAULT_EXPORT_DIR,
        'target_task': target_task,
        'skip_profiles': skip_profiles,
    }


@cli.command()
@click.argument('token', required=False)
@click.argument('enterprise_key', required=False)
@click.option('--config', 'config_file', help="Path to JSON configuration file")
@click.option('--edition', default=None)
@click.option('--url', default=None)
@click.option('--run_id', default=None,
              help='ID of a run to resume in case of failures. Migration will resume by retrying the last completed task')
@click.option('--concurrency', default=None, type=int, help='Maximum number of concurrent requests')
@click.option('--export_directory', default=None,
              help="Root Directory containing all of the SonarQube exports")
@click.option('--target_task',
              help='Name of a specific migration task to complete. All dependent tasks will be be included')
@click.option('--skip_profiles', is_flag=True, default=False,
              help='Skip quality profile migration/provisioning in SonarQube Cloud')
def migrate(token, edition, url, enterprise_key, concurrency, run_id, export_directory, target_task, config_file, skip_profiles):
    """Migrate SonarQube Server configurations to SonarQube Cloud. User must run the structure and mappings command and add the SonarQube Cloud organization keys to the organizations.csv in order to start the migration

    TOKEN is a user token that has admin permissions at the enterprise level and all organizations
    ENTERPRISE_KEY is the key of the SonarQube Cloud enterprise. All migrating organizations must be added to the enterprise

    You can also use --config to specify a JSON configuration file instead of command-line arguments.
    """
    # Load config file if provided
    try:
        p = _build_migrate_params(config_file, token, enterprise_key, edition, url, run_id,
                                   concurrency, export_directory, target_task, skip_profiles)
    except (FileNotFoundError, ValueError) as e:
        click.echo(f"Error loading config file: {e}")
        return
    token, enterprise_key = p['token'], p['enterprise_key']
    edition, url, run_id = p['edition'], p['url'], p['run_id']
    concurrency, export_directory = p['concurrency'], p['export_directory']
    target_task, skip_profiles = p['target_task'], p['skip_profiles']
    export_directory = os.path.realpath(export_directory)

    # Validate required arguments
    if not token or not enterprise_key:
        click.echo("Error: TOKEN and ENTERPRISE_KEY are required (either as arguments or in config file)")
        return

    create_plan = False
    configure_client(url=url, cert=None, server_version="cloud", token=token)
    api_url = url.replace('https://', 'https://api.')
    configure_client(url=api_url, cert=None, server_version="cloud", token=token)
    configs = get_available_task_configs(client_version='cloud', edition=edition)
    if run_id is None:
        run_id = generate_run_id(export_directory)
        create_plan = True
    run_dir, completed = validate_migration(directory=export_directory, run_id=run_id)
    extract_mapping = get_unique_extracts(directory=export_directory)
    configure_logger(name='http_request', level='INFO', output_file=os.path.join(run_dir, REQUESTS_LOG))
    if target_task is not None:
        target_tasks = [target_task]
    else:
        target_tasks = list(
            [k for k in configs.keys() if not any([k.startswith(i) for i in ['get', 'delete', 'reset']])])
    if skip_profiles:
        target_tasks = [t for t in target_tasks if 'Profile' not in t and 'profile' not in t]
    completed = completed.union(MIGRATION_TASKS)
    if create_plan:
        plan = generate_task_plan(
            target_tasks=target_tasks,
            task_configs=configs, completed=completed)
        with open(os.path.join(run_dir, 'plan.json'), 'wt') as f:
            json.dump(
                dict(
                    plan=plan,
                    version='cloud',
                    edition=edition,
                    completed=list(completed),
                    url=url,
                    target_tasks=target_tasks,
                    available_configs=list(configs.keys()),
                    run_id=run_id,
                ), f
            )
    else:
        with open(os.path.join(run_dir, 'plan.json'), 'rt') as f:
            plan = json.load(f)['plan']
    plan = filter_completed(plan=plan, directory=run_dir)
    execute_plan(execution_plan=plan, inputs=dict(url=url, api_url=api_url, enterprise_key=enterprise_key),
                 concurrency=concurrency,
                 task_configs=configs,
                 output_directory=export_directory, current_run_id=run_id,
                 run_ids=set(extract_mapping.values()).union({run_id}))

    try:
        from analysis_report import generate_final_analysis_report
        run_dir_real = os.path.realpath(run_dir)
        cwd_base = os.path.realpath(os.getcwd())
        if not (run_dir_real.startswith(cwd_base + os.sep) or run_dir_real == cwd_base):
            raise ValueError("run_dir is outside the working directory")
        report_rows = generate_final_analysis_report(run_directory=run_dir_real)
        if report_rows:
            success_count = sum(1 for r in report_rows if r['outcome'] == 'success')
            failure_count = sum(1 for r in report_rows if r['outcome'] == 'failure')
            click.echo(f"Final Analysis Report: {os.path.join(run_dir, 'final_analysis_report.csv')}")
            click.echo(f"  Total API calls: {len(report_rows)}, Successful: {success_count}, Failed: {failure_count}")
    except Exception:
        click.echo("Warning: Could not generate final analysis report.")


@cli.command()
@click.argument('token')
@click.argument('enterprise_key')
@click.option('--edition', default='enterprise', help="SonarQube Cloud License Edition")
@click.option('--url', default='https://sonarcloud.io/', help="Url of the SonarQube Cloud")
@click.option('--concurrency', default=25, help="Maximum number of concurrent requests")
@click.option('--export_directory', default='/app/files/', help="Directory to place all interim files")
def reset(token, edition, url, enterprise_key, concurrency, export_directory):
    """Resets a SonarQube cloud Enterprise back to its original state. Warning, this will delete everything in every organization within the enterprise.

    TOKEN is a user token that has admin permissions at the enterprise level and all organizations

    ENTERPRISE_KEY is the key of the SonarQube Cloud enterprise that will be reset.

    """

    configs = get_available_task_configs(client_version='cloud', edition=edition)
    if not url.endswith('/'):
        url = f"{url}/"
    configure_client(url=url, cert=None, server_version="cloud", token=token)
    api_url = url.replace('https://', 'https://api.')
    configure_client(url=api_url, cert=None, server_version="cloud", token=token)
    run_id = generate_run_id(export_directory)
    run_dir = os.path.join(export_directory, run_id)
    os.makedirs(run_dir, exist_ok=True)

    configure_logger(name='http_request', level='INFO', output_file=os.path.join(run_dir, REQUESTS_LOG))
    target_tasks = list([k for k in configs.keys() if k.startswith('delete')])
    plan = generate_task_plan(
        target_tasks=target_tasks,
        task_configs=configs)
    with open(os.path.join(run_dir, 'clear.json'), 'wt') as f:
        json.dump(
            dict(
                plan=plan,
                version='cloud',
                edition=edition,
                enterprise_key=enterprise_key,
                url=url,
                target_tasks=target_tasks,
                available_configs=list(configs.keys()),
                run_id=run_id,
            ), f
        )
    execute_plan(execution_plan=plan, inputs=dict(url=url, api_url=api_url, enterprise_key=enterprise_key),
                 concurrency=concurrency,
                 task_configs=configs,
                 output_directory=export_directory, current_run_id=run_id,
                 run_ids={run_id})


@cli.command()
@click.argument('secrets_file')
@click.argument('sonar_token')
@click.argument('sonar_url')
@click.option('--input_directory', default='/app/files/', help="Directory to find all migration files")
@click.option('--output_directory', default=None, help="Directory to place all interim files")
def pipelines(secrets_file, sonar_token, sonar_url, input_directory, output_directory):
    with open(os.path.join(input_directory, secrets_file), 'rt') as f:
        secrets = json.load(f)
    extract_mapping = get_unique_extracts(directory=input_directory, key='plan.json')

    if output_directory is None:
        output_directory = input_directory
    if extract_mapping:
        pipeline_dir = os.path.join(input_directory, str(max(extract_mapping.values())))
    elif os.path.exists(os.path.join(input_directory, 'generateOrganizationMappings')):
        pipeline_dir = input_directory
    else:
        click.echo("No Migrations Found")
        return
    run_id = generate_run_id(output_directory)
    run_dir = os.path.join(output_directory, run_id)
    os.makedirs(run_dir, exist_ok=True)
    configure_logger(name='http_request', level='INFO', output_file=os.path.join(pipeline_dir, REQUESTS_LOG))
    results = asyncio.run(
        update_pipelines(
            input_directory=pipeline_dir, output_directory=run_dir, org_secret_mapping=secrets, sonar_token=sonar_token,
            sonar_url=sonar_url
        )
    )
    click.echo(f"Repositories Updated: {list(results.keys())}")


@cli.command()
@click.argument('run_id')
@click.option('--export_directory', default='/app/files/',
              help="Root Directory containing all of the SonarQube exports")
def analysis_report(run_id, export_directory):
    """Generate a final analysis report CSV from a migration run's requests.log.

    RUN_ID is the ID of the migration run to analyze
    """
    from analysis_report import generate_final_analysis_report
    run_dir = os.path.join(export_directory, run_id)
    if not os.path.isdir(run_dir):
        click.echo(f"Run directory not found: {run_dir}")
        return
    report_rows = generate_final_analysis_report(run_directory=run_dir)
    if report_rows:
        success_count = sum(1 for r in report_rows if r['outcome'] == 'success')
        failure_count = sum(1 for r in report_rows if r['outcome'] == 'failure')
        click.echo(f"Final Analysis Report: {os.path.join(run_dir, 'final_analysis_report.csv')}")
        click.echo(f"  Total API calls: {len(report_rows)}, Successful: {success_count}, Failed: {failure_count}")
    else:
        click.echo("No POST requests found in requests.log")


if __name__ == '__main__':
    cli()
