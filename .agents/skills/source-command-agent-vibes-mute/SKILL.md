---
name: 'source-command-agent-vibes-mute'
description: 'Mute all AgentVibes TTS output (project-specific by default)'
---

# source-command-agent-vibes-mute

Use this skill when the user asks to run the migrated source command `agent-vibes-mute`.

## Command Template

# Mute AgentVibes TTS

Mute TTS for this project only (default):

```bash
# Get the project root (where .claude/ directory is located)
PROJECT_ROOT="${CLAUDE_PROJECT_DIR:-$(pwd)}"
while [[ "$PROJECT_ROOT" != "/" ]] && [[ ! -d "$PROJECT_ROOT/.claude" ]]; do
  PROJECT_ROOT=$(dirname "$PROJECT_ROOT")
done

if [[ -d "$PROJECT_ROOT/.claude" ]]; then
  # Remove project unmute file if it exists
  rm -f "$PROJECT_ROOT/.claude/agentvibes-unmuted"

  # Create project mute file
  touch "$PROJECT_ROOT/.claude/agentvibes-muted"

  echo "🔇 **AgentVibes TTS muted for this project.** All voice output is now silenced."
else
  echo "⚠️ No .claude directory found. Cannot create project-local mute file."
  exit 1
fi
```

**Advanced Options:**

To mute globally across ALL projects, use:

```bash
touch "$HOME/.agentvibes-muted"
```

To unmute, use `/agent-vibes:unmute`

## MANUAL MIGRATION REQUIRED

Migrated from source command `agent-vibes-mute` into a Codex skill. Invoke it as `$source-command-agent-vibes-mute` and manually rewrite any slash-command behavior that depended on provider-specific runtime expansion.
