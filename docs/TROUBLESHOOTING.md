# Troubleshooting

This guide covers common issues when using the SonarQube to SonarCloud migration tool.

---

## Finding Logs

- **Request logs**: `./files/<extract_id>/requests.log` -- Detailed log of every HTTP request and response. Check here first when diagnosing issues.
- **Analysis report**: Run `sonar-migration-tool analysis_report <RUN_ID> --export_directory ./files/` to generate a CSV summary of a migration run.

---

## Common Errors

### "Token does not have sufficient permissions"

**Cause**: The admin token does not have the required permissions.

**Solution**:

- **SonarQube Server** token needs: Administer System, Quality Gates, Quality Profiles
- **SonarCloud** token needs: Enterprise admin privileges, Organization admin for all target organizations

---

### "Organization not found"

**Cause**: The `sonarcloud_org_key` in `organizations.csv` does not match any organization in SonarCloud.

**Solution**:

1. Log in to SonarCloud and verify the organization exists.
2. Check for typos in the org key.
3. Make sure the organization is part of your SonarCloud enterprise.

---

### "Request timeout"

**Cause**: Large datasets or slow network can exceed the default timeout.

**Solution**: Increase the timeout:

```bash
sonar-migration-tool extract <URL> <TOKEN> --timeout 120 --export_directory ./files/
```

---

### "Connection refused" or SSL errors

**Cause**: Network connectivity issues or certificate problems.

**Solution**:

1. Verify the SonarQube URL is accessible (try opening it in a browser or using `curl`).
2. For self-signed certificates, use mTLS options:

```bash
sonar-migration-tool extract <URL> <TOKEN> \
  --pem_file_path ./certs/client.pem \
  --key_file_path ./certs/client.key \
  --export_directory ./files/
```

---

### Migration task fails midway

**Cause**: API error, rate limiting, or network interruption.

**Solution**: Resume using the `--run_id` flag:

```bash
sonar-migration-tool migrate <TOKEN> <ENTERPRISE_KEY> --run_id <RUN_ID> --export_directory ./files/
```

Completed tasks are skipped automatically.

---

### No Projects Extracted

- Verify your token has admin-level permissions.
- Confirm projects exist in your SonarQube instance.
- Review `requests.log` for API errors.

---

### Authentication Errors

- Verify tokens are valid and not expired.
- **SonarQube Server < 10**: Uses Basic authentication.
- **SonarQube Server >= 10**: Uses Bearer token authentication.
- SonarCloud tokens must belong to a user with enterprise admin permissions.

---

### API Rate Limiting

Reduce concurrency and increase timeout:

```bash
sonar-migration-tool extract <URL> <TOKEN> --concurrency 5 --timeout 120 --export_directory ./files/
```

---

## Reducing Memory Usage

For large instances (50,000+ projects), lower the concurrency:

```bash
sonar-migration-tool extract <URL> <TOKEN> --concurrency 10 --export_directory ./files/
```

---

## Resetting After a Bad Migration

Use the `reset` command to delete all migrated content in the SonarCloud enterprise:

```bash
sonar-migration-tool reset <TOKEN> <ENTERPRISE_KEY> --export_directory ./files/
```

**Warning: This is destructive.** It deletes all migrated projects, quality profiles, quality gates, and organization configurations.

---

## Getting Help

1. Check `files/*/requests.log` for detailed error information.
2. Generate an analysis report: `sonar-migration-tool analysis_report <RUN_ID> --export_directory ./files/`
3. Search the [repository issues](https://github.com/sonar-solutions/sonar-migration-tool/issues).
4. Open a new issue with: the command you ran (redact tokens), the full error message, and relevant log excerpts.
