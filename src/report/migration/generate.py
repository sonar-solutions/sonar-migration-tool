from report.common import generate_server_markdown, generate_devops_markdown, generate_pipeline_markdown, \
    generate_application_markdown, generate_plugin_markdown, generate_profile_markdown, \
    generate_project_metrics_markdown, generate_gate_markdown, generate_portfolio_markdown, generate_permission_template_markdown

TEMPLATE = """
# SonarQube Utilization Assessment

## Table of Contents

* Instance Overview
* Devops Integrations
* CI Environment Overview
* Permissions
* Governance
    * Installed Plugins
    * Custom Quality Profiles
    * Portfolios
    * Applications
* Installed Plugins
* Project Metrics
* Appendix

{instance_overview}
{devops_integrations}
{pipeline_overview}

## Governance
{permission_templates}

{active_quality_profiles}

{active_quality_gates}

{active_portfolios}

{active_applications}

{plugins}

{project_metrics}

## Appendix
{project_scan_details}

{unused_quality_gates}

{unused_quality_profiles}

{empty_portfolios}

{empty_applications}

"""

def generate_markdown(extract_directory, extract_mapping):
    server_markdown, server_id_mapping,  projects = generate_server_markdown(directory=extract_directory, extract_mapping=extract_mapping)
    pipeline_overview, project_scan_details, project_scans = generate_pipeline_markdown(
        directory=extract_directory,
        extract_mapping=extract_mapping,
        server_id_mapping=server_id_mapping
    )
    devops_markdown, *_ = generate_devops_markdown(directory=extract_directory, extract_mapping=extract_mapping,
                                               server_id_mapping=server_id_mapping)
    permissions, permission_templates = generate_permission_template_markdown(directory=extract_directory, extract_mapping=extract_mapping,
                                       server_id_mapping=server_id_mapping, projects=projects)

    plugins_md, plugins = generate_plugin_markdown(directory=extract_directory, extract_mapping=extract_mapping,
                                       server_id_mapping=server_id_mapping)
    active_profiles, inactive_profiles, _, projects = generate_profile_markdown(
        directory=extract_directory,
        extract_mapping=extract_mapping,
        server_id_mapping=server_id_mapping,
        projects=projects,
        plugins=plugins
    )
    active_portfolios, inactive_portfolios = generate_portfolio_markdown(directory=extract_directory,
        extract_mapping=extract_mapping,
        server_id_mapping=server_id_mapping)
    active_gates, inactive_gates = generate_gate_markdown(directory=extract_directory,
        extract_mapping=extract_mapping,
        server_id_mapping=server_id_mapping,
        projects=projects)

    active_app_md, inactive_app_md = generate_application_markdown(
        directory=extract_directory,
        extract_mapping=extract_mapping,
        server_id_mapping=server_id_mapping
    )
    md = TEMPLATE.format(
        instance_overview=server_markdown,
        devops_integrations=devops_markdown,
        pipeline_overview=pipeline_overview,
        active_applications=active_app_md,
        permission_templates=permissions,
        active_quality_profiles=active_profiles,
        active_quality_gates=active_gates,
        active_portfolios=active_portfolios,
        plugins=plugins_md,
        project_metrics=generate_project_metrics_markdown(projects=projects, project_scans=project_scans),
        project_scan_details=project_scan_details,
        unused_quality_gates=inactive_gates,
        unused_quality_profiles=inactive_profiles,
        empty_portfolios=inactive_portfolios,
        empty_applications=inactive_app_md,
    )
    return md