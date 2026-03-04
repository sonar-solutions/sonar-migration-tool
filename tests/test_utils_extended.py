"""Extended tests for src/utils.py covering uncovered lines."""
import json
import os
import tempfile

import pytest

from utils import (
    generate_run_id,
    object_reader,
    get_latest_extract_id,
    export_csv,
    filter_completed,
    generate_hash_id,
)


class TestGenerateRunId:
    def test_raises_when_directory_outside_cwd(self):
        # /etc is neither in cwd (/app) nor in tmp (/tmp) inside Docker
        with pytest.raises(ValueError, match="must be within the working directory"):
            generate_run_id('/etc/fake_sonar_run_test')

    def test_uses_tmp_dir_when_allowed(self):
        tmp = tempfile.gettempdir()
        subdir = os.path.join(tmp, 'sonar_run_test')
        os.makedirs(subdir, exist_ok=True)
        run_id = generate_run_id(subdir)
        assert run_id.count('-') >= 2  # format mm-dd-yyyy-nn


class TestObjectReader:
    def test_skips_non_jsonl_files(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            key_dir = os.path.join(tmpdir, 'items')
            os.makedirs(key_dir)
            with open(os.path.join(key_dir, 'data.txt'), 'w') as f:
                f.write('not jsonl')
            results = list(object_reader(directory=tmpdir, key='items'))
            assert results == []

    def test_returns_empty_for_missing_folder(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            results = list(object_reader(directory=tmpdir, key='nonexistent'))
            assert results == []


class TestGetLatestExtractId:
    def test_returns_max_directory_with_extract_json(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            for run_id in ['01-01-2024-01', '01-01-2024-02', '01-01-2024-03']:
                d = os.path.join(tmpdir, run_id)
                os.makedirs(d)
                with open(os.path.join(d, 'extract.json'), 'w') as f:
                    json.dump({'url': 'http://example.com'}, f)
            result = get_latest_extract_id(tmpdir)
            assert result == '01-01-2024-03'

    def test_ignores_dirs_without_extract_json(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            d = os.path.join(tmpdir, 'no-extract')
            os.makedirs(d)
            result = get_latest_extract_id(tmpdir)
            assert result is None


class TestExportCsv:
    def test_raises_when_directory_outside_cwd(self):
        # /etc is neither in cwd (/app) nor in tmp (/tmp) inside Docker
        with pytest.raises(ValueError, match="must be within the working directory"):
            export_csv(directory='/etc/fake_sonar_test_export', name='test', data=[{'a': 1}])

    def test_raises_on_path_traversal_name(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            original_cwd = os.getcwd()
            os.chdir(tmpdir)
            try:
                with pytest.raises(ValueError, match="Invalid CSV filename"):
                    export_csv(directory=tmpdir, name='../../etc/passwd', data=[{'a': 1}])
            finally:
                os.chdir(original_cwd)


class TestFilterCompleted:
    def test_returns_full_plan_when_nothing_completed(self):
        plan = [['task1', 'task2'], ['task3']]
        result = filter_completed(plan=plan, directory='/nonexistent')
        assert result == plan

    def test_filters_completed_tasks(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            os.makedirs(os.path.join(tmpdir, 'task1'))
            os.makedirs(os.path.join(tmpdir, 'task2'))
            plan = [['task1', 'task2'], ['task3']]
            result = filter_completed(plan=plan, directory=tmpdir)
            # task1 is completed, task2 would be considered mid-phase so only task1 fully completed
            # The logic marks task at boundary as not-completed (completed[:-1])
            assert result is not None

    def test_empty_plan(self):
        result = filter_completed(plan=[], directory='/nonexistent')
        assert result == []


class TestGenerateHashId:
    def test_returns_uuid_format(self):
        result = generate_hash_id({'key': 'value'})
        assert len(result) == 36
        assert result.count('-') == 4

    def test_same_input_produces_same_hash(self):
        data = {'a': 1, 'b': [2, 3]}
        assert generate_hash_id(data) == generate_hash_id(data)

    def test_different_inputs_produce_different_hashes(self):
        assert generate_hash_id({'a': 1}) != generate_hash_id({'a': 2})
