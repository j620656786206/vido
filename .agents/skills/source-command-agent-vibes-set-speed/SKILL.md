---
name: "source-command-agent-vibes-set-speed"
description: "Set TTS speech speed for Piper voices"
---

# source-command-agent-vibes-set-speed

Use this skill when the user asks to run the migrated source command `agent-vibes-set-speed`.

## Command Template

# Set Speech Speed

Control the speech rate for Piper TTS voices (Piper TTS doesn't support speed control).

## Usage

```bash
/agent-vibes:set-speed 2x              # Set main voice to 2x slower
/agent-vibes:set-speed target 2x       # Set target language to 2x slower
/agent-vibes:set-speed 0.5x            # Set main voice to 2x faster
/agent-vibes:set-speed target 3x       # Set target language to 3x slower
/agent-vibes:set-speed normal          # Reset to normal speed (1.0)
/agent-vibes:set-speed target normal   # Reset target to normal speed
```

## Speed Values

- `0.5x` or `-2x` = 2x faster (half duration)
- `1x` or `normal` = Normal speed
- `2x` or `+2x` = 2x slower (double duration, great for learning)
- `3x` or `+3x` = 3x slower (triple duration, very slow)

## Examples

```bash
# Make Spanish 2x slower for learning
/agent-vibes:set-speed target 2x

# Make main voice faster
/agent-vibes:set-speed 0.5x

# Reset target language to normal speed
/agent-vibes:set-speed target normal
```

!bash .claude/hooks/speed-manager.sh $ARGUMENTS

## MANUAL MIGRATION REQUIRED

Migrated from source command `agent-vibes-set-speed` into a Codex skill. Invoke it as `$source-command-agent-vibes-set-speed` and manually rewrite any slash-command behavior that depended on provider-specific runtime expansion.

Provider argument placeholders like `$ARGUMENTS` or `$1` were preserved as text; rewrite them into natural-language instructions for Codex.

Review unsupported command metadata manually: `argument-hint`.
