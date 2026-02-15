import time
from datetime import datetime
import uuid
from protobufs.scanner_report_pb2 import Metadata, Component, Measure, ActiveRule, Changesets, LineCoverage, Issue, Duplication
from scripts.generate_history import result_dir
import shutil

required = [
    "changesets", # git blame information
    'activerules', # used for issue active issues views
    'component', # files and projects
    'coverages', # coverage details
    'duplications', # target
    'issues', #required
    'measures', #required
    'metadata', #required
    'source', #required
]



import httpx
from pathlib import Path
import tempfile
import zipfile
import os
for project_key in os.listdir(result_dir):
    for scan in sorted(os.listdir(os.path.join(result_dir, project_key)))[6:]:
        zip_path = os.path.join(result_dir, project_key, scan)
        # shutil.copy('/Users/perry.stallings/Code/sonar-reports/activerules.pb', zip_path)
        # Create a temporary zip file
        with tempfile.NamedTemporaryFile(suffix='.zip', delete=False) as temp_zip:
            temp_zip_path = temp_zip.name

        with zipfile.ZipFile(temp_zip_path, 'w', zipfile.ZIP_DEFLATED) as zipf:
            for root, dirs, files in os.walk(zip_path):
                for file in files:
                    file_path = os.path.join(root, file)
                    arc_name = os.path.relpath(file_path, zip_path)
                    zipf.write(file_path, arc_name)

        async with httpx.AsyncClient(base_url="https://sc-staging.io",
                                     headers=dict(
                                         Authorization='Bearer ')) as client:
            with open(temp_zip_path, "rb") as f:
                resp = await client.post(
                    "/api/ce/submit",
                    params={"organization": "do-more-thing"},
                    data={
                        "projectKey": project_key,
                        "projectName": project_key,
                    },
                    files={"report": ("scanner-report.zip", f, "application/zip")},
                    timeout=300,
                    headers={"Accept": "application/x-protobuf"},  # optional
                )
            resp.raise_for_status()
            print(resp.status_code, resp.headers.get("content-type"))
            print(resp.content)  # protobuf on success

        os.remove(temp_zip_path)
        break
        time.sleep(30)
