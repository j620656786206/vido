---
name: "source-command-agent-vibes-replay-target"
description: "Replay the last target language audio (for language learning mode)"
---

# source-command-agent-vibes-replay-target

Use this skill when the user asks to run the migrated source command `agent-vibes-replay-target`.

## Command Template

Replay the last message that was spoken in your target language during language learning mode.

This is useful when learning a new language - you can hear the translation again without triggering a new one.

Usage:

- `/agent-vibes:replay-target` - Replay the last target language audio

**Note:** This only works when language learning mode is active (`/agent-vibes:learn`).

!bash .claude/hooks/replay-target-audio.sh

## MANUAL MIGRATION REQUIRED

Migrated from source command `agent-vibes-replay-target` into a Codex skill. Invoke it as `$source-command-agent-vibes-replay-target` and manually rewrite any slash-command behavior that depended on provider-specific runtime expansion.
