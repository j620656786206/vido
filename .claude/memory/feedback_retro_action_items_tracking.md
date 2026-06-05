---
name: Retro Action Items Must Be Tracked Stories
description: All retrospective action items (HIGH/MEDIUM/LOW) must become tracked entries in sprint-status.yaml — no exceptions. Established in Epic 8 retro.
type: feedback
---

ALL retro action items, regardless of priority, must be created as tracked entries in sprint-status.yaml immediately after the retro is completed.

**Why:** Epic 7 retro had 6 action items; only 1 was completed (17% follow-through). Items left only in retro documents were forgotten when the next epic started. The husky pre-commit fix (HIGH) was not done and caused CI failures again in Epic 8.

**How to apply:** When running retrospective workflow step 11 (save & update), create a `retro-{epic}-{id}-{name}` entry in sprint-status.yaml for every action item. Use `backlog` status. These are tracked alongside regular stories.
