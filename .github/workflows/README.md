# GitHub Actions Workflows

This directory contains automated workflows for the sonar-reports project.

## Active Workflows

### 1. `test.yml` - Pull Request Testing
**Trigger:** Pull requests to `main` branch
**Purpose:** Run automated tests using Docker Compose
**What it does:**
- Checks out the PR code
- Runs the test suite in Docker containers
- Validates code quality before merging

### 2. `build.yml` - Docker Image Publishing
**Trigger:** Push to `main` branch
**Purpose:** Build and publish the Docker image to GitHub Container Registry
**What it does:**
- Builds the Docker image from the repository
- Pushes to `ghcr.io/sonar-solutions/sonar-reports:latest`
- Requires `CR_PAT` secret for authentication
