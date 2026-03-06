# Vido - NAS Media Management Platform

## UX Design Screenshots Workflow

When `ux-design.pen` is modified, you MUST regenerate and commit screenshots:

1. Run `python3 scripts/export-pen-screenshots.py` (requires Pencil.app running)
2. Screenshots are saved to `_bmad-output/screenshots/` organized by flow:
   - `flow-a-browse-desktop/` — Empty → Loading → Grid → List
   - `flow-b-hover-detail-desktop/` — Hover → Context Menu → Detail (Movie/TV) → Detail Context Menu
   - `flow-c-search-filter-settings-desktop/` — Search+Filter → Batch Ops → Settings
   - `flow-d-browse-mobile/` — Empty → Loading → Grid → Sort → Filter
   - `flow-e-interaction-mobile/` — Context Menu → Detail → Detail Context Menu
   - `flow-f-batch-settings-mobile/` — Batch Ops → Settings
3. If new screens are added to the .pen file, update the SCREENS dict in `scripts/export-pen-screenshots.py`
4. Commit updated screenshots alongside the .pen file changes

## Key Paths

- UX Design: `ux-design.pen` (Pencil app, read via MCP tools only)
- Design Screenshots: `_bmad-output/screenshots/`
- Planning Docs: `_bmad-output/planning-artifacts/`
- Implementation Specs: `_bmad-output/implementation-artifacts/`
