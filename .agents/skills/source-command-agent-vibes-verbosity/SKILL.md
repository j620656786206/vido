---
name: "source-command-agent-vibes-verbosity"
description: "Get or set AgentVibes verbosity level (low/medium/high)"
---

# source-command-agent-vibes-verbosity

Use this skill when the user asks to run the migrated source command `agent-vibes-verbosity`.

## Command Template

Get or set the AgentVibes verbosity level to control how much Claude speaks while working.

## Usage

- `/agent-vibes:verbosity` - Show current verbosity level
- `/agent-vibes:verbosity low` - Set to LOW (minimal)
- `/agent-vibes:verbosity medium` - Set to MEDIUM (balanced)
- `/agent-vibes:verbosity high` - Set to HIGH (maximum transparency)

## Verbosity Levels

### LOW (Minimal)

- ✅ Acknowledgments only (start of task)
- ✅ Completions only (end of task)
- 🔇 No reasoning spoken during work
- Perfect for: Quiet work sessions, minimal distraction

### MEDIUM (Balanced)

- ✅ Acknowledgments
- 🤔 Major decisions ("I'll use grep to search")
- ✓ Key findings ("Found 12 instances")
- ✅ Completions
- Perfect for: Understanding major decisions without full narration

### HIGH (Maximum Transparency)

- ✅ Acknowledgments
- 💭 All reasoning ("Let me search for all instances")
- 🤔 All decisions ("I'll use grep for this")
- ✓ All findings ("Found it at line 1323")
- ✅ Completions
- Perfect for: Full transparency, learning mode, debugging complex tasks

## How It Works

Claude uses **emoji markers** in its reasoning text:

- 💭 = Reasoning/thinking
- 🤔 = Decisions
- ✓ = Findings/results

Based on your verbosity level, AgentVibes automatically detects these markers and speaks them aloud.

## Examples

### LOW Verbosity

```
User: "Find all TODO comments"
[TTS: "I'll search for those"]
[Work happens silently...]
[TTS: "Found 12 TODO comments"]
```

### MEDIUM Verbosity

```
User: "Find all TODO comments"
[TTS: "I'll search for those"]
🤔 I'll use grep to search all files
[Work happens...]
✓ Found 12 TODO comments across 5 files
[TTS: "Found 12 TODO comments"]
```

### HIGH Verbosity

```
User: "Find all TODO comments"
[TTS: "I'll search for those"]
💭 Let me search through the codebase for TODO comments
🤔 I'll use the Grep tool with pattern "TODO"
[Grep runs...]
✓ Found 12 TODO comments across 5 files
💭 Let me organize these by file
[TTS: "Found 12 TODO comments in 5 files"]
```

## Notes

- Changes take effect on next Claude Code session restart
- Verbosity is saved per-project (if `.claude/` exists) or globally
- You can also control verbosity via MCP: `mcp__agentvibes__set_verbosity(level="high")`

## Related Commands

- `/agent-vibes:personality` - Set voice personality style
- `/agent-vibes:switch` - Change voice
- `/agent-vibes:provider` - Switch TTS provider

## MANUAL MIGRATION REQUIRED

Migrated from source command `agent-vibes-verbosity` into a Codex skill. Invoke it as `$source-command-agent-vibes-verbosity` and manually rewrite any slash-command behavior that depended on provider-specific runtime expansion.

Review unsupported command metadata manually: `tags`.
