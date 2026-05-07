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

# Determine 3 years ago in seconds
if date --version &>/dev/null; then
    THRESHOLD=$(date -d "3 years ago" +%s)
    IS_GNU_DATE=true
else
    THRESHOLD=$(date -v-3y +%s)
    IS_GNU_DATE=false
fi

# Reusable function to check a GitHub repo's status
# Returns a string in the format "status|pushed_at"
function get_repo_status() {
    local REPO=$1
    local RESPONSE=$(gh api "repos/$REPO" 2>/dev/null)

    if [ $? -ne 0 ]; then
        echo "unknown|"
        return
    fi

    local ARCHIVED=$(echo "$RESPONSE" | jq -r '.archived')
    local PUSHED_AT=$(echo "$RESPONSE" | jq -r '.pushed_at')

    if [ "$PUSHED_AT" = "null" ] || [ -z "$PUSHED_AT" ]; then
        echo "unknown|"
        return
    fi

    local LAST_COMMIT_SEC
    if $IS_GNU_DATE; then
        LAST_COMMIT_SEC=$(date -d "$PUSHED_AT" +%s)
    else
        LAST_COMMIT_SEC=$(date -j -f "%Y-%m-%dT%H:%M:%SZ" "$PUSHED_AT" +%s)
    fi

    local STATUS="fine"
    if [ "$ARCHIVED" = "true" ]; then
        STATUS="archived"
    elif [ "$LAST_COMMIT_SEC" -lt "$THRESHOLD" ]; then
        STATUS="stale"
    fi

    echo "$STATUS|$PUSHED_AT"
}

echo "Fetching all direct and indirect dependencies..."

# Get the main module name to identify direct dependencies
MAIN_MODULE=$(go list -m)

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

    # 2. Extract the base GitHub repo (owner/name) and check its status
    REPO=$(echo "$MODULE" | sed -E 's|^github\.com/([^/]+)/([^/]+).*|\1/\2|')
    RESULT=$(get_repo_status "$REPO")

    STATUS="${RESULT%%|*}"
    PUSHED_AT="${RESULT##*|}"

    # 3. Process if the repository is stale or archived
    if [ "$STATUS" = "archived" ] || [ "$STATUS" = "stale" ]; then
        FORMATTED_DATE=$(echo "$PUSHED_AT" | cut -d'T' -f1)

        REASONS=()
        if [ "$STATUS" = "archived" ]; then
            REASONS+=("Archived")
        fi
        if [ "$STATUS" = "stale" ]; then
            REASONS+=("Inactive > 3 years")
        fi
        REASON_STR=$(IFS=", "; echo "${REASONS[*]}")

        # 4. Check the immediate parent module in the dependency chain
        # Strip comments/empty lines, get the second to last line (the parent)
        PARENT_MOD=$(echo "$WHY_OUTPUT" | grep -v '^#' | sed '/^$/d' | tail -n 2 | head -n 1)
        PARENT_MSG=""

        if [ -n "$PARENT_MOD" ] && [ "$PARENT_MOD" != "$MODULE" ]; then
            if [ "$PARENT_MOD" = "$MAIN_MODULE" ]; then
                PARENT_MSG="Notice: This is a direct dependency. You may want to replace it."
            elif [[ "$PARENT_MOD" == github.com/* ]]; then
                # Get the parent's base repository
                PARENT_REPO=$(echo "$PARENT_MOD" | sed -E 's|^github\.com/([^/]+)/([^/]+).*|\1/\2|')

                # Check the parent's status
                PARENT_RESULT=$(get_repo_status "$PARENT_REPO")
                PARENT_STATUS="${PARENT_RESULT%%|*}"

                if [ "$PARENT_STATUS" = "fine" ]; then
                    PARENT_MSG="Notice: Brought in by '$PARENT_MOD' which is active. This dependency is still fine."
                fi
            fi
        fi

        # 5. Print the formatted report
        echo "[!] $MODULE"
        echo "    Status: $REASON_STR"
        echo "    Last updated: $FORMATTED_DATE"

        # Output the parent notice if it exists
        if [ -n "$PARENT_MSG" ]; then
            echo "    $PARENT_MSG"
        fi

        echo "    Dependency chain (go mod why):"
        echo "$WHY_OUTPUT" | sed 's/^/      /'
        echo ""
    fi
done

echo "Scan complete."