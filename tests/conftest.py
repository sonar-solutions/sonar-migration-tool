import uuid
import pytest
import shutil
from plan import get_available_task_configs
import os
import json
from functools import partial
import httpx
from urllib.parse import urlparse, parse_qs
import respx
from respx.patterns import M
from click.testing import CliRunner
from main import cli
from .mocks.mock_platform import MockPlatform

TEST_DIR = os.path.dirname(__file__)
PROJECT_ROOT = os.path.dirname(TEST_DIR)
FILES_DIR = os.path.join(PROJECT_ROOT, 'files')
PIPELINE_MOCK_DIR = os.path.join(TEST_DIR, 'mocks/pipelines/')
PIPELINE_MOCKS = [os.path.join(PIPELINE_MOCK_DIR, i) for i in os.listdir(PIPELINE_MOCK_DIR)]
PIPELINE_CASES = [os.path.join(PIPELINE_MOCK_DIR, i) for i in os.listdir(PIPELINE_MOCK_DIR) if
                  os.path.exists(os.path.join(PIPELINE_MOCK_DIR, i) and 'pycache' not in i)]


@pytest.fixture(scope='session', autouse=True, params=['github'])
def platform(request):
    return request.param


@pytest.fixture(scope='session', params=PIPELINE_CASES)
def pipeline_case(request):
    return request.param


@pytest.fixture(scope='session')
def token():
    return "sq_" + str(uuid.uuid4())


@pytest.fixture(scope='session', params=['migration', 'maturity'])
def report_type(request):
    return request.param


@pytest.fixture(scope='session')
def root_dir():
    return os.path.dirname(__file__)


@pytest.fixture(scope='session')
def custom_mock():
    with respx.mock(assert_all_called=False) as mock:
        yield mock


@pytest.fixture(scope='session', params=['community', 'enterprise', 'developer'])
def edition(request):
    return request.param


@pytest.fixture(scope='session', params=['9.9', '10.7'])
def version(request):
    return request.param


@pytest.fixture(scope='session')
def server_url():
    return 'https://sonarqube:9000'


@pytest.fixture(scope='session')
def endpoints(root_dir, edition, version):
    paths = dict()
    with open(f"{root_dir}/mocks/{edition}/{version}/webservices.json") as f:
        js = json.load(f)
        for service in js['webServices']:
            for endpoint in service['actions']:
                paths[f"{service['path']}/{endpoint['key']}"] = endpoint
    with open(f"{root_dir}/mocks/{edition}/{version}/responses.json") as f:
        js = json.load(f)
        for path, response in js.items():
            paths[path]['response'] = response
    return paths


@pytest.fixture(scope='session')
def output_dir(root_dir, edition, version):
    import shutil
    path = os.path.join(FILES_DIR, edition, version) + "/"
    shutil.rmtree(path, ignore_errors=True)
    os.makedirs(os.path.join(FILES_DIR, edition), exist_ok=True)
    os.makedirs(path, exist_ok=True)
    return path


@pytest.fixture(scope='session')
def report_output_dir(output_dir, report_type):
    path = f"{output_dir}{report_type}/"
    os.makedirs(path, exist_ok=True)
    return path


@pytest.fixture(scope='session')
def request_mocks(custom_mock, server_url, edition, version, endpoints):
    custom_mock.get(f"{server_url}/api/server/version").mock(return_value=httpx.Response(200, text=f"{version}.0"))
    for k, v in endpoints.items():
        if k == 'api/server/version':
            continue
        custom_mock.route(M(url=f"{server_url}/{k}")).mock(side_effect=partial(validate_api_input, endpoint=v))
    return custom_mock


def validate_api_input(request, endpoint):
    params = parse_qs(urlparse(str(request.url)).query)
    allowed_params = set()
    required_params = set()
    for i in endpoint.get('params', []):
        allowed_params.add(i['key'])
        if i.get('required', False):
            required_params.add(i['key'])
    if required_params > set(params.keys()):
        raise ValueError(f"Missing required parameters: {required_params - set(params.keys())}")
    if set(params.keys()) > allowed_params:
        raise ValueError(f"Invalid parameters: {set(params.keys()) - allowed_params}")
    return httpx.Response(200, json=endpoint.get('response', dict()))


@pytest.fixture(scope='session')
def extracts(request_mocks, server_url, token, report_output_dir, report_type):
    runner = CliRunner()
    result = runner.invoke(cli, ['extract', server_url, token, f'--export_directory={report_output_dir}',
                                 f'--extract_type={report_type}'], catch_exceptions=False)
    return result.output


@pytest.fixture(scope='session')
def empty_results():
    path = os.path.join(FILES_DIR, 'empty') + "/"
    configs = get_available_task_configs(client_version=1000000, edition='enterprise')
    shutil.rmtree(path, ignore_errors=True)
    os.makedirs(path, exist_ok=True)
    for task, config in configs.items():
        task_dir = os.path.join(path, task)
        os.makedirs(task_dir, exist_ok=True)
        with open(os.path.join(task_dir, 'results.1.json'), 'wt') as f:
            f.write('{}')
    return path


@pytest.fixture()
def custom_platform(mocker, platform, pipeline_case):
    mocker.patch('pipelines.process.get_platform_module',
                 return_value=MockPlatform(platform=platform, case=pipeline_case))
    return mocker
