# Disc：各季寫法卡片 A（後端）——按「確認」不再刪掉某一季的官方寫法

Status: review

**Source:** ⚖️ 2026-10-18 Alexyu 選「每季照官方」（`disc-2026-10-glossary-season-variants-card-design`，PR #811）。原本一張單，Bob 建檔時照 Story Splitting Rule（跨前後端、每邊超過 3 項就拆）拆成 A（後端，本單）與 B（前端，`…-card-b`）。A 先做：它單獨上線就能擋掉「按確認會把第 1 季官方寫法刪掉」。

## 查到的事（2026-10-18）

- `GlossaryService.Confirm`／`Edit` 成功後跑 `clearDrawersOf` → 刪掉該詞所有**未確認**的季別寫法。
- `GlossaryService.ConfirmAll` 成功後跑 `DeleteSeasonDrawers(scope)` → 刪光整部劇未確認的季別寫法。
- `Delete` 全劇列也只清未確認的季別寫法；已確認的留下變孤兒。
- 讀端兩處（`subtitle/glossary_store.go` 的 `LookupFor`、`transcription_service.go` 的 `overlaySeasonDrawer`）：全劇那個詞只要被確認過，季別寫法就**不套用**（scope-a CR 1「使用者說了算」）。所以就算季別寫法沒被刪，按過確認的詞第 1 季也會翻成全劇寫法。
- 正式機《末日光明》目前全部 `confirmed=0`，還沒有人按過，資料是好的。

## 新規則（照設計稿 F6s Note）

1. 確認全劇列＝整張卡片確認：全劇列＋它的季別寫法一起改成已確認，不刪。
2. 編輯全劇列只改全劇譯名（並確認它），季別寫法不動。
3. 刪除全劇列＝整張卡片刪：連同它的季別寫法（不論確認與否）一起刪。
4. 全部確認＝全劇列與所有季別寫法都改成已確認，不刪。
5. 讀端：某一季有季別寫法，就用它，不管全劇那個詞有沒有被確認。要讓那一季改用全劇寫法，就刪掉那一行季別寫法。
6. 季別寫法自己的編輯／刪除維持原樣（只動那一列）。
7. 重新學習官方字幕（miner）只清未確認的季別寫法——已確認的保留，原本就是這樣，不改。

## Tasks

- [x] T1 repository：`ConfirmSeasonDrawersOfTerm(id)`、`DeleteAllSeasonDrawersOfTerm(id)`（含已確認）、`ConfirmAllByScope` 連季別一起確認。
- [x] T2 service：Confirm／Edit／Delete／ConfirmAll 改用新規則；`clearDrawersOf` 退場。
- [x] T3 讀端：`LookupFor` 與 `overlaySeasonDrawer` 拿掉「已確認就不覆蓋」。
- [x] T4 測試：回歸「確認 Baba Voss 之後，第 1 季的集仍拿到巴巴禾斯」（service＋讀端兩條路徑）。

## Acceptance Criteria

1. 確認全劇 Baba Voss → 全劇列與 s1 列都 `confirmed=1`，s1 列還在。
2. 編輯全劇 Baba Voss → s1 列不變。
3. 刪除全劇 Baba Voss → s1、s2 列（含已確認的）都刪掉。
4. 全部確認 → 全劇與季別列全部 `confirmed=1`，一列都沒少。
5. 第 1 季的集：全劇 Baba Voss 已確認為巴霸沃斯、s1 為巴巴禾斯 → 翻譯拿到巴巴禾斯（`LookupFor` 與 `loadGlossary` 兩條路徑都測）。
6. `go test ./...`、`go vet`、staticcheck 綠。

## 不做

- 面板改卡片：B 單。

## Dev Agent Record

- repository：`ClearSeasonDrawersOfTerm` 換成 `ConfirmSeasonDrawersOfTerm`／`DeleteSeasonDrawersOfTerm`（共用 `drawerPatternOfTerm`，季別列 id 一律 no-op）；`ConfirmAllByScope` 用 `scope = ? OR scope LIKE '<scope>:s%'` 一起確認。
- service：`clearDrawersOf` 退場；Confirm → `confirmDrawersOf`、Delete → `deleteDrawersOf`、Edit 不碰季別、ConfirmAll 不再 `DeleteSeasonDrawers`。
- 讀端：`glossaryStoreRepository.LookupFor` 與 `overlaySeasonDrawer` 拿掉 confirmed 鎖；同一個詞大小寫不同時以季別那筆取代（不再兩筆並存）。
- 測試：repository 兩條（卡片確認保留並確認、卡片刪除連已確認一起刪、別的詞／別部劇不動；全部確認連季別確認）、service 一條、讀端兩條（`LookupFor`、`overlaySeasonDrawer`）改成新規則。`go test ./...`、`go vet`、staticcheck 綠。
- miner 的 `DeleteSeasonDrawers` 不改：本來就只清未確認的，已確認的季別寫法重學也不會被洗掉。
