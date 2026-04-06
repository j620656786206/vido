#!/bin/bash
# Pre-git hook: Remove stale index.lock before git add/commit
# Root cause: Zed editor's background git status races with git operations
# See: project-context.md Rule 12

LOCK_FILE="${CLAUDE_PROJECT_DIR:-.}/.git/index.lock"

if [ -f "$LOCK_FILE" ]; then
  # Check if the lock is stale (older than 5 seconds)
  if [ "$(uname)" = "Darwin" ]; then
    lock_age=$(( $(date +%s) - $(stat -f %m "$LOCK_FILE") ))
  else
    lock_age=$(( $(date +%s) - $(stat -c %Y "$LOCK_FILE") ))
  fi

  if [ "$lock_age" -gt 5 ]; then
    rm -f "$LOCK_FILE"
    echo "Removed stale .git/index.lock (age: ${lock_age}s)" >&2
  fi
fi
