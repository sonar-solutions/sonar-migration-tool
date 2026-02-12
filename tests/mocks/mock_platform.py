import os
from hashlib import sha256
from uuid import uuid4
from importlib import import_module
import difflib

TESTS_DIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
FILES_DIR = os.path.join(os.path.dirname(TESTS_DIR), 'files')

class MockPlatform:
    def __init__(self, platform: str, case: str):
        self.platform = platform
        self.case = case
        self.test_dir = os.path.join(case, platform)
        mod = import_module(f'pipelines.platforms.{self.platform}')
        self.updated_files = dict()
        self.platform_mod = mod
        self.expected = self.load_expected()

    def load_expected(self):
        expected = dict()
        output_dir = os.path.join(self.test_dir, 'output') + '/'
        for root, dirs, files in os.walk(output_dir):
            for file in files:
                with open(os.path.join(root, file), 'rt') as f:
                    expected[os.path.join(root.replace(output_dir, ""), file)] = f.read()
        return expected

    def format_file(self, file_name, default=None):
        file_path = os.path.join(self.test_dir, 'input', file_name)
        file = None
        if os.path.exists(file_path):
            with open(file_path, 'rt') as f:
                file = dict(
                    file_path=file_name,
                    content=f.read(),
                    sha=sha256(uuid4().hex.encode()).hexdigest()
                )
        return file if file is not None else default

    async def get_content(self, token, repository, file_path, branch_name, extra_args=None):
        file = self.format_file(file_name=file_path, default=dict(
            file_path=file_path,
            content='',
            sha=sha256(uuid4().hex.encode()).hexdigest()
        ))
        assert isinstance(file['file_path'], str)
        return file, extra_args

    @staticmethod
    async def create_org_secret(organization, token, secret_name, secret_value):
        return dict()

    @staticmethod
    async def get_default_branch(token, repository):
        return dict(
            name='main',
            sha=sha256(uuid4().hex.encode()).hexdigest()
        )

    async def get_pipeline_files(self, token, repository, branch_name):
        files = []
        locations = self.platform_mod.get_pipeline_file_paths()
        for f in locations.get('files', []):
            file = self.format_file(file_name=f)
            if file:
                files.append((file, dict))
        for folder in locations.get('folders', []):
            for file in os.listdir(os.path.join(self.test_dir, 'input', folder)):
                file = self.format_file(file_name=os.path.join(folder, file))
                if file:
                    files.append((file, dict))
        return files

    async def create_branch(self, token, repository, branch_name, base_branch_name, **_):
        return dict(
            name=branch_name,
            repository=repository,
            sha=sha256(uuid4().hex.encode()).hexdigest(),
        )

    def _write_output_file(self, output_dir, file_path, content):
        os.makedirs(output_dir, exist_ok=True)
        with open(os.path.join(output_dir, file_path.split('/')[-1]), 'wt') as f:
            f.write(content)

    async def create_or_update_file(self, token, repository, message, branch_name, file_path, content: str, sha,):
        self.updated_files[file_path] = content
        output_dir = os.path.join(FILES_DIR, self.case.split('/')[-1])
        self._write_output_file(output_dir, file_path, content)
        return dict(
            content=content,
            branch_name=branch_name,
            sha=sha256(uuid4().hex.encode()).hexdigest(),
            file_path=file_path,
        )

    async def create_pull_request(self, token, repository, source_branch, target_branch, title, body):
        for k,v in self.expected.items():
            changes = "\n"
            if self.updated_files.get(k, '').strip() != v.strip():
                for pos, op in enumerate(difflib.ndiff(self.updated_files.get(k,'').splitlines(), v.splitlines())):
                    if op[0] == ' ':
                        continue
                    if op[0] == '+':
                        changes += f"line {pos}: + |{op[2:]}\n"
                    elif op[0] == '-':
                        changes += f"line {pos}: - |{op[2:]}\n"
                raise AssertionError(changes)
        return {
            "title": title,
            "body": body,
            "head": source_branch,
            "base": target_branch
        }
    def generate_repository_string(self, **kwargs):
        return self.platform_mod.generate_repository_string(**kwargs)

    def get_available_pipelines(self, **kwargs):
        return self.platform_mod.get_available_pipelines(**kwargs)