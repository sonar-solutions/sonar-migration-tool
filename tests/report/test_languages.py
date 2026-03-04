"""Tests for src/report/maturity/languages.py"""
from report.maturity.languages import process_languages, generate_language_markdown


class TestProcessLanguages:
    def test_basic_language_extraction(self):
        measures = {
            'server1': {
                'proj1': {'ncloc_language_distribution': 'java=1000;python=500'},
            }
        }
        result = process_languages(measures=measures, profile_map={})
        assert 'java' in result
        assert result['java']['loc'] == 1000
        assert result['java']['projects'] == 1
        assert 'python' in result
        assert result['python']['loc'] == 500

    def test_aggregates_across_projects(self):
        measures = {
            'server1': {
                'proj1': {'ncloc_language_distribution': 'java=1000'},
                'proj2': {'ncloc_language_distribution': 'java=500'},
            }
        }
        result = process_languages(measures=measures, profile_map={})
        assert result['java']['loc'] == 1500
        assert result['java']['projects'] == 2

    def test_skips_empty_distribution(self):
        measures = {
            'server1': {
                'proj1': {'ncloc_language_distribution': ''},
            }
        }
        result = process_languages(measures=measures, profile_map={})
        assert result == {}

    def test_skips_malformed_distribution_entry(self):
        measures = {
            'server1': {
                'proj1': {'ncloc_language_distribution': 'java=1000;malformed_no_equals'},
            }
        }
        result = process_languages(measures=measures, profile_map={})
        assert 'java' in result
        assert 'malformed_no_equals' not in result

    def test_profile_map_enriches_language_data(self):
        measures = {
            'server1': {
                'proj1': {'ncloc_language_distribution': 'java=1000'},
            }
        }
        profile_map = {
            'server1': {
                'java': {
                    'sonar way': {
                        'rules': ['r1', 'r2', 'r3'],
                        'projects': ['p1'],
                        'is_built_in': True,
                    },
                    'custom-java': {
                        'rules': ['r1', 'r2', 'r3', 'r4', 'r5'],
                        'projects': ['p1'],
                        'is_built_in': False,
                    }
                }
            }
        }
        result = process_languages(measures=measures, profile_map=profile_map)
        assert result['java']['base_rules'] == 3
        assert result['java']['custom_profiles'] == 1
        assert result['java']['profiles'] == 2

    def test_profile_without_projects_is_skipped(self):
        measures = {
            'server1': {
                'proj1': {'ncloc_language_distribution': 'java=1000'},
            }
        }
        profile_map = {
            'server1': {
                'java': {
                    'unused-profile': {
                        'rules': ['r1'],
                        'projects': [],
                        'is_built_in': False,
                    }
                }
            }
        }
        result = process_languages(measures=measures, profile_map=profile_map)
        assert result['java']['profiles'] == 0

    def test_profile_for_unknown_language_is_skipped(self):
        measures = {}
        profile_map = {
            'server1': {
                'cobol': {
                    'some-profile': {'rules': [], 'projects': ['p1'], 'is_built_in': False}
                }
            }
        }
        result = process_languages(measures=measures, profile_map=profile_map)
        assert 'cobol' not in result

    def test_missing_ncloc_field_falls_back_to_empty(self):
        measures = {
            'server1': {
                'proj1': {},  # no ncloc_language_distribution
            }
        }
        result = process_languages(measures=measures, profile_map={})
        assert result == {}

    def test_generate_language_markdown_returns_string(self):
        measures = {
            'server1': {
                'proj1': {'ncloc_language_distribution': 'java=500'},
            }
        }
        markdown, languages = generate_language_markdown(measures=measures, profile_map={})
        assert isinstance(markdown, str)
        assert 'java' in languages
