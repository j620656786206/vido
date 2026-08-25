# Fix: green stops saying「做完了」— the vocabulary heals

Status: ready-for-review

> Closes P1#3, the last P1 from the 夜行複測 critique. 固定詞彙規則: green
> means IN PROGRESS; spending it on done-ness devalues the green of 已連線 —
> the one asset PRODUCT.md says must never be diluted.

## Five sites, done-ness → neutral

| site | was | now |
|---|---|---|
| QBittorrentForm 設定已儲存 | success-text | neutral note + Check icon |
| CacheManagement 已清除 N 筆 banner | success-tint/text | bg-tertiary / text-secondary |
| LogsViewer 已清除 N 筆 banner | same | same |
| ScannerSettings success notification | same | bg-tertiary / text-primary |
| BackupTable 完成 label + dot | success-text / success dot | text-secondary / muted dot |

## What deliberately KEEPS green (current state, not done-ness)

已連線 (status), ConnectionTestResult 連線成功 (the connection works NOW),
LibraryCard 可自動字幕 (current capability), ApiKeysForm key-test OK.

## Verification

- 2 new vocabulary guard tests (CacheManagement banner, BackupTable 完成),
  each individually falsified — including a re-do after discovering prettier's
  line-split had made the first BackupTable falsification a no-op.
- Settings suites 316/316; visual suite zero churn against current baselines
  (the changed states will churn once #307's confirm flow and this land
  together — expected, small).
- The CacheManagement guard clicks twice, which is correct under BOTH the
  pre-#307 (one-click) and post-#307 (arm+confirm) behaviour, so this PR does
  not stack on #307.
