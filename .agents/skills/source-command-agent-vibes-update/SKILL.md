---
name: 'source-command-agent-vibes-update'
description: 'Update AgentVibes to the latest version'
---

# source-command-agent-vibes-update

Use this skill when the user asks to run the migrated source command `agent-vibes-update`.

## Command Template

Updates AgentVibes to the latest version from the npm registry.

This will update:

- All slash commands
- TTS scripts and hooks
- Personality templates (new ones added, existing ones updated)
- Output styles

Your custom settings and voice configurations will be preserved.

Usage examples:

- `/agent-vibes:update` - Update with confirmation prompt
- `/agent-vibes:update --yes` - Update without confirmation

!bash npx agent-vibes update $ARGUMENTS

## MANUAL MIGRATION REQUIRED

Migrated from source command `agent-vibes-update` into a Codex skill. Invoke it as `$source-command-agent-vibes-update` and manually rewrite any slash-command behavior that depended on provider-specific runtime expansion.

Provider argument placeholders like `$ARGUMENTS` or `$1` were preserved as text; rewrite them into natural-language instructions for Codex.

Review unsupported command metadata manually: `argument-hint`.
