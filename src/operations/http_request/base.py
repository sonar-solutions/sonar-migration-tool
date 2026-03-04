import httpx
from httpx import AsyncClient
from tenacity import retry, stop_after_attempt, wait_random_exponential
from asyncio import gather
from parser import extract_path_value
from io import BytesIO
from logs import log_event
from datetime import datetime, timezone

httpx_client: tuple = None, None

CLIENTS = dict()


def get_client(url: str):
    global CLIENTS
    return CLIENTS.get(url)


def configure_client(url: str, cert: tuple | None, server_version, token: str, timeout=60, concurrency=25):
    global CLIENTS
    CLIENTS[url] = httpx.AsyncClient(
        base_url=url, cert=cert, timeout=timeout, limits=httpx.Limits(max_connections=2 * concurrency),
        headers=generate_auth_headers(token=token, server_version=server_version)
    )
    return CLIENTS


def configure_client_cert(key_file_path: str, pem_file_path: str, cert_password: str) -> tuple | None:
    cert = tuple([i for i in [pem_file_path, key_file_path, cert_password] if i is not None])
    return cert if cert else None


def generate_auth_headers(token, server_version):
    from base64 import b64encode
    headers = dict()
    if server_version == 'cloud' or server_version >= 10:
        headers['Authorization'] = f"Bearer {token}"
    else:
        encoded_auth = b64encode((token + ':').encode('utf-8')).decode('utf-8')
        headers['Authorization'] = f"Basic {encoded_auth}"
    return headers


def format_response_body(response):
    js = dict()
    try:
        js = response.json()
    except ValueError:
        js['content'] = response.text
    return js


def safe_json_request(host, method, url, stop=stop_after_attempt(3), reraise=True, raise_over=500,
                      wait=wait_random_exponential(multiplier=.01, max=1), log_attributes=None, **kwargs):
    client = get_client(url=host)
    if client is None:
        client = AsyncClient(base_url=host)
    if log_attributes is None:
        log_attributes = {
            k: v for k, v in kwargs.items() if k not in ['headers', 'files']
        }
    log_event(level='info', status='success', process_type='request_started', payload=log_attributes,
              logger_name='http_request')
    args = dict(
        client=client, method=method, url=url, stop=stop, reraise=reraise, wait=wait,
        raise_over=raise_over,
        log_attributes=log_attributes, **kwargs
    )
    return async_safe_json_request(**args)


async def async_safe_json_request(client, method, url, log_attributes: dict, raise_over, reraise,
                                  stop, wait, **kwargs):
    import httpx
    @retry(stop=stop, reraise=reraise, wait=wait)
    async def make_async_request():
        resp = await client.request(method=method, url=url, **kwargs)
        log_event(
            level='info', status='success', process_type='request_completed', payload=dict(
                method=method,
                url=url,
                status=resp.status_code,
                created_ts=datetime.now(tz=timezone.utc).isoformat(timespec='seconds'),
                response=resp.text,
                **log_attributes,
            ),
            logger_name='http_request'
        )
        status, json_response = process_response(
            resp=resp, raise_over=raise_over
        )

        return status, json_response

    try:
        status_code, js = await make_async_request()
    except (httpx.RequestError, httpx.HTTPStatusError) as exc:
        status_code, js = process_errors(
            url=url, exc=exc, method=method, log_attributes=log_attributes
        )
    except Exception as exc:
        status_code, js = None, dict()
        log_event(
            level='error', status='failure', process_type='request_completed', payload=dict(
                method=method,
                url=url,
                status=None,
                created_ts=datetime.now(tz=timezone.utc).isoformat(timespec='seconds'),
                error=str(exc),
                **log_attributes,
            ),
            logger_name='http_request'
        )
    return status_code, js


def process_response(resp, raise_over):
    status = resp.status_code
    js = format_response_body(response=resp)
    if status is not None and status >= raise_over:
        resp.raise_for_status()
    return status, js


def process_errors(exc, method, url, log_attributes):
    status_code, js = None, dict()
    if isinstance(exc, httpx.RequestError):
        log_event(
            level='error', status='failure', process_type='request_completed', payload=dict(
                method=method,
                url=url,
                status=status_code,
                created_ts=datetime.now(tz=timezone.utc).isoformat(timespec='seconds'),
                **log_attributes,
            ),
            logger_name='http_request'
        )
    elif isinstance(exc, httpx.HTTPStatusError):
        js = format_response_body(response=exc.response)
        status_code = exc.response.status_code
        log_event(
            level='error', status='failure', process_type='request_completed', payload=dict(
                method=method,
                url=url,
                status=status_code,
                content=exc.response.text,
                created_ts=datetime.now(tz=timezone.utc).isoformat(timespec='seconds'),
                **log_attributes,
            ),
            logger_name='http_request'
        )
    return status_code, js


async def process_request_chunk(chunk, max_threads):
    request_chunk = list()
    results = list()
    for c in chunk:
        request_chunk.append(c)
        if len(request_chunk) >= max_threads:
            results = await process_sub_chunk(results=results, request_chunk=request_chunk)
            request_chunk = list()
    if request_chunk:
        results = await process_sub_chunk(results=results, request_chunk=request_chunk)
    return results


async def process_sub_chunk(results, request_chunk):
    sub_results = await gather(
        *[
            safe_json_request(
                host=request_chunk[idx]['kwargs']['client'],
                method=request_chunk[idx]['kwargs']['method'],
                raise_over=300,
                url=payload['kwargs']['url'],
                files=None if payload['kwargs'].get('file') is None else {
                    request_chunk[idx]['kwargs']['file']['name']: (
                    'backup.xml', BytesIO(request_chunk[idx]['kwargs']['file']['content'].encode('utf-8')),
                    request_chunk[idx]['kwargs']['file']['contentType'])
                },
                data={k: v for k, v in payload['kwargs']['payload'].items() if v is not None} if
                payload['kwargs']['encoding'] in ['x-www-form-urlencoded', 'multipart/form-data'] else None,
                json={k: v for k, v in payload['kwargs']['payload'].items() if v is not None} if
                payload['kwargs']['encoding'] == 'json' else None,
            ) for idx, payload in enumerate(request_chunk)
        ]
    )
    results.extend(
        [
            [
                extract_path_value(
                    obj=result[1],
                    path=request_chunk[idx]['kwargs']['resultKey']
                )
            ] for idx, result in enumerate(sub_results)
        ]
    )
    return results
