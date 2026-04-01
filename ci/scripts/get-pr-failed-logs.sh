#!/bin/bash

# Exit on error, undefined variables, and pipe failures
set -euo pipefail

# Check for required arguments
if [ -z "${1:-}" ]; then
    echo "Usage: $0 <PR_NUMBER> [REPO]"
    echo "Example: $0 123"
    echo "         $0 123 hyperledger/fabric"
    exit 1
fi

PR_NUMBER=$1
REPO_ARG=""

if [ -n "${2:-}" ]; then
    REPO_ARG="--repo $2"
fi

# Ensure gh CLI is installed
if ! command -v gh &> /dev/null; then
    echo "Error: GitHub CLI (gh) is not installed. Please install it first."
    exit 1
fi

echo "🔍 Finding branch for PR #$PR_NUMBER..."
BRANCH_NAME=$(gh pr view "$PR_NUMBER" $REPO_ARG --json headRefName --jq '.headRefName')

if [ -z "$BRANCH_NAME" ]; then
    echo "❌ Could not retrieve branch name for PR #$PR_NUMBER."
    exit 1
fi

echo "✅ Found branch: $BRANCH_NAME"
echo "🔍 Fetching the latest failed workflow run..."

# 1. Get ONLY the last (most recent) failed run ID for the branch
LAST_RUN_ID=$(gh run list $REPO_ARG --branch "$BRANCH_NAME" --status failure -L 1 --json databaseId --jq '.[0].databaseId')

# jq returns "null" if the array is empty
if [ -z "$LAST_RUN_ID" ] || [ "$LAST_RUN_ID" == "null" ]; then
    echo "✅ No failed runs found for PR #$PR_NUMBER (Branch: $BRANCH_NAME)."
    exit 0
fi

# Create an output directory
LOG_DIR="pr_${PR_NUMBER}_failed_logs"
mkdir -p "$LOG_DIR"
echo "📂 Logs will be saved in: ./${LOG_DIR}/"
echo "Inspecting latest failed Run ID: $LAST_RUN_ID"

# 2. Get specific jobs within the run that have a "failure" conclusion
FAILED_JOBS=$(gh run view "$LAST_RUN_ID" $REPO_ARG --json jobs --jq '.jobs[]? | select(.conclusion == "failure") | "\(.databaseId)|\(.name)"')

if [ -z "$FAILED_JOBS" ]; then
    echo "  -> No specific failed jobs found in this run (could be a cancellation or workflow-level error)."
    exit 0
fi

# 3. Process each failed job and fetch its specific log
while IFS='|' read -r JOB_ID JOB_NAME; do
    if [ -z "$JOB_ID" ]; then
        continue
    fi

    # Sanitize the job name to create a safe filename
    SAFE_JOB_NAME=$(echo "$JOB_NAME" | tr -s ' /:<>|' '_')
    LOG_FILE="${LOG_DIR}/${SAFE_JOB_NAME}_${JOB_ID}.log"

    echo "  ⬇️  Downloading and cleaning log for job: $JOB_NAME ($JOB_ID)"

    # Use sed to strip everything up to and including the ISO8601 timestamp
    gh run view --job="$JOB_ID" $REPO_ARG --log | \
        sed -E 's/^.*[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]+Z[[:space:]]*//' > "$LOG_FILE"

done <<< "$FAILED_JOBS"

echo "🚀 Done! Cleaned failed job logs have been extracted."