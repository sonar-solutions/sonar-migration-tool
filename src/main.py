import json
from datetime import datetime, UTC

from constants import REPORT_TASKS, MIGRATION_TASKS
import click
import os
import asyncio
from execute import execute_plan
from logs import configure_logger
from operations.http_request import configure_client, configure_client_cert, get_server_details
from plan import generate_task_plan, get_available_task_configs
from utils import get_unique_extracts, export_csv, load_csv, filter_completed, get_latest_extract_id
from validate import validate_migration
from importlib import import_module
from pipelines.process import update_pipelines
from config import load_config_file, merge_config_with_cli


@click.group()
def cli():
    pass


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
    # Load config file if provided
    if config_file:
        try:
            config = load_config_file(config_file)
            # Merge CLI args with config file (CLI takes precedence)
            cli_args = {
                'url': url,
                'token': token,
                'export_directory': export_directory,
                'extract_type': extract_type,
                'pem_file_path': pem_file_path,
                'key_file_path': key_file_path,
                'cert_password': cert_password,
                'target_task': target_task,
                'concurrency': concurrency,
                'timeout': timeout,
                'extract_id': extract_id,
            }
            config = merge_config_with_cli(config, cli_args)
            url = config.get('url')
            token = config.get('token')
            export_directory = config.get('export_directory', '/app/files/')
            extract_type = config.get('extract_type', 'all')
            pem_file_path = config.get('pem_file_path')
            key_file_path = config.get('key_file_path')
            cert_password = config.get('cert_password')
            target_task = config.get('target_task')
            concurrency = config.get('concurrency', 25)
            timeout = config.get('timeout', 60)
            extract_id = config.get('extract_id')
        except (FileNotFoundError, json.JSONDecodeError) as e:
            click.echo(f"Error loading config file: {e}")
            return
    else:
        # Set defaults if not using config file
        if export_directory is None:
            export_directory = '/app/files/'
        if extract_type is None:
            extract_type = 'all'
        if concurrency is None:
            concurrency = 25
        if timeout is None:
            timeout = 60

    # Validate required arguments
    if not url or not token:
        click.echo("Error: URL and TOKEN are required (either as arguments or in config file)")
        return

    if not url.endswith('/'):
        url = f"{url}/"
    if extract_id is None:
        extract_id = str(int(datetime.now(UTC).timestamp()))
    cert = configure_client_cert(key_file_path, pem_file_path, cert_password)
    server_version, edition = get_server_details(url=url, cert=cert, token=token)
    extract_directory = os.path.join(export_directory, extract_id + '/')
    os.makedirs(extract_directory, exist_ok=True)
    configure_logger(name='http_request', level='INFO', output_file=os.path.join(extract_directory, 'requests.log'))
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
def migrate(token, edition, url, enterprise_key, concurrency, run_id, export_directory, target_task, config_file):
    """Migrate SonarQube Server configurations to SonarQube Cloud. User must run the structure and mappings command and add the SonarQube Cloud organization keys to the organizations.csv in order to start the migration

    TOKEN is a user token that has admin permissions at the enterprise level and all organizations
    ENTERPRISE_KEY is the key of the SonarQube Cloud enterprise. All migrating organizations must be added to the enterprise

    You can also use --config to specify a JSON configuration file instead of command-line arguments.
    """
    # Load config file if provided
    if config_file:
        try:
            config = load_config_file(config_file)
            # Merge CLI args with config file (CLI takes precedence)
            cli_args = {
                'token': token,
                'enterprise_key': enterprise_key,
                'edition': edition,
                'url': url,
                'run_id': run_id,
                'concurrency': concurrency,
                'export_directory': export_directory,
                'target_task': target_task,
            }
            config = merge_config_with_cli(config, cli_args)
            token = config.get('token')
            enterprise_key = config.get('enterprise_key')
            edition = config.get('edition', 'enterprise')
            url = config.get('url', 'https://sonarcloud.io/')
            run_id = config.get('run_id')
            concurrency = config.get('concurrency', 25)
            export_directory = config.get('export_directory', '/app/files/')
            target_task = config.get('target_task')
        except (FileNotFoundError, json.JSONDecodeError) as e:
            click.echo(f"Error loading config file: {e}")
            return
    else:
        # Set defaults if not using config file
        if edition is None:
            edition = 'enterprise'
        if url is None:
            url = 'https://sonarcloud.io/'
        if concurrency is None:
            concurrency = 25
        if export_directory is None:
            export_directory = '/app/files/'

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
        run_id = str(int(datetime.now(UTC).timestamp()))
        create_plan = True
    run_dir, completed = validate_migration(directory=export_directory, run_id=run_id)
    extract_mapping = get_unique_extracts(directory=export_directory)
    configure_logger(name='http_request', level='INFO', output_file=os.path.join(run_dir, 'requests.log'))
    if target_task is not None:
        target_tasks = [target_task]
    else:
        target_tasks = list(
            [k for k in configs.keys() if not any([k.startswith(i) for i in ['get', 'delete', 'reset']]) ])
    completed = completed.union(MIGRATION_TASKS)
    plan = None
    if create_plan:
        plan = generate_task_plan(
            target_tasks=target_tasks,
            task_configs=configs, completed=completed)
        with open(os.path.join(run_dir, 'migrate.json'), 'wt') as f:
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
        with open(os.path.join(run_dir, 'migrate.json'), 'rt') as f:
            plan = json.load(f)['plan']
    plan = filter_completed(plan=plan, directory=run_dir)
    execute_plan(execution_plan=plan, inputs=dict(url=url, api_url=api_url, enterprise_key=enterprise_key),
                 concurrency=concurrency,
                 task_configs=configs,
                 output_directory=export_directory, current_run_id=run_id,
                 run_ids=set(extract_mapping.values()).union({run_id}))


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
    run_id = str(int(datetime.now(UTC).timestamp()))
    run_dir = os.path.join(export_directory, run_id)
    os.makedirs(run_dir, exist_ok=True)

    configure_logger(name='http_request', level='INFO', output_file=os.path.join(run_dir, 'requests.log'))
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
@click.argument('token')
@click.argument('enterprise_key')
@click.option('--edition', default='enterprise', help="SonarQube Cloud License Edition")
@click.option('--url', default='https://sonarcloud.io/', help="Url of the SonarQube Cloud")
@click.option('--concurrency', default=25, help="Maximum number of concurrent requests")
@click.option('--export_directory', default='/app/files/', help="Directory to place all interim files")
@click.option('--latest_only', default=True, is_flag=True, help="Only process latest scan for each project")
def history(token, edition, url, enterprise_key, concurrency, export_directory, latest_only):
    """Migrate latest scan data for migrated projects to SonarQube Cloud.
    
    Generates scan data using real source code, issues, and measures from extracted data,
    then uploads to SonarQube Cloud.
    
    TOKEN is a user token that has admin permissions at the enterprise level and all organizations
    ENTERPRISE_KEY is the key of the SonarQube Cloud enterprise
    """
    if not url.endswith('/'):
        url = f"{url}/"
    
    # Load migration and extract mappings
    migration_mapping = get_unique_extracts(directory=export_directory, key='migrate.json')
    extract_mapping = get_unique_extracts(directory=export_directory)
    
    if not migration_mapping:
        click.echo("No migration data found. Please run 'migrate' command first.")
        return
    
    if not extract_mapping:
        click.echo("No extract data found. Please run 'extract' command first.")
        return
    
    # Get migrated projects
    from history.projects import get_new_projects
    from history.process import process_project
    
    projects = get_new_projects(export_directory, migration_mapping)
    
    if not projects:
        click.echo("No migrated projects found.")
        return
    
    # Create output directory for scans
    run_id = str(int(datetime.now(UTC).timestamp()))
    output_dir = os.path.join(export_directory, run_id)
    os.makedirs(output_dir, exist_ok=True)
    
    # Configure logging
    configure_logger(name='http_request', level='INFO', output_file=os.path.join(output_dir, 'requests.log'))
    
    # Create semaphore to limit concurrent operations
    semaphore = asyncio.Semaphore(concurrency)
    
    # Process all projects concurrently
    loop = asyncio.get_event_loop()
    tasks = [
        process_project(
            source_project_key, project, semaphore,
            export_directory, extract_mapping, migration_mapping,
            output_dir, token, url
        )
        for source_project_key, project in projects.items()
    ]
    results = loop.run_until_complete(asyncio.gather(*tasks))
    
    # Summary
    successful = sum(1 for r in results if r['status'] == 'success')
    failed = len(results) - successful
    click.echo(f"\nSummary: {successful} successful, {failed} failed")
    
    return results




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
    run_id = str(int(datetime.now(UTC).timestamp()))
    run_dir = os.path.join(output_directory, run_id)
    os.makedirs(run_dir, exist_ok=True)
    loop = asyncio.get_event_loop()
    configure_logger(name='http_request', level='INFO', output_file=os.path.join(pipeline_dir, 'requests.log'))
    results = loop.run_until_complete(
        update_pipelines(
            input_directory=pipeline_dir, output_directory=run_dir, org_secret_mapping=secrets, sonar_token=sonar_token,
            sonar_url=sonar_url
        )
    )
    click.echo(f"Repositories Updated: {list(results.keys())}")


@cli.command()
@click.argument('config_file')
def full_migrate(config_file):
    """Complete end-to-end migration from SonarQube to SonarCloud using a single config file.

    CONFIG_FILE is a JSON file containing all configuration for the migration.

    This command will automatically:
    1. Extract data from SonarQube Server
    2. Generate organization structure
    3. Update organizations with your SonarCloud org key
    4. Generate all mappings
    5. Migrate everything to SonarCloud

    Example config file:
    {
      "sonarqube": {
        "url": "http://localhost:9000",
        "token": "YOUR_TOKEN"
      },
      "sonarcloud": {
        "url": "https://sonarcloud.io/",
        "token": "YOUR_TOKEN",
        "enterprise_key": "YOUR_ENTERPRISE",
        "org_key": "YOUR_ORG"
      },
      "settings": {
        "export_directory": "./files",
        "concurrency": 10,
        "timeout": 60
      }
    }
    """
    try:
        config = load_config_file(config_file)
    except (FileNotFoundError, json.JSONDecodeError) as e:
        click.echo(f"Error loading config file: {e}")
        return

    # Extract configuration values
    try:
        sonarqube_url = config['sonarqube']['url']
        sonarqube_token = config['sonarqube']['token']
        sonarcloud_url = config['sonarcloud']['url']
        sonarcloud_token = config['sonarcloud']['token']
        sonarcloud_enterprise_key = config['sonarcloud']['enterprise_key']
        sonarcloud_org_key = config['sonarcloud']['org_key']
        export_directory = config.get('settings', {}).get('export_directory', './files')
        concurrency = config.get('settings', {}).get('concurrency', 10)
        timeout = config.get('settings', {}).get('timeout', 60)
    except KeyError as e:
        click.echo(f"Error: Missing required configuration key: {e}")
        click.echo("\nRequired structure:")
        click.echo("  sonarqube.url, sonarqube.token")
        click.echo("  sonarcloud.url, sonarcloud.token, sonarcloud.enterprise_key, sonarcloud.org_key")
        return

    # Ensure URLs end with /
    if not sonarqube_url.endswith('/'):
        sonarqube_url = f"{sonarqube_url}/"
    if not sonarcloud_url.endswith('/'):
        sonarcloud_url = f"{sonarcloud_url}/"

    # Create export directory
    os.makedirs(export_directory, exist_ok=True)
    export_dir_abs = os.path.abspath(export_directory)

    click.echo("=" * 60)
    click.echo("SonarQube to SonarCloud Full Migration")
    click.echo("=" * 60)
    click.echo(f"SonarQube URL: {sonarqube_url}")
    click.echo(f"SonarCloud URL: {sonarcloud_url}")
    click.echo(f"SonarCloud Org: {sonarcloud_org_key}")
    click.echo(f"Export Directory: {export_dir_abs}")
    click.echo(f"Concurrency: {concurrency}")
    click.echo("=" * 60)
    click.echo()

    # Step 1: Extract from SonarQube
    click.echo("Step 1/5: Extracting data from SonarQube...")
    extract_id = str(int(datetime.now(UTC).timestamp()))
    cert = configure_client_cert(None, None, None)
    server_version, edition = get_server_details(url=sonarqube_url, cert=cert, token=sonarqube_token)
    extract_directory = os.path.join(export_dir_abs, extract_id + '/')
    os.makedirs(extract_directory, exist_ok=True)
    configure_logger(name='http_request', level='INFO', output_file=os.path.join(extract_directory, 'requests.log'))
    configure_client(url=sonarqube_url, cert=cert, server_version=server_version, token=sonarqube_token,
                     concurrency=concurrency, timeout=timeout)

    configs = get_available_task_configs(client_version=server_version, edition=edition)
    target_tasks = list([k for k in configs.keys() if k.startswith('get')])
    plan = generate_task_plan(target_tasks=target_tasks, task_configs=configs)

    with open(os.path.join(extract_directory, 'extract.json'), 'wt') as f:
        json.dump(dict(plan=plan, version=server_version, edition=edition, url=sonarqube_url,
                      target_tasks=target_tasks, available_configs=list(configs.keys()),
                      run_id=extract_id), f)

    execute_plan(execution_plan=plan, inputs=dict(url=sonarqube_url), concurrency=concurrency,
                 task_configs=configs, output_directory=export_dir_abs, current_run_id=extract_id,
                 run_ids={extract_id})
    click.echo("✓ Data extracted successfully\n")

    # Step 2: Generate structure
    click.echo("Step 2/5: Generating organization structure...")
    from structure import map_organization_structure, map_project_structure
    extract_mapping = get_unique_extracts(directory=export_dir_abs)
    bindings, projects = map_project_structure(export_directory=export_dir_abs, extract_mapping=extract_mapping)
    organizations = map_organization_structure(bindings)
    export_csv(directory=export_dir_abs, name='organizations', data=organizations)
    export_csv(directory=export_dir_abs, name='projects', data=projects)
    click.echo("✓ Organization structure generated\n")

    # Step 3: Update organizations.csv with SonarCloud org key
    click.echo("Step 3/5: Updating organizations with SonarCloud org key...")
    orgs_file = os.path.join(export_dir_abs, 'organizations.csv')
    if not os.path.exists(orgs_file):
        click.echo(f"Error: organizations.csv not found at {orgs_file}")
        return

    # Read and update organizations
    orgs = load_csv(directory=export_dir_abs, filename='organizations.csv')
    for org in orgs:
        org['sonarcloud_org_key'] = sonarcloud_org_key
    export_csv(directory=export_dir_abs, name='organizations', data=orgs)
    click.echo(f"✓ Updated {len(orgs)} organization(s) with org key: {sonarcloud_org_key}\n")

    # Step 4: Generate mappings
    click.echo("Step 4/5: Generating mappings...")
    from structure import map_templates, map_groups, map_profiles, map_gates, map_portfolios
    projects = load_csv(directory=export_dir_abs, filename='projects.csv')
    project_org_mapping = {p['server_url'] + p['key']: p['sonarqube_org_key'] for p in projects}

    mapping = dict(
        templates=map_templates(project_org_mapping=project_org_mapping, extract_mapping=extract_mapping,
                               export_directory=export_dir_abs),
        profiles=map_profiles(extract_mapping=extract_mapping, project_org_mapping=project_org_mapping,
                             export_directory=export_dir_abs),
        gates=map_gates(project_org_mapping=project_org_mapping, extract_mapping=extract_mapping,
                       export_directory=export_dir_abs),
        portfolios=map_portfolios(export_directory=export_dir_abs, extract_mapping=extract_mapping)
    )
    mapping['groups'] = map_groups(project_org_mapping=project_org_mapping, profiles=mapping['profiles'],
                                   extract_mapping=extract_mapping, export_directory=export_dir_abs,
                                   templates=mapping['templates'])

    for k, v in mapping.items():
        export_csv(directory=export_dir_abs, name=k, data=v)
    click.echo("✓ Mappings generated successfully\n")

    # Step 5: Migrate to SonarCloud
    click.echo("Step 5/5: Migrating to SonarCloud...")
    click.echo("This may take several minutes depending on the number of projects...")

    run_id = str(int(datetime.now(UTC).timestamp()))
    run_dir, completed = validate_migration(directory=export_dir_abs, run_id=run_id)

    configure_client(url=sonarcloud_url, cert=None, server_version="cloud", token=sonarcloud_token)
    api_url = sonarcloud_url.replace('https://', 'https://api.')
    configure_client(url=api_url, cert=None, server_version="cloud", token=sonarcloud_token)

    configure_logger(name='http_request', level='INFO', output_file=os.path.join(run_dir, 'requests.log'))

    configs = get_available_task_configs(client_version='cloud', edition='enterprise')
    target_tasks = list([k for k in configs.keys() if not any([k.startswith(i) for i in ['get', 'delete', 'reset']])])
    completed = completed.union(set(MIGRATION_TASKS))

    plan = generate_task_plan(target_tasks=target_tasks, task_configs=configs, completed=completed)

    with open(os.path.join(run_dir, 'plan.json'), 'wt') as f:
        json.dump(dict(plan=plan, version='cloud', edition='enterprise', completed=list(completed),
                      url=sonarcloud_url, target_tasks=target_tasks,
                      available_configs=list(configs.keys()), run_id=run_id), f)

    execute_plan(execution_plan=plan, inputs=dict(url=sonarcloud_url, api_url=api_url,
                 enterprise_key=sonarcloud_enterprise_key), concurrency=concurrency,
                 task_configs=configs, output_directory=export_dir_abs, current_run_id=run_id,
                 run_ids=set(extract_mapping.values()).union({run_id}))

    click.echo("✓ Migration completed successfully\n")

    # Summary
    click.echo("=" * 60)
    click.echo("Migration Complete!")
    click.echo("=" * 60)
    click.echo(f"Export data: {export_dir_abs}")
    click.echo(f"Migration logs: {run_dir}/requests.log")
    click.echo()
    click.echo("View your projects at:")
    click.echo(f"  {sonarcloud_url.rstrip('/')}/organizations/{sonarcloud_org_key}/projects")
    click.echo()
    click.echo("IMPORTANT:")
    click.echo("  • Historical analysis data, issues, and code coverage were NOT migrated")
    click.echo("  • You need to re-scan your projects to populate code and issues")
    click.echo("  • Configure DevOps integrations for automatic analysis")
    click.echo()


if __name__ == '__main__':
    cli()
