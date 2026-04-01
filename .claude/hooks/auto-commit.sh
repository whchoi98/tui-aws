#!/bin/bash
# Auto-commit after Claude Code session ends.
# Triggered by the Stop hook — commits all staged+unstaged changes.

cd "$(git rev-parse --show-toplevel 2>/dev/null)" || exit 0

# Skip if no changes
if git diff --quiet && git diff --cached --quiet && [ -z "$(git ls-files --others --exclude-standard)" ]; then
    exit 0
fi

# Stage all changes
git add -A

# Generate commit message from changed files
CHANGED=$(git diff --cached --name-only | head -5)
FILE_COUNT=$(git diff --cached --name-only | wc -l | tr -d ' ')

if [ "$FILE_COUNT" -eq 0 ]; then
    exit 0
fi

if [ "$FILE_COUNT" -le 3 ]; then
    MSG="auto: update $(echo $CHANGED | tr '\n' ', ' | sed 's/,$//')"
else
    MSG="auto: update ${FILE_COUNT} files"
fi

git commit -m "$MSG" --no-verify 2>/dev/null || true

echo "[auto-commit] Committed ${FILE_COUNT} file(s)"
