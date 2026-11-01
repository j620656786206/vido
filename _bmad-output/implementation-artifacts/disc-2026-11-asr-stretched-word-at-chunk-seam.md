# Story disc-2026-11-asr-stretched-word-at-chunk-seam：聽寫切段處的第一個字不再跑到 20 秒前

Status: done

**Source:** ⚖️ Alexyu 2026-11-01「1 修」（聽寫改善討論後，先修切段時間錯位）。

## 問題

《特種部隊：母獅》S01E01 的聽寫稿（10/28，Scribe v2）有一句：

```
150  00:14:00,000 --> 00:14:07,000  That'll
151  00:14:22,940 --> 00:14:24,020  do, Cruz.
```

原本以為 14:00 那句是憑空生出來的（issue #940）。2026-11-01 實驗發現「That'll do, Cruz.」是真的台詞，實際在 14:22。14:00 剛好是 120 秒切段的格線（840 秒）：這段音訊開頭是 22 秒的配樂，Scribe 把第一個字「That'll」從 0 秒一路拉到 22.86 秒。Vido 把這個字當成一句，只把結尾截到 7 秒（`scribeCueMaxSeconds`），開頭留在 14:00。

同一段重送 Scribe 一次，這次時間是對的（22.56–22.86），所以這是 Scribe 偶爾會發生的狀況，不是每次都有。

## 做法

`ai/scribe.go` 分句之前多一步 `reanchorStretchedWords`：一個字超過 2 秒、而且下一個字在它結束後 0.8 秒內就開始（也就是後面緊接著有人說話），代表它的**結尾**才是真的時間 → 保留結尾，開頭改成「結尾往前推這個字念得完的時間」（每個字母約 0.08 秒，0.3～1 秒）。後面沒有緊接的字（例如一聲長長的尖叫）照舊，由原本的 7 秒上限處理。

## Acceptance Criteria

1. [x] 母獅那段的字（That'll 0–22.86、do, 22.94、Cruz. 23.26）分出來是一句「That'll do, Cruz.」，從 22.3 秒左右開始。
2. [x] 正常長度的字、後面沒有緊接說話的長字、中間隔著音效的長字都不動；原本「一個字拉 15 秒只截到 7 秒」的測試照過。
3. [x] 不改輸入的陣列。

## Dev Agent Record

- `scribe.go`：新常數 `scribeMaxWordSeconds`／`scribeSecondsPerWidth`／`scribeReanchorMinSeconds`／`scribeReanchorMaxSeconds`，`reanchorStretchedWords`，`scribeCues` 開頭呼叫。
- 測試：`TestScribeCues_StretchedWordBeforeSpeechMovesToWhereItWasSaid`、`TestReanchorStretchedWords_LeavesNormalAndIsolatedWordsAlone`；`go test ./internal/ai/`、`go vet` 通過。
- 實驗紀錄：同一段音訊 840–960 秒重送 Scribe，第一個字時間正確，正式跑那次的原始字時間已不在快取裡，無法逐字比對；修法只依賴「後面緊接著說話」這個條件，不靠猜。

## Change Log

- 2026-11-01：建立＋實作。
