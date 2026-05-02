#!/bin/bash

# Target the repository path (defaults to the current directory if not provided)
REPO_PATH="${1:-.}"

# Verify if the target directory is a valid git repository
if ! git -C "$REPO_PATH" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    echo "Error: '$REPO_PATH' is not a valid git repository."
    exit 1
fi

echo "Extracting deduplicated Signed-off-by emails with counts from: $REPO_PATH"
echo "------------------------------------------------------------"

# Fetch logs, filter sign-offs, extract emails, count duplicates, and sort by count descending
git -C "$REPO_PATH" log | \
    grep -i "^[[:space:]]*Signed-off-by:" | \
    sed -n 's/.*<\([^>]*\)>.*/\1/p' | \
    sort | \
    uniq -c | \
    sort -nr