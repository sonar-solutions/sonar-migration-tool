"""Tests for unix shell script parsing and project key extraction"""
import pytest
from pipelines.runtimes.unix import update_script, process_command_parts, process_command_list


class TestUpdateScript:
    def test_simple_sonar_scanner_command(self):
        script = "sonar-scanner -Dsonar.projectKey=my-project -Dsonar.projectName=My Project"
        updated, mapping = update_script(script=script, root_dir='./', dir_project_mapping={})

        assert 'sonar.projectKey' not in updated
        assert 'sonar.projectName' not in updated
        assert './' in mapping
        assert 'my-project' in mapping['./']['projects']

    def test_maven_command(self):
        script = "mvn sonar:sonar -Dsonar.projectKey=my-project"
        updated, mapping = update_script(script=script, root_dir='./', dir_project_mapping={})

        assert 'sonar.projectKey' not in updated
        assert './' in mapping
        assert 'maven' in mapping['./']['scanners']

    def test_gradle_command(self):
        script = "gradle sonarqube -Dsonar.projectKey=my-project"
        _, mapping = update_script(script=script, root_dir='./', dir_project_mapping={})

        assert './' in mapping
        assert 'gradle' in mapping['./']['scanners']

    def test_no_sonar_command(self):
        script = "npm install && npm test"
        updated, mapping = update_script(script=script, root_dir='./', dir_project_mapping={})

        assert updated  # script preserved
        # No sonar scanners or projects detected
        assert all(not v['scanners'] and not v['projects'] for v in mapping.values())

    def test_cd_then_sonar_scanner(self):
        script = "cd subproject && sonar-scanner -Dsonar.projectKey=sub-project"
        updated, mapping = update_script(script=script, root_dir='./', dir_project_mapping={})

        assert 'sonar.projectKey' not in updated
        # Project should be tracked under the subdirectory
        assert any('subproject' in k for k in mapping.keys())

    def test_project_key_in_script_root_dir_not_in_mapping(self):
        """Bug fix: root_dir KeyError when sonar commands only found in subdirectory"""
        # Simulate: process_command_list found sonar in a subdir, so root_dir not in mapping
        # but script still has sonar.projectKey tokens
        preloaded_mapping = {
            './subdir': {'projects': set(), 'scanners': {'cli'}}
        }
        script = "sonar-scanner -Dsonar.projectKey=my-project"
        # Should not raise KeyError
        _, mapping = update_script(script=script, root_dir='./', dir_project_mapping=preloaded_mapping)

        assert './' in mapping
        assert 'my-project' in mapping['./']['projects']

    def test_parse_error_falls_back_gracefully(self):
        """Unparseable script should not raise, mapping falls back to root_dir"""
        script = "echo 'unclosed"  # malformed shell — may trigger parse error
        # Should not raise
        try:
            update_script(script=script, root_dir='./', dir_project_mapping={})
        except Exception as exc:
            pytest.fail(f"update_script raised unexpectedly: {exc}")

    def test_empty_script(self):
        script = ""
        updated, _ = update_script(script=script, root_dir='./', dir_project_mapping={})

        assert updated == ""


class TestProcessCommandParts:
    def _make_word_part(self, word):
        """Minimal stand-in for bashlex word part"""
        class Part:
            kind = 'word'
            def __init__(self, w):
                self.word = w
        return Part(word)

    def _make_non_word_part(self):
        class Part:
            kind = 'operator'
            word = ''
        return Part()

    def test_sonar_scanner_command(self):
        parts = [self._make_word_part('sonar-scanner'),
                 self._make_word_part('-Dsonar.projectKey=my-key')]
        _, runs_sonar, project, scanners = process_command_parts(
            parts=parts, current_dir='./', root_directory='./'
        )
        assert runs_sonar is True
        assert project == 'my-key'
        assert 'cli' in scanners

    def test_cd_changes_directory(self):
        parts = [self._make_word_part('cd'), self._make_word_part('subdir')]
        current_dir, runs_sonar, _, _ = process_command_parts(
            parts=parts, current_dir='./', root_directory='./'
        )
        assert 'subdir' in current_dir
        assert runs_sonar is False

    def test_cd_bare_resets_to_root(self):
        """cd with no arg resets to root_directory"""
        parts = [self._make_word_part('cd')]
        current_dir, _, _, _ = process_command_parts(
            parts=parts, current_dir='./subdir', root_directory='./'
        )
        assert current_dir == './'

    def test_unrelated_command(self):
        parts = [self._make_word_part('echo'), self._make_word_part('hello')]
        _, runs_sonar, project, scanners = process_command_parts(
            parts=parts, current_dir='./', root_directory='./'
        )
        assert runs_sonar is False
        assert project is None
        assert not scanners

    def test_non_word_parts_ignored(self):
        parts = [self._make_non_word_part(), self._make_word_part('sonar-scanner')]
        _, runs_sonar, _, _ = process_command_parts(
            parts=parts, current_dir='./', root_directory='./'
        )
        # sonar-scanner not at index 0 after non-word leading part, so not detected
        assert runs_sonar is False
