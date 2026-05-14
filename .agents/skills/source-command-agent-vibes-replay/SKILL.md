---
name: 'source-command-agent-vibes-replay'
description: 'Replay recently played TTS audio'
---

# source-command-agent-vibes-replay

Use this skill when the user asks to run the migrated source command `agent-vibes-replay`.

## Command Template

Replay previously played TTS audio from history.

Usage:

- `/agent-vibes:replay` - Replay last audio (most recent)
- `/agent-vibes:replay 1` - Replay last audio
- `/agent-vibes:replay 2` - Replay second-to-last audio
- `/agent-vibes:replay 3` - Replay third-to-last audio

The system keeps the last 10 audio files in history. This is useful for:

- Hearing a summary again
- Checking what was just said
- Comparing different voice samples

!bash .claude/hooks/voice-manager.sh replay $ARGUMENTS

## MANUAL MIGRATION REQUIRED

Migrated from source command `agent-vibes-replay` into a Codex skill. Invoke it as `$source-command-agent-vibes-replay` and manually rewrite any slash-command behavior that depended on provider-specific runtime expansion.

Provider argument placeholders like `$ARGUMENTS` or `$1` were preserved as text; rewrite them into natural-language instructions for Codex.

Review unsupported command metadata manually: `argument-hint`.
