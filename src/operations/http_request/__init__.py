import asyncio
from .get import extract_chunk
from .base import generate_auth_headers, configure_client_cert as configure_client_cert, configure_client as configure_client, process_request_chunk

MAPPING = dict(
    GET=extract_chunk,
)


def get_server_details(url, cert, token):
    from httpx import Client
    edition_mapper = {
        "Data": "datacenter",
        "Developer": "developer",
        "Enterprise": "enterprise",
        "Community": "community"
    }
    edition = None
    sync_client = Client(base_url=url, cert=cert)
    server_version_resp = sync_client.get("/api/server/version")
    server_version = float('.'.join(server_version_resp.text.split(".")[:2]))
    headers = generate_auth_headers(token=token, server_version=server_version)
    server_details_resp = sync_client.get("/api/system/info", headers=headers)
    for k, v in edition_mapper.items():
        if k.lower() in server_details_resp.json()['System']['Edition'].lower():
            edition = v
            break
    return server_version, edition


def process_chunk(chunk):
    results = asyncio.run(
        MAPPING.get(
            chunk[0]['kwargs']['method'], process_request_chunk
        )(
            chunk=chunk,
            max_threads=25
        )
    )
    return results
