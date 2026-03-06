# Vido - NAS Media Management Platform

## UX Design Screenshots Workflow

**IMPORTANT:** After ANY modification to `ux-design.pen` (whether via Pencil MCP tools in this session, or externally), you MUST regenerate and commit screenshots before finishing.

### Steps:
1. Run `python3 scripts/export-pen-screenshots.py` (requires Pencil.app running)
   - The script starts its own MCP HTTP connection — safe to run even when Pencil MCP is already active
2. Screenshots are saved to `_bmad-output/screenshots/` organized by flow:
   - `flow-a-browse-desktop/` — Empty → Loading → Grid → List
   - `flow-b-hover-detail-desktop/` — Hover → Context Menu → Detail (Movie/TV) → Detail Context Menu
   - `flow-c-search-filter-settings-desktop/` — Search+Filter → Batch Ops → Settings
   - `flow-d-browse-mobile/` — Empty → Loading → Grid → Sort → Filter
   - `flow-e-interaction-mobile/` — Context Menu → Detail → Detail Context Menu
   - `flow-f-batch-settings-mobile/` — Batch Ops → Settings
3. If new screens are added to the .pen file, update the `SCREENS` dict in `scripts/export-pen-screenshots.py`
4. `git add` both the `.pen` file changes AND the updated screenshots, commit together

### Commit convention:
- If only design changed: `feat: update UX design — [what changed]`
- Include both `ux-design.pen` and `_bmad-output/screenshots/` in the same commit

## Key Paths

- UX Design: `ux-design.pen` (Pencil app, read via MCP tools only)
- Design Screenshots: `_bmad-output/screenshots/`
- Screenshot Export Script: `scripts/export-pen-screenshots.py`
- Design Brief: `_bmad-output/planning-artifacts/epic5-media-library-design-brief.md`
- Planning Docs: `_bmad-output/planning-artifacts/`
- Implementation Specs: `_bmad-output/implementation-artifacts/`
