# Docker Guide

## Overview

The SonarQube-to-SonarCloud migration tool is published as a ready-to-use Docker image at:

```
ghcr.io/sonar-solutions/sonar-reports:latest
```

You do not need Python, pip, or any other dependencies installed on your machine — just Docker. Everything the tool needs is already baked into the image.

## Basic Usage

Mount a local directory so that generated files (CSVs, configs, logs) persist between commands:

```bash
docker run -v /path/to/local/files:/app/files ghcr.io/sonar-solutions/sonar-reports:latest <command>
```

Replace `<command>` with any of the available migration commands described below.

## Using the Interactive Wizard

The wizard walks you through the migration step by step with prompts. Because it is interactive, you need the `-it` flags so Docker connects your terminal to the container:

```bash
docker run -it -v ./files:/app/files ghcr.io/sonar-solutions/sonar-reports:latest wizard
```

> **Note:** The `-it` flag is required for interactive mode. Without it the wizard will not be able to read your input and will exit immediately.

## Step-by-Step Commands

If you prefer to run each migration phase individually, here is the full sequence:

### 1. Extract

Pull project data from your SonarQube Server:

```bash
docker run --rm -v "$(pwd)/files:/app/files" ghcr.io/sonar-solutions/sonar-reports:latest extract https://your-sonarqube-server.com YOUR_SONARQUBE_TOKEN
```

### 2. Structure

Generate the organizational structure files:

```bash
docker run --rm -v "$(pwd)/files:/app/files" ghcr.io/sonar-solutions/sonar-reports:latest structure
```

### 3. Edit organizations.csv manually

Open `files/organizations.csv` on your host machine in any spreadsheet editor or text editor. Map your SonarQube projects to the desired SonarCloud organizations. Save the file when you are done — because the directory is mounted, the container will see your changes on the next run.

### 4. Mappings

Generate the mapping files based on your edited organizations:

```bash
docker run --rm -v "$(pwd)/files:/app/files" ghcr.io/sonar-solutions/sonar-reports:latest mappings
```

### 5. Migrate

Run the actual migration to SonarCloud:

```bash
docker run --rm -v "$(pwd)/files:/app/files" ghcr.io/sonar-solutions/sonar-reports:latest migrate
```

## Docker Compose

For repeated use, a `docker-compose.yaml` file can save you from typing long commands. Here is an example:

```yaml
version: "3.8"

services:
  sonar-migration:
    image: ghcr.io/sonar-solutions/sonar-reports:latest
    volumes:
      - ./files:/app/files
      # Uncomment the line below if you need client certificates (mTLS)
      # - ./certs:/app/certs
```

Run commands through Compose like this:

```bash
docker compose run sonar-migration extract https://your-sonarqube-server.com YOUR_SONARQUBE_TOKEN
docker compose run sonar-migration structure
docker compose run sonar-migration mappings
docker compose run sonar-migration migrate
```

Or use the wizard interactively:

```bash
docker compose run sonar-migration wizard
```

## Client Certificates (mTLS)

If your SonarQube Server requires mutual TLS (client certificates), mount your certificate files into the container and pass the paths as arguments:

```bash
docker run -v ./files:/app/files -v ./certs:/app/certs ghcr.io/sonar-solutions/sonar-reports:latest extract https://your-server.com YOUR_TOKEN --pem_file_path /app/certs/client.pem --key_file_path /app/certs/client.key --cert_password YOUR_PASSWORD
```

Make sure the paths inside the container (`/app/certs/...`) match the volume mount target.

## Important: Accessing localhost

If your SonarQube Server is running on `localhost` on your host machine, the container cannot reach it using `localhost` (that would refer to the container itself). Instead, use Docker's special hostname:

```
host.docker.internal
```

For example:

```bash
docker run --rm -v "$(pwd)/files:/app/files" ghcr.io/sonar-solutions/sonar-reports:latest extract http://host.docker.internal:9000 YOUR_TOKEN
```

This works on Docker Desktop for macOS and Windows. On Linux you may also need to add `--add-host=host.docker.internal:host-gateway` to the `docker run` command.

## Building the Docker Image Locally

If you want to build the image from source (for example, to test local changes), run:

```bash
docker build -t sonar-reports:local .
```

Then use `sonar-reports:local` in place of `ghcr.io/sonar-solutions/sonar-reports:latest` in any of the commands above.

## Migration Script (Automated)

The repository includes `scripts/execute_full_migration.sh`, a shell script that wraps the Docker commands for a fully automated end-to-end migration. To use it:

1. Open the script and fill in your SonarQube and SonarCloud credentials.
2. Make it executable and run it:

```bash
chmod +x scripts/execute_full_migration.sh
./scripts/execute_full_migration.sh
```

This is a good option when you want a repeatable, hands-off migration process.
