"""Tests for src/dependencies.py"""
import json
import os
import tempfile

import pytest

from dependencies import (
    load_dependencies,
    load_dependency,
    load_all,
    load_chunk,
    load_each,
    load_map,
    clean_entity,
    find_required_keys,
    plan_dependency_values,
)


def _write_jsonl(directory, key, objects):
    """Write objects as JSONL into directory/key/results.1.jsonl."""
    folder = os.path.join(directory, key)
    os.makedirs(folder, exist_ok=True)
    path = os.path.join(folder, 'results.1.jsonl')
    with open(path, 'w') as f:
        for obj in objects:
            f.write(json.dumps(obj) + '\n')


class TestLoadAll:
    def test_returns_all_objects(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            run_id = 'run-01'
            run_dir = os.path.join(tmpdir, run_id)
            os.makedirs(run_dir)
            _write_jsonl(run_dir, 'items', [{'a': 1}, {'a': 2}])
            dependency = {'key': 'items'}
            results = list(load_all(dependency=dependency, directory=tmpdir, run_ids={run_id}))
            assert len(results) == 1
            assert results[0] == [{'a': 1}, {'a': 2}]

    def test_empty_directory_yields_empty_list(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            dependency = {'key': 'items'}
            results = list(load_all(dependency=dependency, directory=tmpdir, run_ids={'run-01'}))
            assert results == [[]]


class TestLoadChunk:
    def test_yields_chunks(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            run_id = 'run-01'
            run_dir = os.path.join(tmpdir, run_id)
            os.makedirs(run_dir)
            _write_jsonl(run_dir, 'items', [{'i': n} for n in range(5)])
            dependency = {'key': 'items', 'chunkSize': 2}
            chunks = list(load_chunk(dependency=dependency, directory=tmpdir, run_ids={run_id}))
            # 5 items, chunk size 2 → [2, 2, 1]
            assert len(chunks) == 3
            assert len(chunks[0]) == 2
            assert len(chunks[1]) == 2
            assert len(chunks[2]) == 1

    def test_remainder_chunk_is_yielded(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            run_id = 'run-01'
            run_dir = os.path.join(tmpdir, run_id)
            os.makedirs(run_dir)
            _write_jsonl(run_dir, 'items', [{'i': 1}, {'i': 2}, {'i': 3}])
            dependency = {'key': 'items', 'chunkSize': 10}
            chunks = list(load_chunk(dependency=dependency, directory=tmpdir, run_ids={run_id}))
            assert len(chunks) == 1
            assert len(chunks[0]) == 3


class TestLoadEach:
    def test_yields_individual_objects(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            run_id = 'run-01'
            run_dir = os.path.join(tmpdir, run_id)
            os.makedirs(run_dir)
            _write_jsonl(run_dir, 'items', [{'x': 1}, {'x': 2}])
            dependency = {'key': 'items'}
            results = list(load_each(dependency=dependency, directory=tmpdir, run_ids={run_id}))
            assert results == [{'x': 1}, {'x': 2}]


class TestLoadMap:
    def test_groups_by_group_key(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            run_id = 'run-01'
            run_dir = os.path.join(tmpdir, run_id)
            os.makedirs(run_dir)
            _write_jsonl(run_dir, 'items', [
                {'org': 'a', 'val': 1},
                {'org': 'a', 'val': 2},
                {'org': 'b', 'val': 3},
            ])
            dependency = {'key': 'items', 'groupKey': 'org'}
            results = list(load_map(dependency=dependency, directory=tmpdir, run_ids={run_id}))
            assert len(results) == 1
            grouped = results[0]
            assert len(grouped['a']) == 2
            assert len(grouped['b']) == 1


class TestLoadDependency:
    def test_uses_each_strategy_by_default(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            run_id = 'run-01'
            run_dir = os.path.join(tmpdir, run_id)
            os.makedirs(run_dir)
            _write_jsonl(run_dir, 'items', [{'x': 1}])
            dependency = {'key': 'items'}
            results = list(load_dependency(dependency=dependency, directory=tmpdir, run_ids={run_id}))
            assert results == [{'x': 1}]

    def test_uses_all_strategy(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            run_id = 'run-01'
            run_dir = os.path.join(tmpdir, run_id)
            os.makedirs(run_dir)
            _write_jsonl(run_dir, 'items', [{'x': 1}, {'x': 2}])
            dependency = {'key': 'items', 'strategy': 'all'}
            results = list(load_dependency(dependency=dependency, directory=tmpdir, run_ids={run_id}))
            assert results == [[{'x': 1}, {'x': 2}]]

    def test_none_strategy_returns_empty(self):
        dependency = {'key': 'items', 'strategy': 'none'}
        results = list(load_dependency(dependency=dependency, directory='/any', run_ids=set()))
        assert results == []


class TestLoadDependencies:
    def test_no_dependencies_yields_empty_input(self):
        task_config = {}
        results = list(load_dependencies(
            task='my_task',
            inputs={'x': 1},
            task_config=task_config,
            concurrency=10,
            output_directory='/tmp',
            run_ids=set(),
        ))
        assert results == [[{'obj': {}, 'inputs': {'x': 1}}]]

    def test_single_dependency_yielded(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            run_id = 'run-01'
            run_dir = os.path.join(tmpdir, run_id)
            os.makedirs(run_dir)
            _write_jsonl(run_dir, 'items', [{'x': 1}, {'x': 2}])
            task_config = {'dependencies': [{'key': 'items', 'strategy': 'each'}]}
            results = list(load_dependencies(
                task='my_task',
                inputs={},
                task_config=task_config,
                concurrency=10,
                output_directory=tmpdir,
                run_ids={run_id},
            ))
            assert len(results) == 1  # one chunk with both items
            chunk = results[0]
            assert len(chunk) == 2
            assert chunk[0]['items'] == {'x': 1}
            assert chunk[1]['items'] == {'x': 2}

    def test_dependencies_with_none_strategy_skipped(self):
        task_config = {'dependencies': [{'key': 'items', 'strategy': 'none'}]}
        results = list(load_dependencies(
            task='my_task',
            inputs={},
            task_config=task_config,
            concurrency=10,
            output_directory='/tmp',
            run_ids=set(),
        ))
        assert results == [[{'obj': {}, 'inputs': {}}]]


class TestCleanEntity:
    def test_keeps_only_required_keys(self):
        entity = {'a': 1, 'b': 2, 'c': 3}
        result = clean_entity(entity, {'a', 'c'})
        assert result == {'a': 1, 'c': 3}


class TestFindRequiredKeys:
    def test_finds_path_key(self):
        field = {'path': '$.projectKey'}
        result = find_required_keys(task='myTask', field=field)
        assert 'myTask' in result
        assert 'projectKey' in result['myTask']

    def test_finds_nested_path(self):
        field = {'nested': {'path': '$.name', 'source': 'otherTask'}}
        result = find_required_keys(task='myTask', field=field)
        assert 'otherTask' in result
        assert 'name' in result['otherTask']

    def test_handles_list(self):
        field = [{'path': '$.key'}]
        result = find_required_keys(task='myTask', field=field)
        assert 'myTask' in result

    def test_non_dict_non_list_returns_empty(self):
        result = find_required_keys(task='myTask', field='plain_string')
        assert result == {}


class TestPlanDependencyValues:
    def test_extracts_from_operations(self):
        config = {'operations': {'path': '$.projectKey'}}
        result = plan_dependency_values(config=config, task='myTask')
        assert 'myTask' in result

    def test_extracts_from_prefilters(self):
        config = {'prefilters': {'path': '$.name'}}
        result = plan_dependency_values(config=config, task='myTask')
        assert 'myTask' in result

    def test_empty_config(self):
        result = plan_dependency_values(config={}, task='myTask')
        assert result == {}
