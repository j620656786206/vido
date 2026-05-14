---
name: "source-command-agent-vibes-version"
description: "Show the installed AgentVibes version"
---

# source-command-agent-vibes-version

Use this skill when the user asks to run the migrated source command `agent-vibes-version`.

## Command Template

Display the currently installed version of AgentVibes.

Usage:

- `/agent-vibes:version` - Show version information

!bash npx agent-vibes --version

## MANUAL MIGRATION REQUIRED

Migrated from source command `agent-vibes-version` into a Codex skill. Invoke it as `$source-command-agent-vibes-version` and manually rewrite any slash-command behavior that depended on provider-specific runtime expansion.
