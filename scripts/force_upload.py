import httpx
from pathlib import Path
import httpx
import zipfile
import tempfile
import os
from pathlib import Path
from urllib.parse import urlencode
from protobufs.scanner_report_pb2 import Metadata


from datetime import datetime
import uuid
with open('/Users/perry.stallings/Code/sonar-reports/.scannerwork/scanner-report/metadata.pb', 'rb') as f:
    m = Metadata()
    m.ParseFromString(f.read())
m.analysis_date = int(datetime.utcnow().timestamp() * 1000)
m.analysis_uuid = str(uuid.uuid4())

with open('/Users/perry.stallings/Code/sonar-reports/.scannerwork/scanner-report/metadata.pb', 'wb') as f:
    f.write(m.SerializeToString())

token = ""
zip_path = Path("/Users/perry.stallings/Code/sonar-reports/.scannerwork/scanner-report")

# Create a temporary zip file
with tempfile.NamedTemporaryFile(suffix='.zip', delete=False) as temp_zip:
    temp_zip_path = temp_zip.name

with zipfile.ZipFile(temp_zip_path, 'w', zipfile.ZIP_DEFLATED) as zipf:
    for root, dirs, files in os.walk(zip_path):
        for file in files:
            file_path = os.path.join(root, file)
            arc_name = os.path.relpath(file_path, zip_path)
            zipf.write(file_path, arc_name)


async with httpx.AsyncClient(base_url="https://sonarcloud.io", headers=dict(Authorization='Bearer')) as client:
    with open(temp_zip_path, "rb") as f:
        resp = await client.post(
            "/api/ce/submit",
            params={"organization": "testing-things"},
            data={
                "projectKey": "testing-things_backdate",
                "projectName": "backdate",
            },
            files={"report": ("scanner-report.zip", f, "application/zip")},
            timeout=300,
            headers={"Accept": "application/x-protobuf"},  # optional
        )
    resp.raise_for_status()
    print(resp.status_code, resp.headers.get("content-type"))
    print(resp.content)  # protobuf on success