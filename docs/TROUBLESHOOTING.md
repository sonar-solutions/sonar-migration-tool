# Troubleshooting

This guide covers common issues you might run into when using the SonarQube to SonarCloud migration tool, along with solutions to get you back on track.

---

## Finding Logs

When something goes wrong, logs are your best friend. Here is where to find them:

- **Request logs**: `./files/<extract_id>/requests.log` -- This file contains a detailed log of every HTTP request and response made during the migration. It is the first place to look when diagnosing issues.
- **Docker logs**: Run `docker logs <container_id>` to view the output from the Docker container.
- **Analysis report**: Run the `analysis_report <RUN_ID>` command to generate a CSV summary of everything that happened during a migration run. This is a great way to get a high-level view of what succeeded and what failed.

---

## Common Errors

### "Token does not have sufficient permissions"

**Cause**: The admin token you provided does not have the required permissions.

**Solution**:

- For your **SonarQube Server** token, ensure it has the following permissions:
  - Administer System
  - Quality Gates
  - Quality Profiles
- For your **SonarCloud** token, ensure the user has:
  - Enterprise admin privileges
  - Organization admin privileges for all target organizations

---

### "Organization not found"

**Cause**: The `sonarcloud_org_key` value in your `organizations.csv` file does not match any organization in SonarCloud.

**Solution**:

1. Log in to SonarCloud and verify that the organization exists.
2. Double-check for typos in the org key -- even a single character difference will cause this error.
3. Make sure the organization is part of your SonarCloud enterprise.

---

### "Request timeout"

**Cause**: Large datasets or a slow network connection can cause requests to take longer than the default timeout allows.

**Solution**: Increase the timeout value using the `--timeout` flag. For example:

```bash
sonar-reports extract --timeout 120
```

A value of 120 seconds is a good starting point for larger instances.

---

### "Connection refused" or SSL errors

**Cause**: Network connectivity issues or certificate problems, especially with self-signed certificates.

**Solution**:

1. Verify that the SonarQube URL is accessible from the machine running the migration tool. Try opening it in a browser or using `curl`.
2. For self-signed certificates, use the `--pem_file_path` and `--key_file_path` options to provide your certificate files.
3. If you are running the tool via Docker, make sure to mount the certificate files into the container:

```bash
docker run -v /path/to/certs:/certs \
  sonar-reports extract \
  --pem_file_path /certs/my-cert.pem \
  --key_file_path /certs/my-key.key
```

---

### Migration task fails midway

**Cause**: This can happen due to an API error, rate limiting, or a network interruption.

**Solution**: Resume the migration using the `--run_id` flag with the ID from the failed run:

```bash
sonar-reports extract --run_id <RUN_ID>
```

The tool keeps track of which tasks have already been completed and will skip them automatically. You do not need to start over from scratch.

---

### No Projects Extracted

If the tool completes without extracting any projects, check the following:

- Verify that your token has admin-level permissions on the SonarQube instance.
- Confirm that projects actually exist in your SonarQube instance.
- Review the `requests.log` file for any API errors that might explain why projects were not returned.

---

### Authentication Errors

If you are seeing authentication failures:

- Verify that your tokens are valid and have not expired.
- Be aware that SonarQube handles authentication differently depending on the version:
  - **SonarQube Server < 10**: Uses Basic authentication.
  - **SonarQube Server >= 10**: Uses Bearer token authentication.
- For SonarCloud, the token must belong to a user with enterprise-level admin permissions.

---

### API Rate Limiting

If you are hitting API rate limits, try reducing the concurrency and increasing the timeout:

```bash
sonar-reports extract --concurrency 5 --timeout 120
```

This slows down the number of parallel requests, which helps avoid triggering rate limits.

---

### Docker: "localhost" Not Working

If your SonarQube instance is running on your local machine and you are using Docker to run the migration tool, `localhost` will not work. Docker containers have their own network namespace, so `localhost` inside the container refers to the container itself, not your host machine.

**Solution**: Use `host.docker.internal` instead of `localhost` in your SonarQube URL:

```bash
docker run sonar-reports extract --sq_url http://host.docker.internal:9000
```

---

## Reducing Memory Usage

For large SonarQube instances with 50,000 or more projects, the migration tool can consume a significant amount of memory. To reduce memory usage, lower the concurrency setting to 10 or fewer:

```bash
sonar-reports extract --concurrency 10
```

This limits the number of operations being processed in parallel, which reduces the amount of data held in memory at any given time.

---

## Resetting After a Bad Migration

If a migration went wrong and you need to start fresh, use the `reset` command to wipe everything in the SonarCloud enterprise and start over:

```bash
sonar-reports reset
```

**Warning: This is a destructive operation.** The reset command deletes all migrated data in your SonarCloud enterprise, including projects, quality profiles, quality gates, and organization configurations. Use this only when you are certain you want to start from scratch.

---

## Getting Help

If you are still stuck after reviewing this guide, here is how to get help:

1. Check the log files located in `files/*/requests.log` for detailed error information.
2. Generate an analysis report using the `analysis_report` command to get a summary of what happened.
3. Search the repository for similar issues -- someone else may have encountered the same problem.
4. If none of the above helps, open a new issue in the repository and include the following:
   - The command you ran (make sure to redact any tokens or sensitive credentials).
   - The full error message you received.
   - Relevant excerpts from the log files.
