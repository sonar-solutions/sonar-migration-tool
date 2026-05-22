#!/usr/bin/env bash
#
# delete_all_portfolios.sh
#
# One-off script to delete every portfolio in a SonarQube Cloud enterprise.
#
# Usage:
#   ./delete_all_portfolios.sh <SQC_URL> <TOKEN> <ENTERPRISE_KEY>
#
# Example:
#   ./delete_all_portfolios.sh https://sonarcloud.io squ_xxx my-enterprise
#
# Requirements: bash, curl, jq.
#
# What it does:
#   1. Looks up the enterprise UUID from the provided enterprise key
#      via GET /enterprises/enterprises.
#   2. Paginates GET /enterprises/portfolios?enterpriseId=<uuid> to list
#      every portfolio in that enterprise.
#   3. Issues DELETE /enterprises/portfolios/<id> for each one.
#
# Notes:
#   - The enterprise endpoints live on api.<SQC_URL_HOST> (e.g.
#     https://api.sonarcloud.io), not on the public SQC host.
#   - This is destructive and irreversible. Re-confirm the target before running.

set -euo pipefail

if [[ $# -ne 3 ]]; then
  echo "usage: $0 <SQC_URL> <TOKEN> <ENTERPRISE_KEY>" >&2
  exit 2
fi

SQC_URL="$1"
TOKEN="$2"
ENTERPRISE_KEY="$3"

for cmd in curl jq; do
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "error: '$cmd' is required but not installed" >&2
    exit 2
  fi
done

# Normalise the URL and derive the enterprise API host.
SQC_URL="${SQC_URL%/}"
case "$SQC_URL" in
  https://api.*) API_URL="$SQC_URL" ;;
  https://*)     API_URL="${SQC_URL/https:\/\//https://api.}" ;;
  http://api.*)  API_URL="$SQC_URL" ;;
  http://*)      API_URL="${SQC_URL/http:\/\//http://api.}" ;;
  *)
    echo "error: SQC URL must start with http(s)://" >&2
    exit 2
    ;;
esac

AUTH_HEADER="Authorization: Bearer ${TOKEN}"

# Wrapper: GET that fails loudly on non-2xx.
http_get() {
  local url="$1"
  local body status
  body=$(curl -sS -H "$AUTH_HEADER" -H "Accept: application/json" \
              -w $'\n%{http_code}' "$url")
  status="${body##*$'\n'}"
  body="${body%$'\n'*}"
  if [[ "$status" -lt 200 || "$status" -ge 300 ]]; then
    echo "GET $url returned HTTP $status: $body" >&2
    return 1
  fi
  printf '%s' "$body"
}

# Wrapper: DELETE that returns the status code (no body in 204).
http_delete() {
  local url="$1"
  curl -sS -o /dev/null -w '%{http_code}' \
       -X DELETE -H "$AUTH_HEADER" "$url"
}

echo "Looking up enterprise '$ENTERPRISE_KEY' on $API_URL..."
ENT_LIST_JSON=$(http_get "$API_URL/enterprises/enterprises")
# The SQC API returns either a bare array of enterprises or an envelope
# {"enterprises": [...]}. Normalise to an array, then filter by key.
ENTERPRISE_ID=$(jq -r --arg key "$ENTERPRISE_KEY" '
                   (if type == "array" then . else (.enterprises // []) end)
                   | .[]? | select(.key == $key) | .id
                ' <<<"$ENT_LIST_JSON")

if [[ -z "$ENTERPRISE_ID" || "$ENTERPRISE_ID" == "null" ]]; then
  echo "error: no enterprise with key '$ENTERPRISE_KEY' is visible to this token" >&2
  exit 1
fi
echo "Enterprise UUID: $ENTERPRISE_ID"

# Page through GET /enterprises/portfolios?enterpriseId=...
PAGE_SIZE=50
page=1
PORTFOLIOS_JSON='[]'
while :; do
  url="$API_URL/enterprises/portfolios?enterpriseId=$ENTERPRISE_ID&pageIndex=$page&pageSize=$PAGE_SIZE"
  page_json=$(http_get "$url")
  count=$(jq '.portfolios | length' <<<"$page_json")
  total=$(jq '.page.total // 0' <<<"$page_json")
  PORTFOLIOS_JSON=$(jq -s '.[0] + (.[1].portfolios // [])' \
                       <(printf '%s' "$PORTFOLIOS_JSON") \
                       <(printf '%s' "$page_json"))
  fetched=$(jq 'length' <<<"$PORTFOLIOS_JSON")
  echo "  page $page: fetched $count (running total: $fetched / $total)"
  if [[ "$count" -lt "$PAGE_SIZE" ]] || [[ "$total" -gt 0 && "$fetched" -ge "$total" ]]; then
    break
  fi
  page=$((page + 1))
done

total_to_delete=$(jq 'length' <<<"$PORTFOLIOS_JSON")
if [[ "$total_to_delete" -eq 0 ]]; then
  echo "No portfolios to delete."
  exit 0
fi

echo "Found $total_to_delete portfolio(s) to delete."
deleted=0
failed=0
while IFS=$'\t' read -r id name; do
  [[ -z "$id" ]] && continue
  status=$(http_delete "$API_URL/enterprises/portfolios/$id")
  if [[ "$status" -ge 200 && "$status" -lt 300 ]]; then
    echo "  deleted [$id] $name"
    deleted=$((deleted + 1))
  else
    echo "  FAILED [$id] $name (HTTP $status)" >&2
    failed=$((failed + 1))
  fi
done < <(jq -r '.[] | "\(.id)\t\(.name)"' <<<"$PORTFOLIOS_JSON")

echo "Done. Deleted $deleted, failed $failed (out of $total_to_delete)."
if [[ "$failed" -gt 0 ]]; then
  exit 1
fi
