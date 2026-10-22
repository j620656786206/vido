# disc-2026-10-reparse-dialog-misses-instant-pass：重新解析的視窗不會再卡在「等待開始…」

Status: review

**Source:** 2026-10-22 `testsprite-2026-10-quota-run`：TC115 兩次 FAIL（Rule 24 lane ③ 立案）。Alexyu 排序：三個 bug 裡第二個修。

## 為什麼要做

在媒體庫勾幾部片按「重新解析」，視窗會說「已排入比對 N 項，等待開始…」，接著應該變成「比對中」→「比對完成 成功 X・失敗 Y」。

TestSprite 經過通道（像從外面連回家的慢連線）跑了兩次，視窗都停在「等待開始…」超過 60 秒；但那兩部片在媒體庫裡早就變成「失敗」，代表後端已經跑完了。

原因：視窗是在「送出重新解析」**之後**才開始聽進度（SSE）。如果比對很快就結束（例如 TMDb 連不到，一瞬間全部失敗），「完成」通知比視窗開始聽還早送出，視窗就永遠收不到。本機連線很快，所以看不出來。9 月修過一樣的問題：小片庫一下就掃完、看不到完成提示（#609），當時的解法是「先連上 SSE，再送出」。

## Acceptance Criteria

1. **先連上再送出。** 按下確認後，先打開進度連線並等它連上，才送出重新解析的請求。連不上時最多等 3 秒，之後照樣送出（壞掉的進度連線不能擋住重新解析）。
2. **瞬間完成也收得到。** 比對在請求送出後立刻結束時，視窗仍會顯示「比對完成」和成功／失敗數。
3. **沒排入任何項目時不留連線。** 重新解析一部都沒排入（全部失敗）時，關掉這條連線，視窗照舊不顯示比對狀態。
4. **其他行為不變。** 刪除、匯出不開連線；第二次重新解析會從「已排入」重新開始；離開頁面時關掉連線。
5. **測試。** hook：要求開始時建立連線，`open` 之後才回報已連上；逾時也會回報；`enrich_complete` 在送出前就到也會被收到。頁面：確認後先等連線、再送出請求（連線未連上時請求還沒送）；排入 0 項時關掉連線。

## 不要做的事

- 不改後端（`/events` 已經是先登記、再回第一個位元組，跟 #609 一樣）。
- 不改視窗的版面與文字。
- 不碰掃描的 SSE（`useScanProgress`）。

## Tasks

- [x] `useEnrichmentRefresh`：改成由頁面在送出前呼叫「開始聽」，回傳一個連上（或逾時）就完成的 Promise（AC 1、2、4）
- [x] `LibraryBrowseV2`：重新解析先 await 連線再送出；排入 0 項時停止（AC 1、3）
- [x] 測試（AC 5）
- [x] :8090 本機確認重新解析的視窗會走到「比對完成」

## Dev Notes

- 現在的流程：`executeBatchAction('reparse')` → POST → `successCount > 0` 才 `setWatchEnrichment(true)` → hook 的 effect 開 `EventSource`。
- 參考 #609 的 `requestScanTracking`（`hooks/useScanProgress.ts`）：`SCAN_TRACKING_CONNECT_TIMEOUT_MS = 3000`、`es.onopen` 代表之後的事件都收得到。
- `LibraryBrowseV2.spec.tsx` 用 `enrichmentRefresh(enabled, runId)` mock 這個 hook，介面改了要一起改。

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（claude-opus-5-5），2026-10-22。

### Completion Notes List

- `useEnrichmentRefresh` 不再用 `enabled`／`runId` 開關，改成回傳 `{ matching, watch, stop }`：`watch()` 開一條新的 EventSource（關掉舊的、狀態回到「已排入」），連上（`onopen`）、出錯（`onerror`）或 3 秒逾時就 resolve；`stop()` 關掉並清空；離開頁面時關掉。
- `LibraryBrowseV2`：重新解析先 `await watch()` 再送出；一部都沒排入或請求丟例外就 `stop()`。刪除、匯出不開連線。
- 測試：hook 8 條（連上才 resolve、逾時與出錯也會 resolve、連上後立刻來的「完成」收得到、第二次重新開始、stop 清空、壞資料不當機…）；頁面新增「連線還沒好之前不會送出請求」，並把 mock 換成新的介面。library＋hook vitest 23 檔 321 條全綠。
- 本機 :8090（重新 build）：用 Playwright 把進度連線故意延遲 2 秒（模擬慢連線）。請求在連線之後才送出（`GET /events` → 2 秒後 `POST /library/batch/reparse`），視窗走到「比對完成 成功 2・失敗 1 — 清單已更新」。
- 已知小事：連線慢時，按下確認到開始比對之間最多會多等 3 秒（等連線）；視窗這段時間顯示「重新解析中…」。

### File List

- `apps/web/src/hooks/useEnrichmentRefresh.ts`、`useEnrichmentRefresh.spec.ts`
- `apps/web/src/components/library/LibraryBrowseV2.tsx`、`LibraryBrowseV2.spec.tsx`
- 本檔、`sprint-status.yaml`（也把上一張 `disc-2026-10-batch-mixed-selection-wrong-type` 標 done，PR #900）
