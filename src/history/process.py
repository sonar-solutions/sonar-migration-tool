"""Process projects concurrently for history migration"""
import asyncio

import click

from history.generate_latest import generate_latest_scan
from history.upload import upload_scan


async def process_project(source_project_key, project, semaphore, extract_directory, extract_mapping,
                          migration_mapping, output_dir, token, base_url):
    """Process a single project: generate scan and upload"""
    async with semaphore:
        try:
            click.echo(f"Processing project: {project['key']} (source: {source_project_key})")
            
            # Generate latest scan (run in thread pool to avoid blocking)
            loop = asyncio.get_running_loop()
            scan_dir = await loop.run_in_executor(
                None,
                generate_latest_scan,
                extract_directory,
                extract_mapping,
                migration_mapping,
                source_project_key,
                output_dir
            )
            
            # Upload scan
            organization = project['sonarCloudOrgKey']
            await upload_scan(
                scan_dir=scan_dir,
                project_key=project['key'],
                project_name=project['name'],
                organization=organization,
                token=token,
                base_url=base_url,
                timeout=300
            )
            
            click.echo(f"Successfully uploaded scan for {project['key']}")
            return {
                'project_key': project['key'],
                'source_project_key': source_project_key,
                'status': 'success',
                'scan_dir': scan_dir
            }
        except Exception as e:
            click.echo(f"Error processing {project.get('key', source_project_key)}: {str(e)}", err=True)
            return {
                'project_key': project.get('key', 'unknown'),
                'source_project_key': source_project_key,
                'status': 'error',
                'error': str(e)
            }

