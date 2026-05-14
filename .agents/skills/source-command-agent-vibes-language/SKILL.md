---
name: 'source-command-agent-vibes-language'
description: 'Set your main/native language for learning mode'
---

# source-command-agent-vibes-language

Use this skill when the user asks to run the migrated source command `agent-vibes-language`.

## Command Template

Set your main/native language. This is the language you already know and will hear first when learning mode is enabled.

Usage:

```
/agent-vibes:language english
/agent-vibes:language spanish
/agent-vibes:language french
```

The main language uses your currently selected voice. When learning mode is ON, TTS will speak in your main language FIRST, then translate to your target language.

Default: english

Supported languages: english, spanish, french, german, italian, portuguese, chinese, japanese, korean, hindi, arabic, polish, dutch, turkish, swedish, russian, and 15+ more.

After setting your main language:

1. Set your target language with `/agent-vibes:target <language>`
2. Set target voice with `/agent-vibes:target-voice <voice>`
3. Enable learning mode with `/agent-vibes:learn`

## MANUAL MIGRATION REQUIRED

Migrated from source command `agent-vibes-language` into a Codex skill. Invoke it as `$source-command-agent-vibes-language` and manually rewrite any slash-command behavior that depended on provider-specific runtime expansion.
