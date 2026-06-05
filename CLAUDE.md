# Vido - NAS Media Management Platform

## UX Design Screenshots Workflow

**IMPORTANT:** After ANY modification to `ux-design.pen` (whether via Pencil MCP tools in this session, or externally), you MUST regenerate and commit screenshots before finishing.

### Steps:

1. Run `python3 scripts/export-pen-screenshots.py` (requires Pencil.app running)
   - The script spawns its own Pencil MCP server in **stdio** mode (Pencil 1.1.61 removed the old `--http`/`--http-port` transport) — safe to run even when Pencil MCP is already active
2. Screenshots are saved to `_bmad-output/screenshots/`, one folder per **user flow** (A–J merged-block convention, 2026-06-05 rework). Each flow folder holds both desktop (`-d`) and mobile (`-m`) screens; filenames are the canvas frame codes (e.g. `b3-d.png`, `b3-m.png`):
   - `flow-a-browse/` — Empty / Loading / Grid / List / Sort / Filter
   - `flow-b-detail-interaction/` — Hover / Context Menus / Detail (Movie/TV) / Fallbacks / Tech Badges / Image-load Fallback spec (B9)
   - `flow-c-search-settings/` — Search+Filter / Batch Ops / Settings / Backup
   - `flow-d-downloads/` — Download management
   - `flow-e-scanner/` — Scanner settings / Scan progress / Complete toast / Filtered-unmatched
   - `flow-f-subtitle/` — Subtitle search dialog / Preview-download / Batch progress
   - `flow-g-ai-subtitle/` — AI correction / Transcription progress / Translation confirm
   - `flow-h-homepage/` — Homepage TV Wall / Loading skeleton / Block CRUD modal / ExploreBlock spec
   - `flow-i-advanced-search/` — Filter chips / Suggestions dropdown / Save preset / Filter sheet
   - `flow-j-specs/` — Design-decision spec screens (e.g. PosterCard info-density)
   - `design-system/` — Design System Reference + Component Library docs
   - Canvas naming + block-layout convention: see `.claude/memory/project_pen_flow_layout_convention.md`
3. If new screens are added to the .pen file, update the `SCREENS` dict in `scripts/export-pen-screenshots.py` (key = node ID, value = `(flow-folder, code)`)
4. `git add` both the `.pen` file changes AND the updated screenshots, commit together
   - ⚠️ **A full regen is non-deterministic** — every PNG re-renders with byte diffs at the same dimensions. Only stage the screenshots whose **design actually changed**; `git checkout` the rest to avoid committing re-render noise.

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

<!-- nx configuration start-->
<!-- Leave the start & end comments to automatically receive updates. -->

## General Guidelines for working with Nx

- For navigating/exploring the workspace, invoke the `nx-workspace` skill first - it has patterns for querying projects, targets, and dependencies
- When running tasks (for example build, lint, test, e2e, etc.), always prefer running the task through `nx` (i.e. `nx run`, `nx run-many`, `nx affected`) instead of using the underlying tooling directly
- Prefix nx commands with the workspace's package manager (e.g., `pnpm nx build`, `npm exec nx test`) - avoids using globally installed CLI
- You have access to the Nx MCP server and its tools, use them to help the user
- For Nx plugin best practices, check `node_modules/@nx/<plugin>/PLUGIN.md`. Not all plugins have this file - proceed without it if unavailable.
- NEVER guess CLI flags - always check nx_docs or `--help` first when unsure

## Scaffolding & Generators

- For scaffolding tasks (creating apps, libs, project structure, setup), ALWAYS invoke the `nx-generate` skill FIRST before exploring or calling MCP tools

## When to use nx_docs

- USE for: advanced config options, unfamiliar flags, migration guides, plugin configuration, edge cases
- DON'T USE for: basic generator syntax (`nx g @nx/react:app`), standard commands, things you already know
- The `nx-generate` skill handles generator discovery internally - don't call nx_docs just to look up generator syntax

<!-- nx configuration end-->
