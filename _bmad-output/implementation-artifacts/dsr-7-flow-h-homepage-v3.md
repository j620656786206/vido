# Story DSR.7: Flow H 首頁 v3——程式碼與設計稿雙向對齊 H1–H8

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As someone who opens Vido's homepage more often than any other page,
I want 首頁上同一個詞只穿一種顏色、金額照它自己的規則顯示、設計稿畫的就是我真的會看到的東西,
so that 「整理中」不會在同一排裡一個赭一個金，而稿也不再是別人憑印象畫的另一個首頁。

## Context

`epic-dsr` 的第八張（dsr-4／4b 之外已收 5、9、10、11、12、13）。**剩下的 flow 裡最小的一塊**——`components/homepage/` 9 個元件檔 3,812 行，9 張設計稿。

⚠️ **方向不是單向的，而且這次「碼對稿錯」佔多數。** 首頁 v3 的程式碼是 2026-08-26 從 `home-v3-identity-brief.md` 一路做到 `ux3-1-8` 出貨的，寫得比稿新、比稿嚴謹（scrim pill、`formatUsdShort` 折疊、fail-soft 降級文案都在程式碼裡而稿上沒有）。所以下面每一條都標了方向：`碼→稿`（程式碼要改）或 `稿→碼`（設計稿要改）。**dev 不要無腦把程式碼改成跟稿一樣。**

⚠️ **驗收基準是 `.pen` 節點值，不是 `_bmad-output/screenshots/flow-h-*/` 的 PNG。** Flow H 不在 `READABLE_FLOWS` 裡，那 9 張現在是 400px 縮圖，讀不到字。Task 1 先修這件事。

✅ **視覺基準風險低。** `HomeReadoutBand` / `RecentlyAddedRowV2` / `HomeBrowseV2` **沒有任何 gallery 夾具**（既有基準只有 `homepage-explore-block` / `-skeleton` / `-blocks-list` / `-hero-banner` / `-trailer-modal` 五個），而本 story 要改的兩個元件都不在其中。預期 `-linux` 基準線變動 **0 張**——若 CI 開了 bootstrap PR，那代表改到了預期外的東西，回頭看。

### 設計稿節點（逐字抄，不要重查）

| 代號 | 節點 | 內容 |
| --- | --- | --- |
| `H1-D-v3` | `k2Otv` | 首頁 v3 桌機全景 |
| `H2-M-v3` | `uGCAU` | 首頁 v3 手機 |
| `H4-D-v3` | `B7UO8` | 載入骨架 |
| `H5-D-v3` | `RvS6c` | 空片庫・首跑 |
| `H6-D-v3` | `zRyNS` | 自家內容載入失敗 |
| `H7-D-v3` | `EoCQ4` | TMDb 降級態 |
| `H8-SPEC-v3` | `iWUSV` | 讀數帶金額顯示規則 |
| `H5-D` | `Y5XvRv` | ExploreBlock 優化（bugfix-10-6 歷史規格稿） |
| `H3` | `Paqlk` | Block 編輯 Modal — ⛔ **不在本 story 範圍**（見「不要做的事」） |

### 對照表

| # | 項目 | 設計稿 | 程式碼 | 方向 |
| --- | --- | --- | --- | --- |
| 1 | 「整理中」藥丸顏色 | `$warning-tint` / `$warning-text`（赭） | `--warning-tint` / `--warning-text`（赭） | **兩邊都錯 → 都改金** |
| 2 | 同一排海報卡的「整理中」徽章 | — | `TINT.accent`（泥金，dsr-11 裁定） | ✅ 正確，不動 |
| 3 | Hero 漸層資訊層底色 | 寫死 `#0c1512 → #0c151200`（不隨主題翻轉） | `from-[var(--bg-primary)] via-.../70 to-transparent` | **稿→碼** |
| 4 | Hero「最新入庫」與年份列的底 | 沒有 scrim pill，字用會翻轉的 token | `bg-[var(--overlay-scrim)]` + `text-[var(--text-on-scrim)]` | **稿→碼** |
| 5 | H5-D 三個 ExploreBlock scrim | 寫死 `#0c1512` | `from-[var(--bg-primary)] to-transparent` | **稿→碼** |
| 6 | 金額小數位 | `$1.20/$5.00` | `formatUsdShort` → `$1.2/$5` | **稿→碼** ⚠️稿違反自己的 spec |
| 7 | 「需要注意」赭色戴在哪 | 標籤赭、數字中性 | 標籤中性、數字赭 | **稿→碼** |
| 8 | 手機第三格 | H2-M 一行且沒有分母；H8-SPEC 說兩行 | 一行、有分母 | ⚖️**已裁定：兩行，兩邊都改** |
| 9 | 手機版 TMDb 探索區 | H2-M 沒畫 | 不分斷點都渲染 | **稿→碼** |
| 10 | 手機版 hero 的「查看詳情」與切換圓點 | H2-M 沒畫 | 沒有任何斷點隱藏 | **稿→碼** |
| 11 | 區塊失敗文案 | `無法載入，請稍後再試` | 兩行具名橫幅＋`重試` | **稿→碼** |
| 12 | TMDb 降級文案 | `探索內容目前無法載入（…）` | `探索區塊目前無法載入（…）` | **稿→碼** |
| 13 | ExploreBlock 空狀態 | 只畫 `沒有符合條件的內容` | 還有一句 `這排的作品你都已經擁有了` | **碼→稿** |
| 14 | 讀數帶標籤字級 | `$Type/Label/Size` = **12** | `text-[11px]`（2 處） | **碼→稿** |
| 15 | `ExploreBlocksList.tsx`／`TrailerModal.tsx` 檔頭 | — | 指向 `sAaCR`——**該節點已不存在** | **碼→稿** |
| 16 | `ExploreBlock.tsx` 檔頭 | 節點在，名字已改成 `H5-D` | 寫 `HP-5 ExploreBlock Polish` | **碼→稿** |
| 17 | `H5-D` vs `H5-D-v3` | 兩張不同的稿，代號只差 `-v3` | — | **稿改名** |
| 18 | 最近新增／探索的 12 張海報 | 全部叫「你的名字 2016 107 分」 | 真實資料各不相同 | **稿→碼** |
| 19 | 讀數帶四格標籤與數字文案 | 繁中字幕／今天處理／需要注意／進行中 | 逐字相同 | ✅ |
| 20 | 空片庫首跑：`繁中字幕 · 開始掃描`、`0/0`、`一切正常` | — | 逐字相同 | ✅ |
| 21 | 區塊順序 讀數帶→Hero→最近新增→探索 | — | `HomeBrowseV2.tsx:42/48/61/68` 同序 | ✅ |
| 22 | 空片庫時 hero 不渲染（例外訊號原則） | 註記寫明 | `HeroBanner` 無 backdrop 即 absent | ✅ |
| 23 | `--accent-text` 而非稿上的 `--accent-primary` | 稿用 `$accent-text` | 程式碼註解已寫明理由（4.40:1 被裁出 text 角色） | ✅ **不要改** |

---

## Acceptance Criteria

1. **Flow H 的稿要讀得到。** `scripts/export-pen-screenshots.py` 的 `READABLE_FLOWS` 加入 `"flow-h-homepage-v3"` 與 `"flow-h-homepage"`，重跑匯出，9 張從 400px 長邊變成 2x／最小寬 1400。**只 stage Flow H 這 9 張**，其餘 `git checkout` 丟掉（全檔重產是非決定性的，見 CLAUDE.md）。

2. **首頁不准讓「整理中」同時穿兩個顏色。** `RecentlyAddedRowV2.tsx:149` 的藥丸是 `bg-[var(--warning-tint)] text-[var(--warning-text)]`（赭），但**同一排下面每一張海報卡**的「整理中」徽章走 `pickPosterBadge → deriveLifecycleStatus`，dsr-11（⚖️ Alexyu 2026-09-11）已經把它改成 `TINT.accent`（泥金）。改藥丸為 `bg-[var(--accent-tint)] text-[var(--accent-text)]`，與 `utils/libraryStatus.ts:64` 的 `TINT.accent` 逐字一致。
   - **同時改掉 `RecentlyAddedRowV2.tsx:134-138` 的註解**——它現在寫著「the same items' poster badges wear amber 整理中, and one screen may not dress one truth in two colours」，那個前提在 dsr-11 之後是假的，而它正是造成這次矛盾的理由。新註解要引 dsr-11 的裁定。
   - **同時改掉 `apps/web/src/styles.css:76` 的註解** `/* warning badge bg, 整理中 pill */`——`--warning-tint` 不再服務整理中。
   - ⛔ **不要動 `PosterCardV2` 或 `utils/libraryStatus.ts`**——它們是對的，而且 `PosterCardV2` 在 dsr-1 的範圍裡（同檔衝突）。
   - 加一條 spec 測試守住藥丸用 accent 而非 warning。

3. **三張 hero 的漸層是寫死的夜行色，改成會翻轉的變數。** `vKtFI`（H1-D-v3）／`PAxHw`（H7-D-v3）／`WhSG1`（H2-M-v3）的 `漸層資訊層` fill 是寫死的 `#0c1512 → #0c151200`。`#0c1512` 是 `--bg-primary` 在**夜行模式的字面值**——日巡時這道漸層不會跟著變淺，但壓在上面的 `$text-primary`／`$text-secondary`／`$text-muted` 會翻成深墨綠，等於黑字壓黑底。改成 `$bg-primary → transparent`，對上 `HeroBanner.tsx:165`。
   📌 這是 P0 單子 `disc-2026-09-text-on-artwork-flips-with-theme` 點名的證據畫面（「`H1-D-v3 — 日巡` 首圖標題『沙丘：第二部』讀不到」）。**本 story 不需要等那張單子的 (a)/(b) token 裁定**——程式碼已經示範了第三條路（見 AC #4），Flow H 照抄即可。結案後在該條目補一行說明 Flow H 已收。

4. **Hero 的「最新入庫」與年份列要坐在 scrim pill 上。** `HeroBanner.tsx:189` 與 `:208` 把這兩塊包在 `bg-[var(--overlay-scrim)]` + `text-[var(--text-on-scrim)]` 的圓角 pill 裡（兩個 token 在兩個主題下都是固定值，不翻轉）；設計稿是裸字直接壓在漸層上。三張 hero 的 `最新入庫標籤`（`XTc4A`／`d28I7v`／`d0Woug`）與 `狀態列`（`xxEXp` 及對應節點）補上 pill 底，字改 `$text-on-scrim`。
   - `片名` 維持 `$text-primary`（程式碼 `HeroBanner.tsx:196` 就是這樣，漸層底下最厚的地方讀得到）。
   - `字幕徽章` 維持 `$success-tint`/`$success-text`（程式碼走 `badge.className`，同一套）。

5. **H5-D 的三個 scrim 也是寫死的。** `uO36W`（aScrim）／`tLOP3`（leftScrim）／`rQ6T2`（rightScrim）同樣寫死 `#0c1512`，改成 `$bg-primary`，對上 `ExploreBlock.tsx:184/189` 的 `from-[var(--bg-primary)] to-transparent`。改完 Flow H 的寫死色只剩畫布標籤（`b0X61`／`r8FsQ2`／`OivWL`／`n8bIY`，那是畫布 chrome 不是畫面內容）與 `HPpRF` 的 `#00000000`（全透明）。

6. **金額要照 H8-SPEC 自己寫的規則顯示。** 稿上五處寫 `$1.20/$5.00`，但同一張 H8-SPEC 的規則白紙黑字寫「小於 $10 顯示一位小數（$1.2）」。`formatUsdShort.ts:24` 還會 `trimTrailingZero`，所以 `5.0` 出來是 `$5` 不是 `$5.0`。改成 `$1.2/$5`：
   - `XqtJj`（H1-D-v3）、`rK3eF`（H7）、`ovxT4`（H6）、`ODERd`（H8「現況」）→ `2 部失敗 · $1.2/$5`
   - `sJ87f`（H2-M）→ 見 AC #7
   - H8 的另外兩組（`$123/$500`、`$1.2k/$5k`）已經對，不動。

7. **「需要注意」的赭色戴錯位置。** H7 的 `SGhuq` 與 H6 的 `ac05t` 把 `$warning-text` 放在**標籤**上、數字留 `$text-primary`；程式碼 `HomeReadoutBand.tsx:130/148` 是反過來的——標籤永遠 `--text-muted`，**數字**在 `exception` 時才轉 `--warning-text`。程式碼是對的（赭色標記的是那個異常的讀數，不是那個格子的名字）。兩處稿改成：標籤 `$text-muted`、數字 `$warning-text`。

8. **⚖️ 手機第三格改成兩行（Alexyu 2026-09-16 裁定：照規格稿做）。**
   - H8-SPEC 的規則：「手機版（2×2 格）改為兩行：第一行『N 部失敗』，第二行金額『$12/$50』。」
   - H2-M（`sJ87f`）畫的是一行 `2 部失敗 · $1.20`——**連分母都沒有**，跟 spec 與程式碼都不一樣。
   - 程式碼（`attentionText`）不分斷點都回一個字串 `2 部失敗 · $1.2/$5`。
   - ⚖️ **裁定：`H8-SPEC-v3` 勝出，兩行。** 理由（Alexyu）：金額是這個產品護城河的第一層（花費上限＋同意流程），不該在最小的螢幕上被擠掉；`2 部失敗 · $9.9k/$99k` 這種最壞情況在 390px 的 2×2 格裡一行放不下。
   - **碼**：`HomeReadoutBand` 在 `md` 以下把失敗數與金額拆成兩行。`attentionText` 的**回傳型別不要改成陣列**——改成 `{ failures, spend, text, exception }`（`text` 維持原本的單行合併字串給桌機與既有斷言用），新增的兩個欄位只有手機分支讀。這樣既有 spec 對 `text` 的斷言一行都不用動。
   - **稿**：`sJ87f`（H2-M 需要注意格）改成兩行 —— 第一行 `2 部失敗`（`$warning-text`）、第二行 `$1.2/$5`（`$warning-text`，等寬）。**分母不准再省略**（現況畫成 `$1.20` 少了 `/$5.00`，那是第三種版本，兩邊都不是）。
   - **驗收**：390px 下兩行不溢出、不截斷；`md` 以上維持一行。加一條 spec 斷言守住兩個斷點各自的形狀。

9. **H2-M 缺兩塊東西。**
   - **缺整個 TMDb 探索區。** `uGCAU` 的 `內容捲動區` 只有 讀數帶／Hero／最近新增；`HomeBrowseV2.tsx:68` 的 `<ExploreBlocksList />` 沒有任何斷點隱藏，手機一定會渲染。照 H1-D-v3 的 `y8Zgsw` 補一個（標題「熱門電影」＋過濾說明＋橫向捲動卡片列）。
   - **hero 缺「查看詳情」與手動切換圓點。** `HeroBanner.tsx:262` 的 CTA 與 `:361` 的圓點 pill 都沒有 `hidden`／`md:` 之類的斷點隱藏（全檔唯一的斷點是 `:332` 的 `h-[250px] md:h-[400px]`）。`iQhCJ` 底下補上，形狀照 `EwfRT`／`yNRep`。
   📌 這是 dsr-10「手機稿缺活動記錄區塊」同一類錯誤的再犯。

10. **H6 的區塊失敗文案照程式碼重畫。** 稿只有一句 `無法載入，請稍後再試`；程式碼（`RecentlyAddedRowV2.tsx:179-195`）是三件事：具名標題 `最近新增目前無法載入`（`$error-text` ＋ AlertCircle 圖示）、說明 `這個區塊的資料暫時無法取得，其他首頁內容仍可使用。`（`$text-secondary`）、`重試` 按鈕（`$error-text`）。程式碼版本較誠實——它說出是**哪一區**壞了，也說明其餘照常，那正是 F3 fail-soft 想讓人知道的事。`CJvcN` 與 `aWgOu` 依此重畫。
    📌 Flow K 的 `K4-D-v2` 用的是短句版 `無法載入，請稍後再試`。**本 story 不統一跨 flow 的錯誤文案**——那是文案正典裁定，已立案（見 Discovery Triage）。

11. **H7 的降級文案一字之差。** 稿寫 `探索內容目前無法載入（TMDb 未設定或無法連線）`，程式碼 `ExploreBlocksList.tsx:154` 是 `探索區塊目前無法載入（TMDb 未設定或無法連線）`。`UMdF2` 改成「探索**區塊**」。`lGr26` 的 `前往連線設定` 已逐字相符，不動。
    - 程式碼還有第二種降級文案（`${count} 個探索區塊的內容目前無法載入`，reason='content'），稿上完全沒有。在 H7 的註記文字裡補一行說明這兩種，不另畫畫面。

12. **ExploreBlock 空狀態少畫一句。** `ExploreBlock.tsx:265` 是三元：`allItems.length > 0 ? '這排的作品你都已經擁有了' : '沒有符合條件的內容'`。H5-D 的 `XWLNl` 只畫了後者。在同一區補上前者的說明（「這排全部已擁有」是首頁真的會出現的狀態——H1-D-v3 的 `jCmVZ` 自己就寫著「已擁有的作品不會出現在這裡」）。

13. ~~**兩處 `text-[11px]` 收斂到字階。**~~ ⚖️ **已撤銷（Alexyu 2026-09-16）** —— 見「對抗式 Code Review」第 13 項。標籤維持 11px。原文： `Type/Label/Size` = **12**，11 不在字階上。`HomeReadoutBand.tsx:130`（格子標籤）與 `:236`（骨架標籤）**必須一起改**成 `text-xs`——`:236` 的註解寫明骨架是「borrowing the type means the skeleton measures itself」，只改一邊骨架高度就會跟真內容錯開。改完 `grep -c 'text-\[1[0-9]px\]' apps/web/src/components/homepage/*.tsx` 全為 0，並照 `LoginForm.spec.tsx` 的 `puts every label on the type scale` 先例加一條守門測試。

14. **三個檔頭修正（其中兩個指向已刪除的節點）。**
    - `ExploreBlocksList.tsx` 現在寫 `Screen HP-1 Homepage Desktop (sAaCR)`。**`sAaCR` 已不存在**（Pencil MCP `Get("sAaCR")` → `Can't find node`，它是 2026-08-26 v3 切換時刪掉的 v2 首頁）。改成 `// Design ref: ux-design.pen Screen H1-D-v3 (k2Otv) · H7-D-v3 (EoCQ4)`。
    - `TrailerModal.tsx` 同樣寫 `(sAaCR)`。這個元件在 Home v3 **完全沒有掛載點**（見 Discovery Triage），改成缺口變體：`// Design ref: ux-design.pen — no current screen frame; {理由}`。
    - `ExploreBlock.tsx` 寫 `Screen HP-5 ExploreBlock Polish (Y5XvRv)`——節點還在，但它現在叫 `H5-D · ExploreBlock 優化（桌面）`。改成 AC #15 定案後的代號。
    - 其餘 6 個元件檔的標頭已驗證正確，**不動**。
    🚨 標頭那一行必須以 `(節點ID)` 結尾、說明另起一行——`local/implements-pen-node-id` 的 `DESIGN_REF_RE` 要求如此，2026-09-11 有 8 個檔案踩過。

15. **拆掉 `H5-D` / `H5-D-v3` 的代號衝突。** 兩張完全不同的稿（ExploreBlock 歷史規格 vs 空片庫首跑）代號只差 `-v3`，截圖檔名是 `h5-d.png` 與 `h5-d-v3.png`。把 `Y5XvRv` 更名為 `H9-SPEC · ExploreBlock 優化（bugfix-10-6）`，同步：`export-pen-screenshots.py` 的 `SCREENS` 值改 `("flow-h-homepage", "h9-spec")`、刪掉舊的 `h5-d.png`、`CLAUDE.md` 的 flow-h 說明、AC #14 的 `ExploreBlock.tsx` 標頭。稿上的標題文字 `HP-5 · ExploreBlock Polish — bugfix-10-6`（`Q1WDtP`）一併改成新代號。
    📌 順手把 `ro1S5`／`TwaV0` 兩處已經過期的行號引用（`ExploreBlock.tsx ~L101-120` / `~L154-161`，實際在 `~L184-210` / `~L93-98`）改成不帶行號的說法——規格稿不該綁行號。

16. **12 張海報不准全叫同一部片。** H1-D-v3 的最近新增 6 張（`e6cF9`/`SLfzh`/`fph4S`/`OPppD`/`G0u1Z`/`oawTn`）＋探索 6 張（`dhB94`/`FnGdU`/`dnBDb`/`IQU0x`/`a1KOtQ`/`SD76q`）、H2-M 的 3 張（`sP70k`/`KZSRp`/`FqY46`），目前**全部**是「你的名字 · 2016 · 107 分 · 繁中」。換成各不相同的真片名／年份／片長，徽章依 `pickPosterBadge` 的優先序至少出現兩種（例如 `繁中`、`整理中`），讓 AC #2 的顏色改動在稿上看得見。
    📌 這是 `disc-2026-09-flow-a-m-screen-content` 已經判過的一類錯（瀏覽網格 12 張重複已修），當時沒掃到 Flow H。

17. **既有測試全數保留通過。** `HomeReadoutBand.spec.tsx`／`RecentlyAddedRowV2.spec.tsx`／`HeroBanner.spec.tsx`／`ExploreBlocksList.spec.tsx`／`ExploreBlock.spec.tsx` 現有斷言，只有 AC #2（藥丸顏色）、AC #8（手機兩行）、AC #13（字級）指名的地方可以改，其餘一律不動。特別是 fail-soft 那幾條（一區壞掉其他區照常渲染）與 `HomeBrowseV2.spec` 的 D3 順序斷言。

18. **CI 全綠。** `pnpm run lint`（0 errors）、`pnpm nx run web:typecheck --skip-nx-cache`、`pnpm run format:check`、`python3 scripts/check-design-tokens.py`、`pnpm nx test web`、`pnpm nx test api`。⛔ 絕不用 `run_in_background` 跑測試。

## Tasks / Subtasks

- [x] **Task 1 — 先讓稿讀得到（AC: #1）**
  - [x] `READABLE_FLOWS` 加入 `"flow-h-homepage-v3"` 與 `"flow-h-homepage"`
  - [x] `python3 scripts/export-pen-screenshots.py`（照 💡 與 dsr-10 先例，改動全做完後只跑一次）
  - [x] 只有 Flow H 的 10 個檔案有差異，**其餘 182 張零位元變動**，不需要 `git checkout` 丟棄任何東西
  - [x] 打開 `h1-d-v3.png`／`h2-m-v3.png`／`h6-d-v3.png` 逐一看過，字讀得到
  - 💡 dsr-10 的經驗：設計稿內容用 Pencil MCP 直接讀就好，**不必**先匯出再讀圖；把改腳本與改設計稿併在一起、最後只匯出一次，可省掉一輪 10 分鐘的重複匯出

- [x] **Task 2 — 程式碼：顏色與字級（AC: #2, #13）**
  - [x] 先改 spec 讓它變紅，再改 `RecentlyAddedRowV2.tsx:149` 的 `warning-*` → `accent-*`
  - [x] 重寫 `RecentlyAddedRowV2.tsx:134-138` 的註解（舊前提已被 dsr-11 推翻）
  - [x] 改 `styles.css:76` 的註解，拿掉「整理中 pill」
  - [x] `HomeReadoutBand.tsx:130` 與 `:236` 兩處 `text-[11px]` → `text-xs`（一起改）
  - [x] 加守門測試：首頁區沒有任何 `text-[\d+px]` 任意值（真內容＋骨架兩個分支都掃）
  - [x] ⛔ 不要動 `PosterCardV2.tsx`、`utils/libraryStatus.ts`（dsr-1 範圍／已正確）

- [x] **Task 3 — 程式碼：手機讀數帶兩行（AC: #8）** ⚖️ 已裁定，直接做
  - [x] `attentionText` 回傳擴充成 `{ text, exception, failures?, spend? }`，`text` 維持原字串（既有斷言零改動）
  - [x] `ReadoutCell` 新增 `valueParts`：**一份 DOM，CSS 決定軸向**（`flex-col md:flex-row` ＋ 桌機才顯示的 `·`），文字內容在任何寬度都逐字相同
  - [x] 加 3 條 spec：兩半才斷行／只有金額不斷行／只有失敗數不斷行
  - [ ] 390px 實機看一次不溢出、不截斷 — 併入 Task 8 的 UX 驗證

- [x] **Task 4 — 程式碼：三個檔頭（AC: #14）**
  - [x] `ExploreBlocksList.tsx` → `Screen H1-D-v3 (k2Otv)` ＋ companion 行 `H7-D-v3 (EoCQ4)`
  - [x] `TrailerModal.tsx` → 缺口變體 `— no current screen frame; …`（並指向 `disc-2026-09-trailer-modal-unmounted`）
  - [x] `ExploreBlock.tsx` → `Screen H9-SPEC (Y5XvRv)`（Task 7 讓 `.pen` 跟上）
  - [x] `npx eslint apps/web/src/components/homepage/` → **0 errors**（12 warnings 全為既有的 spec `any`）

- [x] **Task 5 — 設計稿：寫死色與金額（AC: #3, #4, #5, #6, #7）**
  - [x] 三張 hero 的 `漸層資訊層` → `$bg-primary → transparent`
  - [x] 三張 hero 的 `最新入庫標籤` 與 `狀態列` 補 `$overlay-scrim` pill、字改 `$text-on-scrim`
  - [x] H9-SPEC 三個 scrim → `$bg-primary`
  - [x] 五處 `$1.20/$5.00` → `$1.2/$5`（含 H8 手機樣本的第二行）
  - [x] 赭色從標籤搬到數字 —— **不只 H7／H6**：H1（`V3Bale`）、H2-M（`uWZja`）、H8 三組桌機樣本也同錯，五張稿共 11 個節點一起歸位；警告圖示一併改中性
  - [x] 掃一次剩餘寫死色：只剩 4 個畫布標籤 ＋ 1 個全透明 ＋ 6 個漸層的**透明端停止點**（`#00000000`，見 Dev Notes）
  - [x] ⚠️ 全程 Pencil MCP，未用 Read/Grep 碰 `.pen`

- [x] **Task 6 — 設計稿：缺漏與文案（AC: #9, #10, #11, #12, #16）**
  - [x] H2-M 補 TMDb 探索區（由 H2-M 自己的「最近新增」複製，改標題＋過濾說明、拿掉整理中藥丸）
  - [x] H2-M hero 補「查看詳情」 —— ⚠️ **AC 有一處寫錯：手動切換圓點本來就有**（`EpmNH`，在 depth 4，建單時漏看）。順手把三張 hero 的 CTA 高度 40 → 44（程式碼是 `min-h-[44px]`，N5 觸控底線）
  - [x] H6 錯誤區重畫成具名橫幅（標題＋說明＋重試），並拿掉稿上多畫的 1px 外框、圓角 `$radius-md` → `$radius-lg` 對齊程式碼
  - [x] H7 `UMdF2` → 「探索區塊目前無法載入（…）」，新增註記說明兩種降級文案
  - [x] H9-SPEC 空狀態：在規格說明裡寫明兩句文案與各自條件（spec 稿用文字承載比再畫一張圖準確）
  - [x] **36 張**海報卡（不只 15）換成各不相同的片 —— H1／H2-M 之外，H5／H6／H7 各 6 張完全沒覆寫、全部吃母版預設的「你的名字」，建單時沒掃到
  - [x] 溢出檢查：Flow H 只剩 5 個，全部是刻意的（兩張規格稿的示意裁切、手機橫向捲動列、手機捲動區的下半部）

- [x] **Task 7 — 設計稿：H5-D 改名（AC: #15）**
  - [x] `Y5XvRv` frame name、標題 `Q1WDtP`、畫布標籤 `OivWL` 三處同步成 `H9-SPEC`
  - [x] `ro1S5` / `TwaV0` 的過期行號拿掉；`ro1S5` 的裸 `shadow-lg` 一併改成 `shadow-[var(--shadow-lg)]`（dsr-9 之後不得用裸值）
  - [x] `export-pen-screenshots.py` 的 `SCREENS["Y5XvRv"]` → `("flow-h-homepage", "h9-spec")`
  - [x] `git rm _bmad-output/screenshots/flow-h-homepage/h5-d.png`
  - [x] `CLAUDE.md` 的 `flow-h-homepage` 說明同步

- [x] **Task 8 — 收尾驗證（AC: #17, #18）**
  - [x] `pnpm run lint` → **0 errors**（128 warnings，全是既有的 spec `any`；本 story 引入 0）
  - [x] `pnpm nx run web:typecheck --skip-nx-cache` → PASS
  - [x] `pnpm run format:check` → PASS（先修掉一個既有問題，見 Completion Notes 的 Pre-existing fix）
  - [x] `python3 scripts/check-design-tokens.py` → 五份來源一致
  - [x] `pnpm nx test web` **3401/3401** 與 `pnpm nx test api` PASS；測試後無孤兒 process
  - [x] 重跑匯出：**192 張只有 10 個檔案有差異**，全部在 Flow H
  - [x] 390px 真實瀏覽器驗證（見 Completion Notes 的 UX Verification）

## Dev Notes

### 這張跟 dsr-10 / dsr-12 的差別

dsr-12 是純「碼追稿」，dsr-10 是六稿五碼。**這張是 13 條稿要改、5 條碼要改**——比例最偏向設計稿的一次。

原因很具體：首頁 v3 的稿（`ux3-1-5`，2026-08-26）是 Pencil Inline Agent 依 Sally 的提示詞畫的，畫完之後 `ux3-1-6/7/8` 才把後端與前端做出來，而做的過程中出現了三件稿上沒有的東西——`overlay-scrim` pill、`formatUsdShort` 的金額折疊、具名的 fail-soft 降級橫幅。**稿停在「畫的時候以為會長這樣」，碼走到了「實際做出來更好的樣子」。**

**不要把程式碼改成跟稿一樣**——特別是 hero 漸層那一條。程式碼用 `var(--bg-primary)` 是對的，稿寫死 `#0c1512` 才是錯的。

### 固定詞彙

- **AC #2 是這張 story 最重要的一條。** 「同一個畫面不准把一個事實穿成兩種顏色」是 `RecentlyAddedRowV2.tsx` 自己的註解在 2026-08-26 寫下的規矩，而 dsr-11 在 2026-09-11 把海報徽章改成泥金之後，這個檔案就變成了它自己禁止的東西。藥丸與它正下方那排卡片，中間隔不到 40px。
- **赭色只有一個意思：「你要求了，但它沒發生」。** 整理中正在發生 → 泥金。AC #7 的「需要注意」數字是真的沒發生（失敗）→ 赭，戴在數字上不是戴在格子名字上。
- **`--accent-text` vs `--accent-primary` 是刻意的差異，不要「修正」。** `HomeReadoutBand.tsx` 的註解寫明：稿上是 `--accent-primary`（#c9a24b），但 styles.css 記錄它當文字量到 4.40:1、已被裁出 text 角色，所以程式碼用 AA-safe 的 `--accent-text`。這是遵守意圖，不是漂移。

### 不要做的事

- **不要動 `H3`（`Paqlk`，Block 編輯 Modal）。** 它畫的是 `components/settings/ExploreBlockEditModal.tsx`，檔案住在 `settings/`，`dsr-3-flow-c-settings` 的條目已經點名了它。跨進去只會製造同檔衝突。
- **不要動 `H5-D` 的 AC3（emoji → lucide Film/Tv）。** 同上，落點是 `ExploreBlocksSettings.tsx`，屬 dsr-3。
- **不要動 `PosterCardV2.tsx`。** 它在 `components/library/`，是 dsr-1 的範圍；而且它的「整理中」徽章已經是對的。
- **不要動 `ContinueWatchingSlot.tsx`。** 它的標頭已經是誠實的缺口變體（v3 移除了這個 slot，Epic 17 才會回來），沒有稿可以對。
- **不要刪 `TrailerModal.tsx`。** 它確實沒有掛載點，但刪不刪是產品裁定，已立案（見 Discovery Triage），本 story 只改它的標頭。
- **不要為了對稿而改 `HomeBrowseV2` 的區塊順序。** 讀數帶→Hero→最近新增→探索，稿與碼一致（`HomeBrowseV2.tsx:42/48/61/68` vs `k2Otv` 的 `BjSjk`/`F9N92Y`/`kk5yY`/`y8Zgsw`）。
- **不要統一跨 flow 的區塊失敗文案。** Flow K 用短句、Flow H 用具名兩行——哪個是正典要裁定，已立案。

### Source tree

```
apps/web/src/components/homepage/RecentlyAddedRowV2.tsx   ← Task 2（藥丸顏色＋註解）
apps/web/src/components/homepage/HomeReadoutBand.tsx      ← Task 2（字級 ×2）、Task 3（手機兩行）
apps/web/src/components/homepage/ExploreBlocksList.tsx    ← Task 4（檔頭）
apps/web/src/components/homepage/TrailerModal.tsx         ← Task 4（檔頭）
apps/web/src/components/homepage/ExploreBlock.tsx         ← Task 4（檔頭）
apps/web/src/styles.css                                   ← Task 2（僅註解）
apps/web/src/components/homepage/*.spec.tsx               ← Task 2, 3
apps/web/src/utils/libraryStatus.ts                       ← 只讀，AC #2 的事實基礎
apps/web/src/utils/formatUsdShort.ts                      ← 只讀，AC #6 的事實基礎
scripts/export-pen-screenshots.py                         ← Task 1, 7
CLAUDE.md                                                 ← Task 7
ux-design.pen（k2Otv/uGCAU/B7UO8/RvS6c/zRyNS/EoCQ4/iWUSV/Y5XvRv）← Task 5,6,7，只能用 Pencil MCP
```

### Project Structure Notes

- `sprint-status.yaml` 的 `dsr-7` 條目有**三處已經過期**，dev 以本 story 為準：
  - 「① `ExploreBlock.tsx`／`RecentlyAddedRowV2.tsx` 有 raw shadow-*」——**已被 dsr-9 修掉**，現在四處全是 `shadow-[var(--shadow-lg)]` token 形式，`local/no-raw-shadow` 也擋著了。
  - 「③ 只有 8/18 檔有 Design ref 標頭」——18 是把 9 個 `.spec.tsx` 也算進去了，但 Rule 21 明文豁免 spec/test。**9 個元件檔全都有標頭**；真正的問題是其中 **2 個指向已刪除的節點**（`sAaCR`）、1 個寫著舊名字——那比缺標頭更糟，因為它看起來是對的。（dsr-10 遇過一模一樣的情況。）
  - 「④ 四個狀態要對照 h4–h7 逐一驗」——已在本 story 的對照表逐條驗完，剩下的是執行不是調查。
- 唯一真正剩下的原始缺口是「② 1 檔舊字級」（= `HomeReadoutBand` 的 2 處 `text-[11px]`，見 AC #13）。
- `routes/index.tsx`／`routes/test/*` 免 Rule 21（route 檔）。

### Time-dependent visual coverage

- **Does this story add/modify any `apps/web/src/components/**/*.{ts,tsx}` that reads `Date.now()` / `new Date()` / `Date.UTC()` / `Date.parse()`?**
  - **N/A — no wall-clock-reading components touched.** 已驗證：`grep -n "Date.now()\|new Date(\|Date.parse\|Date.UTC\|formatRelativeTime" apps/web/src/components/homepage/*.tsx`（排除 spec）**零命中**。「今天處理」是後端算好的 `processedToday`，前端只渲染字串。
  - 本 story 也**不新增任何 gallery 夾具**（`HomeReadoutBand`／`RecentlyAddedRowV2` 目前沒有夾具，AC 沒有要求新增）。若 dev 想順手加，`recent`/`stale` 兩態與 `withFixedClock(page, iso)` 的規矩照 Rule 23，但預設**不要**——會多一輪 `-linux` 基準線 CI。
- Reference: `project-context.md` Rule 23；audit `_bmad-output/audit/time-bomb-fixtures-2026-05.md`。

### References

- [Source: `ux-design.pen` Screen H1-D-v3 (k2Otv) / H2-M-v3 (uGCAU) / H4-D-v3 (B7UO8) / H5-D-v3 (RvS6c) / H6-D-v3 (zRyNS) / H7-D-v3 (EoCQ4) / H8-SPEC-v3 (iWUSV) / H5-D (Y5XvRv)] — 內容以 Pencil MCP `Get(..., {resolveInstances:true})` 展開 instance 後讀出，非依賴縮圖
- [Source: `apps/web/src/utils/libraryStatus.ts:55-66`] — `deriveLifecycleStatus` 的 `pending → TINT.accent`（dsr-11 裁定），AC #2 的事實基礎
- [Source: `apps/web/src/components/homepage/RecentlyAddedRowV2.tsx:134-152`] — 藥丸的 `warning-*` 與那段已失效的註解
- [Source: `apps/web/src/utils/formatUsdShort.ts:12-29`] — `trimTrailingZero` ⇒ `$1.2` / `$5`，AC #6 的事實基礎
- [Source: `apps/web/src/components/homepage/HeroBanner.tsx:165, 189, 196, 208, 262, 332, 361`] — 漸層、scrim pill、CTA、圓點，AC #3/#4/#9 的事實基礎
- [Source: `apps/web/src/components/homepage/HomeReadoutBand.tsx:53-72, 130, 148, 236`] — `attentionText`、標籤／數字的赭色歸屬、兩處 11px，AC #6/#7/#8/#13 的事實基礎
- [Source: `apps/web/src/components/homepage/ExploreBlocksList.tsx:48, 154, 162, 167`] — `ExploreDegradedNotice` 的兩種文案，AC #11 的事實基礎
- [Source: `apps/web/src/components/homepage/ExploreBlock.tsx:184, 189, 265`] — scrim 漸層與空狀態三元，AC #5/#12 的事實基礎
- [Source: `apps/web/src/components/homepage/HomeBrowseV2.tsx:42, 48, 61, 68`] — D3 區塊順序，對照表第 21 條
- [Source: DESIGN.md#Colors（固定詞彙）] — 赭＝要求了但沒發生；泥金＝正在跑
- [Source: project-context.md#Rule 21 / #Rule 23 / #Rule 24]
- [Source: `sprint-status.yaml` → `epic-dsr` / `dsr-7-flow-h-homepage-v3` / `dsr-11-flow-l-requests-v2`（整理中改金的裁定）/ `dsr-9-flow-j-specs-and-design-system`（shadow token 收斂）/ `disc-2026-09-text-on-artwork-flips-with-theme`（P0）/ `disc-2026-09-flow-a-m-screen-content`（假資料重複判例）]
- [Source: `_bmad-output/planning-artifacts/home-v3-identity-brief.md`] — 讀數帶四格與 hero identity 的原始裁定

## Dev Agent Record

### Agent Model Used

Claude Opus 5 (1M context) — `claude-opus-5[1m]`，BMAD dev agent（Amelia）2026-09-16

### Debug Log References

- `npx vitest run …/RecentlyAddedRowV2.spec.tsx`（RED）：1 failed / 5 passed —— 逐一比對 `deriveLifecycleStatus` className 的迴圈先炸，正是想要的守法
- `npx vitest run …/HomeReadoutBand.spec.tsx`（RED ①）：`text-[11px]` × 8 個節點被字階守門測試抓出
- `npx vitest run …/HomeReadoutBand.spec.tsx`（RED ②）：手機兩行 1 failed / 32 passed
- `pnpm nx test web`（GREEN）：**3401 / 3401 passed**（+5 新測試：藥丸色、字階、手機兩行 ×3）
- `pnpm nx test api`：PASS
- `pnpm run lint`：**0 errors**、128 warnings
- `pnpm nx run web:typecheck --skip-nx-cache`：PASS
- `python3 scripts/check-design-tokens.py`：一致（82 變數／192 畫面／73 母版）
- Playwright（臨時夾具，已移除）：390px 日巡實機量到 value 96.3×48（兩行）落在 179×84 的格子裡、`scrollWidth-clientWidth = 0`、分隔號 `display:none`；1280px 時 `flex-direction: row`、分隔號 `display:block`
- 匯出：192 張，只有 10 個檔案變動（Flow H 7 張 v3 ＋ h3 改 2x ＋ h9-spec 新增 ＋ h5-d 刪除）

### Completion Notes List

- 🔗 **AC Drift: FOUND —— 兩條，其中一條改寫了這張 story 的前提。**
  檢查方式：`grep -rln "整理中" _bmad-output/implementation-artifacts/*.md`（15 檔命中）＋ 逐一讀 `ux3-1-7`／`fix-homepage-critique-r2`／`dsr-11`。

  1. **`fix-homepage-critique-r2.md:26` → 本 story AC #2（修復既有漂移，非新漂移）。**
     R2 當初把藥丸定成赭色，理由白紙黑字是「chip now matches the badge exactly（整理中, warning tint/text）」。
     dsr-11（⚖️ Alexyu 2026-09-11）把 `libraryStatus.ts` 的徽章改成 `TINT.accent` 泥金，**但沒有同步藥丸**——
     R2 的理由因此在兩天後就失效了，而藥丸留在原地。本 story 是把 dsr-11 沒走完的最後一哩補上，
     不是推翻 R2：R2 要的是「兩者同色」，現在同色的值是泥金。

  2. **`ux3-1-7-home-v3-readout-band-fe.md` AC #3 → 本 story AC #8 與 AC #13。⚠️ 這條改寫了 story 的前提。**
     原文：「Desktop: single-row flex, 4 equal cells, **11px labels** + mono digits；mobile: 2×2 grid, digits 16px,
     **the 需要注意 cell breaks to two lines（第一行 N 部失敗 / 第二行金額）per H8-SPEC-v3**」。

     - **AC #8 不是新功能，是一條出貨時漏做的 AC。** SM 建單時判成「H8-SPEC vs H2-M vs 程式碼三方不一致、需要裁定」，
       實際查了上游才發現 `ux3-1-7` AC #3 **早就指定了兩行**，程式碼從頭到尾沒做。Alexyu 2026-09-16 的裁定
       （「兩行，照規格稿做」）因此不是新決定，而是**恢復原本就寫好的契約**。story 的敘述已在 AC #8 保留裁定紀錄，
       此處補記真正的出處。
     - **AC #13（11px → 12px）確實推翻了一條已出貨的 AC，需要 Alexyu 知情。**
       `ux3-1-7` AC #3 明寫 11px，來源是 `home-v3-identity-brief.md` §2 的散文（「11px labels over mono digits」）。
       但 `.pen` 的 `H1-D-v3` 標籤節點（`Q7kbtf` 等）綁的是 `$Type/Label/Size`，而該變數的值是 **12**。
       也就是說：brief 寫 11 → 稿畫 12（因為用了 token）→ 程式碼實作 11。
       **判定改 12**，依據是 epic-dsr 的前提（「設計稿是新事實，程式碼追回來」）＋ dsr-9／dsr-10 的字階收斂先例。
       若 Alexyu 認為 11px 才是要的，改回去只需一行，並應同步把 `.pen` 的標籤節點從 token 改成字面 11
       （否則下一輪 dsr 會再撿一次）。

- 📎 **Contract Stamps: FOUND（1 個，跨 1 個上游檔）。** `ux3-1-7-home-v3-readout-band-fe.md:6` 的 `[@contract-v1]` 指的是
  **上游 `ux3-1-6` 的 `GET /api/v1/home-summary` wire contract**，不是讀數帶自己的視覺契約。本 story 不動 API、不動
  `homeSummaryService` 的型別、不動 SSE，該契約**未 bump**，維持 v1。本 story 自身不定義新契約（無 `[@contract-v*]`）。

- 🎭 **A11y Pre-Flight: PASS**（7 個 component 檢查：`HomeReadoutBand`／`RecentlyAddedRowV2`／`ExploreBlock`／`ExploreBlocksList`／`TrailerModal` ＋ 兩支 spec；jsx-a11y warning 0，本 story 引入 0）。四類回歸項：沒有新增圖片、沒有新增 aria-modal、沒有新增非同步揭露內容、沒有新增自訂 widget。手機兩行是**同一份 DOM 換軸向**，格子的 `aria-label` 一字未動；新增的分隔號帶 `aria-hidden="true"`（它是桌機的標點，不是內容）。

- 🎨 **UX Verification: PASS（真的開瀏覽器量的，不是讀程式碼推的）。** 用一個**臨時** gallery 夾具在 390px 實機渲染讀數帶，塞進最壞情況金額 `$9.9k/$99k`：
  | 量到的 | 值 |
  | --- | --- |
  | 文字 | `2 部失敗 · $9.9k/$99k` |
  | 高度 | 48px＝**兩行** |
  | 軸向 | `flex-direction: column`（手機）／1280px 時 `row`（桌機） |
  | 分隔號 | 手機 `display:none`／桌機 `display:block` |
  | 寬度 | value 96.3px ⊂ 格子 179px |
  | 溢出 | `scrollWidth - clientWidth = 0` —— **沒有截斷** |
  | 顏色 | `rgb(123,58,6)`（日巡的 `--warning-text`）壓在 `rgb(240,231,211)` 上 |
  夾具與臨時 spec **已全數移除**（`git status` 對 `routes/test/` 與 `tests/` 皆為 clean），**沒有留下任何視覺基準線**。
  設計稿側逐張看過 `h1-d-v3`／`h2-m-v3`／`h6-d-v3` 的匯出圖，與程式碼行為相符。

- 🔧 **Pre-existing fix（Epic 9c Retro AI-2 lane 1）**：`pnpm run format:check` 在本機一直是紅的，7 個檔案——`apps/api/coverage/*`（6，`nx test api --coverage` 產生的 HTML 報告）與 `testsprite_tests/testsprite-mcp-test-report.html`。兩者都是**跑測試產生的產物**，卻沒有被 `.gitignore` 蓋到（只有 root 的 `/coverage` 與 `testsprite_tests/tmp/`）。屬「15 分鐘內可修」，就地補兩條 ignore 規則並附上原因註解。CI 不受影響（乾淨 checkout 沒有這些檔案），但本機每一支 story 都會踩到，而且有被誤 commit 的風險。

- ⚠️ **建單時寫錯、實作時更正的三處**（story 內文已同步改）：
  1. AC #9 說「H2-M hero 缺手動切換圓點」——**圓點本來就有**（`EpmNH`，藏在 depth 4，建單時的結構掃描只到 depth 3）。真正缺的只有「查看詳情」按鈕。
  2. AC #7 只點名 H7／H6 的「需要注意」標籤穿錯色——實際上 **H1、H2-M、H8 的三組桌機樣本也同錯**，五張稿共 11 個節點。
  3. AC #16 說 15 張海報卡重複——實際是 **36 張**：H5／H6／H7 各 6 張**完全沒有覆寫**，直接吃母版預設的「你的名字 · 2016 · 107 分」，建單時只掃了有 descendants 的卡。

### Discovery Triage

- **Did this story discover any work outside its current scope?** **YES —— 一項就地吸收、五項立案。**

- **① expand-scope-in-place（由既有 AC 追蹤，不另開單）**
  - **海報徽章只該在例外狀態出現。** AC #16 原本要我「讓徽章至少出現兩種」，實作時查 `pickPosterBadge`（`libraryStatus.ts:203-209`）才確認：**已入庫＋繁中是穩定態，函式回 `null`、程式碼根本不畫徽章**。稿上每張卡都掛一顆「繁中」徽章，等於在畫面上宣告一個出貨版不存在的例外。已在 Flow H 的 instance 層把 8 張穩定態卡片的徽章關掉，只留「整理中」那幾張（並隨 AC #2 改成泥金）。由 AC #2／#16 共同追蹤。

- **③ backlog-with-carry-forward-link —— create-story 當下立的三案（2026-09-16）**
  - **`disc-2026-09-trailer-modal-unmounted`** — `TrailerModal.tsx`（含 FocusScope 與 YouTube nocookie 嵌入）在全 app **沒有任何掛載點**，唯一的非測試引用是 `routes/test/-gallery.fixtures.tsx:1696` 的視覺夾具。它是 Story 10-2 為舊首頁 hero 的「預告片」按鈕做的；Home v3 的 hero 只有「查看詳情」。刪或留是產品裁定（留＝Epic 17 播放路徑可能用得上；刪＝少一個有視覺基準線的死元件）。與 `dsr-2` 的 `MediaDetailPanel.tsx` 同類，建議一起裁。本 story 只改它的標頭，不刪。
  - **`disc-2026-09-block-failure-copy-two-canons`** — 區塊載入失敗的文案有兩套：Flow K 是短句 `無法載入，請稍後再試`＋`重試`；Flow H 是具名兩行 `最近新增目前無法載入`＋`這個區塊的資料暫時無法取得，其他首頁內容仍可使用。`＋`重試`。兩者都合理（短句省空間、具名較誠實），但同一個產品對同一件事有兩個講法。統一是文案正典裁定，牽動 Flow F／I 的同類區塊。
  - **`disc-2026-09-pen-hardcoded-hex-unguarded`** — `check-design-tokens.py` 只比對**五份 token 定義**是否一致，**不掃 `.pen` 節點裡的寫死 hex**。本 story 抓到 6 個寫死 `#0c1512` 的 scrim 漸層，全是 `--bg-primary` 的夜行字面值，日巡下不會翻轉——它們躲過了 token 檢查器、躲過了 `fix/flow-a-m-screen-content` 的 186 處清理、也躲過了 CI。需要一條守門規則（掃 fill，遇到等於某 token 某主題值的字面 hex 就報）。

- **③ backlog-with-carry-forward-link —— 實作中發現的兩案（2026-09-16）**
  - **`disc-2026-09-postercard-v2-badge-shape-drift`** — `Component/PosterCard-v2` 母版的狀態徽章與出貨的 `PosterCardV2.tsx` 有三處形狀差異：位置（稿**左上** vs 碼 `absolute right-1.5 top-1.5` **右上**）、底色（稿 `$overlay-scrim` vs 碼「不透明 `--bg-secondary` 墊底再疊 tint」——那是 critique R1 P0 為了壓在任意海報上仍可讀而特意加的）、以及稿多了一個**打勾圖示**（碼的徽章沒有任何圖示，`Check` 在該檔是「選取」用的）。**本 story 沒有動母版**：它被 Flow A／B／L 大量 instance，屬 `components/library/**`＝`dsr-1` 的範圍，改一次會動到三個流程的視覺基準。
  - **`disc-2026-09-pen-poster-artwork-all-identical`** — Flow H 的 36 張海報卡片名／年份／片長現在都不同了，但**海報圖仍是同一張「你的名字」**（母版的 image fill，instance 未覆寫）。一排六張時，六張一樣的圖比六個一樣的片名還醒目。要修得逐卡 `Generate(..., "stock"|"ai")`，成本不低且橫跨 A／B／H／I 同一個母版。

- **雙向連結**：五個條目的 `sprint-status.yaml` 敘述都寫明「filed by dsr-7」，本區塊亦逐一寫明條目 ID。
- Reference: `project-context.md` Rule 24

### File List

**修改（程式碼）：**
- `apps/web/src/components/homepage/RecentlyAddedRowV2.tsx` — 「整理中」藥丸赭→泥金＋重寫失效的註解
- `apps/web/src/components/homepage/RecentlyAddedRowV2.spec.tsx` — 藥丸守門測試改成比對 `deriveLifecycleStatus` 的 className
- `apps/web/src/components/homepage/HomeReadoutBand.tsx` — 兩處字級上字階、`attentionText` 加 `failures`/`spend`、`ReadoutCell` 加 `valueParts`（手機兩行）
- `apps/web/src/components/homepage/HomeReadoutBand.spec.tsx` — ＋4 條（字階守門、手機兩行、兩種單行情況）
- `apps/web/src/components/homepage/ExploreBlocksList.tsx` — Rule 21 標頭（原指向已刪除的 `sAaCR`）
- `apps/web/src/components/homepage/TrailerModal.tsx` — Rule 21 標頭改缺口變體（原指向已刪除的 `sAaCR`）
- `apps/web/src/components/homepage/ExploreBlock.tsx` — Rule 21 標頭改 `H9-SPEC`
- `apps/web/src/styles.css` — `--warning-tint` 的註解（僅註解，值不變）
- `.prettierignore` — pre-existing fix：`apps/api/coverage/` 與 `testsprite_tests/*.html`（CR 後從 `.gitignore` 改放這裡，`.gitignore` 完全未動）
- `tests/e2e/homepage-layout.spec.ts` — ＋5 條真實瀏覽器測試（390／768／1024／1280 ＋ CLS）

**修改（設計與腳本）：**
- `ux-design.pen` — Flow H 九張稿：6 個寫死漸層改吃變數、3 張 hero 補 scrim pill、五處金額、11 個「需要注意」節點的赭色歸位、H2-M 補探索區與 CTA、H6 錯誤橫幅重畫、H7 文案與新註記、H9-SPEC 更名與規格文字、36 張海報卡資料、3 個整理中藥丸改泥金、手機讀數帶格高 64→84
- `scripts/export-pen-screenshots.py` — `READABLE_FLOWS` ＋2、`SCREENS["Y5XvRv"]` → `h9-spec`
- `CLAUDE.md` — `flow-h-homepage` 說明同步 H9-SPEC 更名
- `_bmad-output/pen-tokens.json` — 快照同步
- `_bmad-output/screenshots/flow-h-homepage-v3/*.png`（7 張）
- `_bmad-output/screenshots/flow-h-homepage/h3.png`（改 2x 可讀）
- `_bmad-output/screenshots/flow-h-homepage/h9-spec.png`（新增）
- `_bmad-output/screenshots/flow-h-homepage/h5-d.png`（刪除，已更名）
- `_bmad-output/implementation-artifacts/sprint-status.yaml` — 狀態流轉＋5 筆新立案＋P0 條目補雙向連結

**參考（AC drift，未修改）：**
- `_bmad-output/implementation-artifacts/ux3-1-7-home-v3-readout-band-fe.md`（AC drift reference — see Completion Notes）
- `_bmad-output/implementation-artifacts/fix-homepage-critique-r2.md`（AC drift reference — see Completion Notes）

## 對抗式 Code Review（/ship，2026-09-16）

獨立 reviewer（fresh context，只拿到 diff 與 repo 讀取權）回報 **18 項**。分類與處置：

### 🔴 修掉的真缺陷

| # | 問題 | 處置 |
| --- | --- | --- |
| 1 | **`md:flex-row` 在最窄的地方把修正撤銷了。** 斷點設在 `md`(768)，但那時讀數帶已經是四格橫排，一格內容寬 ≈155px——**比 390px 手機的 ≈154px 還擠**，而且字級已經升到 `text-lg`。等於在「比原本更窄」的地方換回一行。我原本只量了 390 與 1280，**768–1023 從來沒看過**。 | 斷點改 `lg`；並補一支**跨 390／768／1024／1280 的真實瀏覽器測試**鎖住 |
| 2 | **載入骨架不再等高。** 骨架存在的唯一理由是擋掉一個實測 54px／CLS 0.0832 的跳動，但真實格子在 `lg` 以下可能兩行、骨架永遠一行 → 資料到了band 會長高 8px。 | 高度下限移到 `CELL_BOX`（`min-h-[84px] lg:min-h-[44px]`），骨架與實際值**由同一條規則決定**而不是兩串手工對齊的 class；84 是**量出來的**，不是算的 |
| 3 | **唯一的行為改動沒有任何自動化覆蓋。** jsdom 不套 Tailwind、沒有斷點，class 字串斷言不論功能在不在都會綠；我用來驗證的臨時夾具驗完就刪了。 | 在 `tests/e2e/homepage-layout.spec.ts` 新增 5 條**永久**測試（4 個寬度 ＋ 1 條 CLS）。CI 的 e2e 不帶 grep，所以**這些會在 CI 跑到** |
| 4 | 三條新測試有兩條**把功能整個拿掉也會綠**（只斷言某個 testid 不存在）。 | 改成斷言單行時**不該**帶 `flex-col`——正好也擋住「class 又被寫成無條件」的回頭路 |
| 5 / 6 | `attentionText` 的回傳型別允許三種它永遠產不出來的狀態，導致一個永遠為真的守衛；兩半情況下 `text` 是死碼，改成垃圾也沒有測試會紅。 | 改成 discriminated union，兩半情況**不再產生** `text`，守衛消失 |
| 7 | 註解宣稱「文字在任何寬度逐字相同」——那只是 `textContent` 不看 CSS 的假象，實際渲染出來的讀法會變。 | 註解改寫成誠實的說法 |
| 8 | **金額對螢幕閱讀器完全不存在。** 格子的 `aria-label` 會**取代**內容，而它只講「2 部失敗待處理」。AC #8 的立論正是「金額是護城河第一層，不該被擠掉」——結果對一整類使用者被擠掉了。我原本還把「aria-label 一字未動」當成優點寫進 a11y 紀錄。 | `aria-label` 補上金額，並加測試 |
| 9 / 10 | **`.gitignore` 那一刀下錯位置。** (a) 註解說那是 Go 覆蓋率，實際是 **istanbul/nyc 的 JS 報告**寫進了 Go 專案資料夾；(b) TestSprite 的 HTML 報告**歷史上是刻意 commit 的**（有五個 commit，其中一個還叫「prettier-format generated HTML report」），把它放進 `.gitignore` 是在一張首頁 CSS 的單子裡偷改追蹤政策。 | `.gitignore` **完全還原**；兩條改放 `.prettierignore`（那是本 repo 既有的慣例，只影響 prettier 讀什麼、不影響 git）。JS 覆蓋率寫錯位置另行立案 |
| 11 | Rule 21 的修法把節點 ID 從**被 lint 驗證的那一行**挪到了下面的自由文字——那正是 `sAaCR` 指向空無三週的成因。 | 兩個檔頭的 ID 全部放回 `Design ref:` 那一行 |
| 12 | 藥丸測試讀的是 `deriveLifecycleStatus`，但卡片渲染的是 `pickPosterBadge`（多一層優先序）。把優先序改掉，徽章變色而測試照樣綠。 | 改成斷言 `pickPosterBadge` |
| 14 | 字階守門只掃自己渲染出來的 DOM，AC #13 要的是整個資料夾。 | 補一條掃 `components/homepage/*.tsx` 原始碼的測試 |
| 15 | `el.className` 對 `<svg>` 是 `SVGAnimatedString`，那個正則永遠不會命中——形同虛設。 | 改用 `getAttribute('class')` |
| 16 | `toContain('hidden')` 會被 `md:hidden` 蒙混過去。 | 改成比對 class token 陣列 |
| 18 | `flex-row` 沒有 `justify-center`，溢出時該格會左靠、與其他三格不一致。 | 補 `justify-center` |

### ⚖️ 裁定後改回去的一項

| # | 問題 | 處置 |
| --- | --- | --- |
| 13 | **11px → 12px 推翻了一條已出貨的 AC，而且製造了新的不一致。** 實查：全站有 **44 處**（不是 reviewer 估的 ~24）`text-[11px]`，包括 `MobileTabBar`、`SidebarGroupLabel`、以及 **`PosterCardV2.tsx:154` 那顆徽章**——就在藥丸下方 40px、正是這張單子特意去對齊顏色的那一顆。改成 12px 會讓讀數帶變成全站唯一的 12px 微標籤。 | ⚖️ **Alexyu 2026-09-16 裁定改回 11px**，AC #13 撤銷。字階守門測試一併移除（它守的規則不成立了），改成守「骨架標籤與真標籤同尺寸」——那才是骨架註解真正在意的事。設計系統要不要補一個 11px 的 Micro 階，立案 `disc-2026-09-11px-micro-label-not-on-type-scale`。 |

### 📝 已立案，不在本單

`disc-2026-09-js-coverage-in-apps-api`（誰把 JS 覆蓋率寫進 Go 資料夾）。
另有三項觀察未處理：`sAaCR` 現在以歷史文字形式仍可被 grep 到（誠實但下次掃描會再撞）、H9-SPEC 還有四處行號引用（AC #15 的原則只套用到兩處）、以及 `HeroBanner`／`HomeBrowseV2`／`RecentlyAddedRowV2` 的 companion 行有同樣的 #11 問題（既有，非本單引入，且目前指向的節點都還活著）。

## Change Log

| 日期 | 內容 |
| --- | --- |
| 2026-09-16 | ✅ 收單 —— PR #440 合併進 main（commit b16c7f10），CI 17 項全綠。**視覺基準零變動**，沒有開 bootstrap PR，與建單時的預測一致。中途 `Lint & Format Check` 紅過一次：新測試裡丟進瀏覽器跑的那段用了 `getComputedStyle`，eslint 當成 Node 程式碼看 → 改 `window.getComputedStyle`，同 PR 修掉。 |
| 2026-09-16 | ⚖️ **Alexyu 裁定：標籤改回 11px**，AC #13 撤銷（全站 44 處在用 11px，含藥丸下方 40px 那顆徽章）。字階守門測試移除，改成守「骨架標籤與真標籤同尺寸」。立案 `disc-2026-09-11px-micro-label-not-on-type-scale`。 |
| 2026-09-16 | 🔴 **對抗式 CR（/ship）回報 18 項，修掉 14 項**。最重的三項：① 斷點設在 `md` 等於在**比手機更窄**的地方換回一行（768px 一格 ≈155px < 390px 的 ≈154px，且字級已升到 `text-lg`）——改 `lg`；② 載入骨架不再等高，資料到了會跳 8px——高度下限移到 `CELL_BOX`（84px 是量出來的）；③ 唯一的行為改動**沒有任何自動化覆蓋**（jsdom 不套 Tailwind）——補 5 條真實瀏覽器測試進 `homepage-layout.spec.ts`，CI 的 e2e 不帶 grep 所以會跑到。另修：aria-label 補金額（螢幕閱讀器原本完全聽不到）、`.gitignore` 還原改用 `.prettierignore`、節點 ID 放回被 lint 驗證的那一行、藥丸測試改斷言 `pickPosterBadge`、兩條「拿掉功能也會綠」的測試改成有鑑別力。 |
| 2026-09-16 | ✅ Task 8 閘門全綠：web **3401/3401**、api PASS、lint 0 errors、typecheck PASS、prettier PASS、token 五份一致。匯出 192 張只有 10 個檔案變動，全在 Flow H。 |
| 2026-09-16 | 🎨 UX 驗證用**真的瀏覽器**做：臨時 gallery 夾具在 390px 日巡下渲染讀數帶，塞最壞情況 `$9.9k/$99k` —— 兩行、96.3px ⊂ 179px、`scrollWidth-clientWidth = 0`、分隔號手機隱藏桌機顯示。夾具與臨時 spec 已全數移除，**沒有留下任何視覺基準線**。 |
| 2026-09-16 | 🔧 Pre-existing fix：`format:check` 在本機一直紅（7 個檔案），全是 `apps/api/coverage/` 與 `testsprite_tests/*.html` 這類測試產物沒被 ignore 蓋到。補兩條 `.gitignore` 規則。CI 不受影響，但每支 story 都會踩。 |
| 2026-09-16 | Task 7 — `H5-D` → `H9-SPEC`（frame／標題／畫布標籤三處），腳本 `SCREENS`、`h5-d.png` 刪除、`CLAUDE.md` 同步。規格稿裡兩處過期行號拿掉，順手把裸 `shadow-lg` 改成 token 形式。 |
| 2026-09-16 | Task 6 — H2-M 補 TMDb 探索區與「查看詳情」（**圓點本來就有，建單寫錯**）；H6 錯誤橫幅改成具名兩行＋拿掉多畫的外框；H7 文案一字之差修正＋補兩種降級文案的註記；H9-SPEC 空狀態補第二句文案；**36 張**海報卡（不是 15）換成各不相同的片，並依 `pickPosterBadge` 關掉 8 張穩定態卡的徽章。三張 hero 的 CTA 高度 40 → 44（N5 觸控底線）。 |
| 2026-09-16 | Task 5 — 6 個寫死 `#0c1512` 漸層改吃 `$bg-primary`（透明端留 `#00000000`，那是「沒有顏色」不是「凍結的顏色」，與程式碼的 `to-transparent` 逐字對等）；三張 hero 的「最新入庫」與狀態列補 `$overlay-scrim` pill＋`$text-on-scrim`；五處 `$1.20/$5.00` → `$1.2/$5`（稿違反自己 H8-SPEC 的折疊規則）；**11 個**「需要注意」節點的赭色從標籤搬到數字。 |
| 2026-09-16 | Task 4 — 三個檔頭。`ExploreBlocksList` 與 `TrailerModal` 指向的 `sAaCR` **三週前就被刪了**（v3 切換），是「看起來對、其實指向虛空」的壞標頭；`ExploreBlock` 的 `HP-5` 是舊名。三個都修好並註明原委。eslint 0 errors。 |
| 2026-09-16 | Task 3 — 手機「需要注意」格斷兩行。做法是**一份 DOM、CSS 決定軸向**（`flex-col md:flex-row` ＋ `hidden md:inline` 的 `·`），所以文字內容在任何寬度都逐字相同，既有斷言與 aria-label 零改動。`attentionText` 只加 `failures`/`spend` 兩個可選欄位。只有「失敗數＋金額」兩半俱在才斷行（H8-SPEC 的「金額升到第一行」）。web 3401/3401 綠。 |
| 2026-09-16 | Task 2 — 「整理中」藥丸從赭改泥金（先紅後綠）。守門測試不比對字面色碼，改成**讀 `deriveLifecycleStatus` 回傳的 className 逐一比對**，讓藥丸與海報徽章在結構上無法再分家。順手修 `styles.css:76`（`--warning-tint` 的註解還寫著「整理中 pill」）與元件註解（R2 的舊前提）。兩處 `text-[11px]` → `text-xs`，骨架分支一起改（它刻意借用同一個字級來自我量高）。web 3398/3398 綠。 |
| 2026-09-16 | Task 1（部分）— `READABLE_FLOWS` 加入兩個 flow-h 資料夾。匯出併入 Task 8 只跑一次（設計稿內容全程用 Pencil MCP 直讀，不依賴縮圖）。 |
| 2026-09-16 | 🔗 **AC-drift 檢查改寫了一條前提**：`ux3-1-7` AC #3 早就指定手機「需要注意」格要斷成兩行——AC #8 不是新功能，是出貨時漏做的 AC。同一條 AC 也寫著「11px labels」，與 `.pen` 綁的 `$Type/Label/Size`=12 牴觸，判定以設計稿為準（AC #13），已記在 Completion Notes 供 Alexyu 覆核。 |
| 2026-09-16 | ⚖️ **Alexyu 裁定：手機「需要注意」格做兩行**，照 `H8-SPEC-v3`。AC #8 從「需要裁定」改為已定案，Task 3 解除阻塞。 |
| 2026-09-16 | Story 建立（SM Bob, create-story）。設計稿八張以 Pencil MCP 逐節點讀出後與 `components/homepage/` 九個元件逐條比對，得 23 條對照：13 條稿改、5 條碼改、5 條已一致。sprint-status 條目的三處過期描述在 Project Structure Notes 更正。 |

## 裁定紀錄

### ⚖️ 2026-09-16 · 手機版「需要注意」格：兩行

三個版本互相打架：`H8-SPEC-v3`（規格稿）說兩行、`H2-M-v3`（手機稿）畫一行且**沒有分母**、出貨的程式碼是一行有分母。

**Alexyu 選：兩行，照規格稿做。** 金額是護城河的第一層，不該在最小的螢幕上被擠掉。

落點：AC #8 / Task 3。`H2-M-v3` 的 `sJ87f` 重畫成兩行且補回分母；`HomeReadoutBand` 加一個 `md` 斷點分支。
