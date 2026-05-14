---
name: 'source-command-agent-vibes-target-voice'
description: 'Set the voice for your target language'
---

# source-command-agent-vibes-target-voice

Use this skill when the user asks to run the migrated source command `agent-vibes-target-voice`.

## Command Template

Set which voice to use when speaking your target language. This should typically be a multilingual voice that supports your target language.

Usage:

```
/agent-vibes:target-voice Antoni
/agent-vibes:target-voice Rachel
/agent-vibes:target-voice Domi
```

Recommended multilingual voices:

- **Antoni** - Best for Spanish, Portuguese
- **Rachel** - Best for French, English
- **Domi** - Best for German, European languages
- **Bella** - Best for Italian, Romance languages
- **Charlotte** - European languages
- **Matilda** - Latin languages

These voices support 30+ languages using Piper TTS' Multilingual v2 model.

After setting your target voice:

- Enable learning mode with `/agent-vibes:learn`
- Check your setup with `/agent-vibes:learn status`

## MANUAL MIGRATION REQUIRED

Migrated from source command `agent-vibes-target-voice` into a Codex skill. Invoke it as `$source-command-agent-vibes-target-voice` and manually rewrite any slash-command behavior that depended on provider-specific runtime expansion.
