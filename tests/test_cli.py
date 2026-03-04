
import pytest
from click.testing import CliRunner
from main import cli
from .mocks.mock_platform import MockPlatform
from .conftest import FILES_DIR

@pytest.fixture(scope='session')
def runner():
    return CliRunner()

def test_extract(runner, request_mocks, server_url, token, output_dir):
    result = runner.invoke(cli, ['extract', server_url, token, f'--export_directory={output_dir}'], catch_exceptions=False)
    assert result.exit_code == 0


def test_report(runner, server_url, extracts, report_output_dir, report_type):
    result = runner.invoke(cli, ['report', f'--export_directory={report_output_dir}', f'--report_type={report_type}'],
                           catch_exceptions=False)
    assert result.exit_code == 0


def test_report_completes_with_empty_results(runner, server_url, empty_results, report_type):
    result = runner.invoke(cli, ['report', f'--export_directory={empty_results}', f'--report_type={report_type}'],
                           catch_exceptions=False)
    assert result.exit_code == 0


def test_structure(runner, server_url, extracts, output_dir):
    result = runner.invoke(cli, ['structure', f'--export_directory={output_dir}'], catch_exceptions=False)
    assert result.exit_code == 0


def test_structure_completes_with_empty_results(runner, server_url, empty_results):
    result = runner.invoke(cli, ['structure', f'--export_directory={empty_results}'], catch_exceptions=False)
    assert result.exit_code == 0


def test_mappings(runner, server_url, extracts, output_dir):
    runner.invoke(cli, ['structure', f'--export_directory={output_dir}'], catch_exceptions=False)
    result = runner.invoke(cli, ['mappings', f'--export_directory={output_dir}'], catch_exceptions=False)
    assert result.exit_code == 0

def test_mappings_completes_with_empty_results(runner, server_url, empty_results):
    runner.invoke(cli, ['structure', f'--export_directory={empty_results}'], catch_exceptions=False)
    result = runner.invoke(cli, ['mappings', f'--export_directory={empty_results}'], catch_exceptions=False)
    assert result.exit_code == 0

def test_pipelines(mocker, respx_mock, runner, platform, pipeline_case):
    mocker.patch('pipelines.process.get_platform_module',
                 return_value=MockPlatform(platform=platform, case=pipeline_case))
    result = runner.invoke(cli, ['pipelines', f'{pipeline_case}/secrets.json', 'test_url', 'test_token', f'--input_directory={pipeline_case}', f'--output_directory={FILES_DIR}/'],  catch_exceptions=False)
    print(result.stdout)
    assert result.exit_code == 0
    assert '[]' not in result.output