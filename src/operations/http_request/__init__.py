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
    system_info = server_details_resp.json()
    if 'System' not in system_info:
        raise PermissionError(
            "The token provided does not have System Administrator privileges. "
            "The 'extract' command requires a token with admin access to /api/system/info. "
            "Please provide a System Administrator token and try again."
        )
    for k, v in edition_mapper.items():
        if k.lower() in system_info['System']['Edition'].lower():
            edition = v
            break
    return server_version, edition


_loop = None

def _get_event_loop():
    global _loop
    if _loop is None or _loop.is_closed():
        _loop = asyncio.new_event_loop()
        asyncio.set_event_loop(_loop)
    return _loop

def process_chunk(chunk):
    loop = _get_event_loop()
    results = loop.run_until_complete(
        MAPPING.get(
            chunk[0]['kwargs']['method'], process_request_chunk
        )(
            chunk=chunk,
            max_threads=25
        )
    )
    return results
