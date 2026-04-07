#!/usr/bin/env bash
set -euo pipefail

# validate.sh — compare checker test-run results against expectations
# Usage: validate.sh <ndjson_file> <expectations_file>

NDJSON_FILE="${1:-}"
EXPECTATIONS_FILE="${2:-}"

if [[ -z "$NDJSON_FILE" || -z "$EXPECTATIONS_FILE" ]]; then
  echo "Usage: $0 <ndjson_file> <expectations_file>" >&2
  exit 1
fi

if ! command -v jq &>/dev/null; then
  echo "Error: jq is required but not installed." >&2
  exit 1
fi

if [[ ! -f "$NDJSON_FILE" ]]; then
  echo "Error: NDJSON file not found: $NDJSON_FILE" >&2
  exit 1
fi

if [[ ! -f "$EXPECTATIONS_FILE" ]]; then
  echo "Error: Expectations file not found: $EXPECTATIONS_FILE" >&2
  exit 1
fi

# Print header
printf "\n%-30s %-10s %-10s %s\n" "CHECK NAME" "EXPECTED" "ACTUAL" "RESULT"
printf "%-30s %-10s %-10s %s\n" "----------" "--------" "------" "------"

matched=0
total=0
exit_code=0

# Iterate over each expected check
while IFS= read -r check_name; do
  expected_status=$(jq -r --arg name "$check_name" '.[$name]' "$EXPECTATIONS_FILE")

  # Find the last matching entry in the NDJSON by check_name
  actual_status=$(grep -v '^$' "$NDJSON_FILE" \
    | jq -r --arg name "$check_name" 'select(.check_name == $name) | .status' 2>/dev/null \
    | tail -1)

  total=$((total + 1))

  if [[ -z "$actual_status" ]]; then
    result="MISSING"
    exit_code=1
    printf "%-30s %-10s %-10s %s\n" "$check_name" "$expected_status" "(none)" "$result"
    continue
  fi

  if [[ "$actual_status" == "$expected_status" ]]; then
    result="OK"
    matched=$((matched + 1))
  else
    result="MISMATCH"
    exit_code=1
  fi

  printf "%-30s %-10s %-10s %s\n" "$check_name" "$expected_status" "$actual_status" "$result"

done < <(jq -r 'keys[]' "$EXPECTATIONS_FILE")

printf "\n%d/%d checks matched\n\n" "$matched" "$total"

exit "$exit_code"
