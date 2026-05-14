---
name: 'source-command-agent-vibes-whoami'
description: 'Show the current active voice and TTS provider'
---

# source-command-agent-vibes-whoami

Use this skill when the user asks to run the migrated source command `agent-vibes-whoami`.

## Command Template

Display the currently selected TTS voice and active provider (Piper TTS or Piper).

!bash .claude/hooks/voice-manager.sh whoami

## MANUAL MIGRATION REQUIRED

Migrated from source command `agent-vibes-whoami` into a Codex skill. Invoke it as `$source-command-agent-vibes-whoami` and manually rewrite any slash-command behavior that depended on provider-specific runtime expansion.
