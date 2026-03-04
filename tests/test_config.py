"""Tests for src/config.py"""
import json
import os
import tempfile

import pytest

from config import (
    load_config_file,
    merge_config_with_cli,
    get_config_value,
    validate_required_keys,
    _validate_config_paths,
)


class TestValidateConfigPaths:
    def test_valid_path_within_cwd(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            config = {'export_directory': tmpdir + '/subdir'}
            os.makedirs(tmpdir + '/subdir')
            original_cwd = os.getcwd()
            os.chdir(tmpdir)
            try:
                _validate_config_paths(config)
                assert os.path.isabs(config['export_directory'])
            finally:
                os.chdir(original_cwd)

    def test_path_outside_cwd_raises(self):
        # /etc is neither in cwd (/app) nor in tmp (/tmp) inside Docker
        config = {'export_directory': '/etc/fake_sonar_test_path'}
        with pytest.raises(ValueError, match="must be within the working directory"):
            _validate_config_paths(config)

    def test_empty_path_skipped(self):
        config = {'export_directory': ''}
        _validate_config_paths(config)
        assert config['export_directory'] == ''

    def test_none_path_skipped(self):
        config = {'export_directory': None}
        _validate_config_paths(config)
        assert config['export_directory'] is None

    def test_non_path_key_ignored(self):
        config = {'other_key': '/some/random/path'}
        _validate_config_paths(config)  # should not raise

    def test_path_in_tmp_dir_is_allowed(self):
        tmp = tempfile.gettempdir()
        subdir = os.path.join(tmp, 'sonar_test_subdir')
        os.makedirs(subdir, exist_ok=True)
        config = {'export_directory': subdir}
        _validate_config_paths(config)
        assert config['export_directory'] == os.path.realpath(subdir)


class TestLoadConfigFile:
    def test_loads_valid_json_file(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            config_data = {'key': 'value', 'number': 42}
            config_path = os.path.join(tmpdir, 'config.json')
            with open(config_path, 'w') as f:
                json.dump(config_data, f)
            original_cwd = os.getcwd()
            os.chdir(tmpdir)
            try:
                result = load_config_file(config_path)
                assert result == config_data
            finally:
                os.chdir(original_cwd)

    def test_raises_if_file_not_found(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            original_cwd = os.getcwd()
            os.chdir(tmpdir)
            try:
                with pytest.raises(FileNotFoundError):
                    load_config_file(os.path.join(tmpdir, 'nonexistent.json'))
            finally:
                os.chdir(original_cwd)

    def test_raises_if_path_outside_cwd(self):
        with tempfile.TemporaryDirectory() as tmpdir1:
            with tempfile.TemporaryDirectory() as tmpdir2:
                config_path = os.path.join(tmpdir2, 'config.json')
                with open(config_path, 'w') as f:
                    json.dump({'key': 'val'}, f)
                original_cwd = os.getcwd()
                os.chdir(tmpdir1)
                try:
                    with pytest.raises(ValueError, match="must be within the working directory"):
                        load_config_file(config_path)
                finally:
                    os.chdir(original_cwd)

    def test_raises_on_invalid_json(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            config_path = os.path.join(tmpdir, 'bad.json')
            with open(config_path, 'w') as f:
                f.write('not valid json {{{')
            original_cwd = os.getcwd()
            os.chdir(tmpdir)
            try:
                with pytest.raises(json.JSONDecodeError):
                    load_config_file(config_path)
            finally:
                os.chdir(original_cwd)


class TestMergeConfigWithCli:
    def test_cli_overrides_config(self):
        config = {'a': 1, 'b': 2}
        cli_args = {'b': 99, 'c': 3}
        result = merge_config_with_cli(config, cli_args)
        assert result == {'a': 1, 'b': 99, 'c': 3}

    def test_none_cli_values_not_overriding(self):
        config = {'a': 1, 'b': 2}
        cli_args = {'b': None}
        result = merge_config_with_cli(config, cli_args)
        assert result['b'] == 2

    def test_empty_cli_args(self):
        config = {'a': 1}
        result = merge_config_with_cli(config, {})
        assert result == {'a': 1}

    def test_does_not_mutate_original_config(self):
        config = {'a': 1}
        merge_config_with_cli(config, {'a': 2})
        assert config['a'] == 1


class TestGetConfigValue:
    def test_returns_existing_key(self):
        assert get_config_value({'x': 42}, 'x') == 42

    def test_returns_default_for_missing_key(self):
        assert get_config_value({}, 'missing', default='fallback') == 'fallback'

    def test_returns_none_default_by_default(self):
        assert get_config_value({}, 'missing') is None


class TestValidateRequiredKeys:
    def test_all_keys_present(self):
        validate_required_keys({'a': 1, 'b': 2}, ['a', 'b'])  # no exception

    def test_missing_key_raises(self):
        with pytest.raises(ValueError, match="Missing required configuration keys"):
            validate_required_keys({'a': 1}, ['a', 'b'])

    def test_none_value_treated_as_missing(self):
        with pytest.raises(ValueError, match="b"):
            validate_required_keys({'a': 1, 'b': None}, ['a', 'b'])

    def test_empty_required_keys(self):
        validate_required_keys({}, [])  # no exception
