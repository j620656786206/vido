# Disc：Gemini 免費方案每分鐘到底幾次——複量後把 13 秒改成 5 秒

Status: done

**Source:** 2026-10-14 FB Ming Jwo：「gemini-3.5-flash-lite 只要確保每分鐘 15 個 request 內就免費，我是每個 request 睡 4 秒」。我們 2026-10-14 早上量到 5/分鐘就 429，所以 #786 的預設是 13 秒一批。Alexyu 選 A 兩張都跑。

## 查到的事（2026-10-14 晚上，本機直接打 Gemini OpenAI 相容端點）

- 15 次、每 4 秒一次（＝15/分）：15/15 成功，72 秒。
- 12 次、每 2 秒一次（＝30/分）：12/12 成功，33 秒。
- 20 次**真實大小**的翻譯 prompt（2,300 input tokens、系統提示＋段落），每 4 秒一次：20/20 成功，102 秒。
- 結論：flash-lite 免費方案的每分鐘限制是 15 次（跟 Ming Jwo 說的一樣）；早上的 429 是 **gemini-3.8-flash 每天 20 次的額度**（考卷記錄裡寫得很清楚）以及沒控速的連發，不是 flash-lite 的 RPM。13 秒是把兩件事混在一起的保守值。
- 既有數字：一集 835 句 ≈ 84 批；13 秒 → 18 分鐘；5 秒 → 7 分鐘。

## Acceptance Criteria

1. `VIDO_GEMINI_MIN_INTERVAL` 預設 13 → **5**（12/分，留 15/分 的餘裕）：`config.go`、`services.DefaultGeminiMinInterval`、`.env.example`、兩份 compose、`docs/development.md`、`docs/deployment.md` 一起改。
2. 既有測試改成 5 秒；pacing 估算（估價加的分鐘數）跟著變小、不另改程式。
3. 註解寫清楚為什麼早上量到 5（3.8-flash 的每日額度），免得之後又改回去。
4. 正式機：`update_container` 後 log `gemini_min_interval=5s`。

## Tasks / Subtasks

- [x] T1 預設值＋文件（AC 1、3）
- [x] T2 測試（AC 2）
- [x] T3 正式機（AC 4）：2026-10-15 09:51 `update_container` 升到 c6e5202f，log `gemini_min_interval=5s`。

## Dev Notes

- 正式機容器 env 裡沒有明設 `VIDO_GEMINI_MIN_INTERVAL`（模板沒這個欄位），所以升級後直接吃新預設。
- 沒動 `cmd/grade --pace`（考卷用）。

### Time-dependent visual coverage

N/A — 純後端與文件。

## Dev Agent Record

### Agent Model Used

Claude Fable 5.1（claude-fable-5-1），2026-10-14。

### Completion Notes List

- 三組量測腳本在 session scratchpad（`gemini_rpm.py`、`gemini_rpm2.py`），只打 flash-lite、總共 47 次請求、免費方案 US$0。

### Discovery Triage

N/A — no out-of-scope work discovered.

### File List

- `apps/api/internal/config/config.go`、`apps/api/internal/services/{translation_dispatcher.go,translation_dispatcher_test.go}`、`.env.example`、`docker-compose.yml`、`docs/{docker-compose.nas.yml,development.md,deployment.md}`
