---
name: Auto-Execute Stories Without Asking
description: When executing story pipeline (CS→DS→TA→CR), don't pause between stories to ask permission — just continue
type: feedback
---

When running a multi-story development pipeline, proceed automatically to the next story without asking for confirmation.

**Why:** User explicitly said "以後的項目不用問我 時間到就執行" — they don't want to be asked between each story.

**How to apply:** After completing one story's full cycle (CS→DS→TA→CR→Commit), immediately start the next story. Only pause if blocked by an error or decision point.
