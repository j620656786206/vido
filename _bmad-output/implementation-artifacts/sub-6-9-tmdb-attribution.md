# Story 6.9: TMDb attribution —— logo + 未經認可聲明（前端，合規）

Status: done（Alexyu 2026-09-06 裁定直接改 done，未走換模型 CR）— 原 review 註記： — **七條 AC 全數滿足**。程式／測試／文件隨 PR #385 合併；AC #1 的 logo 檔與 AC #5 延後的 gallery fixture 於同日隨 PR #386 補齊（`backlog-tmdb-logo-asset` 已 CLOSED）

## Story

As the operator of a public self-hosted app,
I want Vido to display the TMDB logo and the required non-endorsement notice wherever TMDB data is used,
so that we comply with the TMDB API Terms of Use §3 before the app reaches anyone outside this house.

## Context — 這個 story 為什麼存在

party-mode 2026-09-03 查證：TMDB 條款第 3 條要求 logo 與「This application uses TMDB and the TMDB APIs but is not endorsed, certified, or otherwise approved by TMDB.」JustWatch 的 attribution 已做（`StreamingAvailability.tsx:179`，註解標「mandatory licensing requirement」），**TMDb 的完全沒有**。內嵌預設 TMDb key（P1-7）與任何對外測試都以此為前提。

## Acceptance Criteria

1. **設定頁「資料來源」段。** 在設定 shell（Screen C4-D `6UCtX`）的 API 金鑰區塊（`ApiKeysForm.tsx` TMDB 列下方）新增 attribution：TMDB 官方 logo（SVG，從 TMDB brand 頁下載的「TMDB — The Movie Database」primary short logo，放 `apps/web/public/images/tmdb-logo.svg`，檔頭註明來源 URL 與下載日期）＋兩行文案：英文原句（條款要求原文）＋ zh-TW 說明「本應用程式使用 TMDB 與 TMDB API，但未經 TMDB 認可、認證或核准。」logo 連到 `https://www.themoviedb.org/`（`rel="noopener noreferrer"`）。

2. **詳情頁頁尾。** 電影／影集詳情頁（TMDB 資料的主要消費面）在 `StreamingAvailability` 的 JustWatch attribution 同區塊下方加一行小字「資料來源：TMDB」＋小 logo（沿用同一 className 與 `data-testid="tmdb-attribution"`）。

3. **設計。** 兩處都 ride 既有 frame：設定頁循 `ApiKeysForm.tsx:1` 的 `// Design ref: … no current screen frame` 先例；詳情頁 attribution 列循 JustWatch 既有樣式。**不改 `.pen`**（純文字＋小 logo 行，Sally 在 party-mode 已認可「合規行不佔設計位」）；若 Sally 要求入 `.pen`，改走 `CLAUDE.md` 截圖流程。

4. **主題與 a11y。** logo 在 light/dark 都可讀（TMDB 提供的藍綠漸層版兩者皆可；若對比不足用單色版）；`alt="TMDB"`；連結有可見焦點。

5. **測試。** `ApiKeysForm.spec.tsx`／`StreamingAvailability.spec.tsx` 加：logo `img` 存在、alt、英文原句逐字、連結 href；visual gallery fixture 更新（`-darwin` 本機、`-linux` 等 CI）。

## Tasks / Subtasks

- [x] **Task 1 — 資產與設定頁（AC: #1, #3, #4）** — `TmdbAttribution` 元件 + `ApiKeysForm` TMDB 列下方；logo 檔由 Alexyu 於 2026-09-05 提供並落地（PR #386）
- [x] **Task 2 — 詳情頁列（AC: #2）** — `TMDbDetailV2` 與 `LocalDetailV2` 頁尾（⚠️ 位置偏離 AC 字面，理由見下）
- [x] **Task 3 — 測試與 fixtures（AC: #5）** — 11 個新測試；visual gallery fixture 於 logo 落地時補上（`ui/TmdbAttribution` 兩個 variant，PR #386）

## Dev Notes

- 英文原句**不得改寫**（條款原文）。
- `README.md` + `README.zh-TW.md`（Rule 17）若有「資料來源」段一併補 TMDB 一行。
- 純前端 ≤3 task。

### Time-dependent visual coverage

- N/A — no wall-clock-reading components touched。

### References

- TMDB API Terms of Use §3 — https://www.themoviedb.org/api-terms-of-use
- eval-1「後續 Backlog」P0-9；`apps/web/src/components/media/StreamingAvailability.tsx:176-192`、`apps/web/src/components/settings/ApiKeysForm.tsx`

## Dev Agent Record

### Agent Model Used

Claude Code on the web（2026-09-05）

### Completion Notes List

**✅ AC #1 的 logo 檔：已於 2026-09-05 稍晚補齊（PR #386）**

Alexyu 從 <https://www.themoviedb.org/about/logos-attribution> 下載並提供 SVG，已進 repo，
檔頭帶來源 URL 與下載日期，資產閘門由「檔案不在就跳過」轉為**真的驗證 provenance 並通過**。
AC #5 延後的 visual gallery fixture 一併補上（延後理由「唯一的視覺變數還沒進 repo」已消失）。
⚖️ **變體選擇**：Alexyu 提供了官方的 long（13.8:1）／primary short（堆疊兩行，2.33:1）／
alt short（單行橫式，7.7:1）三種。採 **alt short**，非 AC 字面的 primary short。理由是尺寸：
本 story 的兩個位置都是**單行文字脈絡**（金鑰欄位下的說明段、詳情頁頁尾的「資料來源：」），
不是品牌標頭；堆疊版在 16px 高度下每行字只剩約 7px 會糊掉，單行版的字高等於整體高度。
設定頁 h-5（20px × 154px）、詳情頁 h-4（16px × 123px）。要換變體只需換檔 + 調一個 height class。

以下為當時裁定「後補」的記錄：

**🔴 原：AC #1 的 logo 檔 ⚖️ 裁定後補（Alexyu 2026-09-05：「logo 我會之後再補」）**

carry-forward 至 `backlog-tmdb-logo-asset`（Rule 24 lane ③，雙向）。該條目帶一條硬約束：
**BLOCKS sub-7-7**（內嵌預設 TMDb key）——用我們的名義大量呼叫 TMDb 的 API 卻沒放標記，
會把這條殘留從「未完成」升級成「風險」。

`tmdb-logo-asset.spec.ts` 隨之從「檔案必須存在」改為「**有檔就必須帶 provenance**」：
沒有關掉任何測試，斷言本身仍是真的（後補時若沒有來源 URL 與下載日期就會紅——一個沒有
provenance 的檔案，和某人用設計工具匯出的仿冒品無法區分）。「存不存在」那半交給 backlog
條目，因為一支長期紅燈、大家學會忽略的測試保護不了任何東西。

以下為原始交付記錄：

**官方 logo 檔為何不在 commit 裡**

`apps/web/public/images/tmdb-logo.svg` **沒有進 commit**：這個沙盒的網路政策把
`themoviedb.org` 擋在外面（proxy 對 CONNECT 回 403），抓不到官方 brand 頁的 SVG。
第三方重繪／重新上色的版本（例如 simple-icons 的單色 mark）被**否決**：這支 story
的立案理由就是遵守 TMDB 的品牌條款，用非官方素材等於一邊宣稱合規一邊違反同一份條款。
Alexyu 於 2026-09-05 裁定由他手動放檔。

處理方式（三層，都不需要再改任何程式碼）：

1. `TmdbLogo` 有 `onError` fallback → 檔案不在時顯示純文字 "TMDB" 字標。畫面不會出現
   破圖，條款要求的那句話今天就在線上，連結也保有可讀名稱。這是 `ProviderLogo`
   （`StreamingAvailability.tsx`）既有的「絕不畫破圖」慣例。
2. `tmdb-logo-asset.spec.ts` 原本寫成**紅燈閘門**（檔案不存在就 fail，失敗訊息即下載步驟）。
   ⚖️ 上述裁定後改為「有檔就必須帶 provenance」，缺檔改由 `backlog-tmdb-logo-asset` 追蹤。
3. 該 spec 仍檢查檔頭有來源 URL 與下載日期（AC #1 的 provenance 要求）——後補時沒帶會紅。

放檔步驟：<https://www.themoviedb.org/about/logos-attribution> → primary short logo(SVG)
→ 存成 `apps/web/public/images/tmdb-logo.svg` → 檔頭加一行
`<!-- TMDB primary short logo — https://www.themoviedb.org/about/logos-attribution — downloaded YYYY-MM-DD -->`。

**⚠️ 三點裁量／偏離待 Alexyu 過目**

1. **詳情頁位置（AC #2）**：AC 標題寫「詳情頁**頁尾**」，內文卻寫「在 `StreamingAvailability`
   的 JustWatch attribution 同區塊下方」——兩者矛盾。採**頁尾**。理由：JustWatch 那行只在
   `StreamingAvailability` 的**有資料分支**渲染，該元件另有 loading／error／`此區域暫無串流資訊`
   三個分支；掛在它下面，等於「這部片在台灣沒有任何串流平台」就沒有 TMDB 聲明——而整頁的
   海報、標題、簡介全都是 TMDB 資料。改放兩個詳情頁容器的最後一個子元素，並以 `tmdbId > 0`
   收斂（未匹配的本機檔沒用到 TMDB 資料，宣告一個沒用到的來源是另一種謊）。代價是動到兩個檔案
   （`TMDbDetailV2`、`LocalDetailV2`）而非一個。
2. **visual gallery fixture（AC #5）未加**：`ApiKeysForm` 與 `StreamingAvailability` 本來就
   都不在 gallery（`-gallery.fixtures.tsx` 的 SCOPE NOTE：settings/* 家族在 19-4b 待辦）,
   所以沒有既有基準線需要更新。新元件本身可以入 gallery，但它唯一的視覺變數就是那個還沒進 repo
   的品牌資產——現在建基準線會把 fallback 字標烤進 `-linux` 基準（且 CLAUDE.md 規定 `-linux`
   只能由 CI bootstrap PR 產），檔案一放又要立刻重烤一次。**建議**：logo 落地那次 commit 再一起加。
3. **README 既有那句是舊版且不合規**（順手修）：`README.md` 的「第三方服務與資料來源」寫的是
   `This product uses the TMDB API but is not endorsed or certified by TMDB.`，與條款 §3 現行
   原句（`This application uses TMDB and the TMDB APIs but is not endorsed, certified, or
   otherwise approved by TMDB.`）不一致。已改為原句 + zh-TW 說明。`README.zh-TW.md` 不存在，
   Rule 17 雙語孿生在此不適用（既有欠債見 `backlog-deployment-doc-zh-tw-twin`）。

**測試**：10 個新測試（`TmdbAttribution` 5、`ApiKeysForm` 3、`LocalDetailV2` 2）+ 1 個資產閘門
（現為預期中的紅燈）。`LocalDetailV2.spec` 的兩個案例特別有意義：該檔把 `StreamingAvailability`
mock 成 `null`，所以它們證明的正是「聲明不依賴串流區塊」。全回歸：web 3195 passed / 1 failed
（= 資產閘門），lint 0 errors，prettier clean，`tsc` 於本次改動的檔案 0 error（既有 150 個
error 全在 `-gallery.fixtures.tsx` 等未觸及檔案）。

### Discovery Triage

- **既有 README 聲明用的是舊版條款字串**（見上，已於本 story 修正，不另立條目）。
- **`.pen` 未動**：依 AC #3 與 party-mode 裁定，合規行不佔設計位；本 story 未新增 frame，
  故不觸發 CLAUDE.md 的截圖重產流程。
- 🆕 **`backlog-tmdb-attribution-coverage-sweep`**（本 story 交付後自審發現）：本 story 依 party-mode
  圈定只覆蓋設定頁與兩個詳情頁，但**圖書館格狀／列表頁、搜尋結果、首頁 TV 牆**同樣整片都是 TMDB
  海報，那些路由上沒有任何聲明可及。需裁定覆蓋策略（全域頁尾 vs 現行的逐面放置）。
- 查證後**不成立**的疑慮（記錄免重查）：(a) `MediaDetailPanel`（吃 `MovieDetails|TVShowDetails`，
  是 TMDB 資料面）**沒有生產消費者** —— 只剩 gallery fixture 引用，v1 shell 已被 LocalDetailV2／
  TMDbDetailV2 取代，故非缺口；(b) 兩個詳情元件不會同時掛載（`routes/media/$type.$id.tsx:77/80`
  早退二選一），`data-testid="tmdb-attribution"` 不會在同一頁出現兩次；(c) 全 repo 已無舊版聲明
  字串殘留（唯一命中是本檔引述它的那行）。

### File List

- `apps/web/src/components/ui/TmdbAttribution.tsx`（新）
- `apps/web/src/components/ui/TmdbAttribution.spec.tsx`（新）
- `apps/web/src/components/ui/tmdb-logo-asset.spec.ts`（新，資產閘門）
- `apps/web/src/components/settings/ApiKeysForm.tsx`
- `apps/web/src/components/settings/ApiKeysForm.spec.tsx`
- `apps/web/src/components/media/TMDbDetailV2.tsx`
- `apps/web/src/components/media/LocalDetailV2.tsx`
- `apps/web/src/components/media/LocalDetailV2.spec.tsx`
- `README.md`
- `apps/web/public/images/tmdb-logo.svg`（Alexyu 提供，2026-09-05，PR #386）
- `apps/web/src/routes/test/-gallery.fixtures.tsx`（AC #5 的 gallery fixture，PR #386）
