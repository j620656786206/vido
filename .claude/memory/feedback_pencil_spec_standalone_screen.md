---
name: Pencil Spec Screens Stand Alone
description: Design-decision/spec annotations in .pen go in their own standalone screen, never crammed into an existing screen mockup
type: feedback
originSessionId: 074ed7e0-596b-4683-afe4-dd5950bece14
---
When a UX pass needs to document design decisions / specs in `ux-design.pen` (e.g. a bugfix polish bundle), create a NEW standalone top-level screen for it (placed in empty canvas space, generous padding ~48px, clear section headings, Before/After comparisons + demo blocks + spec text). Do NOT append the spec as a section inside an existing screen mockup like HP-1.

**Why:** On 2026-05-11 (bugfix-10-6) I appended a "b10-6-polish-spec" frame to the bottom of Screen HP-1. Alexyu: "HP-1 的設計稿我看得不是很明白...標題好像都重複，並且都跟設計稿整個重疊在一起" — same dark bg, tiny text, demo rows reused the real block titles → unreadable and confusing.

**How to apply:** Use `find_empty_space_on_canvas` to place the spec screen; give it a distinct-but-themed bg (`$bg-secondary`) with demo sub-cards in `$bg-primary`; register it in `scripts/export-pen-screenshots.py` SCREENS dict so it gets a screenshot; keep the real screen mockups (HP-1 etc.) untouched except for the actual design change.
