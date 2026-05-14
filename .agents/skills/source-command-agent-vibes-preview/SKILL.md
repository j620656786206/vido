---
name: "source-command-agent-vibes-preview"
description: "Preview TTS voices by playing audio samples (provider-aware)"
---

# source-command-agent-vibes-preview

Use this skill when the user asks to run the migrated source command `agent-vibes-preview`.

## Command Template

Preview TTS voices by playing audio samples from your active provider.

Usage examples:

- `/agent-vibes:preview` - Preview first 3 voices (default)
- `/agent-vibes:preview 5` - Preview first 5 voices
- `/agent-vibes:preview Jessica` - Preview Jessica Anne Bogart voice (Piper TTS)
- `/agent-vibes:preview lessac` - Preview Lessac voice (Piper)
- `/agent-vibes:preview "Northern Terry"` - Preview Northern Terry voice
- `/agent-vibes:preview first 10` - Preview first 10 voices
- `/agent-vibes:preview last 5` - Preview last 5 voices

!bash .claude/hooks/provider-commands.sh preview $ARGUMENTS

## MANUAL MIGRATION REQUIRED

Migrated from source command `agent-vibes-preview` into a Codex skill. Invoke it as `$source-command-agent-vibes-preview` and manually rewrite any slash-command behavior that depended on provider-specific runtime expansion.

Provider argument placeholders like `$ARGUMENTS` or `$1` were preserved as text; rewrite them into natural-language instructions for Codex.

Review unsupported command metadata manually: `argument-hint`.
