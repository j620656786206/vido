# Fix: the two unguarded clears join the two-step confirm

Status: ready-for-review

> Closes P1#2 from the 夜行複測 critique. The finding was not that these
> actions lacked ceremony — it was that their ceremony CONTRADICTED the
> pattern ten pixels below them (CacheTypeCard's per-type clears confirm;
> the bulk clears on the same pages did not).

## What changed

`CacheManagement` 清除 30 天前的快取 and `LogsViewer` 清除 30 天前 now use the
exact CacheTypeCard grammar: first click ARMS (label flips to 確認清除…, button
goes `--error` fill, a 取消 escape appears beside it), second click executes,
取消 or completion disarms. No new pattern was invented — the third instance of
an existing one.

## Verification

- 6 new behaviour tests (arm / confirm / cancel × 2 components), falsified:
  removing the arming step turns 5 red.
- The legacy one-click test now double-clicks BY DESIGN — its old expectation
  was the bug.
- Settings suites 320/320; full suite green except the 16 pre-existing
  eslint-rules env failures; visual suite zero churn (armed state is
  interaction-only, not in gallery default states); prettier clean.

## Critique remainder

P1#3 green-says-done (`/impeccable clarify`) and the P2s remain, each separate.
