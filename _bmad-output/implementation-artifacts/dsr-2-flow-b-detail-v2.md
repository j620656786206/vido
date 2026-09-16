# Story DSR.2: Flow B 詳情頁——程式碼與設計稿雙向對齊（電影／影集詳情、狀態、延伸區塊）

Status: review

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As the person who clicks into a movie or series from 媒體庫 every day,
I want 詳情頁在淺色主題也讀得到字、圖片載不出來時不顯示破圖、伺服器出錯時不說「找不到這部影片」,
so that 這一頁說的每一句話都是真的，而且設計稿指到的每一個畫面、每一個元件都真的存在。

## Context

`epic-dsr` 的第十張（已收 1、4、5、7、9、10、11、12、13）。

⚠️ **已拆成兩張。**

- **本張（dsr-2）＝詳情頁已經出貨的部分**：B3p-D 電影／B4p-D 影集／B6p-D 找不到／B7p-D 骨架／B8p-D 延伸區塊／B3p-M 手機，＋ B9-D 圖片載入失敗規格，＋ 約 20 個檔頭，＋ **一個小的後端修正**（AC #8）。
- **拆出 `dsr-2b`＝「沒有中繼資料」的兩個狀態**（B6-M 比對失敗／B7-M 正在比對）。程式碼 v2 **完全沒做**（`FallbackFailed`／`FallbackPending` 只活在視覺夾具裡，e2e 有 4 條 `test.skip` 在等），而且稿是 v1 側邊抽屜的形狀，要先重畫才能做。這是補功能，不是對齊。

⚠️ **驗收基準是 `.pen` 節點值，不是 PNG。** Flow B 兩個資料夾都不在 `READABLE_FLOWS` 裡，16 張都是 400px 縮圖，讀不到字。Task 1 先修。

⚠️ **方向不是單向的。** 下面每一條都標了方向：`稿→碼`（程式碼要改）或 `碼→稿`（設計稿要改）。**dev 不要無腦把程式碼改成跟稿一樣**——hero 漸層那一條，碼是對的、稿是錯的（dsr-7 已在首頁證明過）。

### 🔴 建單時查到、sprint-status 條目沒寫的四件事

1. **P0「技術徽章用狀態色」的程式碼那一半，改錯檔案會白做。** 條目指向 `TechBadge.tsx:15-18`，但 **`TechBadge.tsx` 在正式環境沒有任何掛載點**（只有 `TechBadgeGroup` 與視覺夾具在用，而 `TechBadgeGroup` 自己也沒掛載）。詳情頁真正畫出技術徽章的是 **`DetailTechInfoV2.tsx:28-34` 的私有 `Badge`**：`bg-[var(--accent-tint)] text-[var(--accent-text)]`、4px 圓角。**改 `TechBadge.tsx` 使用者什麼都看不到。**
2. **詳情頁的 hero 漸層，程式碼已經是對的，是設計稿還寫死。** `DetailHeroV2.tsx:55` 用會翻轉的 `var(--bg-primary)`。反而是設計稿 B3p-D／B4p-D／B5-D 與「Light · B3p-D 詳情」證據畫面還寫死 `#0c1512`——日巡證據畫面上片名「你的名字」確實讀不到（已用 Pencil 截圖確認）。
3. **「找不到這部影片」在伺服器出錯時也會出現，而且前後端兩層都有錯。** 後端 `movie_handler.go:99-103`／`series_handler.go:112-116` 對**任何**錯誤都回 404；前端 `libraryService.ts:24-39`／`tmdb.ts:48-62` 丟的是 `new Error(message)`，**狀態碼與 `error.code` 都丟掉了**。只改前端做不出來。
4. **Flow B 的設計稿有一半是在畫已經死掉的 v1 元件。** B8-M、B9-D、B9-M 畫的是 `MediaDetailPanel`（側邊面板＋手機抽屜），它**沒有任何 route 掛載**；B9-D／B9-M 上甚至還留著「播放」與「加入片單」兩顆按鈕（PR #411 清播放鈕時漏掉了規格畫面）。

### 設計稿節點（逐字抄，不要重查）

| 代號 | 節點 | 內容 | 本張 |
| --- | --- | --- | --- |
| `B3p-D` | `uRGu2` | 電影詳情（桌機） | ✅ |
| `B4p-D` | `N2fmG6` | 影集詳情（桌機） | ✅ |
| `B6p-D` | `Z42zy` | 找不到這部影片 | ✅ |
| `B7p-D` | `Tqy3E` | 載入骨架 | ✅ |
| `B8p-D` | `UH0sk` | 延伸區塊：預告片／觀看平台／相似推薦／豆瓣 | ✅ |
| `B3p-M` | `SzNRb` | 電影詳情（手機） | ✅ |
| `B9-D` | `Tn4Gz` | Spec — 圖片載入 Fallback | ✅ 改指向 |
| `B5-D` ／ `B5-M` | `7mdTJ` ／ `APfjC` | 詳情頁「⋯」選單 | ✅ 只拿掉「匯出中繼資料」（⚖️ 裁定 B，AC #12） |
| `B2-D` ／ `B2-M` | `auArc` ／ `1UHzI` | 海報卡右鍵選單（在媒體庫網格上） | ✅ 只拿掉「匯出中繼資料」（⚖️ 裁定 B，AC #12） |
| `B8-M` | `6OR3z` | v1 手機抽屜（技術徽章） | 刪（AC #11） |
| `B9-M` | `jH6rM` | v1 手機抽屜（圖片 fallback） | 刪（AC #11） |
| `B6-M` ／ `B7-M` | `2m1Pv` ／ `7UnDy` | 比對失敗／正在比對 | ⛔ `dsr-2b` |
| `B1-D` | `Qm662` | 媒體庫網格上的 hover 卡片 | ⛔ 已由 dsr-1 對齊（`PosterCard-v2/Hover`） |
| `L8-D-v2` | `G0xib` | 未入庫的片的詳情頁（想要流程） | 只用在檔頭（`TMDbDetailV2`） |
| `J3-D` | `Z54xAd` | 分集列字幕入口 | 只用在檔頭（`EpisodeList`） |
| `Light · B3p-D 詳情` | `m3N3ng` | 日巡證據畫面（在 Design System 群組） | ✅ 跟著 hero 一起改 |
| hero 漸層節點 | `XLwlb`（B3p-D）／`BLTZw`（B4p-D）／`yQSHM`（B5-D）／`Gc2xH`（Light）／`HU1Qk`（B3p-M） | 寫死 `#0c1512` | ✅ AC #5 |

### 程式碼地圖（正式環境真的會渲染的）

```
routes/media/$type.$id.tsx
├─ LocalDetailV2        （UUID → 媒體庫裡的片）
│  ├─ DetailHeroV2      hero：背景圖＋漸層＋返回鍵＋海報＋狀態徽章＋標題＋meta＋動作列
│  ├─ DetailStatesV2    骨架／找不到
│  ├─ DetailTechInfoV2  檔案資訊（技術徽章在這裡，不在 TechBadge.tsx）
│  ├─ CreditsSection · SeasonAccordion › EpisodeList
│  ├─ TrailerSection › TrailerEmbed · StreamingAvailability · DoubanSection · RelatedContent › PosterCard(v1)
│  ├─ DualRatingDisplay · NfoLocalizeAction · MetadataEditorDialog · ManageSubtitleDialogV2(Flow F)
└─ TMDbDetailV2         （TMDb 數字 id → 探索／首頁上還沒入庫的片；共用 hero、狀態與延伸區塊）
```

**只活在視覺夾具裡（v1 遺留）**：`media/` 的 `MediaDetailPanel`、`DetailPanelMenu`、`FileInfo`、`MetadataSourceBadge`、`TVShowInfo`、`TechBadge`、`TechBadgeGroup`、`FallbackFailed`、`FallbackPending`、`ColorPlaceholder` 元件本體（它的 `filenameToGradient` 函式**有**在用）；`metadata-editor/` 的 `CastEditor`、`PosterUploader`、`GenreSelector`（barrel 有匯出，但正式環境沒有人渲染）。

### 對照表

| # | 項目 | 設計稿 | 程式碼 | 方向 |
| --- | --- | --- | --- | --- |
| 1 | 技術徽章底色 | 中性 `$bg-tertiary` ＋ `$text-secondary`（P0 裁定） | `DetailTechInfoV2:30` 泥金 `accent-tint`／`accent-text` | **稿→碼** |
| 2 | 技術徽章圓角 | `tbb` 用 `$radius-md`（8px） | `radius-sm`（4px） | **兩邊都錯**：DESIGN.md §Shapes 點名 TechBadge＝藥丸 |
| 3 | 檔案資訊的字幕 | 一列事實：`字幕軌　繁中（繁體）· English` | 一顆**狀態色**藥丸（`deriveSubtitleStatus`，`:65-69`），跟 hero 的字幕徽章重複 | **稿→碼**，但值的寫法**稿要跟碼**（資料推不出「繁體」，見 AC #4） |
| 4 | 檔案資訊標籤 | `檔案大小`／`字幕軌`／`路徑`；標籤 `text-secondary`、值 `text-primary` | `大小`／`路徑`；標籤 `text-muted`、值 `text-secondary` | **稿→碼** |
| 5 | hero 漸層 | 寫死 `#0c1512`，日巡讀不到 | `var(--bg-primary)`，跟著主題翻轉 | **碼→稿** |
| 6 | hero 狀態徽章 | 一顆合併的 `已入庫 · 繁中字幕`，整顆綠色 | 兩顆（生命週期＋字幕），各自的顏色；字幕標籤是 `繁中` | **碼→稿**（「已入庫＋缺字幕」合成一顆綠色會說謊） |
| 7 | 返回鍵 hover | — | `DetailHeroV2:64` `hover:bg-[var(--bg-tertiary)]` 配固定淺色字：**日巡 hover 時變淺底淺字** | **碼要修** |
| 8 | 海報縮圖陰影 | 無 | `DetailHeroV2:72` `shadow-[var(--shadow-xl)]`（不是浮層，dsr-9 規則只准浮層有陰影） | **稿→碼** |
| 9 | 圖片載入失敗 | B9-D 規格：切換成片名雜湊漸層＋首字 | `DetailHeroV2:48/74` 的 `<img>` **沒有 `onError`**，圖片載入失敗時顯示破圖 | **稿→碼**；B9-D 改指向 `DetailHeroV2` |
| 10 | 伺服器出錯 | B6p-D 只畫「找不到」 | 後端任何錯誤都回 404；前端丟掉狀態碼；`LocalDetailV2:133`／`TMDbDetailV2:52-53` `isError \|\| !data` 一律顯示找不到 | **前後端都要修**（ux2-3 AC #7 原本就要求分開） |
| 11 | 找不到的說明 | `這個項目可能已從媒體庫移除，或連結已失效。` | `這個項目可能已被移除，或連結有誤。` | **稿→碼** |
| 12 | 兩套 404 | — | route 層 `$type.$id.tsx:45-65`：`404`／`找不到該媒體內容`；元件層：`找不到這部影片` | **碼要收斂** |
| 13 | 電影區塊順序 | 簡介 → 演員/製作 → 檔案資訊 | 簡介 → 檔案資訊 → 演員 | **稿→碼**（ux2-3 AC #4 寫的也是稿的順序） |
| 14 | 延伸區塊順序 | 預告片 → 觀看平台 → 推薦 → 豆瓣 | 預告片 → 觀看平台 → **豆瓣 → 推薦**（`LocalDetailV2:282-298`） | **稿→碼** |
| 15 | 演員區標題 | `演員 / 製作`（H3 區塊標題） | **沒有區塊標題**，只有子標題 `導演`／`創作者`／`演員陣容` | **稿→碼**（補 h2） |
| 16 | 串流區標題 | `觀看平台` | `可在哪裡觀看` | **稿→碼** |
| 17 | 推薦區標題 | `相似推薦` | `相關推薦` | **碼→稿**（TMDb recommendations 不是 similar，「相似」是在宣稱沒做過的事） |
| 18 | 豆瓣區標題 | `豆瓣` | `豆瓣評論` | **稿→碼**（2026-08-30 已裁定停抓短評，標題不該再承諾評論） |
| 19 | 豆瓣區內容 | 「豆瓣資料暫時無法取得」＋重試 | 只剩「查看豆瓣頁面」連結 | **碼→稿** |
| 20 | 影集分集區 | 標題 `分集`、季下拉、三集、`查看全部 25 集 →` | 標題 `季與劇集`、手風琴；**沒有「查看全部」頁面** | **碼→稿** |
| 21 | 分集字幕狀態 | 文字藥丸「繁中」 | 純圖示 `SubtitleStatusIcon`（J2-D 圖示語彙） | **碼→稿**（吸收 `backlog-episodelist-status-pill-vs-icon-drift`，SM 當時預判 (a)） |
| 22 | 片長單位 | `107 分`、`24 分` | 電影 `分`；`EpisodeList:169` `分鐘` | **稿→碼** |
| 23 | 半形逗號 | — | `SeasonAccordion:190`、`StreamingAvailability:116`、`RelatedContent:66` 的錯誤句用 `,` | **碼要修**（全形 `，`） |
| 24 | 推薦卡片 | `PosterCard-v2` | `RelatedContent` 用 v1 `PosterCard`（帶想要按鈕） | ⛔ 不動，立案（AC #13） |
| 25 | 動作列按鈕底色 | 次要鈕 `$bg-tertiary` | `bg-secondary`，hover `bg-tertiary` | 待驗（記進 Dev Notes，不強改） |
| 26 | 骨架形狀 | B7p-D | `DetailSkeletonV2` | 待驗（記進 Dev Notes） |
| 27 | `text-[13px]` | — | `EpisodeList:261`、`NfoLocalizeAction:70/283` | ⛔ **不動**（等 `disc-2026-09-type-scale-even-migration` 三項裁定） |
| 28 | `text-[11px]` | — | `DetailHeroV2:97`、`PosterCard`、`AvailabilityBadge` | ⛔ **不動**（⚖️ 2026-09-16：11px 是全站微標籤標準） |

---

## Acceptance Criteria

1. **Flow B 的稿要讀得到。** `READABLE_FLOWS` 加入 `"flow-b-detail-v2"` 與 `"flow-b-detail-interaction"`，**每一筆都附日期與理由註解**（dsr-1 CR 第 14 項）。匯出併到最後一次跑。

2. **檔頭：已掛載的 15 個檔案指向真的存在的畫面。** 已用 Pencil MCP 逐一驗證：`RgSxQ`、`2ltBl`、`wQOkg`、`407vK`、`KNI8F` **全部不存在**；`uRGu2`／`N2fmG6`／`Tqy3E`／`Z42zy`／`UH0sk`／`SzNRb`／`Tn4Gz`／`G0xib`（L8-D-v2）／`Z54xAd`（J3-D）／`zMYsL`（J6-D）／`XlFIq`（J1-D）／`7mdTJ`（B5-D）存在。

   | 檔案 | 現在 | 改成 |
   | --- | --- | --- |
   | `DetailHeroV2.tsx` | `Implements: Component/DetailHero-v2 (uRGu2)`——**形式錯**：`uRGu2` 是畫面不是元件 | `Design ref: ux-design.pen Screen B3p-D (uRGu2) + Screen B4p-D (N2fmG6) + Screen B3p-M (SzNRb) + Screen B9-D (Tn4Gz)` |
   | `DetailStatesV2.tsx` | `Implements: Component/Detail-Loading (Tqy3E) + …`——形式錯 | `Design ref: ux-design.pen Screen B7p-D (Tqy3E) + Screen B6p-D (Z42zy)` |
   | `TMDbDetailV2.tsx` | `Implements: Component/Detail-Movie-v2 (uRGu2)`——形式錯 | `Design ref: ux-design.pen Screen B3p-D (uRGu2) + Screen L8-D-v2 (G0xib)`（`RequestButton.tsx:1` 已把 L8-D-v2 指為想要流程的詳情頁） |
   | `DetailTechInfoV2.tsx` | 19-8 `pending` 佔位 | `Design ref: ux-design.pen Screen B3p-D (uRGu2)` |
   | `SeasonAccordion.tsx` | 19-8 `pending` 佔位 | `Design ref: ux-design.pen Screen B4p-D (N2fmG6)` |
   | `EpisodeList.tsx` | 19-8 `pending` 佔位 | `Design ref: ux-design.pen Screen B4p-D (N2fmG6) + Screen J3-D (Z54xAd)`（它的夾具與文件註解都引用 J3-D） |
   | `CreditsSection.tsx`、`DualRatingDisplay.tsx` | `RgSxQ`（已刪） | `Design ref: ux-design.pen Screen B3p-D (uRGu2)` |
   | `TrailerEmbed.tsx` | `RgSxQ`（已刪） | `Design ref: ux-design.pen Screen B8p-D (UH0sk)` |
   | `TrailerSection.tsx`、`StreamingAvailability.tsx`、`DoubanSection.tsx`、`RelatedContent.tsx` | 「no current screen frame; postdates the .pen design」——**已過期**，B8p-D 現在畫了這四區 | `Design ref: ux-design.pen Screen B8p-D (UH0sk)`（`RelatedContent` 第二行關於 `PosterCard` 的說明保留） |
   | `AvailabilityBadge.tsx` | `RgSxQ`（已刪） | `Design ref: ux-design.pen — no current screen frame; 「已有」徽章在設計稿裡沒有對應畫面`（已搜過全檔，沒有「已有」這個徽章） |
   | `MetadataEditorDialog.tsx` | 第 2 行 `RgSxQ`（已刪） | 第 2 行改成 no-screen 變體（「修改資訊」對話框沒有設計稿，已搜「編輯媒體資訊」查無）；**第 1 行的 `Time-bomb-exempt` 不動** |

   🚨 節點 ID **必須在 `Design ref:` 那一行**，而且那一行**要以 `)` 結尾、後面不能有任何字**（`implements-pen-node-id.js` 的 `DESIGN_REF_RE`；dsr-7 CR 第 11 項判例）。no-screen 變體的理由寫在分號後面，同一行結束。
   🚨 **ESLint 只驗形狀、不驗節點存不存在**——lint 綠不代表檔頭對。本表的節點已由 SM 驗過；dev 新增任何節點 ID 都要自己用 Pencil MCP 查一次。
   ✅ **不用改、已驗證**：`LocalDetailV2.tsx`（已經是正確形式——`backlog-localdetailv2-rule21-header-grammar` 實際上已被修掉，本張收單時關掉它）、`NfoLocalizeAction.tsx`、`PosterCard.tsx`、`PosterCardSkeleton.tsx`。
   ⛔ **`MediaGrid.tsx`（`KNI8F`）不在本張**：它掛在探索與搜尋頁，屬 Flow I ＝ `dsr-8`。

3. **檔頭：只活在夾具裡的 7 個 v1 檔案，改成誠實的形式（另確認 `DetailPanelMenu` 不動）。** 判準（從 dsr-1 CR 第 3 項推出來的）：**如果某張畫面在正式環境已經有別的元件在實作，死元件就不准指向它**（否則問「誰實作這張稿」會答出一個死元件）；**如果那張畫面根本沒有活的實作，死元件指向它是誠實的**。

   | 檔案 | 現在 | 改成 |
   | --- | --- | --- |
   | `media/MediaDetailPanel.tsx` | `RgSxQ`（已刪）＋ `Tn4Gz` | no-screen 變體：v1 側邊面板、無掛載點，見 `disc-2026-09-unmounted-v1-components`（B9-D 會在 AC #7 改指 `DetailHeroV2`，所以**不能**再指向 `Tn4Gz`） |
   | `media/MetadataSourceBadge.tsx`、`media/TVShowInfo.tsx` | `RgSxQ`／`407vK`（都已刪） | no-screen 變體，同上 |
   | `metadata-editor/CastEditor.tsx`、`metadata-editor/PosterUploader.tsx` | `RgSxQ`（已刪） | no-screen 變體，同上 |
   | `media/FallbackFailed.tsx` | `2ltBl`（已刪） | `Design ref: ux-design.pen Screen B6-M (2m1Pv)`——B6-M 沒有活的實作，指向它是誠實的；**另起一行**註明「測試專用，待 dsr-2b 接回」（不能寫在 Design ref 那一行後面） |
   | `media/FallbackPending.tsx` | `wQOkg`（已刪） | `Design ref: ux-design.pen Screen B7-M (7UnDy)`，同上 |
   | `media/DetailPanelMenu.tsx` | `7mdTJ`（B5-D） | **不動**（⚖️ AC #12 裁定 B：B5-D 保留、目前沒有活的實作，指向它是誠實的） |

   ✅ **`TechBadge.tsx` 不動**——四個母版都存在，而且是對的（中性、藥丸）。
   收尾自檢（dsr-1 CR 第 1 項的教訓）：把本張改過的檔頭裡所有 `disc-*`／`dsr-*` 字串抓出來，逐一確認 sprint-status 真的有那個條目。

4. **P0：詳情頁的技術徽章改成中性藥丸，字幕改成一列事實。** 對照表 #1–#4。
   - `DetailTechInfoV2.tsx` 的 `Badge`：`bg-[var(--bg-tertiary)] text-[var(--text-secondary)]`、`rounded-full`，保留 `font-mono`（技術值是讀數）。
   - 字幕：拿掉那顆狀態色藥丸，改成一列 `字幕軌`。**資料的真實形狀（已查證）**：`subtitleTracks?: string`（`types/library.ts:47,112`），是 JSON 字串 `[{language, format, external, stream_index}]`（`ffprobe_service.go:37-46`）；內嵌軌是 ffprobe 的 ISO 639-2 原始標籤（`chi`、`eng`、`jpn`，缺標籤時是 `und`），外掛字幕檔是檔名後綴原樣；舊資料可能是 `lang` 鍵或根本不是 JSON（`libraryStatus.ts:84-99` 已處理）。
     - **不要自己寫第二個解析器**：把 `utils/libraryStatus.ts` 的 `trackLangs` 匯出來用。
     - 對照：繁中集合 → `繁中`、簡中集合 → `簡中`、`en`／`eng`／`en-*` → `英文`（沿用 app 既有用字）、`und` → `未標示`，其他顯示原始標籤。去重，用 ` · ` 串起來；解析失敗或空清單時**整列不顯示**。
     - ⚠️ `chi`／`zho` **不在**繁／簡集合裡——大部分真實檔案的中文軌會顯示原始標籤 `chi`。這是誠實的（資料本來就分不出繁簡），**不要為了好看猜成繁中**。記進 Dev Notes。
     - **設計稿**：B3p-D 的字幕軌值 `繁中（繁體）· English` 改成程式碼會真的產生的寫法（例如 `繁中 · 英文`）。
   - 標籤 `大小` → `檔案大小`；標籤色 `text-muted` → `text-secondary`、值 `text-secondary` → `text-primary`。
   - **設計稿**：B3p-D `tbb` 四個節點（`PgOwQ`／`XawbQ`／`CjrKz`／`J9e5lX`）圓角 `$radius-md` → `$radius-pill`；B3p-M 的 `tb` 節點與 B5-D、Light · B3p-D 裡的同名節點一起改。
   - 先寫紅測試：`DetailTechInfoV2.spec.tsx:6-25` 那條「字幕藥丸」測試要**改寫**（它斷言的正是要拿掉的東西）；新增：徽章不含任何 `*-tint` class、字幕軌列的文字、無軌道／解析失敗時不存在。
   - 收單時：`disc-2026-09-techbadge-uses-status-colors-as-taxonomy` **關掉**，並在條目寫明「活的程式碼在 `DetailTechInfoV2` 已修；`TechBadge.tsx` 與它的 `media-tech-badge` 基準線、`TechBadge.spec.tsx:14` 仍是狀態色，但元件是死碼，已交給 `disc-2026-09-unmounted-v1-components`」。

5. **P0：設計稿的 hero 漸層改成會翻轉的變數（照 dsr-7 的做法）。** 對照表 #5、#6。
   - 五個漸層節點（表格最後一列）`#0c1512` → `$bg-primary`，透明端留 `#00000000`（dsr-7 判例：「沒有顏色」不是「凍結的顏色」）。
   - B3p-M 的 hero 文字目前是 `$text-on-scrim`（配寫死的深色漸層，自己是一致的，但跟程式碼不同）——改回跟程式碼同一套：片名 `$text-primary`、原名與 meta `$text-secondary`。
   - hero 狀態徽章拆成兩顆：`已入庫`（生命週期）與 `繁中`（字幕——**程式碼的標籤是 `繁中`，不是 `繁中字幕`**，`libraryStatus.ts:132,142`）；B4p-D 的 `已入庫 · 4 季全 · 繁中` 一樣拆。**顏色以 `deriveLifecycleStatus`／`deriveSubtitleStatus` 回傳的 class 為準，不要憑印象。**
   - 改完**重拍 `Light · B3p-D 詳情`（`m3N3ng`）**，片名與 meta 在日巡下讀得到才算完成。
   - 收單時在 `disc-2026-09-text-on-artwork-flips-with-theme` 補一行「Flow B 已收」（該條目剩 Flow F／L／Docs）。

6. **返回鍵與海報縮圖（`DetailHeroV2.tsx`）。** 對照表 #7、#8。
   - 返回鍵 hover：底與字必須**一起翻**或**一起固定**。建議 `hover:bg-[var(--bg-secondary)] hover:text-[var(--text-primary)]`（跟 `DetailNotFoundV2` 的返回鍵同一組）。兩個主題下 hover 態的圖示對比都要 ≥ 3:1。
   - 拿掉海報縮圖的 `shadow-[var(--shadow-xl)]`。

7. **圖片載不出來時不顯示破圖。** 對照表 #9。
   - `DetailHeroV2` 的背景圖與海報 `<img>` 各加 `onError`，失敗後走**已經存在的**漸層分支（`filenameToGradient(title)`，不要另寫一套）。
   - 🚨 **失敗狀態要跟著網址重置。** 推薦卡片連到 `/media/$type/<tmdbId>`（`PosterCard.tsx:135`），會**重用同一個** `TMDbDetailV2` → `DetailHeroV2` 實例；用一個 `useState(false)` 當旗標，上一部片的圖壞掉，下一部片也會永遠顯示漸層。存「失敗的那個網址」並跟目前網址比對，或在 `<img>` 上用 `key={src}`。
   - 兩個漸層分支加 `data-testid`（`detail-backdrop-fallback`／`detail-poster-fallback`；目前背景那個完全沒有、海報那個只有 `aria-hidden`）。
   - 測試：對 `<img>` 觸發 `error` 後漸層出現、`<img>` 消失（背景圖與海報各一條）；**換一個網址 rerender 後 `<img>` 回來**（一條）。
   - **設計稿 B9-D**：標題 `Spec — MediaDetailPanel 圖片載入 Fallback (case B)` → 改指 `DetailHeroV2`；刪掉 `playBtn`（`w0uk3`）與 `addListBtn`（`dF14N`）——Vido 沒有播放路徑（`LocalDetailV2.tsx:10-14`）；最後一行「實作對應」改成 `DetailHeroV2`。
   - 收單時關掉 `disc-2026-09-poster-fallback-purple-gradient`：紫色漸層**只剩**死元件 `MediaDetailPanel.tsx:22`，活的詳情頁與 B9-D 規格早已是片名雜湊漸層。

8. **伺服器出錯時，不要說「找不到這部影片」。** 對照表 #10、#11、#12。**這條前後端都要改，只改前端做不出來。**
   - **後端**：`MovieHandler.GetByID`（`movie_handler.go:99-103`）與 `SeriesHandler.GetByID`（`series_handler.go:112-116`）只在 `errors.Is(err, sql.ErrNoRows)` 時回 404，其他錯誤回 500。repository 已經用 `%w` 包了 `sql.ErrNoRows`（`movie_repository.go:110-111`、`series_repository.go:110-111`）；同檔已有現成寫法（`series_handler.go:388`）。`movie_handler_test.go:322` 與 `series_handler_test.go:213` 現在 mock 的是 `errors.New("not found")` 並期待 404——改成 mock `fmt.Errorf("…: %w", sql.ErrNoRows)`，**再各加一條「其他錯誤 → 500」**。TMDb 的 handler 已經分得對（`tmdb_handler.go:802-821`），不用改。
   - **前端服務層**：`libraryService.ts:24-39` 與 `tmdb.ts:48-62` 的 `fetchApi` 改丟**帶 `status` 與 `code` 的具名錯誤類別**（仍 `extends Error`、`message` 不變，既有呼叫端不受影響；先例：`requestService.ts:43` 的 `RequestApiError`、`scannerService.ts:59` 的 `ScannerApiError`）。
     - 🚨 **這個類別要放在不會被 mock 的模組**（例如 `lib/apiError.ts`）。`routes/media/-$type.$id.spec.tsx:61` 用 `vi.mock` 整個替換掉 `services/tmdb`——類別若定義在 `tmdb.ts`，在那個測試裡 import 進來是 `undefined`，`instanceof` 直接丟 TypeError。
   - **前端畫面**：`LocalDetailV2:133` 與 `TMDbDetailV2:52-53` 只在 `status === 404`（或查無資料）時顯示 `DetailNotFoundV2`；其他錯誤顯示新元件 `DetailLoadErrorV2`（放在 `DetailStatesV2.tsx`，`data-testid="detail-load-error"`，props：`onRetry`、`onBack`、`reassureFiles`）。
   - **載入失敗的文案**，照 dsr-1 已出貨的 `LibraryErrorV2`（`LibraryStatesV2.tsx:101` 起）的形狀：
     - 標題 `無法載入這部影片`；
     - `reassureFiles` 為真時（`LocalDetailV2`）加一句 `詳情資料查詢失敗，你的檔案沒有受影響。`，用 `text-secondary`，**不要用紅色**（dsr-1 CR 第 4 項）；`TMDbDetailV2` 的片不在媒體庫裡，這句會誤導，**不顯示**；
     - 有 `code` 時顯示獨立的等寬錯誤碼膠囊（同 dsr-1）；
     - 按鈕 `重試` ＋ `返回媒體庫`。
   - 找不到的說明改成稿的 `這個項目可能已從媒體庫移除，或連結已失效。`
   - route 層的 `NotFoundComponent`（`$type.$id.tsx:45-65`）改用 `DetailNotFoundV2`，同一件事只剩一套說法。
   - **既有測試會跟著變，這是預期的**（AC #15 點名）：`LocalDetailV2.spec.tsx:263-274`（`isError: true` 沒給錯誤物件 → 期待找不到；改成給 404 錯誤，另加一條 500 → 載入失敗）、`routes/media/-$type.$id.spec.tsx:567-587`（這是**真的** `TMDbDetailV2` 渲染，用 `new Error('TMDB_TIMEOUT')` 期待找不到——改成期待載入失敗，另加一條 404）、`tests/e2e/media-detail.spec.ts:252-262`（斷言 `404` 與 `找不到該媒體內容`）、`tests/e2e/media-detail.spec.ts:277-289`（`getByRole('button', { name: '返回媒體庫' })`——`DetailNotFoundV2` 有**兩顆**同名按鈕（`DetailStatesV2.tsx:41` 的 aria-label 與 `:61` 的文字），route 404 改用它之後會變成 Playwright strict mode 錯誤；把 locator 限定在 `detail-not-found` 裡）。
   - **順帶修好 dsr-1 的錯誤碼膠囊（lane ①）**：`LibraryBrowseV2.tsx:663` 讀 `error.code`，但現在服務層丟的 `Error` 沒有 `code`，**dsr-1 出貨的錯誤碼膠囊在正式環境從來沒顯示過**（它的測試是手動塞 `code` 的，`LibraryBrowseV2.spec.tsx:130`）。服務層改完之後它會開始顯示——確認顯示正確，不要當成意外。
   - **設計稿 B6p-D** 加一段註記說明「載入失敗」變體何時出現與它的文案（照 dsr-1 在 A1p-D 加三態註記的做法，不另開畫面）。

9. **區塊順序與標題。** 對照表 #13–#19、#22、#23。
   - **電影**：`檔案資訊` 移到 `演員` 之後。**影集順序不動**（簡介 → 季與劇集 → 檔案資訊 → 演員）。⚠️ 這表示 `DetailTechInfoV2` 在電影與影集要渲染在不同位置，現在是 `LocalDetailV2.tsx:256-265` 的單一 JSX 區塊——拆成依 `isMovie` 決定位置。
   - **延伸區塊**：`RelatedContent` 移到 `DoubanSection` 前面（`LocalDetailV2:282-298`）。
   - **`TMDbDetailV2`**：現在是預告片 → 觀看平台 → 演員（`:110-123`）；它的檔頭會指向 B3p-D，所以**演員移到最前面**（簡介之後），其餘照 B8p-D 的順序。
   - 新增一條 **DOM 順序測試**（電影、影集、TMDb 各一），現在沒有任何測試守順序。
   - `CreditsSection` 加 h2 `演員 / 製作`（`TMDbDetailV2` 也吃同一個元件，一起受益）；裡面的 `導演`／`創作者`／`演員陣容` 保留為 h3。**不要重排演員列的結構**，稿與碼的排法差異記進 Dev Notes 就好。
   - `StreamingAvailability` 標題 `可在哪裡觀看` → `觀看平台`。
   - `DoubanSection` 標題 `豆瓣評論` → `豆瓣`。
   - `EpisodeList:169` `分鐘` → `分`。
   - 三處半形逗號 `,` → `，`。
   - **設計稿 B8p-D**：`相似推薦` → `相關推薦`；豆瓣區塊的「暫時無法取得＋重試」改成只有一行「查看豆瓣頁面」連結（對上 2026-08-30 的裁定）。
   - 先改測試讓它紅：`grep -rn "可在哪裡觀看\|豆瓣評論\|分鐘\|找不到該媒體內容\|無法載入\|大小" apps/web/src tests/`，逐一確認會被影響的斷言。

10. **影集分集區：設計稿跟上程式碼。** 對照表 #20、#21。
    - B4p-D 的分集區改畫成手風琴（季列＋展開的集列），標題 `季與劇集`；刪掉 `查看全部 25 集 →`（文字節點 `L108tN` 與它的容器）——沒有那一頁。
    - 每集的字幕狀態 `st` 節點（文字分別是 `AK0KY`／`a7B8VW`／`i0MRqo`）從文字藥丸改成 J2-D 的圖示語彙（先讀 J2-D 規格再畫）。
    - 收單時關掉 `backlog-episodelist-status-pill-vs-icon-drift`（本張 AC #10 吸收，lane ①）。

11. **兩張 v1 手機抽屜稿刪掉。** 依 ⚖️ 2026-09-10 Alexyu 的既有判準「有明確後繼版本才刪」：B8-M（`6OR3z`）與 B9-M（`jH6rM`）畫的是 v1 側邊抽屜，後繼版本是 B3p-M（手機詳情）＋改指後的 B9-D（同一個元件、兩個斷點）。
    - 刪節點、`SCREENS` 移除兩筆、`git rm` 兩張 PNG。
    - ⚠️ 刪之前先確認沒有其他畫面 instance 或引用這兩個節點的子節點。

12. **⚖️ 兩個「⋯」選單：稿留著，只拿掉「匯出中繼資料」（Alexyu 2026-09-16 裁定 B）。** 設計稿畫了兩個選單（B2-D／B2-M 海報卡選單、B5-D／B5-M 詳情頁選單），但**正式環境沒有任何地方渲染它們**——`DetailPanelMenu` 只掛在死元件 `MediaDetailPanel` 裡，`PosterCardMenu` 只掛在死元件 `LibraryGrid` 裡，`PosterCardV2` 與 `LocalDetailV2` 都沒有選單。
    - **稿**：四張稿的 `匯出中繼資料` 項目拿掉（文字節點 `SEDGA`（B2-D）／`ohU0V`（B5-D）／`Vk127`（B2-M）／`FPihz`（B5-M）與它們的列容器），因為匯出已經搬到「設定 → 匯出／匯入」。拿掉後選單外框若是固定高度要跟著縮，並用 `ctx.problems` 掃一次。其餘項目（查看詳情／搜尋字幕／重新解析／刪除）**不動**。
    - **碼**：本張**不做**選單功能。已另立 `disc-2026-09-detail-and-poster-action-menus` 追蹤把選單做進 `LocalDetailV2` 與 `PosterCardV2`。
    - `DetailPanelMenu.tsx` 檔頭不動（見 AC #3）。

13. **推薦區卡片不動，立案。** `RelatedContent` 用 v1 `PosterCard`（TMDb 片、帶想要按鈕），稿畫 `PosterCard-v2`（媒體庫片、帶狀態徽章）。兩張卡片服務的資料不同，換卡是設計決定，不是對齊。已立 `disc-2026-09-related-content-card-v1-vs-v2`。

14. **補視覺夾具，走「功能分支刻意改基準」的流程。** 目前**沒有任何夾具**渲染 `DetailHeroV2`／`DetailTechInfoV2`／`DetailStatesV2`——本張改的每一個像素都零覆蓋（dsr-1 Completion Notes 的 reviewer 提醒：「`LibraryStatesV2` 與 `LibraryBrowseV2` 根本沒有夾具，兩處文案改寫零像素覆蓋」）。
    - 在 `routes/test/-gallery.fixtures.tsx` 新增：
      - `media-detail-hero-v2`：`backdropPath`／`posterPath` 都給 `null`（走漸層分支，像素穩定，不要打網路——見 `backlog-flaky-visual-media-detail-panel-backdrop` 的根因）、兩顆徽章、meta、動作列。
      - `media-detail-tech-info-v2`：四顆徽章＋三列事實。
      - `media-detail-not-found-v2` 與 `media-detail-load-error-v2`。
    - **兩個既有夾具會出現像素差異，這是預期的**：`media-credits-section`（`-gallery.fixtures.tsx:1060`，多了 h2）、`media-episode-list-subtitle-entry`（`:4007`，`24 分鐘` → `24 分`）。
    - **基準線流程**（`.claude/memory/project_visual_baseline_intentional_change.md`，CLAUDE.md 那句「合 bootstrap PR 就好」只適用 main）：
      1. 本機先 `nx serve web`，再 `CI=1 npx playwright test --project=visual --update-snapshots=all` 重生 `-darwin`。一定要 `=all`——一個字的改動可能低於 0.001 門檻而不被重寫（`project_visual_update_snapshots_threshold.md`）。新夾具的 `-darwin.png` **要 commit**（repo 裡每個夾具都有 darwin＋linux 兩張）。
      2. `git rm` 上面兩個既有夾具的過期 `-linux.png`，讓「像素差異」變成「缺少」——bootstrap 只處理缺少的。
      3. 把 main 併進功能分支。
      4. `gh workflow run "Visual Regression" --ref <branch>`（PR 事件不會跑產基準的 job），合掉開回本分支的 bootstrap PR。
      5. `--update-snapshots=all` 會順手重生其他漂移的 darwin 圖——只留本張真正改動的，其餘 `git checkout --` 還原。
    - ⛔ **不要在本機產 `-linux.png`。**

15. **既有測試保留通過，但以下是本張刻意改動、要跟著改的**（其餘一律不准改斷言）：
    - `DetailTechInfoV2.spec.tsx:6-25`（字幕藥丸 → 字幕軌列，AC #4）
    - `LocalDetailV2.spec.tsx:263-274`、`routes/media/-$type.$id.spec.tsx:567-587`（找不到 vs 載入失敗，AC #8）
    - `tests/e2e/media-detail.spec.ts:252-262`、`:277-289`（route 404 收斂、locator 限定範圍，AC #8）
    - `movie_handler_test.go:322`、`series_handler_test.go:213`（mock 改用 `sql.ErrNoRows`，AC #8）
    - AC #9 grep 出來的文案斷言
    特別守住：`LocalDetailV2.spec.tsx` 其他 13 條（含「沒有播放按鈕」）、e2e 的返回與直連。**e2e 那 4 條 `test.skip`（3 條引用 `disc-2026-07-v2-detail-fallback-states`、1 條等 parse_status 種子資料）維持 skip**，它們屬 dsr-2b。

16. **CI 全綠**：`pnpm run lint:all`、`pnpm nx run web:typecheck --skip-nx-cache`、`python3 scripts/check-design-tokens.py`、`pnpm nx test web`、`pnpm nx test api`。⛔ 絕不用 `run_in_background` 跑測試。⛔ 用 `nx test web`，不要自己換 `--root`（dsr-1 的假警報）。

## Tasks / Subtasks

- [x] **Task 1 — 先讓稿讀得到（AC: #1）**
  - [x] `READABLE_FLOWS` 加兩個 Flow B 資料夾，附日期與理由
  - [x] 設計稿內容全程用 Pencil MCP 直讀；匯出併到最後一次跑

- [x] **Task 2 — 22 個檔頭（AC: #2, #3）**
  - [x] 已掛載的 15 個（表格照抄）
  - [x] 夾具專用的 7 個（`DetailPanelMenu` 不動，AC #12 裁定 B）
  - [x] `npx eslint apps/web/src/components/media/ apps/web/src/components/metadata-editor/` → **0 errors**（lint 綠 ≠ 節點存在）
  - [x] 收尾自檢：檔頭裡的每個 `disc-*`／`dsr-*` 都在 sprint-status 找得到

- [x] **Task 3 — 技術徽章與字幕軌（AC: #4）** P0
  - [x] 匯出 `trackLangs`，寫字幕軌對照（先紅測試）
  - [x] 改 `DetailTechInfoV2`，改寫 `:6-25` 那條測試
  - [x] 設計稿 `tbb`／`tb` 節點圓角改藥丸、字幕軌值改成程式碼的寫法

- [x] **Task 4 — hero（AC: #5, #6, #7）**
  - [x] 程式碼：返回鍵 hover、拿掉縮圖陰影、兩個 `<img>` 的 `onError`（跟著網址重置；先紅測試，含換網址那條）
  - [x] 設計稿：五個漸層節點、B3p-M 文字 token、狀態徽章拆兩顆（字幕標籤 `繁中`）、B9-D 改指＋刪兩顆按鈕
  - [x] 截圖 `m3N3ng` 確認日巡讀得到
  - [x] ⚠️ 全程 Pencil MCP；每改完一張用 `ctx.problems` 掃一次裁切

- [x] **Task 5 — 後端：404 只給真的找不到（AC: #8）** ← 唯一的後端 task
  - [x] 先改兩條 handler 測試＋各加一條 500 讓它紅
  - [x] `MovieHandler.GetByID`／`SeriesHandler.GetByID` 用 `errors.Is(err, sql.ErrNoRows)` 分流 → 綠

- [x] **Task 6 — 前端：找不到 vs 載入失敗（AC: #8）**
  - [x] `lib/apiError.ts`（不被 mock 的位置）＋ `libraryService`／`tmdb` 的 `fetchApi` 改丟它
  - [x] 紅測試（404 → 找不到、500 → 載入失敗；Local 與 TMDb 各一）→ `DetailLoadErrorV2` ＋ 兩個容器 → 綠
  - [x] route 層 404 改用 `DetailNotFoundV2`，更新兩條 e2e
  - [x] 確認 dsr-1 的錯誤碼膠囊在媒體庫頁正確顯示
  - [x] B6p-D 加註記

- [x] **Task 7 — 區塊順序與文案（AC: #9）**
  - [x] 先 grep 受影響的斷言、改成新文案讓它紅；新增 DOM 順序測試（電影／影集／TMDb）
  - [x] 程式碼：兩種順序、延伸區塊順序、`TMDbDetailV2` 順序、`演員 / 製作` h2、`觀看平台`、`豆瓣`、`分`、三處逗號
  - [x] 設計稿 B8p-D：`相關推薦`、豆瓣區塊

- [x] **Task 8 — 設計稿分集區（AC: #10）**
  - [x] 讀 J2-D 圖示語彙
  - [x] B4p-D 分集區改手風琴、刪「查看全部」、`st` 改圖示

- [x] **Task 9 — 刪兩張 v1 抽屜稿（AC: #11）**
  - [x] 確認無引用 → 刪節點 → `SCREENS` 移除 → `git rm` PNG

- [x] **Task 10 — 兩個選單拿掉「匯出中繼資料」（AC: #12）** ⚖️ 已裁定 B
  - [x] 四張稿各刪一項，縮外框、掃裁切

- [x] **Task 11 — 視覺夾具與基準線（AC: #14）**
  - [x] 四個新夾具，全部不打網路
  - [x] 照 AC #14 的五步走基準線流程——本機能做的步驟 1／2／5 已完成（`--update-snapshots=all` 重生 darwin、只保留 6 個夾具、`git rm` 4 張過期 `-linux`）
  - [x] 步驟 3／4（併 main、`gh workflow run "Visual Regression" --ref fix/dsr-2-flow-b-detail`）**必須先推分支才能跑**，交給 /ship 的開 PR 階段——已寫進 sprint-status 的 dsr-2 條目

- [x] **Task 12 — 收尾（AC: #15, #16）**
  - [x] 全套閘門
  - [x] 重跑匯出；Flow B 兩個資料夾 ＋ `design-system`（`m3N3ng` 那張）stage；其他 flow 位元相同，無須還原
  - [x] 關掉 AC #2／#4／#5／#7／#10 點名的條目，更新 `disc-2026-09-unmounted-v1-components` 的清單

## Dev Notes

### 這張的重點

- **AC #4 是 P0，而且原條目指錯檔案。** 改 `TechBadge.tsx` 使用者什麼都看不到——那個元件沒掛載。真正的徽章在 `DetailTechInfoV2`。
- **AC #8 是最有感的一條。** 只要後端出錯（例如資料庫忙碌），使用者點進一部片就會讀到「找不到這部影片，可能已被移除」——**這是在說謊，而且是會嚇人的那種謊**。而且謊從後端就開始了：handler 對任何錯誤都回 404，前端再怎麼分也分不出來。dsr-1 在媒體庫頁修的是同一件事——結果這次還查到 dsr-1 的錯誤碼膠囊在正式環境從來沒出現過，原因是同一個（服務層丟掉了 `code`）。
- **AC #7 是第二有感的一條。** 只要 TMDb 圖片網址載入失敗（404、或 NAS 暫時連不到圖片 CDN），現在就是瀏覽器破圖。修法很便宜：漸層分支本來就在，只是沒接 `onError`。

### 不要做的事

- **不要改 `TechBadge.tsx`、`MediaDetailPanel.tsx` 等 v1 死元件的樣式或邏輯**——只改檔頭。刪不刪是 `disc-2026-09-unmounted-v1-components` 的裁定。
- **不要把 hero 漸層改成寫死的深色、也不要在程式碼加 `text-on-scrim`**——程式碼的做法是對的（AC #5 是改稿）。
- **不要做 B6-M／B7-M 的比對失敗與正在比對狀態**——那是 `dsr-2b`。
- **不要重排 `CreditsSection` 的結構、不要把 `RelatedContent` 換成 `PosterCardV2`**（AC #9、#13）。
- **不要動 `text-[13px]` 與 `text-[11px]`**（對照表 #27、#28 的判例）。
- **不要動 `ManageSubtitleDialogV2` 與字幕相關元件**——雖然從詳情頁打開，但屬 Flow F ＝ `dsr-6`。它有自己的軌道解析器（`:71-126`），**不要**為了 AC #4 去改它或從它匯出。
- **不要動 `MediaGrid.tsx`**——屬 Flow I ＝ `dsr-8`。
- **不要做選單功能、不要把 `DetailPanelMenu`／`PosterCardMenu` 接回任何頁面**——那是 `disc-2026-09-detail-and-poster-action-menus`（AC #12 裁定 B）。B2／B5 四張稿只拿掉「匯出中繼資料」。
- **不要把 `chi` 猜成繁中**（AC #4）。
- **不要把 `useDoubanReviewSummary` 接回來**——`LocalDetailV2.tsx:93-94` 刻意傳 `false`（2026-08-30 裁定）。

### sprint-status 條目已過期的四處

- 「① `TechBadge.tsx:15-18`」——**死碼**，真正的位置是 `DetailTechInfoV2.tsx:28-34`。
- 「② `MediaDetailPanel.tsx` 要裁定刪除還是標記為死碼」——已併入 `disc-2026-09-unmounted-v1-components`（dsr-1 立），本張只改檔頭。
- 「③ `PosterCard.tsx`（v1）3 處舊字級」——三處都是 `text-[11px]`，⚖️ 2026-09-16 已裁定 11px 是標準，作廢。
- 「④ 播放與加入片單，程式碼端確認沒有反向漂移」——**確認過**：兩顆按鈕只在 `MediaDetailPanel.tsx:256-274`（死碼，只有夾具會傳 `onPlay`）；但**設計稿 B9-D／B9-M 還留著**，AC #7、#11 處理。

### 已知陷阱

- **`$type.$id.tsx` 在 `routes/`，不在 `components/`**——Rule 21 檔頭不適用，不要加。
- **`routes/media/-$type.$id.spec.tsx` 是混合檔。** 裡面的「Media Detail Route」區塊測的是測試內手組的 `SidePanel + MediaDetailPanel` 假環境（改 `LocalDetailV2` 不會讓它紅）；但 `:567-587` 那段是**真的** `TMDbDetailV2` 渲染，AC #8 會讓它紅。另外 `:61` 把整個 `services/tmdb` mock 掉——見 AC #8 的 🚨。
- **`backlog-visual-main-4-workers-flaky-media-detail-focus` 的 flaky 夾具是 `media-media-detail-panel`（死元件）。** 若 CI 在那張紅，先查是不是那個已知 flake，不要以為是本張造成的。
- **B3p-D 的 `act-copypath`（`P1E87`）在桌機是純圖示、手機是圖示＋文字**，程式碼 `LocalDetailV2.tsx:187-202` 已經對上，不用動。
- **行號以本單建立時（2026-09-16，main `6fb735a2`）為準**，動手前重新確認。

### Source tree

```
apps/api/internal/handlers/movie_handler.go                 ← Task 5
apps/api/internal/handlers/series_handler.go                ← Task 5
apps/api/internal/handlers/{movie,series}_handler_test.go   ← Task 5
apps/web/src/lib/apiError.ts                                ← Task 6（新檔）
apps/web/src/services/libraryService.ts                     ← Task 6
apps/web/src/services/tmdb.ts                               ← Task 6
apps/web/src/utils/libraryStatus.ts                         ← Task 3（只匯出 trackLangs）
apps/web/src/components/media/DetailTechInfoV2.tsx          ← Task 2, 3（主要）
apps/web/src/components/media/DetailHeroV2.tsx              ← Task 2, 4（主要）
apps/web/src/components/media/DetailStatesV2.tsx            ← Task 2, 6（主要，新增 DetailLoadErrorV2）
apps/web/src/components/media/LocalDetailV2.tsx             ← Task 6, 7
apps/web/src/components/media/TMDbDetailV2.tsx              ← Task 2, 6, 7
apps/web/src/components/media/CreditsSection.tsx            ← Task 2, 7
apps/web/src/components/media/StreamingAvailability.tsx     ← Task 2, 7
apps/web/src/components/media/DoubanSection.tsx             ← Task 2, 7
apps/web/src/components/media/RelatedContent.tsx            ← Task 2, 7（只改逗號與檔頭）
apps/web/src/components/media/SeasonAccordion.tsx           ← Task 2, 7（只改逗號與檔頭）
apps/web/src/components/media/EpisodeList.tsx               ← Task 2, 7（只改「分鐘」與檔頭）
apps/web/src/components/media/{TrailerSection,TrailerEmbed,DualRatingDisplay,AvailabilityBadge}.tsx ← Task 2
apps/web/src/components/media/{MediaDetailPanel,MetadataSourceBadge,TVShowInfo,FallbackFailed,FallbackPending}.tsx ← Task 2（只改檔頭）
apps/web/src/components/metadata-editor/{MetadataEditorDialog,CastEditor,PosterUploader}.tsx ← Task 2（只改檔頭）
apps/web/src/routes/media/$type.$id.tsx                     ← Task 6
apps/web/src/routes/test/-gallery.fixtures.tsx              ← Task 11
tests/e2e/media-detail.spec.ts                              ← Task 6
tests/visual/components.visual.spec.ts-snapshots/components/** ← Task 11
ux-design.pen（uRGu2/N2fmG6/Z42zy/UH0sk/SzNRb/Tn4Gz/7mdTJ/APfjC/auArc/1UHzI/m3N3ng/6OR3z/jH6rM）← Task 3, 4, 6, 7, 8, 9, 10
scripts/export-pen-screenshots.py                           ← Task 1, 9
```

### Cross-Stack Split Check

後端 task **1 個**（Task 5）、前端／設計 task 約 10 個。後端 ≤ 3 → **不拆**。

### Time-dependent visual coverage

- **Does this story add/modify any `apps/web/src/components/**/*.{ts,tsx}` that reads the wall clock?**
  - **N/A — no wall-clock-reading components touched.** 本張會改的元件都不讀時鐘。唯一讀時鐘的 `MetadataEditorDialog.tsx`（`new Date().getFullYear()`，第 1 行已標 `Time-bomb-exempt`）本張只改第 2 行檔頭，不動時間邏輯。`FallbackFailed.tsx:94`、`TVShowInfo.tsx:31` 的 `new Date(x)` 是解析 prop，不是讀時鐘，而且只改檔頭。AC #14 的四個新夾具都不含日期。
- Reference: `project-context.md` Rule 23。

### References

- [Source: `ux-design.pen` Flow B 16 張（表格）＋ `m3N3ng`] — Pencil MCP 逐節點讀出；`Light · B3p-D 詳情` 已截圖確認片名在日巡下讀不到
- [Source: Pencil MCP 節點存在性驗證] — `RgSxQ`／`2ltBl`／`wQOkg`／`407vK`／`KNI8F` 查無；`G0xib`（L8-D-v2）／`Z54xAd`（J3-D）存在；全檔搜「編輯媒體資訊」「已有」查無對應畫面
- [Source: `apps/web/src/components/media/DetailTechInfoV2.tsx:28-34, 46, 65-69, 76-86`] — AC #4
- [Source: `apps/web/src/types/library.ts:47,112`、`apps/api/internal/services/ffprobe_service.go:37-46, 207-210, 276-282`、`apps/web/src/utils/libraryStatus.ts:84-99, 132, 142`] — 字幕軌資料形狀、`trackLangs`、字幕徽章標籤
- [Source: `apps/web/src/components/media/DetailHeroV2.tsx:47-55, 64, 72-87`、`PosterCard.tsx:135`] — AC #5／#6／#7（含實例重用）
- [Source: `apps/api/internal/handlers/movie_handler.go:99-103`、`series_handler.go:112-116, 388`、`movie_repository.go:110-111`、`series_repository.go:110-111`、`tmdb_handler.go:802-821`] — AC #8 後端
- [Source: `apps/web/src/services/libraryService.ts:24-39`、`tmdb.ts:48-62`、`requestService.ts:43`、`scannerService.ts:59`、`routes/media/-$type.$id.spec.tsx:61, 567-587`、`LibraryBrowseV2.tsx:663`] — AC #8 前端
- [Source: `apps/web/src/components/media/LocalDetailV2.tsx:10-14, 93-94, 133, 187-202, 256-265, 282-298`、`TMDbDetailV2.tsx:52-53, 110-123`] — 沒有播放鈕、豆瓣短評關閉、`isError` 一律找不到、區塊順序
- [Source: `apps/web/src/components/media/DetailStatesV2.tsx:35-66`、`routes/media/$type.$id.tsx:45-65`、`tests/e2e/media-detail.spec.ts:252-262, 277-289`] — AC #8 兩套 404
- [Source: `apps/web/src/components/media/DoubanSection.tsx:28-30, 74-82`] — 標題承諾評論、實際只剩連結
- [Source: `DESIGN.md` §Shapes「新規則：看語意，不看高度」] — TechBadge＝藥丸
- [Source: `_bmad-output/implementation-artifacts/ux2-3-detail-v2.md` AC #4（區塊順序）、AC #7（找不到／載入失敗要分開）] — 本張 AC #8／#9 的原始規格
- [Source: `_bmad-output/implementation-artifacts/dsr-7-flow-h-homepage-v3.md` AC #3／#4] — hero 漸層改 `$bg-primary` 的判例
- [Source: `_bmad-output/implementation-artifacts/dsr-1-flow-a-browse-v2.md` 對抗式 CR 第 1／3／4／14 項、Completion Notes 的 reviewer 提醒] — 檔頭誠實判準、安慰句不用紅色、零像素覆蓋
- [Source: `.claude/memory/project_visual_baseline_intentional_change.md`、`project_visual_update_snapshots_threshold.md`、`.github/workflows/visual-regression.yml`] — AC #14 基準線流程
- [Source: project-context.md#Rule 21 / #Rule 23 / #Rule 24、`apps/web/src/eslint-rules/implements-pen-node-id.js`]
- [Source: `sprint-status.yaml` → `epic-dsr` / `dsr-2-flow-b-detail-v2` / `dsr-2b-flow-b-no-metadata-states` / `disc-2026-09-techbadge-uses-status-colors-as-taxonomy` / `disc-2026-09-text-on-artwork-flips-with-theme` / `disc-2026-09-poster-fallback-purple-gradient` / `disc-2026-09-unmounted-v1-components` / `disc-2026-07-v2-detail-fallback-states` / `backlog-localdetailv2-rule21-header-grammar` / `backlog-episodelist-status-pill-vs-icon-drift` / `bugfix-douban-sec-gate-liveness` / `disc-2026-09-11px-micro-label-not-on-type-scale` / `disc-2026-09-type-scale-even-migration`]

## Dev Agent Record

### Agent Model Used

Claude Opus 5 (1M context) — `claude-opus-5[1m]`，BMAD dev agent（Amelia）2026-09-16

### Debug Log References

- `pnpm nx test web --skip-nx-cache`：**3429 / 3429 passed**（260 files；+26 新測試）
- `pnpm nx test api --skip-nx-cache`：PASS（handler 兩條「資料庫失敗 → 500」新測試先紅後綠）
- `pnpm run lint:all`：**0 errors**、128 warnings（全是既有的）；本張改過的 13 個 web 檔案單獨 eslint：**0 problems**
- `pnpm nx run web:typecheck --skip-nx-cache`：PASS · prettier：PASS · `check-design-tokens.py`：一致（190 張畫面、73 個母版）
- 視覺：`CI=1 npx playwright test --project=visual --update-snapshots=all`（8.8 分鐘）→ 186 張 re-render 雜訊全部還原，只留 6 個夾具
- 匯出：190 張，**只有 Flow B 16 張 ＋ `light-b3p-d` 變動**（Flow B 兩個資料夾這次改成 2x 可讀匯出，所以每張都會變；其他 flow 位元相同）
- ⚠️ **一次自找的假綠**：`LocalDetailV2.tsx` 我在同一個函式裡宣告了第二個 `const credits`（原本就有一個 `credits = isMovie ? movieCredits : tvCredits`）。`npx vitest run src/components/media/` 回報 403 全綠，直到跑路由那支 spec 時 esbuild 才報 `The symbol "credits" has already been declared`。改名 `creditsBlock`／`techInfoBlock` 後 typecheck 與全套都綠。教訓：局部 vitest 綠不代表檔案編得過，改容器後要跑 typecheck。
- ⚠️ `.pen` 匯出讀的是 Pencil 記憶體：改完先 `git status` 看到 `ux-design.pen` **沒變**（mtime 停在 16:15），照記憶檔的做法同值 `Update` 標髒後用 AppleScript 點 File › Save，確認 ` M ux-design.pen`（9,416,082 → 9,355,905 bytes，刪了兩張畫面）後才重跑匯出。

### Completion Notes List

- 🔗 **AC Drift: FOUND**（檢查 `grep -lnE "DetailTechInfoV2|detail-not-found|找不到該媒體內容|可在哪裡觀看|豆瓣評論|DetailHeroV2|找不到這部影片"` 全部 story 檔，5 個命中）
  - `bugfix-10-1-postercard-tmdb-id-404` Task 5.4 — 「TMDb `TMDB_TIMEOUT` → 顯示 404／找不到該媒體內容」→ **逾時顯示載入失敗，只有真的 404 才顯示找不到**（AC #8；該測試已翻轉並註明）
  - `12-4-streaming-platform-availability` AC #1 — 標題「可在哪裡觀看」→「觀看平台」（AC #9）；該 AC 的位置描述「overview 之下、credits 之上」早在 ux2-3 就已經不成立
  - `12-6-douban-integration` UX Note — 標題「豆瓣評論」→「豆瓣」（AC #9，2026-08-30 已停抓短評）
  - `ux2-3-detail-v2` AC #4／#7 — **REUSE**：本張讓程式碼回到它原本規定的區塊順序與「找不到／載入失敗分開」
  - `13-1b-one-click-request` — **REUSE**：只描述 `TMDbDetailV2` 的路由，未受影響
- 📎 **Contract Stamps: NONE**（本張與上述上游 story 都沒有 `[@contract-v*]`；`bugfix-10-1` 的 `[@contract-v1]` 蓋在 `classifyId` AC #2，本張沒有碰 `classifyId`）
- 🎭 **A11y Pre-Flight: PASS**（13 個 web 檔案 eslint 0 problems、0 introduced。四類回歸：① 圖片——`DetailHeroV2` 兩張 `<img>` 新增 `onError`，alt 不變；② modal——沒有新增 aria-modal；③ 非同步揭露——`DetailLoadErrorV2` 帶 `role="alert"`；④ 自訂 widget——無。另外：`CreditsSection` 補上 h2 之後，h3「導演／演員陣容」不再掛在前一個區塊的 h2 底下；`DetailLoadErrorV2` 用 h1，與 `DetailNotFoundV2` 一致。）
- ✅ **Pre-existing failures: NONE**（全套一次綠）
- **dsr-1 的錯誤碼膠囊**：`LibraryBrowseV2.tsx:663` 讀 `error.code`，服務層現在丟 `ApiError` 帶 `code`——由 `libraryService.spec.ts` 新測試（`rejects.toMatchObject({status, code})`）守住共用的 `fetchApi`。
- **字幕軌的誠實性**：`chi`／`zho` 不在繁／簡集合裡，所以大部分真實檔案的中文軌會顯示原始標籤 `chi`。這是刻意的（資料本來就分不出繁簡），測試 `keeps untranslatable tags raw (never guesses 繁中)` 守住。
- **B9-D 多拿掉一塊（lane ①，AC #7 的延伸）**：除了 AC 點名的播放／加入清單，右側示意圖的「字幕」區塊（搜尋字幕／AI 校正／轉錄英文音軌三顆 v1 按鈕）也刪了——它們在 v2 詳情頁不存在。示意圖其餘部分保留並加註「hero 其餘版面以 B3p-D 為準」；B9-D 高度 1160 → 961 收掉留白。
- **B4p-D 分集區**：只畫第 1 季展開的兩集＋第 2 季收合，剛好放進 900px 畫布；body 下緣 padding 改 0（頁面本來就會往下捲）。分集列的縮圖＋播放小圖示一併拿掉（程式碼的分集列沒有縮圖，也沒有播放）。
- **B2-M／B5-M**：刪掉「匯出中繼資料」後抽屜變矮，已把抽屜重新貼齊畫面底部。

#### 🎨 UX Verification（對照 Flow B 可讀版匯出）

| 區域 | 設計稿 | 實作 | 一致？ | 處置 |
| --- | --- | --- | --- | --- |
| 技術徽章 | 中性 `$bg-tertiary`／`$text-secondary`、藥丸 | 同 | ✅ | — |
| 檔案資訊三列 | 檔案大小／字幕軌／路徑，標籤 secondary、值 primary | 同 | ✅ | — |
| 字幕軌值 | `繁中 · 英文` | `繁中 · 英文`（zh-Hant＋eng） | ✅ | — |
| 音訊徽章 | `TrueHD 7.1` | `TrueHD 8ch` | ❌ | 立案 `disc-2026-09-audio-channels-format`（改它會動既有斷言） |
| hero 漸層 | `$bg-primary`，日巡讀得到 | `var(--bg-primary)` | ✅ | 稿已改 |
| hero 狀態徽章 | 兩顆：已入庫／繁中 | 兩顆 | ✅ | 稿已改 |
| 返回鍵 hover | — | 底與字一起翻 | ✅ | — |
| 海報縮圖陰影 | 無 | 無 | ✅ | — |
| 圖片載入失敗 | B9-D：片名雜湊漸層＋首字 | 同（夾具 `media-detail-hero-v2` 可見） | ✅ | — |
| 找不到 | B6p-D 文案 | 同 | ✅ | — |
| 載入失敗 | B6p-D 註記 | `DetailLoadErrorV2`（夾具可見） | ✅ | — |
| 電影區塊順序 | 簡介 → 演員 → 檔案資訊 | 同 | ✅ | — |
| 延伸區塊順序 | 預告片 → 觀看平台 → 相關推薦 → 豆瓣 | 同 | ✅ | — |
| 區塊標題 | 演員 / 製作・觀看平台・相關推薦・豆瓣・季與劇集 | 同 | ✅ | — |
| 豆瓣區 | 只有「查看豆瓣頁面」 | 同 | ✅ | 稿已改 |
| 分集區 | 手風琴＋圓框打勾＋管理字幕＋日期／`24 分` | 同 | ✅ | 稿已改 |
| 演員列排法 | 名字＋角色的一排卡片 | `導演`／`演員陣容` 兩組、3 欄格線 | ⏸️ | AC #9 明訂不重排 |
| 次要動作鈕底色 | `$bg-tertiary` | `bg-secondary`（hover `bg-tertiary`） | ⏸️ | 對照表 #25：待驗、不強改 |
| 骨架 | B7p-D | `DetailSkeletonV2` | ⏸️ | 對照表 #26：待驗、不強改 |
| 推薦卡片 | `PosterCard-v2` | v1 `PosterCard` | ⏸️ | AC #13，已立案 |

### Discovery Triage

- **Did this story discover any work outside its current scope?** **YES** —— 建單當下處理如下。

- **① expand-scope-in-place**
  - `backlog-episodelist-status-pill-vs-icon-drift` → **AC #10**（B4p-D 分集字幕狀態改圖示）
  - `backlog-localdetailv2-rule21-header-grammar` → **AC #2**（查證已被修掉，收單時關掉）
  - `disc-2026-09-poster-fallback-purple-gradient` → **AC #7**（紫色只剩死碼，收單時關掉）
  - **dsr-1 的錯誤碼膠囊在正式環境從未顯示** → **AC #8**（服務層帶上 `code` 後自然修好；建單時由對抗驗證查出）

- **② spawn-blocking-story**：無。

- **③ backlog-with-carry-forward-link**
  - **`dsr-2b-flow-b-no-metadata-states`** — 本張拆出。v2 詳情頁沒有「比對失敗」與「正在比對」狀態（`disc-2026-07-v2-detail-fallback-states` 的主體），稿是 v1 抽屜形狀要先重畫。
  - **`disc-2026-09-related-content-card-v1-vs-v2`** — AC #13。
  - **`disc-2026-09-request-button-requested-on-scrim-light-theme`** — `PosterCard.tsx:321` 在海報上鋪固定深色的 `--overlay-scrim`，上面的 `RequestButton` 已請求態用 `bg-[var(--info-tint)] text-[var(--info-text)]`（`RequestButton.tsx:116`）。日巡下 `--info-text` 是深色、tint 只有 20%——**深字壓深底**。詳情頁的「相關推薦」、首頁探索區、探索頁都會出現。屬 Flow L（`dsr-11` 已收），本張不修。
  - **`disc-2026-09-metadata-editor-no-design`** — 「修改資訊」對話框（`MetadataEditorDialog`，詳情頁主要動作之一）在設計稿裡**沒有任何畫面**，檔頭只能寫 no-screen 變體。
  - **更新 `disc-2026-09-unmounted-v1-components`**：清單從 7 個擴到 19 個（＋`media/` 的 `DetailPanelMenu`、`FileInfo`、`MetadataSourceBadge`、`TVShowInfo`、`TechBadge`、`TechBadgeGroup`、`FallbackFailed`、`FallbackPending`、`ColorPlaceholder` 元件本體；＋`metadata-editor/` 的 `CastEditor`、`PosterUploader`、`GenreSelector`）。其中 `FallbackFailed`／`FallbackPending` 可能被 `dsr-2b` 接回、`DetailPanelMenu`（以及 `library/PosterCardMenu`）可能被 `disc-2026-09-detail-and-poster-action-menus` 接回，裁定時要排除。
  - **更新 `disc-2026-09-dangling-design-refs-outside-library`**：`MediaGrid.tsx` 掛在探索與搜尋頁，歸 `dsr-8` 不是 `dsr-2`；另補 `components/degradation/DegradationBadge.tsx`（也指向已刪的 `RgSxQ`、無掛載點、沒有任何單子涵蓋）。
  - **`disc-2026-09-detail-and-poster-action-menus`** — AC #12 ⚖️ 裁定 B 立的功能單：把 B5 的詳情頁選單與 B2 的海報卡選單做進程式碼。
  - **`disc-2026-09-audio-channels-format`**（dev 時立）— 音訊徽章寫 `8ch`，稿寫 `7.1`。
  - **`dsr-6-flow-f-subtitle-v2` 附帶一則**（dev 時立）— `ManageSubtitleDialogV2` 的軌道語言對照不認 `eng`／`und`，同一條軌在詳情頁顯示「英文」、在對話框顯示 `eng`。

- Reference: `project-context.md` Rule 24

### File List

**後端：**
- `apps/api/internal/handlers/movie_handler.go`、`series_handler.go` — `GetByID` 只在 `sql.ErrNoRows` 回 404，其他 500 `DB_QUERY_FAILED`
- `apps/api/internal/handlers/movie_handler_test.go`、`series_handler_test.go` — mock 改 `%w sql.ErrNoRows`，各加一條 500
- `apps/api/internal/handlers/tmdb_handler.go`（＋`tmdb_handler_test.go`）— `handleTMDbError`／`handleValidationError` 改 `errors.As`，被包過的 TMDb 錯誤保留真實狀態（CR #1）

**前端（新增）：**
- `apps/web/src/lib/apiError.ts`、`apiError.spec.ts` — `ApiError`（status＋code）與 `isNotFoundError`

**前端（修改）：**
- `apps/web/src/services/libraryService.ts`、`tmdb.ts`（＋兩支 spec）— `fetchApi` 丟 `ApiError`
- `apps/web/src/utils/libraryStatus.ts` — 匯出 `trackLangs`
- `apps/web/src/components/media/DetailTechInfoV2.tsx`（＋spec）— 中性藥丸、字幕軌列、標籤
- `apps/web/src/components/media/DetailHeroV2.tsx`（＋spec）— `onError`、返回鍵 hover、拿掉陰影、檔頭
- `apps/web/src/components/media/DetailStatesV2.tsx`（＋spec）— `DetailLoadErrorV2`、找不到文案、檔頭
- `apps/web/src/components/media/LocalDetailV2.tsx`（＋spec）— 404／載入失敗分流、區塊順序
- `apps/web/src/components/media/TMDbDetailV2.tsx` — 同上＋檔頭
- `apps/web/src/routes/media/$type.$id.tsx`（＋`-$type.$id.spec.tsx`）— route 404 改用 `DetailNotFoundV2`；TMDb 錯誤與順序測試
- `apps/web/src/components/media/CreditsSection.tsx`、`StreamingAvailability.tsx`、`DoubanSection.tsx`、`EpisodeList.tsx`、`SeasonAccordion.tsx`、`RelatedContent.tsx`（＋各自 spec）— 標題、`分`、全形逗號、檔頭
- `apps/web/src/components/media/{TrailerSection,TrailerEmbed,DualRatingDisplay,AvailabilityBadge,MediaDetailPanel,MetadataSourceBadge,TVShowInfo,FallbackFailed,FallbackPending}.tsx` — 只改檔頭
- `apps/web/src/components/metadata-editor/{MetadataEditorDialog,CastEditor,PosterUploader}.tsx` — 只改檔頭
- `apps/web/src/routes/test/-gallery.fixtures.tsx` — 四個新夾具
- `tests/e2e/media-detail.spec.ts` — route 404 斷言、返回鍵 locator 限定範圍

**視覺基準線：**
- 新增 `tests/visual/components.visual.spec.ts-snapshots/components/{media-detail-hero-v2,media-detail-tech-info-v2,media-detail-not-found-v2,media-detail-load-error-v2}/default-visual-darwin.png`
- 修改 `media-credits-section/default-visual-darwin.png`、`media-episode-list-subtitle-entry/{default,hover,focus}-visual-darwin.png`
- 刪除 上述兩個夾具的 4 張 `-linux.png`（待 CI bootstrap 補回）

**設計與腳本：**
- `ux-design.pen` — 五個 hero 漸層與狀態徽章、技術徽章圓角與字幕軌值、B3p-M 文字 token、B4p-D 分集區、B6p-D 註記、B8p-D 標題與豆瓣區、B9-D 改指、B2-D／B2-M／B5-D／B5-M 拿掉匯出、刪 B8-M／B9-M 與兩個標籤、B3p-M 左移補洞
- `scripts/export-pen-screenshots.py` — `READABLE_FLOWS` ＋ Flow B 兩個資料夾；`SCREENS` 移除 b8-m／b9-m
- `_bmad-output/pen-tokens.json`、`_bmad-output/screenshots/flow-b-detail-v2/*`（6）、`flow-b-detail-interaction/*`（8 改、2 刪）、`design-system/light-b3p-d.png`
- `_bmad-output/implementation-artifacts/sprint-status.yaml` — 狀態流轉、關 4 案、更新 3 案、立 `disc-2026-09-audio-channels-format`
- `_bmad-output/implementation-artifacts/dsr-2-flow-b-detail-v2.md` — 本檔

## 對抗式 Code Review（/ship，2026-09-16）

獨立 reviewer（fresh context，只讀）回報 **15 項**。**修掉 13 項、2 項立案**。

### 🔴 最重要的一項：TMDb 那一半其實沒修好

| # | 問題 | 處置 |
| --- | --- | --- |
| 1 | **HIGH**：建單時驗證代理說「TMDb handler 已經分得對」——**錯的**。`handleTMDbError` 用型別斷言 `err.(*tmdb.TMDbError)`，但 TMDb client 先把錯誤包了一層（`fmt.Errorf("failed to get movie details: %w", err)`），斷言永遠不中，**所有真的 TMDb 404 都變成 500**。前端的「TMDb 404 → 找不到」測試會綠，只是因為它 mock 了一個假的 `ApiError(404)`。後果：失效的 TMDb 連結會顯示「無法載入」加一顆永遠不會成功的「重試」——比改之前還糟。 | 兩個 helper 改 `errors.As`；新增 handler 測試餵**包過一層與兩層**的 `NewNotFoundError`，先紅後綠。順帶讓 401／429／502 也終於帶著真實狀態出去 |

### 其他修掉的

| # | 問題 | 處置 |
| --- | --- | --- |
| 2 | 錯誤分支沒看 `data`：React Query 背景重新整理失敗時會**保留快取資料但 `isError` 為真**——生成字幕完成後 refetch 撞上資料庫忙碌，整頁（連同打開的對話框）會被換成「無法載入」 | 只在 `!data` 時才接管頁面；新增測試 |
| 3 | 「這個項目可能已從媒體庫移除」也會出現在 TMDb 片與網址錯誤的 404——它們從來不在媒體庫裡 | `DetailNotFoundV2` 加 `inLibrary`，TMDb 與 route 層傳 false，改說「這個連結可能已失效。」；測試 |
| 5 | 「重試」按下去沒有任何回饋，再失敗一次看起來像壞掉的按鈕 | `retrying={query.isFetching}`：停用並顯示「重試中…」；測試 |
| 7 | 註解與測試名稱說「字幕軌不猜繁中」，但共用的 HANT 集合包含裸 `zh` | 改寫註解與測試名稱，另加一條測試明文記錄 `zh → 繁中` |
| 8 | 註解說「推薦連結會重用 hero 實例」——只有下一部片已經在快取時才會 | 改寫成「已在快取時（例如返回上一頁）」 |
| 9 | 兩條測試對整個元件的 `innerHTML` 跑 regex，之後任何地方加一個 `shadow-none` 都會誤紅 | 加 `detail-poster-tile` testid，只看那個元素；字幕軌測試改看該列 |
| 10 | `MediaDetailPanel` 檔頭說「只有視覺夾具引用」，但路由 spec 也當測試環境引用（dsr-1 CR #13 同類） | 補上 |
| 11 | `AvailabilityBadge` 檔頭只說「已有」沒有稿，元件還會畫「已請求」 | 查過 .pen：唯一的「已請求」是 L3-D-v2 季選擇器裡的列藥丸，不是海報角標；檔頭改寫涵蓋兩者 |
| 12 | `StreamingAvailability` 文件註解還寫「可在哪裡觀看」 | 改「觀看平台」 |
| 13 | 只測了電影的 500，影集的 `localSeries.error`／`refetch` 沒有測試 | 新增影集 500 測試（含重試打到影集的 query） |
| 14 | 工作樹混有無關的未追蹤檔（記憶檔、coverage、testsprite 報告） | 只 stage 明確路徑 |
| 15 | TMDb 變體沒有 `reassureFiles` 時畫面只剩標題＋膠囊＋按鈕，一句話都沒有 | 補中性句「詳情資料暫時無法取得，請稍後再試。」；B6p-D 註記同步更新 |

### 立案（不在本張修）

| # | 問題 | 條目 |
| --- | --- | --- |
| 4 | `tmdb.ts` 開始帶出 `code` 後，探索頁的錯誤句第一次會顯示錯誤碼，而且是 dsr-1 否決過的句尾括號形狀 | 附在 `dsr-8-flow-i-discover-v2` |
| 6 | 真的 404 仍會重試一次，骨架多轉約 1 秒。試過在四個 hook 各自設 `retry`，但會蓋掉測試 QueryClient 的 `retry: false`，已還原 | `disc-2026-09-detail-404-still-retried` |

## Change Log

| 日期 | 內容 |
| --- | --- |
| 2026-09-16 | 🔴 **對抗式 CR 回報 15 項，修 13 立 2**。最重要的是第 1 項：**TMDb 那一半其實沒修好**——handler 用型別斷言判斷 TMDb 錯誤，但錯誤被包過一層，所以真的 TMDb 404 全部變成 500，前端測試會綠只是因為 mock 了假的 404。改 `errors.As` 並補包過一層／兩層的測試。另外：背景重新整理失敗不再把整頁換掉、TMDb 片的「找不到」不再提媒體庫、「重試」有進行中狀態、四處不實的註解改掉、兩條看整個 innerHTML 的脆弱測試改成看單一元素、補影集 500 測試。 |
| 2026-09-16 | ✅ **dev-story 完成 → review**（Amelia）。全套閘門綠：web 3429/3429、api PASS、lint 0 errors、typecheck、prettier、token 一致。最有感的三件：① **伺服器出錯不再說「找不到這部影片，可能已被移除」**——前後端一起改，順帶讓 dsr-1 的錯誤碼膠囊第一次真的出現；② **P0 技術徽章**改在真正會畫出來的 `DetailTechInfoV2`（原條目指的 `TechBadge.tsx` 是死碼）；③ **TMDb 圖片載入失敗不再是破圖**。設計稿改 12 張、刪 2 張；新增 4 個視覺夾具。途中一次假綠（同函式裡重複宣告 `credits`，局部 vitest 沒抓到、esbuild 才抓到）已記在 Debug Log。 |
| 2026-09-16 | ⚖️ **AC #12 裁定 B**（Alexyu）：兩個「⋯」選單的稿留著，只拿掉「匯出中繼資料」；選單功能另立 `disc-2026-09-detail-and-poster-action-menus`，本張不做。`DetailPanelMenu.tsx` 檔頭因此不動（夾具專用的檔頭從 8 個變 7 個）。 |
| 2026-09-16 | 🔍 **建單後對抗驗證**（fresh-context 驗證代理，只讀）：4 項 CRITICAL、10 項 SHOULD FIX，**全部併入**。最重要的是 AC #8 原本**做不出來**——後端 handler 對任何錯誤都回 404、前端服務層丟掉狀態碼，只改前端分不出 404 與 5xx；現在加了一個後端 task（不觸發拆單）。另外：AC #15「既有測試全保留」與 AC #8 自相矛盾（改成點名 6 處刻意改動的測試）；AC #14 的基準線步驟在功能分支上會卡死（改成記憶檔裡跑通過的五步流程，並點名兩個會出現像素差異的既有夾具）；字幕軌的資料推不出「繁體」（改用既有的 `trackLangs`、不猜）；圖片失敗狀態會被推薦連結帶到下一部片（要跟著網址重置）；檔頭表漏了 `CastEditor`／`PosterUploader`、`DetailHeroV2` 漏了 B9-D。附帶查到 **dsr-1 的錯誤碼膠囊在正式環境從來沒顯示過**，同一個根因，併入 AC #8。 |
| 2026-09-16 | Story 建立（SM Bob, create-story）。Flow B 16 張稿以 Pencil MCP 逐節點讀出，與 `components/media/` 30 個檔案＋詳情頁掛載鏈比對（背景代理盤點掛載狀態、檔頭、夾具、字級、陰影、寫死色、fallback）。**拆成 dsr-2（已出貨的詳情頁）／dsr-2b（比對失敗與正在比對兩個狀態）**。建單時查到條目沒寫的事：P0 技術徽章指錯檔案（`TechBadge.tsx` 是死碼）、hero 漸層是稿錯不是碼錯、Flow B 有一半的稿在畫已死的 v1 面板。一項待裁定（AC #12 兩個選單）。 |

## 裁定紀錄

### ⚖️ 2026-09-16 · 兩個「⋯」選單：稿留著，之後做進程式碼

設計稿有海報卡選單（B2）與詳情頁選單（B5），正式環境兩個都沒有，只活在視覺夾具裡。SM 建議把稿收回（這張是對齊單不是功能單）。

**Alexyu 選：B，稿留著、之後補做。** 本張只拿掉已經搬去設定頁的「匯出中繼資料」；選單功能另立 `disc-2026-09-detail-and-poster-action-menus`。

落點：AC #12 / Task 10；AC #3 的 `DetailPanelMenu` 檔頭不動。
