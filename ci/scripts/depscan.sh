#!/bin/bash

# Ensure jq is installed
if ! command -v jq &> /dev/null; then
    echo "Error: 'jq' is not installed. Please install it first."
    exit 1
fi

# Ensure GitHub CLI is installed
if ! command -v gh &> /dev/null; then
    echo "Error: GitHub CLI ('gh') is not installed. Please install it from https://cli.github.com/"
    exit 1
fi

# Verify gh is authenticated
if ! gh auth status &> /dev/null; then
    echo "Error: You are not logged into GitHub CLI."
    echo "Please run 'gh auth login' first to authenticate and avoid rate limits."
    exit 1
fi

# Determine 3 years ago in seconds (handles both Linux/GNU and BSD/macOS date versions)
if date --version &>/dev/null; then
    THRESHOLD=$(date -d "3 years ago" +%s)
    IS_GNU_DATE=true
else
    THRESHOLD=$(date -v-3y +%s)
    IS_GNU_DATE=false
fi

echo "Fetching all direct and indirect dependencies..."

# Extract unique GitHub modules exactly as they are known to Go
MODULES=$(go list -m all | awk '{print $1}' | grep '^github\.com/' | sort -u)

echo "Scanning repositories..."
echo "-------------------------------------------------------------------"

for MODULE in $MODULES; do
    # 1. Run `go mod why` first to see if the dependency is actually needed
    WHY_OUTPUT=$(go mod why -m "$MODULE" 2>/dev/null)

    # If the module is not needed by the main module, skip it immediately
    if echo "$WHY_OUTPUT" | grep -q "main module does not need module"; then
        continue
    fi

    # 2. Extract the base GitHub repo (owner/name) from the module path
    REPO=$(echo "$MODULE" | sed -E 's|^github\.com/([^/]+)/([^/]+).*|\1/\2|')

    # 3. Fetch repository metadata using gh api
    RESPONSE=$(gh api "repos/$REPO" 2>/dev/null)

    if [ $? -ne 0 ]; then
        continue
    fi

    # 4. Extract archived status and last push date
    ARCHIVED=$(echo "$RESPONSE" | jq -r '.archived')
    PUSHED_AT=$(echo "$RESPONSE" | jq -r '.pushed_at')

    if [ "$PUSHED_AT" = "null" ] || [ -z "$PUSHED_AT" ]; then
        continue
    fi

    # 5. Convert pushed_at string to seconds
    if $IS_GNU_DATE; then
        LAST_COMMIT_SEC=$(date -d "$PUSHED_AT" +%s)
    else
        LAST_COMMIT_SEC=$(date -j -f "%Y-%m-%dT%H:%M:%SZ" "$PUSHED_AT" +%s)
    fi

    # 6. Evaluate if the repository is stale
    IS_STALE=false
    if [ "$LAST_COMMIT_SEC" -lt "$THRESHOLD" ]; then
        IS_STALE=true
    fi

    # 7. Print the report if it matches the criteria
    if [ "$ARCHIVED" = "true" ] || [ "$IS_STALE" = "true" ]; then
        FORMATTED_DATE=$(echo "$PUSHED_AT" | cut -d'T' -f1)

        REASONS=()
        if [ "$ARCHIVED" = "true" ]; then
            REASONS+=("Archived")
        fi
        if [ "$IS_STALE" = "true" ]; then
            REASONS+=("Inactive > 3 years")
        fi

        REASON_STR=$(IFS=", "; echo "${REASONS[*]}")
        echo "[!] $MODULE"
        echo "    Status: $REASON_STR"
        echo "    Last updated: $FORMATTED_DATE"
        echo "    Dependency chain (go mod why):"

        # Reuse the previously captured WHY_OUTPUT and indent it
        echo "$WHY_OUTPUT" | sed 's/^/      /'
        echo ""
    fi
done

echo "Scan complete."