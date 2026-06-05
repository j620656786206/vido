---
name: Cross-Stack Stories Must Be Split
description: Stories spanning both backend and frontend with >3 tasks on each side must be split into separate stories. Established in Epic 8 retro after Story 8-8 required 3 CR rounds.
type: feedback
---

If a story requires >3 tasks on both backend AND frontend, it must be split into separate backend and frontend stories.

**Why:** Story 8-8 (Manual Subtitle Search UI) had 12 tasks across backend (3 handlers + engine mod) and frontend (service + hook + component + integration). It required 3 rounds of CR (26+ issues total) and the dev agent left tasks unchecked. By contrast, all pure backend stories (8-1 to 8-7, 8-10) completed smoothly with single-round CR.

**How to apply:** During create-story workflow, if the story has significant work on both backend and frontend, split it. Example: "8-8a: Subtitle Search API endpoints" + "8-8b: Subtitle Search Dialog UI".
