---
name: Story Development Follows Loop Timing
description: Each story should start on the loop interval (e.g., every hour), not immediately after the previous one finishes
type: feedback
---

When running a multi-story pipeline with a loop interval (e.g., "每一小時執行一次story的開發"), wait for the next loop cycle before starting the next story. Don't rush to start the next story immediately after finishing one.

**Why:** User wants paced execution tied to the loop timing, not continuous rapid-fire development. This gives them time to review between stories.

**How to apply:** After completing a story's full cycle (CS→DS→TA→CR→Commit), wait for the next scheduled interval before starting the next story. Report completion and what's next, then wait.
