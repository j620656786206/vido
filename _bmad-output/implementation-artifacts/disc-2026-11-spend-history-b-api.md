# Story disc-2026-11-spend-history-b-api：一支 API 列出每次執行的花費、預估、模型與來源

Status: ready-for-dev（依賴 -a）

**Source:** 傘狀 `disc-2026-11-spend-history`。給 -d 的新頁讀。不做：畫面、改月報（`GET /subtitles/spend` 維持原樣）。

## 提議的端點（⚠️ 欄位名要 Alexyu 確認後才標 [@contract-v1]）

`GET /api/v1/subtitles/runs?from=2026-10-01&to=2026-11-01&route=&page=1&page_size=50`

```json
{
  "items": [
    {
      "id": "…",
      "started_at": "2026-10-28T13:33:46Z",
      "completed_at": "2026-10-28T13:38:25Z",
      "status": "completed",
      "failure_reason": "",
      "media": { "id": "…", "type": "episode", "title": "Lioness", "season": 1, "episode": 1 },
      "route": "asr",
      "source": "audio",
      "spent_usd": 0.52,
      "estimate_usd": null,
      "translation_model": "claude-sonnet-5",
      "asr_provider": null,
      "asr_model": null,
      "batch_id": null,
      "triggered_by": "manual"
    }
  ],
  "page": 1, "page_size": 50, "total": 106,
  "series": [ { "date": "2026-10-28", "spent_usd": 3.91, "by_route": { "asr": 3.10, "translate": 0.81 }, "runs": 9 } ]
}
```

- 沒記錄的數字是 `null`，不是 0（PRODUCT.md 原則 1，和月報同一套誠實規則）。
- `translation_model` 只在真的有翻譯的路線（`translate`、`asr`）才給；`deliver_direct`／`convert_then_deliver` 給 `null`（那些路線沒呼叫模型，`model_id` 只是版本欄位）。
- `series` 給圖表用：依伺服器時區分日，涵蓋 `from`～`to`；沒有執行的日子也要有一筆 `spent_usd: 0, runs: 0`（那天是真的 0）。
- 含失敗與中途停止的執行（它們也花了錢）。
- 片名：電影用 `movies.title`、單集要帶劇名＋季集；找不到媒體（已刪）時 `title: null`。

## Acceptance Criteria

1. 端點照上面的形狀回應；欄位名經 Alexyu 確認後標 [@contract-v1]，swag 註解寫齊。
2. 分頁、日期篩選、`route` 篩選；`from`／`to` 格式錯回 400 `VALIDATION_INVALID_FORMAT`（沿用 7-6b）。
3. `series` 每日一筆、含 0 的日子；by_route 的總和等於當日 `spent_usd`。
4. NULL 誠實：舊資料的 `estimate_usd`／`source`／`asr_*` 是 `null`；`spent_usd` NULL 的執行 `spent_usd: null`。
5. Repository 用真的 SQLite 測（Rule 28），含 movie、episode、已刪媒體三種。

## Change Log

- 2026-11-01：建立（Bob）。
