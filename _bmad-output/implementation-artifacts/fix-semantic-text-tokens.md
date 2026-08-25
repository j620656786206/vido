# Fix: semantic colours stop being read — components join the *-text vocabulary

Status: ready-for-review

> Closes the top P1 from the 夜行複測 critique (24/40, 2026-08-25). The token
> layer was gated and correct; 51 component-level call sites were reading
> through the base fill tokens.

## Two sweeps, one measurement-driven exception

1. **Text (and icons beside it): base → `*-text`**, 51 sites across 16 files.
   Fills and dots (`bg-`) untouched — those are graphics, base is correct.
2. **Button labels on solid fills** — measured per fill, not assumed:

   | fill | 白字 | 墨字 | verdict |
   |---|---:|---:|---|
   | 金 accent | **2.40 ✗** | 7.55 | → `--text-on-accent` (9 sites) |
   | 赭 warning | **3.25 ✗** | 5.57 | → `--text-on-accent` (1 site) |
   | 朱砂 error | **5.44 ✓** | 2.63 ✗ | **white KEPT** (4 sites) |

   A blanket swap would have broken the error buttons; the measurement split
   the treatment.

## Live-verified (production build)

| reading | before | after |
|---|---:|---:|
| 已斷線 status text | 3.00 | **6.26** |
| log ERROR badge | 3.17 | **6.62** |
| 建立備份 / 匯出 labels | 2.40 | **7.55** |

## Notes

- 2 spec files updated from base to *-text assertions; settings suites 314/314.
- Visual churn: 4 settings components' darwin baselines regenerated; linux
  already absent on this branch (deleted by the reskin) — the post-merge
  bootstrap covers both at once.
- Remaining from the critique, NOT here: unguarded clears (P1#2), green-says-
  done vocabulary (P1#3), page-header unification (P2) — each its own pass.
