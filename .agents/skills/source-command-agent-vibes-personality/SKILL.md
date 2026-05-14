---
name: 'source-command-agent-vibes-personality'
description: 'Set or customize the personality style for TTS messages'
---

# source-command-agent-vibes-personality

Use this skill when the user asks to run the migrated source command `agent-vibes-personality`.

## Command Template

# /agent-vibes:personality

Set or customize the personality style for TTS messages.

This command allows you to add character and emotion to your TTS announcements by applying personality modifiers to messages.

## Usage

```bash
# Set a personality
/agent-vibes:personality flirty
/agent-vibes:personality sarcastic

# List all personalities
/agent-vibes:personality list

# Add custom personality
/agent-vibes:personality add cowboy "Howdy partner!" "Yeehaw!"

# Show current personality
/agent-vibes:personality get

# Reset to normal
/agent-vibes:personality reset
```

## Available Personalities

- **normal** - Standard professional tone
- **flirty** - Playful and charming
- **angry** - Frustrated and irritated
- **sassy** - Bold with attitude
- **moody** - Melancholic and brooding
- **funny** - Lighthearted and comedic
- **sarcastic** - Dry wit and irony
- **poetic** - Elegant and lyrical
- **annoying** - Over-enthusiastic
- **professional** - Formal and precise
- **pirate** - Seafaring swagger
- **robot** - Mechanical and precise
- **surfer-dude** - Chill beach vibes
- **millennial** - Internet generation speak
- **zen** - Peaceful and mindful
- **dramatic** - Theatrical flair
- **crass** - Edgy and blunt
- **random** - Picks a different personality each time!

## Editing Personalities

Each personality is stored as a markdown file in `.claude/personalities/`. You can:

### Edit existing personalities:

```bash
/agent-vibes:personality edit flirty
```

This shows the file path - edit it directly to customize behavior.

### Create new personalities:

```bash
/agent-vibes:personality add cowboy
```

Creates a new personality file, then edit it to customize.

### Personality files contain:

- **Prefix**: Text added before messages
- **Suffix**: Text added after messages
- **AI Instructions**: How the AI should speak
- **Example Responses**: Sample messages

Files are located in `.claude/personalities/[name].md`

## Implementation

!bash .claude/hooks/personality-manager.sh $ARGUMENTS

## MANUAL MIGRATION REQUIRED

Migrated from source command `agent-vibes-personality` into a Codex skill. Invoke it as `$source-command-agent-vibes-personality` and manually rewrite any slash-command behavior that depended on provider-specific runtime expansion.

Provider argument placeholders like `$ARGUMENTS` or `$1` were preserved as text; rewrite them into natural-language instructions for Codex.

Review unsupported command metadata manually: `argument-hint`.
