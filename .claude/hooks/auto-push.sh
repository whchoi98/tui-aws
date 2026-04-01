#!/bin/bash
# Auto-push after commit. Only pushes if remote is configured and ahead of remote.
# Triggered by Stop hook (after auto-commit.sh).

cd "$(git rev-parse --show-toplevel 2>/dev/null)" || exit 0

# Check if remote exists
REMOTE=$(git remote 2>/dev/null | head -1)
[ -z "$REMOTE" ] && exit 0

# Check if we have a tracking branch
BRANCH=$(git branch --show-current 2>/dev/null)
[ -z "$BRANCH" ] && exit 0

# Check if ahead of remote
AHEAD=$(git rev-list --count "${REMOTE}/${BRANCH}..HEAD" 2>/dev/null)
[ "$AHEAD" = "0" ] && exit 0

# Push
if git push "$REMOTE" "$BRANCH" 2>/dev/null; then
    echo "[auto-push] Pushed ${AHEAD} commit(s) to ${REMOTE}/${BRANCH}"
else
    echo "[auto-push] Push failed — run 'git push' manually"
fi
