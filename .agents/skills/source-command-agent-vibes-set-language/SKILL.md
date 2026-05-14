---
name: "source-command-agent-vibes-set-language"
description: "Run the migrated source command `agent-vibes-set-language`."
---

# source-command-agent-vibes-set-language

Use this skill when the user asks to run the migrated source command `agent-vibes-set-language`.

## Command Template

# /agent-vibes:set-language

Set the language for TTS narration. AgentVibes will automatically use multilingual voices and speak in your chosen language.

## Usage

```bash
# Set language
/agent-vibes:set-language spanish
/agent-vibes:set-language french
/agent-vibes:set-language german

# Show current language
/agent-vibes:set-language

# Reset to English (default)
/agent-vibes:set-language english
/agent-vibes:set-language reset
```

## Supported Languages

Spanish, French, German, Italian, Portuguese, Chinese, Japanese, Korean, Polish, Dutch, Turkish, Russian, Arabic, Hindi, Swedish, Danish, Norwegian, Finnish, Czech, Romanian, Ukrainian, Greek, Bulgarian, Croatian, Slovak, and 20+ more.

## How It Works

1. **Sets Language**: Stores your language preference in `.claude/tts-language.txt`
2. **Voice Selection**: Automatically uses multilingual voices (Antoni, Rachel, Domi, Bella)
3. **BMAD Integration**: Works with BMAD plugin - agents speak in your language
4. **Personality Preserved**: Keeps your current personality/sentiment style

## Multilingual Voice Recommendations

- **Spanish**: Antoni, Matilda
- **French**: Rachel, Charlotte
- **German**: Domi, Charlotte
- **Italian**: Bella
- **Portuguese**: Matilda
- **Other Languages**: Antoni (best all-around multilingual)

## Implementation

This command tells Claude AI to:

1. Validate the language is supported
2. Save to `.claude/tts-language.txt`
3. Switch to an appropriate multilingual voice if needed
4. Inform user of the change

## MANUAL MIGRATION REQUIRED

Migrated from source command `agent-vibes-set-language` into a Codex skill. Invoke it as `$source-command-agent-vibes-set-language` and manually rewrite any slash-command behavior that depended on provider-specific runtime expansion.
