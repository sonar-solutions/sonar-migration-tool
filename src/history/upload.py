"""Upload scan data to SonarQube Cloud"""
import os
import tempfile
import zipfile
import httpx


async def upload_scan(scan_dir, project_key, project_name, organization, token, base_url, timeout=300):
    """
    Upload a scan directory to SonarQube Cloud via /api/ce/submit endpoint.
    
    Args:
        scan_dir: Path to the scan directory containing protobuf files
        project_key: Project key in SonarQube Cloud
        project_name: Project name in SonarQube Cloud
        organization: Organization key in SonarQube Cloud
        token: Authentication token
        base_url: Base URL of SonarQube Cloud instance
        timeout: Request timeout in seconds
    
    Returns:
        Response from the API
    
    Raises:
        httpx.HTTPStatusError: If the upload fails
    """
    # Create a temporary zip file
    with tempfile.NamedTemporaryFile(suffix='.zip', delete=False) as temp_zip:
        temp_zip_path = temp_zip.name
    
    try:
        # Create zip file from scan directory
        with zipfile.ZipFile(temp_zip_path, 'w', zipfile.ZIP_DEFLATED) as zipf:
            for root, dirs, files in os.walk(scan_dir):
                for file in files:
                    file_path = os.path.join(root, file)
                    arc_name = os.path.relpath(file_path, scan_dir)
                    zipf.write(file_path, arc_name)
        
        # Upload to SonarQube Cloud
        async with httpx.AsyncClient(base_url=base_url,
                                     headers=dict(Authorization=f'Bearer {token}')) as client:
            with open(temp_zip_path, "rb") as f:
                resp = await client.post(
                    "/api/ce/submit",
                    params={"organization": organization},
                    data={
                        "projectKey": project_key,
                        "projectName": project_name,
                    },
                    files={"report": ("scanner-report.zip", f, "application/zip")},
                    timeout=timeout,
                    headers={"Accept": "application/x-protobuf"},
                )
            resp.raise_for_status()
            return resp
    finally:
        # Clean up temporary zip file
        if os.path.exists(temp_zip_path):
            os.remove(temp_zip_path)

