---
name: 'source-command-agent-vibes-get'
description: 'Get the currently selected Piper TTS TTS voice'
---

# source-command-agent-vibes-get

Use this skill when the user asks to run the migrated source command `agent-vibes-get`.

## Command Template

Display the currently selected Piper TTS TTS voice.

This shows which voice is currently set as the default for TTS audio generation.

!bash .claude/hooks/voice-manager.sh get

## MANUAL MIGRATION REQUIRED

Migrated from source command `agent-vibes-get` into a Codex skill. Invoke it as `$source-command-agent-vibes-get` and manually rewrite any slash-command behavior that depended on provider-specific runtime expansion.
