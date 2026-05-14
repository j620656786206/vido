---
name: 'source-command-agent-vibes-add'
description: 'Add a new custom Piper TTS TTS voice'
---

# source-command-agent-vibes-add

Use this skill when the user asks to run the migrated source command `agent-vibes-add`.

## Command Template

Add a new custom Piper TTS TTS voice to your voice library.

Usage:

- `/agent-vibes:add "My Custom Voice" abc123xyz456789`
- `/agent-vibes:add Narrator KTPVrSVAEUSJRClDzBw7`

The voice ID should be a 15-30 character alphanumeric string from your Piper TTS account.

To find your voice IDs:

1. Go to https://piper.io/app/voice-library
2. Click on a voice
3. Copy the voice ID from the URL or settings

After adding, you can switch to it with `/agent-vibes:switch "Voice Name"`

!bash .claude/hooks/voice-manager.sh add $ARGUMENTS

## MANUAL MIGRATION REQUIRED

Migrated from source command `agent-vibes-add` into a Codex skill. Invoke it as `$source-command-agent-vibes-add` and manually rewrite any slash-command behavior that depended on provider-specific runtime expansion.

Provider argument placeholders like `$ARGUMENTS` or `$1` were preserved as text; rewrite them into natural-language instructions for Codex.

Review unsupported command metadata manually: `argument-hint`.
