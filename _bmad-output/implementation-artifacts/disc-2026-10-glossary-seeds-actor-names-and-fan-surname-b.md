# disc-2026-10-glossary-seeds-actor-names-and-fan-surname-b：華語劇的角色中文名，TMDb 沒有就從中文維基補

Status: review

<!-- SM Bob create-story 2026-10-27；行號為 main `b7d10094`（-a #928 之後）。傘狀條目 disc-2026-10-glossary-seeds-actor-names-and-fan-surname 的第二張；Alexyu 2026-10-27 選 A：TMDb 沒有中文角色名時改從中文維基補，豆瓣、Google 不做。 -->

**Source:** 《百花殺》在 TMDb 的角色名只有拼音（zh-TW、zh-CN 都是 `Shen Xihe`、`Xiao Huayong`…，10-27 實查），名詞對照表因此一條角色名都沒有；-a 之後連演員名也不收了，這部劇的對照表會是空的。中文維基的「演員陣容」表有「孟子義 → 沈汐和」「何與 → 蕭華雍」。

## 為什麼值得做

中文發音、沒有中文字幕的劇（《百花殺》E01 抽 5 張畫面確認沒有燒進去的字幕）會走聽寫；對照表的中文名會當提示詞送給聽寫服務，角色名寫對，同音字（沈汐和／沈曦禾）比較不會錯。英文字幕翻中文時，拼音角色名也會固定翻成同一組字。

## Acceptance Criteria

1. **什麼時候查**：TMDb 原始語言是中文（`zh`／`cn`），而且有角色的中文名是空的或只有拼音（沒有漢字）時，才去中文維基查。其他情況不發任何請求。
2. **查哪一頁**：依序試「片名 (首播年年電視劇)」→「片名 (電視劇)」→「片名」，再用原文片名試一輪（電影用「電影」），最多 4 頁；跟著重新導向與繁簡標題（`百花杀` 找得到 `百花殺`、`將夜 (電視劇)` 導到 `将夜 (网络剧)`）。
3. **怎麼對**：從頁面的維基表格找「演員」「角色」兩欄（表頭可以是 `!` 也可以是粗體的一般列），也認「演員 飾 角色」的清單寫法；rowspan（一人兩角）會展開。用 TMDb 的演員中文名對維基的演員欄（兩邊都先轉成台灣繁體、去掉空白與間隔號再比）。一人多角（「步疏林／步疏杳」）取第一個。
4. **不是這部劇的頁就不用**：一頁至少要對到 2 個演員（需要補的演員只有 1 個時對到 1 個）才採用；不夠就恢復原狀、試下一頁。
5. **出錯不影響種子**：維基連不上（不是「頁面不存在」）就停止查、角色名維持 TMDb 的；種子照常完成。
6. **只補空的**：TMDb 已經有中文的角色名不動；補上的名字之後照常走簡轉繁與 -a 的姓氏修正。
7. **不受 `ENABLE_WIKIPEDIA` 影響**：那個開關管的是「用維基比對影片」（預設關），這裡只查演員表；client 一秒一個請求。
8. **測試**：表格兩種表頭、rowspan、清單寫法、沒有演員表、繁簡變體標記；client 跟重新導向／變體標題、略過不存在的頁；種子：補上中文角色名（簡體演員名對得到繁體）、不是這部劇的頁不用、非中文片不查、TMDb 已有中文不查、沒接維基不查、維基斷線照舊。

## 不要做的事

- 不接豆瓣、Google。
- 不改 `ENABLE_WIKIPEDIA` 與維基比對影片的行為。
- 不補演員名（-a 的規則：戲劇不收演員名）。

## Tasks

- [x] Task 1 — `wikipedia.Client.GetWikitext`（redirects＋converttitles）、`wikipedia.ParseCast`（AC 2、3）
- [x] Task 2 — `GlossarySeeder`：TMDb 詳細資料一次讀出類型／原始語言／片名；`fillCharactersFromWiki`；配對時記下每一條的演員中文名（AC 1、4、5、6）
- [x] Task 3 — `main.go` 接上 `SetWikiCastSource`（AC 7）
- [x] Task 4 — 測試、vet、staticcheck；真實 TMDb＋維基試跑（AC 8）

## Dev Notes

- 種子：`services/glossary_seeder.go`（`FetchCredits` → `refinePairs` → `fillCharactersFromWiki` → `keepActorsOnlyWhenUnscripted`）、新檔 `services/glossary_wiki_characters.go`。
- 維基：`wikipedia/client.go`（加 `baseURL` 欄位給測試）、新檔 `wikipedia/cast.go`。
- TMDb 詳細資料走 `tmdb.CacheService` 的 `GetTVShowDetails`／`GetMovieDetails`（有快取）。

### Time-dependent visual coverage

- N/A — 只有後端。

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（claude-opus-5-5），2026-10-27。

### Completion Notes List

- `wikipedia.ParseCast`：表格（`!` 表頭或粗體一般列表頭、rowspan 展開、儲存格屬性／連結／ref／模板／`-{}-` 變體標記清掉、尾端括號去掉）＋「飾／饰演」清單。用中文維基《百花殺》全文實跑：40 條演員→角色全部正確（`王會藝 → 余-{}-五娘子` 的標記也清掉了）。
- `GetWikitext(titles...)`：`redirects=1&converttitles=1&formatversion=2`，回傳第一個存在的頁與實際頁名；`IsNotFound`。
- 種子：`titleFacts` 一次讀出類型（-a）、原始語言、片名候選；`pairMovieCredits`／`pairTVCredits` 多回傳每條的演員中文名；`fillCharactersFromWiki` 依 AC 1～6；`refinePairs` 先補角色、再丟演員（-a）。
- 反向驗證：拿掉補角色那行 → 2 個種子測試紅。
- **真實 TMDb＋中文維基試跑（2026-10-27，本機，正式機的 TMDb 金鑰，OpenCC CLI）**：
  - 《百花殺》286506：頁面「百花殺」，10 個角色補上 8 個（沈汐和、蕭華雍、蕭長卿、蕭長旻、蕭長贏、崔晉百、步疏林、珍珠）；另 2 個演員不在維基表裡，維持拼音（之後被種子過濾）。
  - 《將夜》2018 版 83612：「將夜 (電視劇)」導到「将夜 (网络剧)」（表頭是粗體一般列），補上 5 個（桑桑、宁缺、李渔、李仲易、夏侯；簡體會在寫入時轉繁）。
  - 《將夜》2026 動畫 282136：TMDb 已有中文角色名 → 不查。
- `go test ./internal/services/ ./internal/wikipedia/ ./internal/metadata/ ./cmd/...` 全過；`go vet`、`staticcheck` 乾淨。

### File List

- `apps/api/internal/wikipedia/cast.go`（新）、`cast_test.go`（新）、`client.go`
- `apps/api/internal/services/glossary_wiki_characters.go`（新）、`glossary_wiki_characters_test.go`（新）、`glossary_seeder.go`
- `apps/api/cmd/api/main.go`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`、`disc-2026-10-glossary-seeds-actor-names-and-fan-surname-a.md`（標 done）
