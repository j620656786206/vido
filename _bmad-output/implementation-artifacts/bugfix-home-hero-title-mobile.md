# Story: bugfix-home-hero-title-mobile — 首頁 hero 片名在手機上照稿 20px；字級表的 Display 用途更正

Status: done

## Story

身為在手機上用 vido 的人，我要首頁大海報上的片名小一點，
讓它不要蓋掉太多海報圖。

## 背景（查到的事，附親自讀過的位置）

- 上一張 `bugfix-type-heading-mobile-step` 留下一個疑問：DESIGN.md:353 說 Display 用在「詳情頁 hero 標題」，但 `DetailHeroV2.tsx:132` 是 H1（24／30）。
- **查 `.pen`（Pencil MCP，2026-09-28）**：詳情頁片名 B3p-D `E9ZjaG`、B4p-D `E5oihq`、B5-D `CxanW` 都綁 `$Type/H1` → **程式碼對，是表寫錯**。Display 真正的用途：首頁 hero 片名 H1-D-v3 `N8QqX`、H7-D-v3 `W0RdQH`，以及 placeholder 首字 B9-D `NFiZe`、B10p-D `F8u3G`。
- 手機首頁 H2-M-v3（`uGCAU`，`theme:{bp:"mobile"}`）的片名 `rQDOm` 綁 `$Type/H2` → 手機 20。表上 Display 的手機值是 30，程式碼 `HeroBanner.tsx:196` 是 24——**三邊各不同**。
- ⚖️ Alexyu 2026-09-28 選「照設計稿，20px」。

## Acceptance Criteria

1. 首頁 hero 片名：手機（<640）20、640–1023 30、≥1024 36。
2. DESIGN.md：Display 用途改為「首頁 hero 片名、placeholder 首字」；H1 用途加「詳情頁片名」；補一段裁定說明手機首頁例外。
3. 測試鎖住三段字級；mutation 驗證。

## Tasks / Subtasks

- [x] Task 1 — `HeroBanner` 片名 `text-2xl` → `text-xl`（AC #1）
- [x] Task 2 — DESIGN.md 表格與裁定段（AC #2）
- [x] Task 3 — `HeroBanner.spec.tsx` 新測試＋mutation（AC #3）

## Dev Agent Record

### Completion Notes List

- 改一個 class：`text-2xl sm:text-3xl lg:text-4xl` → `text-xl sm:text-3xl lg:text-4xl`，並在旁邊註明裁定與節點。
- 新測試 `[P2] the title is H2 on a phone and Display from lg`。Mutation：改回 `text-2xl` → 紅；還原 → 綠。
- 視覺基準：`homepage-hero-banner` 夾具是桌機寬度，零變動（`--update-snapshots` 沒有重寫任何檔）。
- 驗證：web 全綠、lint 0 errors、prettier 綠。

### File List

- `apps/web/src/components/homepage/HeroBanner.tsx`、`HeroBanner.spec.tsx`
- `DESIGN.md`、`_bmad-output/implementation-artifacts/sprint-status.yaml`

## Change Log

| Date | Change |
| --- | --- |
| 2026-09-28 | 建單＋開發完成（承接 `bugfix-type-heading-mobile-step` 的待查項）→ review |
| 2026-09-28 | PR #570 合併（a14631de），CI 全綠 → done |
