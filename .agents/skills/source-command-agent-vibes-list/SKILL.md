---
name: "source-command-agent-vibes-list"
description: "List available Piper TTS TTS voices with optional filtering"
---

# source-command-agent-vibes-list

Use this skill when the user asks to run the migrated source command `agent-vibes-list`.

## Command Template

List available Piper TTS TTS voices.

Usage examples:

- `/agent-vibes:list` - Show all voices
- `/agent-vibes:list first 5` - Show first 5 voices
- `/agent-vibes:list last 3` - Show last 3 voices

!bash .claude/hooks/voice-manager.sh list $ARGUMENTS

## MANUAL MIGRATION REQUIRED

Migrated from source command `agent-vibes-list` into a Codex skill. Invoke it as `$source-command-agent-vibes-list` and manually rewrite any slash-command behavior that depended on provider-specific runtime expansion.

Provider argument placeholders like `$ARGUMENTS` or `$1` were preserved as text; rewrite them into natural-language instructions for Codex.

Review unsupported command metadata manually: `argument-hint`.
