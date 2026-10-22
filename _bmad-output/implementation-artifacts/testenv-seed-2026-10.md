# testenv-seed-2026-10：測試用的假片庫補上「字幕狀態、生成紀錄、AI 花費」，十月的新畫面才測得到

Status: ready-for-dev

**Source:** 2026-10-22 Party Mode（Murat／John／Bob／Sally／Amelia）＋ founder-pm 判決，Alexyu 選 A：「先補假資料，同時先跑回歸」。上層任務是 `testsprite-2026-10-quota-run`。

## 為什麼要做

本機測試環境（`scripts/serve-test-env.sh`，:8090）的假片庫是 7 月寫的：只有片名、海報、季集。9/29 之後上線的畫面，大多要讀下面這些資料，但假片庫裡**一筆都沒有**：

| 十月上線的畫面 | 要讀的資料 | 假片庫現在 |
|---|---|---|
| 媒體庫「N 部缺中文」一點就篩、依字幕狀態排序（#820–#826） | 片子的字幕狀態＋字幕軌 | 全部同一個狀態，篩不出差別 |
| 季清單每集中文字幕標記（#717–#721） | 每集的 `subtitle_tracks` | 空 |
| 活動記錄每次生成：花多久、花多少、失敗原因（#828/#831/#851/#864） | `subtitle_runs` | 只有 `--failure-fixtures` 時有一筆失敗 |
| AI 帳號沒錢時說去哪儲值（#757–#760） | 帶「額度用完」原因的 run | 無 |
| 本月 AI 花費「檔名解析」那格、首頁「本月 $x」（#637/#892） | `subtitle_runs.spent_usd`＋`ai_side_spend` | 無 |
| 只填 Gemini 金鑰也能用（#836） | 伺服器要有 `ENCRYPTION_KEY` 才存得進金鑰 | 刻意沒設（TC132 靠這點） |

不補的話，十月的 TestSprite 新案例會「拿空盤子去考試」——不是 PASS 得空洞，就是測不出東西。

## Acceptance Criteria

1. **字幕狀態有分布。** 假電影裡至少各有一部是：繁中（片內或外掛）、簡中、只有英文（缺中文）、沒有任何文字字幕（無字幕源）、還沒檢查。媒體庫標題旁的「N 部缺中文」數字＝勾「缺中文字幕」篩出來的筆數，而且不是 0、也不是全部。依「字幕狀態」排序時，缺中文的排最前、繁中的排最後。
2. **影集各集也有分布。** `進擊的巨人` 第 1 季至少一集有繁中、一集缺中文；季清單的標記和影集標頭的「有沒有中文字幕」對得上（沿用 `ChineseSubtitleVerdict` 同一套規則，不另寫判斷）。
3. **生成紀錄。** 本月內至少 5 筆 `subtitle_runs`：成功（有花費、有時長）、失敗（一般原因）、已略過（no-text-source）、處理逾時（queue-timeout）、AI 帳號沒錢。原因字串用 `models/subtitle_run_failure.go` 既有的前綴與分類，不自己發明。活動記錄每一筆都看得到花了多久、花了多少、中文原因；已略過是灰色、失敗是紅色。
4. **AI 花費。** `ai_side_spend` 至少一筆 `kind='parse'`；`GET /api/v1/subtitles/spend` 回的本月總額 ＝ runs 花費＋side spend，且首頁「本月 $x」顯示同一個數。
5. **金鑰開關。** `serve-test-env.sh` 多一個 `--with-encryption-key` 選項：有帶才設一把固定的測試用 `ENCRYPTION_KEY`；沒帶時行為完全不變（TC132 仍要過）。
6. **不破壞舊案例。** 預設跑法（不帶新選項）下，9 月的 108 條案例讀到的東西不變：電影總數、未匹配／整理中筆數、已刪除兩部不出現、`setup_completed` 仍為 true。若某個舊斷言因新增資料而必須改（例如總數），在本單列出並同步改 `testsprite_frontend_test_plan.json`。
7. **寫入走正式路徑。** 跟現在一樣用 repository 層寫，不寫生 SQL（`cmd/seed` 開頭註解的原則）；`ai_side_spend` 若沒有 repository，用既有的 service／repo 方法，沒有就加最小的一個，不在 seed 裡直接 `INSERT`。
8. **測試。** `apps/api/cmd/seed` 加測試：跑完 seed，用 repository 讀回，斷言 AC 1–4 的筆數與 verdict。`go test ./...`、`staticcheck ./...` 綠。

## 動工前要先確認（Confirm Before Coding）

- 電影／影集上決定「缺中文」的欄位實際名字（`vido_chinese_subtitle(status, language, tracks)` 吃的三個欄位）——先讀 `models/movie.go`、`repository/chinese_subtitle_sql.go`，不要猜。
- `subtitle_tracks` 的 JSON 形狀，照 `episode-subtitle-badge-a`（#717）的寫法。
- 「AI 帳號沒錢」那筆 run 的 `error_message` 要長什麼樣，才會被歸到儲值那一類（看 `SubtitleRunFailureReason` 的測試表）。

## 不要做的事

- 不加下載假資料（下載在 qBittorrent 裡，不在 DB）。
- 不加想要清單（Epic 13）資料——founder-pm 判定新用戶第一週碰不到，這個月不測。
- 不動任何正式程式的行為；這張單只改 `cmd/seed`、`serve-test-env.sh`、（必要時）一個 repository 方法。

## Tasks

- [ ] 讀上面三個「先確認」，把實際欄位名寫進 Dev Notes
- [ ] 電影字幕狀態分布（AC 1）
- [ ] 影集各集字幕軌（AC 2）
- [ ] 本月 subtitle_runs 五種（AC 3）
- [ ] ai_side_spend parse 一筆（AC 4）
- [ ] `--with-encryption-key`（AC 5）
- [ ] 舊案例不受影響，必要時同步改測試計畫（AC 6）
- [ ] seed 測試（AC 8）
- [ ] :8090 實際開起來，手動看一次媒體庫、季清單、活動、首頁

## Dev Notes

- 檔案：`apps/api/cmd/seed/main.go`、`scripts/serve-test-env.sh`
- 相關規則：`models.ChineseSubtitleVerdict`、`models.SubtitleTriageRank`、`models.SubtitleRunFailureReason`
- migration 044（`episodes.subtitle_tracks`）、047（`ai_side_spend`）

## Dev Agent Record

### Agent Model Used

### Completion Notes List

### File List
