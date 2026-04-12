#!/bin/bash
set -e

DIR=${INPUT_DIR:-"./..."}
FILTER=${INPUT_FILTER:-""}

echo "Generating structmap for $DIR..."

# Generate the markdown
echo '```mermaid' > structmap.md
structmap "${DIR}" --filter="${FILTER}" >> structmap.md
echo '```' >> structmap.md

# Post to PR if PR event
if [ -n "$GITHUB_EVENT_PATH" ]; then
    PR_NUMBER=$(jq --raw-output .pull_request.number "$GITHUB_EVENT_PATH")
    if [ "$PR_NUMBER" != "null" ]; then
        COMMENT_BODY=$(cat structmap.md)
        # Using GitHub CLI to add a PR comment
        # The GH_TOKEN environment variable must be passed into the container
        if [ -n "$GITHUB_TOKEN" ]; then
            export GH_TOKEN=$GITHUB_TOKEN
            gh pr comment $PR_NUMBER -F structmap.md || echo "Failed to post PR comment"
        else
            echo "GITHUB_TOKEN absent, skipping PR comment"
        fi
    fi
fi
