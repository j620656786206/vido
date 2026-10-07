# Disc：官方字幕兩季譯名不同時，補哪一季的集就用那一季的寫法（後端）

Status: review

**Source:** 2026-10-07 founder-pm 判決＋party mode（John／Winston／Sally／Murat／Bob／Mary 全員同意，修正三點）。資料：See 15 集逐集對照（`eval-see-s01e02-asr-vs-official.md` 第七節）、研究報告（Netflix KNP、CAT 工具 preferred／admitted／forbidden）。

## Story

身為 Vido 的使用者，
我希望用 AI 補第 1 季缺的那一集時，人名寫法跟第 1 季前後集的官方字幕一樣；補第 2 季的就跟第 2 季一樣，
這樣看下去不會在 Vido 補的那集突然換名字。

## 事實（2026-10-07）

- See 官方 zh-TW 是兩位譯者：第 1 季一套（S01E05 例外）、第 2 季另一套；7 個主要名詞分裂。現在「各詞取多數」學到的是**混合套**。
- 專業界（Netflix KNP、memoQ／Trados）不用統計決定譯名；一詞一譯鎖定、靠人維護。沒有媒體伺服器做這件事。
- 觀眾按順序看，早就被官方換過一次；Vido 補的那集只要跟**鄰集**一致就不會製造第二次混亂。
- 7 個詞「行數計票」vs「集數計票（同票取最新季）」：

| 詞 | 行數計票 | 集數計票（15 集，實作後實跑） | 說明 |
|---|---|---|---|
| Jerlamarel | 謝拉馬威 52:22 | **傑拉馬瑞 7:6** | **翻面**（第 2 季 7 集都提到，第 1 季 6 集） |
| Baba Voss | 巴霸沃斯 44:18 | 巴霸沃斯 8 | 一樣 |
| Paris | 巴莉絲 38:21 | 巴莉絲 7 | 一樣 |
| Maghra | 瑪格拉 79:21 | 瑪格拉 8 | 一樣 |
| Kofun | 全劇層**學不到**（56:53 都不到 60%） | 可風 7 | 集數計票才有這條 |
| Tamacti Jun | 全劇層學不到 | 塔瑪克提君恩 6 | 同上 |
| Payan／Baba／Trivantians／Bow Lion／Oloman | — | 皆取第 2 季寫法 | 共 **11 個詞分裂** |

全劇層只在「該季沒有官方字幕」（例如新的第 3 季）時才用；第 3 季最可能跟第 2 季同一家字幕商，所以全劇層幾乎都落在第 2 季寫法是對的方向。另一個意外收穫：**兩季平分的詞（Kofun、Tamacti Jun）全劇層原本一條都學不到**（兩種寫法都不到 60%），分季計票後才出現。

## 設計（Winston）

- **不加欄位、不改 unique index**：官方字幕學到的詞，若各季寫法**不同**，除了全劇層 `tmdb:tv:<id>` 一條（集數計票多數、同票取最新季），再各寫一條**季子層** `tmdb:tv:<id>:s<N>`；各季寫法相同就只有全劇層一條（面板不會無故變長）。
- 翻譯側查名詞表：單集 → 先查 `scope:s<N>`，再疊全劇層（子層覆蓋同名詞）；電影／整劇 → 只查全劇層。
- 季號來自 episode row（miner 逐集跑時就知道；翻譯時 `episodeRowFor` 就有）。只套用 `tmdb:tv`；電影續集不處理。
- 季子層的寫入同樣走 `ReplaceUnconfirmedGuess`；`knownRenderings` 仍只看全劇層。
- 不做：admitted／forbidden 狀態、選擇 UI、以哪季為準的設定。

## Acceptance Criteria

1. 重學 See 後：`tmdb:tv:80752:s1` 有 Jerlamarel→謝拉馬威、`:s2` 有 Jerlamarel→傑拉馬瑞；全劇層依集數計票；各季相同的詞（Haniwa→哈妮娃）只有全劇層一條。
2. 補第 1 季一集 → 翻譯提示拿到謝拉馬威；補第 2 季一集 → 傑拉馬瑞；補沒有官方字幕的第 3 季一集 → 全劇層多數。
3. 電影與整劇查詢行為不變。
4. 真實形狀測試（Murat）：兩季各數集的夾具 → 子層與全劇層各自正確；同票取最新季；全劇層重學會重算。
5. `go test ./...`、staticcheck、lint:all 全綠。
6. **實測：** 正式機重學 See → 查 DB 三層；各挑 S1、S2 一集跑翻譯看人名。

## Tasks / Subtasks

- [x] T1 miner：逐集帶季號、分季 Mine、集數計票＋同票取最新季、子層只在分裂時寫（AC #1、#4）
- [x] T2 翻譯側查詢：單集先子層後全劇層——兩條路都接（聽寫路 `loadGlossary`、片內英文路 `LookupFor`／`feedGlossary`）（AC #2、#3）
- [x] T3 測試（AC #4、#5）
- [ ] T4 實測（AC #6）

## Dev Notes

- 匯出入／面板目前只讀全劇層，子層看不到——B 單處理面板；匯出入先記著。
- `MinePartial` 的「partial」語意不變。

## Dev Agent Record

### Agent Model Used

Claude Fable 5.1（claude-fable-5-1）

### Completion Notes List

- **Adversarial CR（2026-10-07，fresh agent）1H/2M/3L＋nits，處理如下：**
  - H1 使用者在面板確認／改過的全劇層詞會被看不見的季抽屜無聲蓋掉 → 兩條讀取路都加守門：已確認（confirmed）的詞不被抽屜覆蓋（NOCASE 比對）；「刪掉又長回來」要靠寫端清抽屜 → 記到 B 單。
  - M2 抽屜永遠不會被刪（換檔後舊抽屜仍覆蓋）→ 新 repo `DeleteSeasonDrawers(scope)`（`LIKE scope||':s%'`、只刪未確認的 official 列），miner 每次先清再寫。
  - M3 投票語意：原本一集只投「本季多數寫法」，S01E05（第 1 季用第 2 季風格）一票都不投 → 改成一集對它含有的每一種寫法各投一票，跟註解「most episodes」字面一致（兩種算法在 See 上結果相同）。
  - L4 miner 自己複製了一份 `mentionsWord`（不分大小寫、所有格處理不同）→ 匯出 `mine.MentionsWord` 共用；`best==""` 時改 `continue`（避免寫出空譯名）。
  - L5 附加的 season-split 詞順序隨機 → 排序後附加。
  - L6 `local:` scope 的秀也會寫抽屜、永遠沒人讀 → 只在 `tmdb:tv:` 下寫抽屜。
  - 編譯期斷言：`*repository.GlossaryRepository` 實作 miner 的 `GlossaryRepo`、`*glossaryStoreRepository` 實作 `GlossaryEpisodeLookup`、`*GlossaryScopeResolver` 實作 `GlossarySeasonResolver`——未來包一層會直接編譯失敗。
  - 記到 B 單（CR #7）：TMDb 種的「已知」詞（Maghra→瑪格拉）在第 1 季寫法不同時不會有 s1 抽屜——「種子 vs 官方字幕誰大」是產品題。
- 本機 15 集完整重現：11 個詞分裂、全劇層集數計票如上表；單元測試覆蓋 miner（多數／平手取最新季／單季無子層）、resolver `ResolveSeason`、store `LookupFor`、pipeline `feedGlossary`、`loadGlossary` 疊層。
- 🔗 AC Drift: FOUND — sub-7-5a「一詞一條」→ 分裂時多季子層；story 表格的 6:5 是 12 集時的數字，15 集實跑為 7:6（翻面），已更新。
- 📎 Contract Stamps: NONE（`MineResult` 多 `terms_split`，additive）。
- 🎭 A11y Pre-Flight: N/A (100% backend)
- 🎨 UX Verification: SKIPPED — no UI changes（B 單）

### File List

- `apps/api/internal/models/glossary.go`、`apps/api/internal/services/glossary_scope_resolver.go`、`transcription_service.go`、`apps/api/internal/subtitle/glossary_store.go`、`process_item.go`、`apps/api/internal/subtitle/miner/miner.go`（改）＋對應測試
- `_bmad-output/implementation-artifacts/disc-2026-10-glossary-season-scope-a.md`、`sprint-status.yaml`（改）

## Change Log

| 日期 | 內容 |
|---|---|
| 2026-10-07 | CR 1H/2M/3L 修掉：已確認詞不被抽屜蓋、先清再寫、投票語意、共用字詞比對、只在 tmdb:tv 下寫抽屜。 |
| 2026-10-07 | create-story（party mode 後）＋dev 同日；全綠，狀態 review，待正式機重學驗證。 |

