"""Main wizard logic and phase handlers for SonarQube migration"""
import json
import os
from datetime import datetime, UTC

import click

from wizard.state import WizardPhase, WizardState

REQUESTS_LOG = 'requests.log'
ORGANIZATIONS_CSV = 'organizations.csv'
PROJECTS_CSV = 'projects.csv'
SKIPPED_ORG_SENTINEL = 'SKIPPED'
MSG_RETRY_PROMPT = "Would you like to re-enter your information and try again?"

from wizard.prompts import (
    display_welcome,
    display_phase_progress,
    display_phase_start,
    display_phase_complete,
    display_summary,
    display_message,
    display_error,
    display_warning,
    display_success,
    display_resume_info,
    display_wizard_complete,
    prompt_credentials,
    prompt_url,
    prompt_text,
    confirm_action,
    confirm_review,
    display_separator,
)


def run_extract_phase(state: WizardState, export_directory: str) -> WizardState:
    """Collect source URL/token and run extract"""
    from operations.http_request import configure_client, configure_client_cert, get_server_details
    from plan import generate_task_plan, get_available_task_configs
    from execute import execute_plan
    from logs import configure_logger

    display_phase_start(WizardPhase.EXTRACT)

    while True:
        # Collect credentials if not already set
        if not state.source_url:
            state.source_url = prompt_url("SonarQube Server URL (e.g., https://sonarqube.example.com)")

        token = prompt_credentials("SonarQube Server Admin Token")

        if not confirm_review(
            "SonarQube Server information",
            {"URL": state.source_url}
        ):
            state.source_url = None
            continue

        # Optional client certificate
        pem_file_path = None
        key_file_path = None
        cert_password = None
        if confirm_action("Do you need to use a client certificate?", default=False):
            pem_file_path = prompt_text("Path to client certificate PEM file")
            key_file_path = prompt_text("Path to client certificate key file")
            cert_password = prompt_credentials("Certificate password (leave empty if none)", hide_input=True)
            if not cert_password:
                cert_password = None

        # Generate extract ID
        extract_id = str(int(datetime.now(UTC).timestamp()))
        state.extract_id = extract_id

        try:
            cert = configure_client_cert(key_file_path, pem_file_path, cert_password)
            server_version, edition = get_server_details(url=state.source_url, cert=cert, token=token)

            extract_directory = os.path.join(export_directory, extract_id + '/')
            os.makedirs(extract_directory, exist_ok=True)

            configure_logger(name='http_request', level='INFO',
                             output_file=os.path.join(extract_directory, REQUESTS_LOG))
            configure_client(url=state.source_url, cert=cert, server_version=server_version,
                             token=token, concurrency=25, timeout=60)

            configs = get_available_task_configs(client_version=server_version, edition=edition)
            target_tasks = [k for k in configs.keys() if k.startswith('get')]

            plan = generate_task_plan(target_tasks=target_tasks, task_configs=configs)

            with open(os.path.join(extract_directory, 'extract.json'), 'wt') as f:
                json.dump(
                    {
                        "plan": plan,
                        "version": server_version,
                        "edition": edition,
                        "url": state.source_url,
                        "target_tasks": target_tasks,
                        "available_configs": list(configs.keys()),
                        "run_id": extract_id,
                    }, f
                )

            execute_plan(execution_plan=plan, inputs={"url": state.source_url}, concurrency=25,
                         task_configs=configs, output_directory=export_directory,
                         current_run_id=extract_id, run_ids={extract_id})

            display_success(f"Extract complete: {extract_id}")
            state.phase = WizardPhase.STRUCTURE
            state.save(export_directory)
            break

        except Exception as e:
            display_error(f"Extract failed: {str(e)}")
            if confirm_action(MSG_RETRY_PROMPT, default=True):
                state.source_url = None
                continue
            raise

    display_phase_complete(WizardPhase.EXTRACT)
    return state


def run_structure_phase(state: WizardState, export_directory: str) -> WizardState:
    """Run structure analysis to identify organizations"""
    from structure import map_organization_structure, map_project_structure
    from utils import get_unique_extracts, export_csv

    display_phase_start(WizardPhase.STRUCTURE)

    try:
        extract_mapping = get_unique_extracts(directory=export_directory)
        if not extract_mapping:
            display_error("No extracts found. Please run extract first.")
            raise ValueError("No extracts found")

        bindings, projects = map_project_structure(export_directory=export_directory,
                                                   extract_mapping=extract_mapping)
        organizations = map_organization_structure(bindings)

        export_csv(directory=export_directory, name='organizations', data=organizations)
        export_csv(directory=export_directory, name='projects', data=projects)

        display_summary("Structure Analysis Results", {
            "Organizations identified": len(organizations),
            "Projects mapped": len(projects),
            "Source servers": len(extract_mapping),
        })

        state.phase = WizardPhase.ORG_MAPPING
        state.save(export_directory)

    except Exception as e:
        display_error(f"Structure analysis failed: {str(e)}")
        raise

    display_phase_complete(WizardPhase.STRUCTURE)
    return state


def _process_org_mapping(org: dict) -> None:
    """Display info for a single unmapped org and prompt the user to migrate or skip it."""
    org_key = org.get('sonarqube_org_key', 'Unknown')
    display_message(f"  Organization: {org_key}")
    display_message(f"    Server URL: {org.get('server_url', 'Unknown')}")
    display_message(f"    DevOps Binding: {org.get('devops_binding', 'None')}")
    display_message(f"    Projects: {org.get('project_count', 0)}")
    display_message("")

    if confirm_action(f"Migrate '{org_key}' and all its projects?", default=True):
        cloud_key = prompt_text(f"Enter SonarCloud organization key for '{org_key}'")
        org['sonarcloud_org_key'] = cloud_key
        display_separator()
    else:
        org['sonarcloud_org_key'] = SKIPPED_ORG_SENTINEL
        display_warning(
            f"Organization '{org_key}' "
            f"and its {org.get('project_count', 0)} project(s) will be skipped"
        )
        display_separator()


def _map_all_organizations(organizations: list, export_directory: str) -> None:
    """Iterate organizations, prompt for any unmapped ones, then export the result."""
    from utils import export_csv

    unmapped = [org for org in organizations if not org.get('sonarcloud_org_key')]

    if not unmapped:
        display_message("All organizations are already mapped.")
        return

    display_message(f"Found {len(unmapped)} organization(s) to map:")
    display_message("")

    for org in organizations:
        current_cloud_key = org.get('sonarcloud_org_key', '')
        if not current_cloud_key:
            _process_org_mapping(org)
        elif current_cloud_key == SKIPPED_ORG_SENTINEL:
            display_message(f"  {org.get('sonarqube_org_key')} -> SKIPPED")
        else:
            display_message(f"  {org.get('sonarqube_org_key')} -> {current_cloud_key} (already mapped)")

    export_csv(directory=export_directory, name='organizations', data=organizations)
    display_success("Organization mappings saved")


def run_org_mapping_phase(state: WizardState, export_directory: str) -> WizardState:
    """Guide user through organization mapping to SonarQube Cloud"""
    from utils import load_csv

    display_phase_start(WizardPhase.ORG_MAPPING)

    while True:
        # Collect SonarQube Cloud credentials
        if not state.target_url:
            state.target_url = prompt_url("SonarQube Cloud URL (e.g., https://sonarcloud.io/)", default="https://sonarcloud.io/")

        if not state.enterprise_key:
            state.enterprise_key = prompt_text("SonarQube Cloud Enterprise Key")

        if not confirm_review(
            "SonarQube Cloud information",
            {
                "URL           ": state.target_url,
                "Enterprise Key": state.enterprise_key,
            }
        ):
            state.target_url = None
            state.enterprise_key = None
            continue

        display_message("")
        display_message("Organization Mapping")
        display_message("-" * 40)
        display_message("You need to map each SonarQube Server organization to a")
        display_message("SonarQube Cloud organization key.")
        display_message("")

        try:
            organizations = load_csv(directory=export_directory, filename=ORGANIZATIONS_CSV)

            if not organizations:
                display_error("No organizations found. Please run structure analysis first.")
                raise ValueError("No organizations found")

            _map_all_organizations(organizations, export_directory)

            state.organizations_mapped = True
            state.phase = WizardPhase.MAPPINGS
            state.save(export_directory)
            break

        except Exception as e:
            display_error(f"Organization mapping failed: {str(e)}")
            if confirm_action(MSG_RETRY_PROMPT, default=True):
                state.target_url = None
                state.enterprise_key = None
                continue
            raise

    display_phase_complete(WizardPhase.ORG_MAPPING)
    return state


def run_mappings_phase(state: WizardState, export_directory: str) -> WizardState:
    """Run mappings command to map entities to organizations"""
    from structure import map_templates, map_groups, map_profiles, map_gates, map_portfolios
    from utils import get_unique_extracts, load_csv, export_csv

    display_phase_start(WizardPhase.MAPPINGS)

    try:
        extract_mapping = get_unique_extracts(directory=export_directory)
        organizations = load_csv(directory=export_directory, filename=ORGANIZATIONS_CSV)
        skipped_org_keys = {
            org['sonarqube_org_key'] for org in organizations
            if org.get('sonarcloud_org_key') == SKIPPED_ORG_SENTINEL
        }

        projects = load_csv(directory=export_directory, filename=PROJECTS_CSV)
        all_project_count = len(projects)
        projects = [p for p in projects if p['sonarqube_org_key'] not in skipped_org_keys]
        project_org_mapping = {p['server_url'] + p['key']: p['sonarqube_org_key'] for p in projects}

        if skipped_org_keys:
            skipped_project_count = all_project_count - len(projects)
            display_warning(
                f"Skipping {len(skipped_org_keys)} organization(s) "
                f"and {skipped_project_count} associated project(s)"
            )

        mapping = {
            "templates": map_templates(project_org_mapping=project_org_mapping,
                                       extract_mapping=extract_mapping,
                                       export_directory=export_directory),
            "profiles": map_profiles(extract_mapping=extract_mapping,
                                     project_org_mapping=project_org_mapping,
                                     export_directory=export_directory),
            "gates": map_gates(project_org_mapping=project_org_mapping,
                               extract_mapping=extract_mapping,
                               export_directory=export_directory),
            "portfolios": map_portfolios(export_directory=export_directory,
                                         extract_mapping=extract_mapping),
        }
        mapping['groups'] = map_groups(project_org_mapping=project_org_mapping,
                                       profiles=mapping['profiles'],
                                       extract_mapping=extract_mapping,
                                       export_directory=export_directory,
                                       templates=mapping['templates'])

        for k, v in mapping.items():
            export_csv(directory=export_directory, name=k, data=v)

        display_summary("Entity Mappings Generated", {
            "Permission Templates": len(mapping['templates']),
            "Quality Profiles": len(mapping['profiles']),
            "Quality Gates": len(mapping['gates']),
            "Portfolios": len(mapping['portfolios']),
            "Groups": len(mapping['groups']),
        })

        state.phase = WizardPhase.VALIDATE
        state.save(export_directory)

    except Exception as e:
        display_error(f"Mappings generation failed: {str(e)}")
        raise

    display_phase_complete(WizardPhase.MAPPINGS)
    return state


def run_validate_phase(state: WizardState, export_directory: str) -> WizardState:
    """Run pre-flight validation"""
    from utils import load_csv

    display_phase_start(WizardPhase.VALIDATE)

    # Check required files exist
    required_files = [ORGANIZATIONS_CSV, PROJECTS_CSV, 'templates.csv',
                      'profiles.csv', 'gates.csv', 'groups.csv']

    missing_files = []
    for filename in required_files:
        filepath = os.path.join(export_directory, filename)
        if not os.path.exists(filepath):
            missing_files.append(filename)

    if missing_files:
        display_error(f"Missing required files: {', '.join(missing_files)}")
        raise ValueError(f"Missing required files: {', '.join(missing_files)}")

    # Check all organizations have cloud mappings
    organizations = load_csv(directory=export_directory, filename=ORGANIZATIONS_CSV)
    unmapped_orgs = [org['sonarqube_org_key'] for org in organizations
                    if not org.get('sonarcloud_org_key')]

    if unmapped_orgs:
        display_error(f"Unmapped organizations: {', '.join(unmapped_orgs)}")
        display_message("Please complete organization mapping before proceeding.")
        raise ValueError(f"Unmapped organizations found: {', '.join(unmapped_orgs)}")

    # Count entities to migrate
    skipped_org_keys = {
        org['sonarqube_org_key'] for org in organizations
        if org.get('sonarcloud_org_key') == SKIPPED_ORG_SENTINEL
    }
    active_orgs = [org for org in organizations
                   if org.get('sonarcloud_org_key') != SKIPPED_ORG_SENTINEL]

    projects = load_csv(directory=export_directory, filename=PROJECTS_CSV)
    active_projects = [p for p in projects if p.get('sonarqube_org_key') not in skipped_org_keys]
    profiles = load_csv(directory=export_directory, filename='profiles.csv')
    templates = load_csv(directory=export_directory, filename='templates.csv')
    gates = load_csv(directory=export_directory, filename='gates.csv')
    groups = load_csv(directory=export_directory, filename='groups.csv')

    summary = {
        "Organizations": f"{len(active_orgs)} active" + (f", {len(skipped_org_keys)} skipped" if skipped_org_keys else ""),
        "Projects": f"{len(active_projects)} active" + (f", {len(projects) - len(active_projects)} skipped" if skipped_org_keys else ""),
        "Quality Profiles": len(profiles),
        "Permission Templates": len(templates),
        "Quality Gates": len(gates),
        "Groups": len(groups),
    }

    display_summary("Migration Summary", summary)

    display_success("Validation passed!")
    state.validation_passed = True
    state.phase = WizardPhase.MIGRATE
    state.save(export_directory)

    display_phase_complete(WizardPhase.VALIDATE)
    return state


def run_migrate_phase(state: WizardState, export_directory: str) -> WizardState:
    """Execute migration with confirmation"""
    from operations.http_request import configure_client
    from plan import generate_task_plan, get_available_task_configs
    from execute import execute_plan
    from validate import validate_migration
    from utils import get_unique_extracts, filter_completed
    from logs import configure_logger
    from constants import MIGRATION_TASKS

    display_phase_start(WizardPhase.MIGRATE)

    display_warning("This will migrate configurations to SonarQube Cloud.")
    display_warning("Make sure you have backed up any existing data.")
    display_message("")

    if not confirm_action("Do you want to proceed with the migration?", default=False):
        display_message("Migration cancelled.")
        display_message("Your progress has been saved. Run the wizard again to resume.")
        raise SystemExit(0)

    while True:
        # Get cloud token
        token = prompt_credentials("SonarQube Cloud Admin Token")

        try:
            run_id = str(int(datetime.now(UTC).timestamp()))
            state.migration_run_id = run_id

            url = state.target_url
            api_url = url.replace('https://', 'https://api.')

            configure_client(url=url, cert=None, server_version="cloud", token=token)
            configure_client(url=api_url, cert=None, server_version="cloud", token=token)

            configs = get_available_task_configs(client_version='cloud', edition='enterprise')

            run_dir, completed = validate_migration(directory=export_directory, run_id=run_id)
            extract_mapping = get_unique_extracts(directory=export_directory)

            configure_logger(name='http_request', level='INFO',
                             output_file=os.path.join(run_dir, REQUESTS_LOG))

            target_tasks = [k for k in configs.keys()
                            if not any(k.startswith(i) for i in ('get', 'delete', 'reset'))]
            completed = completed.union(MIGRATION_TASKS)

            plan = generate_task_plan(target_tasks=target_tasks, task_configs=configs, completed=completed)

            with open(os.path.join(run_dir, 'plan.json'), 'wt') as f:
                json.dump(
                    {
                        "plan": plan,
                        "version": "cloud",
                        "edition": "enterprise",
                        "completed": list(completed),
                        "url": url,
                        "target_tasks": target_tasks,
                        "available_configs": list(configs.keys()),
                        "run_id": run_id,
                    }, f
                )

            plan = filter_completed(plan=plan, directory=run_dir)

            execute_plan(execution_plan=plan,
                         inputs={"url": url, "api_url": api_url, "enterprise_key": state.enterprise_key},
                         concurrency=25,
                         task_configs=configs,
                         output_directory=export_directory,
                         current_run_id=run_id,
                         run_ids=set(extract_mapping.values()).union({run_id}))

            try:
                from analysis_report import generate_final_analysis_report
                report_rows = generate_final_analysis_report(run_directory=run_dir)
                if report_rows:
                    success_count = sum(1 for r in report_rows if r['outcome'] == 'success')
                    failure_count = sum(1 for r in report_rows if r['outcome'] == 'failure')
                    display_summary("Final Analysis Report", {
                        "Location": os.path.join(run_dir, "final_analysis_report.csv"),
                        "Total API calls": len(report_rows),
                        "Successful": success_count,
                        "Failed": failure_count,
                    })
            except Exception:
                display_warning("Could not generate final analysis report.")

            display_success(f"Migration complete! Run ID: {run_id}")
            state.phase = WizardPhase.PIPELINES
            state.save(export_directory)
            break

        except Exception as e:
            display_error(f"Migration failed: {str(e)}")
            if confirm_action(MSG_RETRY_PROMPT, default=True):
                continue
            raise

    display_phase_complete(WizardPhase.MIGRATE)
    return state


def run_pipelines_phase(state: WizardState, export_directory: str) -> WizardState:
    """Optional pipeline updates"""
    import asyncio
    from pipelines.process import update_pipelines
    from utils import get_unique_extracts
    from logs import configure_logger

    display_phase_start(WizardPhase.PIPELINES)

    display_message("This optional step updates CI/CD pipeline configurations")
    display_message("to point to SonarQube Cloud.")
    display_message("")

    # Check for secrets file
    secrets_file = 'secrets.json'
    secrets_path = os.path.join(export_directory, secrets_file)

    if not os.path.exists(secrets_path):
        display_message(f"No secrets file found at {secrets_path}")
        display_message("Skipping pipeline updates.")
        display_message("")
        display_message("To update pipelines later, create a secrets.json file with")
        display_message("your DevOps platform credentials and run the pipelines command.")

        if confirm_action("Skip pipeline updates and complete the wizard?", default=True):
            state.phase = WizardPhase.COMPLETE
            state.save(export_directory)
            display_phase_complete(WizardPhase.PIPELINES)
            return state

    if not confirm_action("Do you want to update CI/CD pipelines?", default=False):
        display_message("Pipeline updates skipped.")
        state.phase = WizardPhase.COMPLETE
        state.save(export_directory)
        display_phase_complete(WizardPhase.PIPELINES)
        return state

    try:
        with open(secrets_path, 'rt') as f:
            secrets = json.load(f)

        sonar_token = prompt_credentials("SonarQube Cloud Token for pipelines")
        sonar_url = state.target_url

        extract_mapping = get_unique_extracts(directory=export_directory, key='plan.json')

        if extract_mapping:
            pipeline_dir = os.path.join(export_directory, str(max(extract_mapping.values())))
        elif os.path.exists(os.path.join(export_directory, 'generateOrganizationMappings')):
            pipeline_dir = export_directory
        else:
            display_error("No migration data found for pipeline updates")
            raise ValueError("No migration data found")

        run_id = str(int(datetime.now(UTC).timestamp()))
        run_dir = os.path.join(export_directory, run_id)
        os.makedirs(run_dir, exist_ok=True)

        configure_logger(name='http_request', level='INFO',
                         output_file=os.path.join(pipeline_dir, REQUESTS_LOG))

        results = asyncio.run(
            update_pipelines(
                input_directory=pipeline_dir,
                output_directory=run_dir,
                org_secret_mapping=secrets,
                sonar_token=sonar_token,
                sonar_url=sonar_url
            )
        )

        display_success(f"Repositories updated: {list(results.keys())}")

    except Exception as e:
        display_error(f"Pipeline update failed: {str(e)}")
        if not confirm_action("Continue to complete the wizard despite the error?", default=True):
            raise

    state.phase = WizardPhase.COMPLETE
    state.save(export_directory)

    display_phase_complete(WizardPhase.PIPELINES)
    return state


PHASE_HANDLERS = {
    WizardPhase.INIT: lambda state, d: state,  # No-op, just transition
    WizardPhase.EXTRACT: run_extract_phase,
    WizardPhase.STRUCTURE: run_structure_phase,
    WizardPhase.ORG_MAPPING: run_org_mapping_phase,
    WizardPhase.MAPPINGS: run_mappings_phase,
    WizardPhase.VALIDATE: run_validate_phase,
    WizardPhase.MIGRATE: run_migrate_phase,
    WizardPhase.PIPELINES: run_pipelines_phase,
}

PHASE_SEQUENCE = [
    WizardPhase.EXTRACT,
    WizardPhase.STRUCTURE,
    WizardPhase.ORG_MAPPING,
    WizardPhase.MAPPINGS,
    WizardPhase.VALIDATE,
    WizardPhase.MIGRATE,
    WizardPhase.PIPELINES,
    WizardPhase.COMPLETE,
]


def get_next_phase(current_phase: WizardPhase) -> WizardPhase | None:
    """Get the next phase in sequence"""
    if current_phase == WizardPhase.COMPLETE:
        return None
    try:
        current_idx = PHASE_SEQUENCE.index(current_phase)
        return PHASE_SEQUENCE[current_idx + 1] if current_idx + 1 < len(PHASE_SEQUENCE) else None
    except ValueError:
        return PHASE_SEQUENCE[0]


def _handle_resume(state: WizardState, export_directory: str) -> WizardState | None:
    """Handle resume/restart/cancel logic for an existing session.

    Returns the (possibly reset) state, or None if the user cancelled.
    """
    if state.phase == WizardPhase.INIT:
        return state

    display_resume_info(state)
    if confirm_action("Resume from previous session?", default=True):
        display_message("Resuming wizard...")
        return state

    if confirm_action("Start a new wizard session? (This will overwrite previous state)", default=False):
        state = WizardState(phase=WizardPhase.INIT)
        state.save(export_directory)
        return state

    display_message("Wizard cancelled.")
    return None


def _determine_starting_phase(state: WizardState, export_directory: str) -> tuple[WizardState, WizardPhase | None]:
    """Determine which phase to start from based on current state.

    Returns updated state and the starting phase, or None to exit.
    """
    if state.phase == WizardPhase.INIT:
        return state, WizardPhase.EXTRACT

    if state.phase == WizardPhase.COMPLETE:
        display_wizard_complete()
        if confirm_action("Start a new migration?", default=False):
            state = WizardState(phase=WizardPhase.INIT)
            state.save(export_directory)
            return state, WizardPhase.EXTRACT
        return state, None

    return state, state.phase


@click.command()
@click.option('--export_directory', default='/app/files/',
              help="Root directory for export files and wizard state")
def wizard(export_directory):
    """Interactive wizard for SonarQube Server to SonarQube Cloud migration"""
    # Ensure export directory exists
    os.makedirs(export_directory, exist_ok=True)

    # Display welcome
    display_welcome()

    # Load existing state or start fresh
    state = WizardState.load(export_directory)

    try:
        # Check for resume
        state = _handle_resume(state, export_directory)
        if state is None:
            return

        # Determine starting phase
        state, current_phase = _determine_starting_phase(state, export_directory)
        if current_phase is None:
            return

        # Run phases
        while current_phase and current_phase != WizardPhase.COMPLETE:
            display_phase_progress(current_phase)

            handler = PHASE_HANDLERS.get(current_phase)
            if handler:
                state = handler(state, export_directory)

            current_phase = state.phase

        # Complete
        state.phase = WizardPhase.COMPLETE
        state.save(export_directory)
        display_wizard_complete()

    except (KeyboardInterrupt, click.Abort, EOFError):
        display_message("")
        display_message("Wizard interrupted. Your progress has been saved.")
        display_message(f"Run the wizard again to resume from: {state.phase.value}")
        state.save(export_directory)

    except Exception as e:
        display_error(f"Wizard failed: {str(e)}")
        display_message("Your progress has been saved.")
        display_message(f"Run the wizard again to resume from: {state.phase.value}")
        state.save(export_directory)
        raise
