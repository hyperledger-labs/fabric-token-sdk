#!/bin/bash

# Configuration
REPO="hyperledger-labs/fabric-token-sdk"
BASE_URL="https://api.github.com/repos/$REPO"
TOKEN="${GITHUB_TOKEN:-}"

# Helper function for API requests
fetch_github_api() {
    local url="$1"
    local headers=(-H "Accept: application/vnd.github.v3+json" -H "User-Agent: Bash-Script")
    if [[ -n "$TOKEN" ]]; then
        headers+=(-H "Authorization: token $TOKEN")
    fi
    curl -s "${headers[@]}" "$url"
}

# Helper function to get the count from the Link header (for pagination trick)
get_last_page_from_link() {
    local url="$1"
    local headers=(-H "Accept: application/vnd.github.v3+json" -H "User-Agent: Bash-Script" -I)
    if [[ -n "$TOKEN" ]]; then
        headers+=(-H "Authorization: token $TOKEN")
    fi
    
    local link_header=$(curl -s "${headers[@]}" "$url" | grep -i "^link:" || true)
    if [[ -n "$link_header" ]]; then
        # Example: <...page=2>; rel="next", <...page=370>; rel="last"
        echo "$link_header" | sed -n 's/.*page=\([0-9]*\)>; rel="last".*/\1/p'
    else
        echo "1"
    fi
}

# Main script
echo "Collecting stats for $REPO..."

# 1. Basic Metadata
repo_data=$(fetch_github_api "$BASE_URL")
description=$(echo "$repo_data" | jq -r '.description // "None"')
stars=$(echo "$repo_data" | jq -r '.stargazers_count // "0"')
forks=$(echo "$repo_data" | jq -r '.forks_count // "0"')
watchers=$(echo "$repo_data" | jq -r '.subscribers_count // "0"')
language=$(echo "$repo_data" | jq -r '.language // "None"')
license=$(echo "$repo_data" | jq -r '.license.name // "None"')

# 2. Contributors count
contributor_count=$(get_last_page_from_link "$BASE_URL/contributors?per_page=1&anon=true")

# 3. Community Profile
community_data=$(fetch_github_api "$BASE_URL/community/profile")
has_readme=$(echo "$community_data" | jq -r '.files.readme != null')
has_contributing=$(echo "$community_data" | jq -r '.files.contributing != null')
has_coc=$(echo "$community_data" | jq -r '.files.code_of_conduct != null')
has_license=$(echo "$community_data" | jq -r '.files.license != null')
has_security=$(echo "$community_data" | jq -r '.files.security_advisory_declaration != null or .files.security_policy != null')

# Local fallback for security policy
if [[ "$has_security" == "false" && -f "SECURITY.md" ]]; then
    has_security="true"
fi

# 4. Activity Stats (Last 30 Days)
# Get ISO 8601 date for 30 days ago (cross-platform compatible ish)
if date -v-30d >/dev/null 2>&1; then
    # BSD/macOS
    SINCE=$(date -v-30d +"%Y-%m-%dT%H:%M:%SZ")
else
    # GNU
    SINCE=$(date --date="30 days ago" --iso-8601=seconds)
fi

recent_commits=$(get_last_page_from_link "$BASE_URL/commits?since=$SINCE&per_page=1")

# Use Search API for precise counts
fetch_search_count() {
    local query="$1"
    fetch_github_api "https://api.github.com/search/issues?q=$query" | jq -r '.total_count // "0"'
}

open_issues=$(fetch_search_count "repo:$REPO+type:issue+state:open")
closed_issues=$(fetch_search_count "repo:$REPO+type:issue+state:closed")
open_prs=$(fetch_search_count "repo:$REPO+type:pr+state:open")
closed_prs=$(fetch_search_count "repo:$REPO+type:pr+state:closed")

# Output Report
echo -e "\n## GitHub Project Insights: $REPO"
echo -e "\n### General Metadata"
echo "| Metric | Value |"
echo "| :--- | :--- |"
echo "| Description | $description |"
echo "| Stars | $stars |"
echo "| Forks | $forks |"
echo "| Watchers | $watchers |"
echo "| Language | $language |"
echo "| License | $license |"
echo "| Total Contributors | $contributor_count |"

echo -e "\n### Community Standards"
echo "| Standard | Present |"
echo "| :--- | :--- |"
echo "| README | $([[ "$has_readme" == "true" ]] && echo '✅' || echo '❌') |"
echo "| CONTRIBUTING | $([[ "$has_contributing" == "true" ]] && echo '✅' || echo '❌') |"
echo "| CODE_OF_CONDUCT | $([[ "$has_coc" == "true" ]] && echo '✅' || echo '❌') |"
echo "| LICENSE | $([[ "$has_license" == "true" ]] && echo '✅' || echo '❌') |"
echo "| SECURITY Policy | $([[ "$has_security" == "true" ]] && echo '✅' || echo '❌') |"

echo -e "\n### Activity (Last 30 Days / Overall)"
echo "| Metric | Count |"
echo "| :--- | :--- |"
echo "| Commits (Last 30d) | $recent_commits |"
echo "| Open Issues | $open_issues |"
echo "| Closed Issues | $closed_issues |"
echo "| Open Pull Requests | $open_prs |"
echo "| Closed Pull Requests | $closed_prs |"
