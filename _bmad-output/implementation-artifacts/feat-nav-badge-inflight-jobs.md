# Feat: the dead badge prop comes alive — 活動 wears the in-flight count

Status: ready-for-review

> Closes `feat-nav-badge-inflight-jobs`（⚖️ 2026-08-25 ruling「接上它 —— 這是
> 產品護城河」）and `disc-2026-08-home-inflight-chip-dead-end` alongside.

## What was wired

`SidebarNavItem`'s `badge` prop was documented, styled, and called by nobody.
It is now fed by `useInflightJobCount()` — a thin selector over the SAME
`GET /api/v1/activity` query the Activity hub polls (TanStack dedupes across
mounts; no second counting path, per the ruling). Count =
`active_jobs.jobs.length`, only when the section reports `ok`.

Three homes, one readout:
- **Expanded sidebar**: the existing wash pill, now live.
- **Collapsed rail**: badge support ADDED (the rail ignored the prop) — a
  corner pill on the 44px icon; the tooltip carries the count too. The
  collapsed rail is exactly the「回來查」posture; the readout must survive
  collapse.
- **MobileTabBar**: 活動 tab wears a corner pill — on a phone the returning
  user's「現在有東西在跑嗎」no longer costs a tap into a sheet.

Honesty rules carried through:
- **Absent at 0 and while unmeasured/degraded** — exception signal, never a
  number nobody measured.
- **aria-label speaks it**:「活動（N 個任務進行中）」— the Link's aria-label
  replaces subtree text, so a drawn-only badge would be silent to AT.
- **Wash + `--accent-text`, never solid** (配給強調 — a badge is a readout).

## The homepage chip (filed alongside)

進行中 · N on 最近新增: gold → **running green** (固定詞彙: 綠＝正在發生) and
now a **Link to /activity** (was a dead end). The filed third complaint
(recent-20 window) was investigated and **rejected**: `pending.parse_count`
measures the parse-job QUEUE (capped) — live-diverges from item parseStatus
(0 vs 3 on the seeded env) — so the windowed count, scoped to the row the
chip annotates, is the honest one. Recorded in sprint-status so it is not
re-litigated.

## Verification

- Unit: shell 35/35 + homepage row 6/6 (new: badge in all three homes,
  absent-at-zero, AT label, wash-not-solid recipe, chip door + green).
- Falsified: unwiring the sidebar badge → 2 red.
- Live on the rebuilt seeded env: badge ABSENT at 0 jobs (correct); with the
  API mocked to 2 jobs, badge shows「2」in expanded/rail/mobile with the
  right aria-labels (screenshots in scratchpad); chip renders green
  `<a href="/activity">進行中 · 3</a>`. A triggered real scan finishes
  sub-second on the seeded library, so transient visibility was verified via
  the mock, not the scanner.
- Visual suite: fully green, ZERO baseline churn — gallery fixtures carry no
  in-flight state, and an exception-signal badge correctly renders nothing.
