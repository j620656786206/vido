# disc-2026-10-reparse-dialog-empty-followup-pass：混選重新解析的視窗不會再以「成功 0・失敗 0」收尾

Status: done

**Source:** 2026-10-22 `release-0-3-1-tag-and-notes` 發版前實測（正式映像 main-2917eaf、TZ=Asia/Taipei）。Alexyu 選 B：先修再發 v0.3.1。

## 為什麼要做

在「全部」頁同時勾電影和影集，按重新解析。#900 之後前端會分兩次送出（電影一次、影集一次）。

後端收到第一個請求就開始比對；第二個請求到的時候比對還在跑，所以後端記一筆「跑完再補跑一輪」。可是第二批的片子在第一輪就已經一起處理掉了，補跑那一輪是 **0 筆**，它照樣送出「完成：成功 0・失敗 0」，把視窗上第一輪的真實結果（例：成功 2・失敗 2）蓋掉。

使用者看到的是：按了重新解析，結果「成功 0・失敗 0」——像是什麼都沒做。

## Acceptance Criteria

1. 補跑的那一輪（因為「比對中又有片子被設成待處理」而多跑的）如果一筆都沒有，**不送出完成通知**；只寫 log。
2. 其他情況照舊：第一輪（就算 0 筆）照樣送完成通知；補跑有東西時照樣送。
3. 混選電影＋影集重新解析時，視窗最後顯示第一輪的真實結果。
4. 測試：服務層——第一輪跑著時再要求一次、補跑 0 筆 → 只送出一次完成通知（第一輪的）；直接要求一輪、0 筆 → 照樣送出。

## 不要做的事

- 不改前端。
- 不改「補跑」本身要不要跑的判斷（那是 disc-2026-09-batch-reparse-never-runs 的保證）。

## Tasks

- [x] `EnrichmentService.runUntilSettled`／`runPass`：補跑 0 筆不送完成（AC 1、2）
- [x] 測試（AC 4）
- [x] 正式映像或本機實測混選重新解析（AC 3）

## Dev Notes

- `apps/api/internal/services/enrichment_service.go`：`runUntilSettled` 在 `rerunRequested` 時再呼叫 `runPass`；`runPass` 結尾一律 `broadcastComplete(result)`。
- 前端 `useEnrichmentRefresh` 收到 `enrich_complete` 就把視窗設成那一輪的結果（設計如此：排在後面的那一輪會接著顯示）。
- 測試可用 `sse.NewHub()` 的 `AddObserver` 收集事件；`enrichment_request_run_test.go` 已有讓第一輪卡在半路的 `gatedMovieRepo`。

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（claude-opus-5-5），2026-10-22。

### Completion Notes List

- `runUntilSettled` 傳 `followUp`（第一輪 false、補跑 true）給 `runPass`；`runPass` 結尾在「補跑而且 0 筆」時不送 `enrich_complete`，log 照寫。補跑本身照跑。
- 測試 2 條：第一輪跑到一半再要求一次 → 跑 2 輪、只送 1 次完成（改回舊程式會紅）；閒置時直接要求、0 筆 → 照樣送 1 次。`-race` 綠。
- 本機 :8090：進度連線延遲 1 秒，混選 2 部電影＋1 部影集按重新解析。log 一樣是「第一輪 total=4 succeeded=2 failed=2 → 補跑 total=0」，視窗停在「比對完成 成功 2・失敗 2」，3 秒後也沒有被蓋掉。修之前同樣情況最後會變「成功 0・失敗 0」。
- 驗證：`go test ./...`、staticcheck 全綠。

### File List

- `apps/api/internal/services/enrichment_service.go`、`enrichment_request_run_test.go`
- 本檔、`sprint-status.yaml`
