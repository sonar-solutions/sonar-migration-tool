"""Tests for scanner config file update modules"""
import pytest


PROJECT_MAPPINGS = {
    'my-project': {
        'key': 'cloud-org_my-project',
        'name': 'My Project',
        'sonarCloudOrgKey': 'cloud-org',
    }
}


class TestCliScanner:
    def setup_method(self):
        from pipelines.scanners import cli
        self.cli = cli

    def test_get_config_file_name(self):
        names = self.cli.get_config_file_name()
        assert 'sonar-project.properties' in names

    def test_update_existing_content(self):
        content = "sonar.projectKey=old-key\nsonar.projectName=Old Name\nsonar.sources=src\n"
        result = self.cli.update_content(
            content=content,
            projects={'my-project'},
            project_mappings=PROJECT_MAPPINGS
        )

        assert result['is_updated'] is True
        assert 'cloud-org_my-project' in result['updated_content']
        assert 'My Project' in result['updated_content']
        assert 'cloud-org' in result['updated_content']
        # Non-sonar lines preserved
        assert 'sonar.sources=src' in result['updated_content']

    def test_update_adds_missing_keys(self):
        content = "sonar.sources=src\n"
        result = self.cli.update_content(
            content=content,
            projects={'my-project'},
            project_mappings=PROJECT_MAPPINGS
        )

        assert result['is_updated'] is True
        assert 'sonar.projectKey=cloud-org_my-project' in result['updated_content']

    def test_empty_content_with_no_mapping(self):
        """Bug fix: empty content + no matching project should return is_updated=False"""
        result = self.cli.update_content(
            content='',
            projects={'unknown-project'},
            project_mappings=PROJECT_MAPPINGS
        )

        assert result['is_updated'] is False

    def test_empty_content_with_matching_project(self):
        result = self.cli.update_content(
            content='',
            projects={'my-project'},
            project_mappings=PROJECT_MAPPINGS
        )

        assert result['is_updated'] is True
        assert 'cloud-org_my-project' in result['updated_content']

    def test_project_not_in_mappings(self):
        """Project in projects set but not in project_mappings — no update"""
        result = self.cli.update_content(
            content='sonar.projectKey=old-key\n',
            projects={'unknown-project'},
            project_mappings=PROJECT_MAPPINGS
        )

        assert result['is_updated'] is False


class TestGradleScanner:
    def setup_method(self):
        from pipelines.scanners import gradle
        self.gradle = gradle

    def test_get_config_file_names(self):
        names = self.gradle.get_config_file_name()
        assert 'build.gradle' in names
        assert 'build.gradle.kts' in names

    def test_update_content_is_noop(self):
        """Gradle update_content is a no-op stub"""
        content = "sonarqube { properties { property 'sonar.projectKey', 'old' } }"
        result = self.gradle.update_content(content, {'my-project'}, PROJECT_MAPPINGS)

        assert result['is_updated'] is False
        assert result['updated_content'] == content


class TestDotnetScanner:
    def setup_method(self):
        from pipelines.scanners import dotnet
        self.dotnet = dotnet

    def test_get_config_file_name(self):
        names = self.dotnet.get_config_file_name()
        assert len(names) > 0

    def test_update_content_is_noop(self):
        content = "some dotnet sonar config"
        result = self.dotnet.update_content(content, {'my-project'}, PROJECT_MAPPINGS)

        assert result['is_updated'] is False
        assert result['updated_content'] == content


class TestMavenScanner:
    def setup_method(self):
        from pipelines.scanners import maven
        self.maven = maven

    def test_get_config_file_name(self):
        names = self.maven.get_config_file_name()
        assert 'pom.xml' in names

    def test_update_existing_pom(self):
        content = """<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0"
         xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
         xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 http://maven.apache.org/xsd/maven-4.0.0.xsd">
  <modelVersion>4.0.0</modelVersion>
  <properties>
    <sonar.projectKey>old-key</sonar.projectKey>
    <sonar.projectName>Old Name</sonar.projectName>
    <sonar.organization>old-org</sonar.organization>
  </properties>
</project>"""
        result = self.maven.update_content(
            content=content,
            projects={'my-project'},
            project_mappings=PROJECT_MAPPINGS
        )

        assert result['is_updated'] is True
        assert 'cloud-org_my-project' in result['updated_content']
        assert 'cloud-org' in result['updated_content']

    def test_update_pom_without_sonar_properties(self):
        content = """<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0"
         xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
         xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 http://maven.apache.org/xsd/maven-4.0.0.xsd">
  <modelVersion>4.0.0</modelVersion>
  <properties>
    <some.other.prop>value</some.other.prop>
  </properties>
</project>"""
        result = self.maven.update_content(
            content=content,
            projects={'my-project'},
            project_mappings=PROJECT_MAPPINGS
        )

        assert result['is_updated'] is True
        assert 'cloud-org_my-project' in result['updated_content']
