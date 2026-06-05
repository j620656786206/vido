---
name: UX Design Verification Mandatory in Dev Workflow
description: Dev workflow step 9 now requires mandatory UX design screenshot comparison before marking story complete
type: feedback
---

Stories with UI changes are NOT complete until the implementation visually matches the design screenshots in `_bmad-output/screenshots/`.

**Why:** User found that Dev Agent was marking stories as done without verifying UI matches design specs. This led to accumulated design drift across stories 5-2 through 5-5 (filter panel as dropdown instead of sidebar, missing language filter, wrong type filter placement, etc.).

**How to apply:**
- Dev workflow `instructions.xml` step 9 is now "UX Design Verification" (inserted before completion step)
- Dev Agent must compare implementation against relevant design screenshots (flow-a through flow-f)
- Discrepancies must be fixed before the story can be marked as "review"
- `checklist.md` now includes a "🎨 UX Design Verification" section with 9 checkboxes
- UX Design Review reports are stored in `_bmad-output/implementation-artifacts/ux-design-review-*.md`
- This was formalized on 2026-03-15 after Epic 5 design review revealed significant gaps
