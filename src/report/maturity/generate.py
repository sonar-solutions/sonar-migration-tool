from report.common import generate_server_markdown, generate_devops_markdown, generate_pipeline_markdown, \
    generate_plugin_markdown, generate_profile_markdown, \
    generate_permission_template_markdown
from report.common.measures import process_project_measures
from report.common.users import generate_user_markdown
from report.common.tokens import generate_token_markdown
from .coverage import generate_coverage_markdown
from .gates import generate_gate_maturity_markdown
from .ide import generate_ide_markdown
from .issues import generate_issue_markdown
from .portfolios import generate_portfolio_summary_markdown
from .scans import generate_scans_markdown
from .usage import generate_usage_markdown
from .languages import generate_language_markdown
from .profiles import generate_profile_summary
from .permissions import generate_permissions_markdown
from ..common.tasks import generate_task_markdown
from ..common.webhooks import generate_webhook_markdown

TEMPLATE = """
# SonarQube Maturity Assessment

## Table of Contents

* Adoption
    * Instances
    * DevOps Integrations
    * CI Pipeline Overview
    * Usage
    * User Management
* Governance
    * Detected Project Groupings
    * Languages
    * Profiles
    * Gates
    * Active Gates
    * Permissions
    * Portfolios
* Workflow Integration
    * Scans
    * Issues
    * Testing
    * IDE
* Automation
    * API Usage
    * Webhooks

## Adoption
{instances}

{devops}

{pipeline}

{usage}

{user_management}

## Governance
{project_groupings}

{languages}

{profiles}

{gates}

{active_gates}

{permissions}

{portfolios}

## Workflow Integration

{scans}

{issues}

{vulnerabilities}

{bugs}

{code_smells}

{testing}

{ide}

## Automation

{tokens}

{webhooks}

## Appendix

{plugins}

{tasks}
"""


def generate_markdown(extract_directory, extract_mapping):
    server_markdown, server_id_mapping, projects = generate_server_markdown(directory=extract_directory,
                                                                            extract_mapping=extract_mapping)
    pipeline_overview, project_scan_details, scans = generate_pipeline_markdown(
        directory=extract_directory,
        extract_mapping=extract_mapping,
        server_id_mapping=server_id_mapping
    )
    plugin_md, plugins = generate_plugin_markdown(directory=extract_directory, extract_mapping=extract_mapping,
                                                  server_id_mapping=server_id_mapping)
    devops_markdown, pull_requests = generate_devops_markdown(directory=extract_directory,
                                                              extract_mapping=extract_mapping,
                                                              server_id_mapping=server_id_mapping)
    measures = process_project_measures(directory=extract_directory, extract_mapping=extract_mapping,
                                        server_id_mapping=server_id_mapping)
    permissions, permission_templates = generate_permission_template_markdown(directory=extract_directory,
                                                                              extract_mapping=extract_mapping,
                                                                              server_id_mapping=server_id_mapping,
                                                                              projects=projects, only_active=True)
    _, _, profile_map, _ = generate_profile_markdown(directory=extract_directory, extract_mapping=extract_mapping,
                                                    server_id_mapping=server_id_mapping, projects=projects,
                                                    plugins=plugins)
    gate_summary, gate_details = generate_gate_maturity_markdown(directory=extract_directory,
                                                                 extract_mapping=extract_mapping,
                                                                 server_id_mapping=server_id_mapping, projects=projects)
    language_md, languages = generate_language_markdown(measures=measures, profile_map=profile_map)
    profile_markdown = generate_profile_summary(profile_map=profile_map, languages=languages)
    user_md, users, groups = generate_user_markdown(directory=extract_directory, extract_mapping=extract_mapping,
                                                    server_id_mapping=server_id_mapping)
    permissions_md, global_permissions = generate_permissions_markdown(extract_directory=extract_directory,
                                                                       extract_mapping=extract_mapping)
    usage_md = generate_usage_markdown(projects=projects, scans=scans)
    scan_md = generate_scans_markdown(project_scans=scans)
    issue_overview_md, vulnerability_md, bug_md, code_smell_md = generate_issue_markdown(
        extract_directory=extract_directory,
        extract_mapping=extract_mapping,
        server_id_mapping=server_id_mapping
    )
    ide = generate_ide_markdown(users=users)
    coverage = generate_coverage_markdown(measures=measures)
    portfolios = generate_portfolio_summary_markdown(directory=extract_directory, extract_mapping=extract_mapping,
                                                     server_id_mapping=server_id_mapping)
    token_md = generate_token_markdown(directory=extract_directory, extract_mapping=extract_mapping, server_id_mapping=server_id_mapping)
    task_md = generate_task_markdown(directory=extract_directory, extract_mapping=extract_mapping, server_id_mapping=server_id_mapping)
    webhooks_md = generate_webhook_markdown(directory=extract_directory, extract_mapping=extract_mapping, server_id_mapping=server_id_mapping)
    return TEMPLATE.format(
        # Adoption
        instances=server_markdown,
        devops=devops_markdown,
        pipeline=pipeline_overview,
        usage=usage_md,
        user_management=user_md,
        # Governance
        project_groupings=permissions,
        languages=language_md,
        profiles=profile_markdown,
        gates=gate_summary,
        active_gates=gate_details,
        permissions=permissions_md,
        # Workflow Integration
        scans=scan_md,
        issues=issue_overview_md,
        vulnerabilities=vulnerability_md,
        bugs=bug_md,
        code_smells=code_smell_md,
        testing=coverage,
        ide=ide,
        # Automation
        tokens=token_md,
        webhooks=webhooks_md,
        # Reporting
        portfolios=portfolios,
        plugins=plugin_md,
        tasks=task_md
    )
