---
name: "source-command-agent-vibes-sample"
description: "Test a voice with sample text"
---

# source-command-agent-vibes-sample

Use this skill when the user asks to run the migrated source command `agent-vibes-sample`.

## Command Template

Test a specific Piper TTS voice by playing sample text.

Usage:

- `/agent-vibes:sample Cowboy` - Test the Cowboy voice
- `/agent-vibes:sample "Northern Terry"` - Test Northern Terry voice

!bash .claude/hooks/voice-manager.sh sample $ARGUMENTS

## MANUAL MIGRATION REQUIRED

Migrated from source command `agent-vibes-sample` into a Codex skill. Invoke it as `$source-command-agent-vibes-sample` and manually rewrite any slash-command behavior that depended on provider-specific runtime expansion.

Provider argument placeholders like `$ARGUMENTS` or `$1` were preserved as text; rewrite them into natural-language instructions for Codex.

Review unsupported command metadata manually: `argument-hint`.
