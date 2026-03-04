"""Tests for GitHub Actions pipeline parsing"""
import pytest
from ruamel.yaml import YAML


def load_yaml(text):
    return YAML().load(text)


class TestProcessYaml:
    def setup_method(self):
        from pipelines.pipelines import github
        self.github = github

    def test_simple_workflow_with_sonar_action(self):
        yaml = load_yaml("""
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: SonarSource/sonarcloud-github-action@master
        env:
          SONAR_TOKEN: ${{ secrets.SONAR_TOKEN }}
          SONAR_HOST_URL: ${{ secrets.SONAR_HOST_URL }}
""")
        file = {'yaml': yaml, 'file_path': '.github/workflows/sonar.yml', 'content': ''}
        pipelines = self.github.process_yaml(file=file)

        assert len(pipelines) == 1
        assert pipelines[0]['runs_sonar'] is True
        assert len(pipelines[0]['tasks']) == 1
        assert pipelines[0]['tasks'][0]['runner'] == 'plugin'

    def test_workflow_with_script_sonar_scanner(self):
        yaml = load_yaml("""
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Run Sonar
        run: sonar-scanner -Dsonar.projectKey=my-project
        env:
          SONAR_TOKEN: ${{ secrets.SONAR_TOKEN }}
          SONAR_HOST_URL: ${{ secrets.SONAR_HOST_URL }}
""")
        file = {'yaml': yaml, 'file_path': '.github/workflows/sonar.yml', 'content': ''}
        pipelines = self.github.process_yaml(file=file)

        assert pipelines[0]['runs_sonar'] is True
        assert pipelines[0]['tasks'][0]['runner'] == 'script'

    def test_workflow_without_sonar(self):
        yaml = load_yaml("""
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - run: npm install && npm test
""")
        file = {'yaml': yaml, 'file_path': '.github/workflows/ci.yml', 'content': ''}
        pipelines = self.github.process_yaml(file=file)

        assert len(pipelines) == 1
        assert pipelines[0]['runs_sonar'] is False
        assert len(pipelines[0]['tasks']) == 0

    def test_empty_jobs(self):
        yaml = load_yaml("jobs: {}")
        file = {'yaml': yaml, 'file_path': '.github/workflows/empty.yml', 'content': ''}
        pipelines = self.github.process_yaml(file=file)

        assert pipelines == []

    def test_no_jobs_key(self):
        yaml = load_yaml("name: My Workflow\non: push")
        file = {'yaml': yaml, 'file_path': '.github/workflows/nojobs.yml', 'content': ''}
        pipelines = self.github.process_yaml(file=file)

        assert pipelines == []

    def test_sonar_variables_detected(self):
        yaml = load_yaml("""
jobs:
  build:
    runs-on: ubuntu-latest
    env:
      SONAR_TOKEN: ${{ secrets.SONAR_TOKEN }}
      SONAR_HOST_URL: ${{ secrets.SONAR_HOST_URL }}
    steps:
      - uses: SonarSource/sonarcloud-github-action@master
""")
        file = {'yaml': yaml, 'file_path': '.github/workflows/sonar.yml', 'content': ''}
        pipelines = self.github.process_yaml(file=file)

        variables = pipelines[0]['variables']
        assert 'SONAR_TOKEN' in variables
        assert 'SONAR_HOST_URL' in variables


class TestFormatVariable:
    def setup_method(self):
        from pipelines.pipelines import github
        self.github = github

    def test_secret_format(self):
        result = self.github.format_variable(value='MY_SECRET', var_type='secret')
        assert result == '${{ secrets.MY_SECRET }}'

    def test_var_format(self):
        result = self.github.format_variable(value='MY_VAR', var_type='var')
        assert result == '${{ vars.MY_VAR }}'


class TestIsValidPipeline:
    def setup_method(self):
        from pipelines.pipelines import github
        self.github = github

    def test_always_valid(self):
        assert self.github.is_valid_pipeline(file={}) is True
