# Chore chore-pen-subtitle-v2-design-sync: flow-f-subtitle-v2 .pen ↔ shipped-impl sync

Status: ready-for-dev

> **DESIGN-ONLY Pencil chore — executor is the ux-designer agent (Sally), NOT a code dev.** No apps/* code, no migrations, no ESLint/visual-baseline gates. The "gate" is: Pencil MCP self-review vs this spec + screenshot regen + Alexyu eyeball. All edits via Pencil MCP tools ONLY (`.pen` is encrypted — never Read/Grep it). After edits: `python3 scripts/export-pen-screenshots.py`, then commit ONLY the genuinely-changed PNGs (a full regen re-renders every PNG with byte noise — `git checkout` the rest) together with `ux-design.pen`.
>
> **Node IDs below were live-verified via Pencil MCP 2026-07-06** (all 12 targets resolved). Two roadmap/design decisions were ratified by Alexyu via party-mode (Sally+John+Winston, 2026-07-06): see items 4 and 9.

## Story

As the Vido design system,
I want `ux-design.pen`'s flow-f-subtitle-v2 frames reconciled with what shipped in ux3-subtitle-v2 (slice 1) + ux3-subtitle-v2-batch (slice 2),
so that the next person building from the design isn't misled by a stepper missing a stage, dead controls that were never built, or counts/weights that drift from the code — closing the two Sally UX gates' follow-up queue.

## Acceptance Criteria (design-fidelity outcomes)

1. **GenerationProgress-v2 gains the 6th stage (AI校正).** Both the desktop component `XkGvG` and the mobile stepper `fS5is` (in `k8sJl4`) show the frozen 6-stage vocabulary 提取音訊 → 轉錄中 → 翻譯中 → 簡轉繁 → **AI校正** → 完成 (currently draw only 5, missing AI校正 between 簡轉繁 and 完成). Propagates to the Component Library ref (`XkGvG` is referenced override-free in cell `ZHM4p`).
2. **F3-D/F3-M progress-dialog footer says 關閉, not 取消.** Footers in `JbXai` (`H2VIe` → button `tuTpf`, label `L9cIf`) and `k8sJl4` (`tRFbG` → `uHnRS`) relabel 取消 → **關閉** + a "背景繼續" hint (there is no backend cancel route; the SSE chip `I82rS` already implies a background job).
3. **GlossaryRow-v2 shows a delete action + all 3 source-badge variants.** `nDSEd` gains a **刪除** action (destructive, error-text token — impl uses a Radix delete confirm) alongside the drawn 確認 (`wDFKQ`) + 編輯 (`SIp0F`); and the source badge (`WyY3x`, currently only 字幕) is represented in all three variants 字幕 / **中繼資料** / **手動** (impl has all three). Component Library caption `lb5mu` updated to name delete + the 3 sources.
4. **F1 track rows: ⋯ menu removed, Mono filename kept as aspirational.** (Party-mode ruling — Winston's split, Sally+John concur.) In `r1EY9` (rows `ogk8F`/`g7uAsl`/`IxuCc`) and `JkdfH` (mobile counterparts), REMOVE the per-track ⋯ ellipsis menus (`eOgKO`/`vdqhF`/`gs7dP` desktop; `VhazB`/`L0JC2`/`gurW8` mobile) — capability-honor, they have no BE actions (blocked on `track-convert-endpoint`). KEEP the Mono filenames (`VXof3`/`Y5Pxj`/`WMQ1o`; `Hcjm8`/`kZM9o`/`l0YjUq`) but add an **aspirational-data annotation** ("檔名顯示待 production-countries-detail-api 接線；⋯ 單軌動作待 track-convert-endpoint 移除本輪"). Filename = data plumbing (safe to keep as intent); ⋯ = a dead control (removed).
5. **F8-D/F8-M 已選項目 segment shows its Mono count.** Scope segment `fxkko` (`已選項目`, label `yf0WI`) in `i9Nun1`, and `C2mqH` (label `wr93H`) in `H717g`, gain a Mono count node mirroring seg-missing's (`vvyEq`/`FlzYM` "38") — impl draws 已選項目 N. seg-selected is inactive → count uses text-secondary, not accent-600.
6. **F9 已完成 semantics annotated.** `JMqPg` overall `jWgVJ` "已完成 12 / 38" (`hmBwN`) gets a design note: **已完成 = processed (success + fail), preserving 已完成 + 剩餘 = total** (Sally's gate L2 ruling — ACCEPT, not success-only). No new failed row; annotation only.
7. **F8-D/F8-M cost-used amount bumped to weight 600.** `i9Nun1` cost-used `ICPL8` ("$0.42", currently weight normal) → **600**, matching sibling screens F3-D (`Eva59`=600) and F9-D (`MPxUh`=600); ceiling `da87j` stays normal. Same on mobile `H717g` (`kbBNl` normal → 600).
8. **F8-M (and F3-M) sheet header gains a bottom divider.** `H717g` sheet-header `H1CYPa` gets `strokeWidth {bottom:1}` matching desktop F8-D title-bar `tJKu3`; apply the same to F3-M sheet header `B9gEpS` (also missing it) for consistency. (Drag handle `k46gFw` already present — no action.)
9. **F8-D mixed-state frame annotated as a deliberate composite.** (Party-mode ruling — keep one frame, defer split.) `i9Nun1` (`Fkiqd`) keeps its single all-in-one layout (idle scope selector + running progress/queue/cost/SSE/全部取消); add a caption noting **it is an all-in-one spec-reference frame; the shipped UI separates idle-config (scope + preview count, no queue) from running (queue + progress + cost + SSE + 全部取消)**. Do NOT split into two frames this round.

## Tasks / Subtasks (per-node .pen edits)

- [ ] Task 1 — GenerationProgress-v2 AI校正 (AC 1)
  - [ ] `XkGvG`: insert a 6th step "AI校正" (pending state: 6px dot, `$text-muted`/`$bg-tertiary`, Noto Sans TC 12px, hidden `-pct` sibling) between `OtqSr` (簡轉繁) and `jrRbp` (完成); add a 5th connector between them (`$border-subtle`, matching `gp-cn2..4`)
  - [ ] `fS5is`: insert "AI校正" row (Noto 13px, pending) between `YKDDs` (簡轉繁) and `e4Ar7Q` (完成)
- [ ] Task 2 — F3 footer relabel (AC 2)
  - [ ] `JbXai` `L9cIf` 取消 → 關閉 + add 背景繼續 hint text in `H2VIe`
  - [ ] `k8sJl4` `uHnRS` label 取消 → 關閉 + hint
- [ ] Task 3 — GlossaryRow-v2 delete + badges (AC 3)
  - [ ] `nDSEd`: add 刪除 action (error-text, 44px hit) after 編輯; add 中繼資料 + 手動 badge variants (as extra reference instances or an annotated variant strip — designer's call on presentation); update caption `lb5mu`
- [ ] Task 4 — F1 track rows (AC 4)
  - [ ] `r1EY9`: delete `eOgKO`/`vdqhF`/`gs7dP`; keep `VXof3`/`Y5Pxj`/`WMQ1o`; add aspirational-data annotation on `RnqxB`
  - [ ] `JkdfH`: delete `VhazB`/`L0JC2`/`gurW8`; keep filenames; annotation
- [ ] Task 5 — 已選項目 count (AC 5)
  - [ ] `i9Nun1` `fxkko`: add Mono count (text-secondary); `H717g` `C2mqH`: same
- [ ] Task 6 — F9 已完成 note (AC 6)
  - [ ] `JMqPg`: annotation near `jWgVJ`
- [ ] Task 7 — cost-used weight (AC 7)
  - [ ] `i9Nun1` `ICPL8` → 600; `H717g` `kbBNl` → 600
- [ ] Task 8 — mobile sheet dividers (AC 8)
  - [ ] `H717g` `H1CYPa` + `k8sJl4` `B9gEpS`: add `strokeWidth {bottom:1}`
- [ ] Task 9 — F8-D composite annotation (AC 9)
  - [ ] `i9Nun1` `Fkiqd`: add composite-reference caption
- [ ] Task 10 — regen + commit
  - [ ] `python3 scripts/export-pen-screenshots.py`; stage ONLY genuinely-changed PNGs under `_bmad-output/screenshots/flow-f-subtitle-v2/` + `design-system/component-library.png` (`git checkout` re-render noise) + `ux-design.pen`
  - [ ] Pencil MCP self-review vs this spec (all 9 ACs)

**Cross-stack split check:** 0 backend, 0 frontend code tasks (design-only) → no split. ✓

## Dev Notes

### Current-state facts (Pencil MCP, 2026-07-06 — trust these)

- **XkGvG steps** (5, missing AI校正): `Bkbxr`提取音訊(done,check/success) · `A8AT5n`轉錄中(active,loader/accent,600,pct"45%") · `kZVrH`翻譯中 · `OtqSr`簡轉繁 · `jrRbp`完成. Connectors `ewNcK`/`IEjPA`/`Gcvfs`/`wnE8f` (cn1=success green, rest border-subtle). **fS5is** mirror: `x0G3j`/`t6U0O0`/`XPXpx`/`YKDDs`/`e4Ar7Q`.
- **F3 footers**: `JbXai`→`H2VIe`→`tuTpf`(ref `YDPhc` ButtonSecondary)→`L9cIf`"取消"; `k8sJl4`→`tRFbG`→`uHnRS`(full-width)"取消". SSE chip `I82rS` present in F3-D body.
- **nDSEd**: badge `WyY3x`(`z5Xrd`"字幕",info) + status `R353jw`(未確認,warning) + 確認 `wDFKQ`(`u9HLj`,accent-600) + 編輯 `SIp0F`(`KfDYP`,text-secondary). No delete; only 字幕 badge.
- **F1 tracks** (design DRAWS filename+⋯, impl omits both): `r1EY9`/`RnqxB` rows `ogk8F`(file `VXof3` Mono 12px muted + more `eOgKO`) / `g7uAsl`(`Y5Pxj`+`vdqhF`) / `IxuCc`(`WMQ1o`+`gs7dP`); `JkdfH` file Mono 11px `Hcjm8`+more`VhazB` / `kZM9o`+`L0JC2` / `l0YjUq`+`gurW8`.
- **F8-D `i9Nun1`**: scope `V9LuY`(label `STldN`"範圍：") segs `vvyEq`seg-missing("缺字幕的項目"`grLGK` + count `FlzYM`"38", accent-600, fill accent-subtle) / `fxkko`seg-selected("已選項目"`yf0WI`, text-secondary, fill bg-tertiary, NO count). cost `gmdHT` used `ICPL8`"$0.42"(Mono13,text-primary,**normal**) cap `da87j`"$5.00"(text-secondary,normal). title-bar `tJKu3` has bottom:1. dialog root `Fkiqd`.
- **F8-M `H717g`**: sheet `bCZ9p`: handle-wrap `zkKkC`(`k46gFw` 36×4 bg-tertiary) → header `H1CYPa`(NO divider) → body → item-list → footer `GSnOg`(top:1). seg-missing-m `RRXVB`("38"`a0ucP`) / seg-selected-m `C2mqH`(label `wr93H`, no count). cost-used `kbBNl` normal.
- **F9-D `JMqPg`**: banner `GNV6A`"已達本次預算上限（$5.00）— 已完成12部，剩餘26部下次繼續"(nums `mL8Gw`/`Z7I4h`/`h97TVY` Mono 600); overall `jWgVJ`("已完成"`Pjh5l` + `hmBwN`"12 / 38" Mono 20px 600); bar `H8dOP`; rows 完成×3 + 已暫停—下次繼續×2; cost-used `MPxUh`=**600**. No failed row, no 已完成-semantics annotation.
- **F3-D cost** `Eva59`"$0.42" = **600** (the sibling that F8 should match).
- **Component Library `sJzat`**: GenerationProgress-v2 in `progress-v2`(`luza9`) cell `ZHM4p` ref `Thgoj`→`XkGvG` (override-free), caption `bWjaa` claims "…／done／failed" (failed not demonstrated — out of scope, note only). GlossaryRow-v2 in `content-cards-v2`(`m3bVW`) cell `Fx24g` ref `yOVMl`→`nDSEd` (width:640 override), caption `lb5mu`. Editing the mother components propagates to these refs automatically; captions are static text → update manually.

### Ratified decisions (party-mode 2026-07-06, do not re-litigate)

- **Item 4 (F1):** Winston's split — ⋯ menu DROPPED (capability-honor, no BE actions), Mono filename KEPT + aspirational-data annotation. Rationale: filename = data plumbing (`production-countries-detail-api`), not a dead control; ⋯ = needs real actions (`track-convert-endpoint`), and per John's roadmap neither BE ticket is scheduled for the next 2-3 sprints. Precedent: same capability-honor logic as item 2 (F3 footer 取消→關閉).
- **Item 6 (F9):** already Sally's gate ruling — ACCEPT 已完成 = processed (success+fail); annotate only, no success-only copy change (that would be an ADR-lite product decision, not filed).
- **Item 9 (F8-D):** keep single composite frame + annotation; defer split (tagged optional; shipped code is the two-state truth).

### Design-system discipline (feedback memory)

- New-frame font zero-tolerance: the added AI校正 label is Noto Sans TC; 已選項目 count is JetBrains Mono `tabular-nums` — keep numeric/unit splits Mono+Noto. No NEW reusable components created (editing existing registered `XkGvG`/`nDSEd`), so no Component Library registration needed beyond the caption updates.
- ⚠️ Full regen is non-deterministic — stage ONLY genuinely-changed PNGs. Expected changed: `flow-f-subtitle-v2/` f1-d/f1-m (tracks), f2-d?(GenerationProgress is F3 here — verify which screenshot code maps to XkGvG's frame), f3-d/f3-m (footer + AI校正), f6-d/f6-m (glossary delete/badges), f8-d/f8-m (已選項目 count + cost weight + header divider + composite note), f9-d (已完成 note) + `design-system/component-library.png` (XkGvG/nDSEd propagation). Confirm the exact set post-regen by diff, not assumption.

### Project Structure Notes

- Touches ONLY `ux-design.pen` + `_bmad-output/screenshots/`. Zero apps/* code, zero migrations, zero `tests/visual` baselines (component code unchanged — the `// Implements:` node IDs XkGvG/nDSEd are unchanged; only their contents edit). No CI code gate applies.

### Time-dependent visual coverage

- N/A — design-only chore; no `apps/web/src/components/**` files touched.

### References

- [Source: sprint-status.yaml chore-pen-subtitle-v2-design-sync (9 items from 2 Sally UX gates, 2026-07-05/06)]
- [Source: party-mode ruling 2026-07-06 (Sally+John+Winston, Alexyu ratified) — items 4 + 9]
- [Source: ux-design.pen flow-f-subtitle-v2 block (x≈31590 y≈-5791); node IDs verified via Pencil MCP 2026-07-06]
- [Source: CLAUDE.md UX Design Screenshots Workflow; feedback_design_system_conformance_pen memory]

## Dev Agent Record

### Agent Model Used

Pencil Inline AI Agent (drew all 9 edits from the SM-authored node-level prompt) → Sally / ux-designer agent MCP review (Fable 5). Token-efficient division: authoring offloaded to Pencil's own agent; only the review consumed Claude tokens.

### Debug Log References

### Completion Notes List

- **DONE 2026-07-06.** All 9 items executed by the Pencil Inline Agent against the exact node-level prompt (all IDs MCP-pre-verified). Sally MCP review verdict: **APPROVED-WITH-FIXES** — 9/9 PASS, ZERO over-draw (confirmed F4-D `U8rRtv` / F5-D `f6ZxY` are pre-existing grid frames, not F11-style extras; no split of F8-D, no failed row added to F9, no changed node IDs, item-4 not inverted).
- New nodes drawn: AI校正 step `Ptpni` + connector `P2QLk` (XkGvG), row `GOutw` (fS5is); F3 hints `WKtjK`/`wupMI`; glossary delete `Q11NpX`(`zl2HB`) + 3-badge strip `r7rxg0`; F1 annotations `peX3f`(desktop)/`hJV5q`(mobile, in gutter); 已選項目 counts `oRWTl`/`ReTj7`; F9 note `q1PjM`; F8-D composite note `bp5aN`.
- Sally's 2 in-place fixes (via batch_design): (1) F8-M inline stepper `RGssw` width overrides retuned for 6 steps (AI校正 addition had overflowed the 390px card, clipping 提取音訊/完成); (2) F1-M aspirational annotation added (`hJV5q`) — F1-M sheet was full, placed in the right gutter mirroring the desktop `peX3f`.
- Screenshot regen: 143/143 rendered; export was fully deterministic this run (only 11 PNGs genuinely changed, ZERO re-render noise). Staged the genuine set: f1-d/f1-m/f3-d/f3-m/f4-d(AI校正 propagated into failed frame)/f6-d/f6-m(nDSEd delete propagated into glossary screens — Sally's list omitted these; caught at staging)/f8-d/f8-m/f9-d + design-system/component-library.png + ux-design.pen. Content spot-verified: f3-d (6-step stepper + 關閉), f8-d (已選項目 count + composite), f6-d (glossary rows + delete/badges).

### Discovery Triage

- N/A — no out-of-scope work discovered. Parked out-of-scope notes (unchanged, NOT this chore): Component Library caption `bWjaa` claims a "failed" GenerationProgress state that IS demonstrated in the pre-existing F4-D frame but not the library cell (left untouched); F9 success-only copy would be an ADR-lite product decision (not filed — Sally's L2 ruling was ACCEPT processed-semantics + annotate, which this chore did).

### File List

- `ux-design.pen` (9 edits + 2 Sally fixes)
- `_bmad-output/screenshots/flow-f-subtitle-v2/{f1-d,f1-m,f3-d,f3-m,f4-d,f6-d,f6-m,f8-d,f8-m,f9-d}-v2.png`
- `_bmad-output/screenshots/design-system/component-library.png`
- `_bmad-output/implementation-artifacts/chore-pen-subtitle-v2-design-sync.md` (this story)
- `_bmad-output/implementation-artifacts/sprint-status.yaml` (→ done)
