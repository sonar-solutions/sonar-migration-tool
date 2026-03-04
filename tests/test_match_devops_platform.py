"""Tests for src/operations/match_devops_platform.py"""
import pytest

from operations.match_devops_platform import execute, process_chunk


class TestExecute:
    BASE_KWARGS = {
        'output_key': 'result',
        'integration_key': 'int-key',
        'repo_slug': 'my-repo',
        'slug': 'my-slug',
        'label': 'workspace / my-repo',
    }

    def test_bitbucketcloud_matches_when_repository_equals_label(self):
        result = execute(
            alm='bitbucketcloud',
            repository='workspace / my-repo',
            label='workspace / my-repo',
            **{k: v for k, v in self.BASE_KWARGS.items() if k != 'label'},
        )
        assert result == {'result': 'int-key'}

    def test_bitbucketcloud_no_match(self):
        result = execute(
            alm='bitbucketcloud',
            repository='other-repo',
            label='workspace / my-repo',
            **{k: v for k, v in self.BASE_KWARGS.items() if k != 'label'},
        )
        assert result is None

    def test_github_matches_when_repository_equals_repo_slug(self):
        result = execute(
            alm='github',
            repository='my-repo',
            label='something',
            **{k: v for k, v in self.BASE_KWARGS.items() if k not in ('label',)},
        )
        assert result == {'result': 'my-repo'}

    def test_github_no_match(self):
        result = execute(
            alm='github',
            repository='other-repo',
            label='something',
            **{k: v for k, v in self.BASE_KWARGS.items() if k not in ('label',)},
        )
        assert result is None

    def test_gitlab_matches_when_integration_key_equals_repository_string(self):
        result = execute(
            alm='gitlab',
            repository=42,
            integration_key='42',
            label='some-label',
            repo_slug='some-slug',
            slug='some-slug',
            output_key='result',
        )
        assert result == {'result': '42'}

    def test_gitlab_no_match(self):
        result = execute(
            alm='gitlab',
            repository=99,
            integration_key='42',
            label='some-label',
            repo_slug='some-slug',
            slug='some-slug',
            output_key='result',
        )
        assert result is None

    def test_azure_matches_when_label_equals_slug_slash_repository(self):
        result = execute(
            alm='azure',
            repository='my-repo',
            slug='my-slug',
            label='my-slug / my-repo',
            integration_key='int-key',
            repo_slug='rs',
            output_key='result',
        )
        assert result == {'result': 'int-key'}

    def test_azure_no_match(self):
        result = execute(
            alm='azure',
            repository='my-repo',
            slug='my-slug',
            label='wrong-label',
            integration_key='int-key',
            repo_slug='rs',
            output_key='result',
        )
        assert result is None

    def test_unknown_alm_returns_none(self):
        result = execute(
            alm='unknown',
            repository='my-repo',
            label='label',
            slug='slug',
            repo_slug='rs',
            integration_key='ik',
            output_key='result',
        )
        assert result is None


class TestProcessChunk:
    def test_single_match_returns_result(self):
        chunk = [{'kwargs': {
            'alm': 'github',
            'repository': 'my-repo',
            'repo_slug': 'my-repo',
            'slug': 's',
            'label': 'l',
            'integration_key': 'ik',
            'output_key': 'binding',
        }}]
        results = process_chunk(chunk)
        assert results == [[{'binding': 'my-repo'}]]

    def test_no_match_returns_empty_list(self):
        chunk = [{'kwargs': {
            'alm': 'github',
            'repository': 'no-match',
            'repo_slug': 'my-repo',
            'slug': 's',
            'label': 'l',
            'integration_key': 'ik',
            'output_key': 'binding',
        }}]
        results = process_chunk(chunk)
        assert results == [[]]

    def test_multiple_items(self):
        chunk = [
            {'kwargs': {
                'alm': 'github',
                'repository': 'repo-a',
                'repo_slug': 'repo-a',
                'slug': 's',
                'label': 'l',
                'integration_key': 'ik',
                'output_key': 'k',
            }},
            {'kwargs': {
                'alm': 'github',
                'repository': 'repo-b',
                'repo_slug': 'repo-x',  # no match
                'slug': 's',
                'label': 'l',
                'integration_key': 'ik',
                'output_key': 'k',
            }},
        ]
        results = process_chunk(chunk)
        assert results == [[{'k': 'repo-a'}], []]
