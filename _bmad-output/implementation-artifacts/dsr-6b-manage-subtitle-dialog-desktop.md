# Story DSR.6b：「管理字幕」對話框（桌機）對齊設計稿，順手修掉三個真的 bug

Status: review

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As the person who opens 「管理字幕」 on a movie or an episode,
I want 對話框、生成進度、生成失敗三個畫面長得跟設計稿一樣，而且畫面上的每一句話都是中文、都是實話,
so that 我看得懂現在在跑什麼、失敗在哪、重試要花多少錢，而且在分集裡加了名詞之後，條數會馬上跟著變。

## Context

`epic-dsr` 的 `dsr-6`（Flow F 字幕）拆出來的**第二張**（第一張 `dsr-6a` 已完成：付費按鈕帶金額）。⚖️ SM 2026-09-17 把 `dsr-6` 剩下的範圍依「畫面在哪」拆成五張：

| 單子 | 範圍 |
| --- | --- |
| **`dsr-6b`（本張）** | 管理字幕對話框桌機：F1-D-v2、F2-D-v2、F3-D-v2、F4-D-v2、F5-D-v2，以及母版 `Component/GenerationProgress-v2` |
| `dsr-6c` | 名詞對照表：F6-D-v2、F7-D-v2（要先裁定來源徽章顏色，見該條目） |
| `dsr-6d` | 批次生成＋生成工作區：F8、F9、F11–F13（桌機） |
| `dsr-6e` | 同意流程：F14–F20、f18-spec-err（桌機；含 F16 金額標籤、赭色付費徽章裁定） |
| `dsr-6f` | Flow F 所有手機稿：F1-M、F3-M、F6-M、F8-M、F11-M、F15-M、F16-M、F19-M |

F21–F27（自動處理政策）**不是對齊**——功能還沒做、J8-D 有三個待 Alexyu 裁定的問題——另立 `backlog-consent-policy-implementation`。F10（載入骨架）在正式環境到不了，另立單（🔴 #7）。

⚠️ **驗收基準是 `.pen` 節點值**（文字逐字、字級變數、顏色變數、間距）。PNG 只是參考。

### 🔴 建單時查到的事

1. **分集的名詞對照表：加了詞，條數不會變（真的 bug，但不會掉資料）。** `ManageSubtitleDialogV2.tsx:208` 用 `glossaryMediaId ?? mediaId`（整部劇的 id）**算條數**，但 `:766` 打開的 `GlossaryPanelV2` 收到的是 `mediaId`（**分集** id）。自 sub-7-1（PR #395）起後端會把分集 id 解析到整部劇的名詞範圍（`glossary_service.go` 的 `scopeFor`、`glossary_scope_resolver.go:154-170`），所以**詞沒有存錯地方、也不會消失**；壞在前端快取：面板用分集 id 的 query key 讀寫，入口的條數用劇 id 的 key，全域 staleTime 5 分鐘（`queryClient.ts:8`）——在分集裡新增或刪除一個詞，入口的「（N 條）」最多 5 分鐘不會變。檔頭 `:16-18` 與 spec 的「RED LINE 1」註解還寫著舊的「會存錯地方」說法，已過期。
2. **生成進度的說明是英文（真的 bug）。** 後端 `transcription_*` SSE 的 `message` 是英文：「Extracting audio track from media file」「Transcribing audio with Whisper API」「Translating subtitles to Traditional Chinese」「Translating subtitles: 45%」（`transcription_service.go:610, 645, 864, 1301`）。`GenerationProgressV2` 原樣顯示——正式環境的生成進度畫面，進度條下面就是一行英文。批次與生成工作區也顯示同一個欄位。只有「轉錄完成」是中文。全 repo 沒有任何測試或文件引用這四句英文。
3. **只差翻譯的片，對話框說「尚無字幕」（真的 bug）。** `buildTrackRows`（`:118` 附近）只替 `subtitleStatus === 'found'` 產生一列。`untranslated` 的片（英文 SRT 已經生成）會顯示「尚無字幕／此影片目前沒有任何字幕軌」，同一個畫面的說明列卻寫「僅需翻譯，不再重跑語音辨識——這次很快也很便宜」。
4. **設計稿畫了系統給不了的東西（F3）。** `W6snM` 的「正在轉錄音訊（Whisper large-v3）— 12:34 / 45:10」：SSE 沒有已用／總時間、沒有模型名（Rule 23 也禁止本地時鐘）；`fVBQF` 把「45%」放在**轉錄中**——轉錄階段沒有百分比，只有翻譯階段有（`useGenerationProgress.ts:127-128`）；`Gi18m` 本次用量行——沒有用量廣播（9R-17 未做，程式碼的欄位一直是關的）。**這些要改稿，不是改碼。**
5. **F4 的重試在 footer，程式碼在失敗面板裡。** 設計稿 `dg5rH`：「稍後再試」（`RLbWb`，Secondary）＋「重試 $0.39」（`Bfuec`，ButtonCost）；錯誤框 `vgChD` 裡沒有按鈕。程式碼的重試在 `GenerationProgressV2` 的 `gen-failed-panel` 裡（`dsr-6a` 剛加了金額與原因）。**只有管理字幕對話框會傳 `onRetry`**——`GenerationWorkspaceV2:187-194`、`GenerationBatchDialogV2:196-203` 都沒傳、而且把 `failed` 映成 `idle`，不會畫失敗面板。F4 也畫了「現有字幕」區，程式碼在進度畫面整個藏掉。
6. **設計稿上的註記大多過期。** `CJVC5`（四個「BE gaps」全都已上線）、`c4FIoB`（影集不支援、無成本端點、active-jobs 未合併——都已過期）、`lLRJV`（「CTA 永遠可按」與 J9-D 衝突）、`k4bp6`（訊號來源已改成估價的 `translationConfigured`）、`LMH8J`（「儲存後需重啟」——sub-5-2 已熱重載）、F5 副標 `dDOH6` 還寫「並重啟伺服器後再試」、`iQb2i`「查看部署說明」連結在程式碼裡沒有目標頁。
7. **F10 載入骨架在正式環境到不了。** `LocalDetailV2.tsx:417` 與 `SeasonAccordion.tsx:130` 都沒傳 `isLoading`（兩者都在資料載入後才掛對話框）。為一個到不了的畫面對齊骨架是浪費，**本張不碰 F10**，另立 `disc-2026-09-manage-subtitle-skeleton-unreachable`。
8. **11px 凍結。** `disc-2026-09-11px-micro-label-not-on-type-scale`（⚖️ Alexyu 2026-09-16）：「在裁定之前不要再有人『順手』把 11 改成 12」。本張所有 `text-[11px]` **原樣保留**；只收斂 13px 與 15px（先例：dsr-10、dsr-11）。
9. **關閉 X 是共用元件。** 設計稿的關閉鈕是 44×44 點擊區、18px 圖示、`$text-secondary`；程式碼在 `ui/Dialog.tsx:66`（16px、`--text-muted`、沒有 44px 點擊區），改它會動到全 app 的對話框與一堆基準線。**本張不改**，另立 `disc-2026-09-dialog-close-target-44px`。

### 設計稿節點

| 代號 | 節點 | 本張要看的 |
| --- | --- | --- |
| F1-D-v2 | `r1EY9`（對話框 `gD99f`） | 外框、標題列 `Nl85C`／`tO72N`、內容 `pKQCA`、「現有字幕」`XB5ys`、軌道列 `ogk8F`／膠囊 `gBXbU`／`PGiMH`、來源 `dYrRl`、名詞對照表 `MlIIH`／`h43R3`、註記 `lLRJV`／`k4bp6`／`oTTwd` |
| F2-D-v2 | `S9Rbrq` | 空狀態 `cb8Bf`／`mPiuH`／`nE56F`、生成區 `j2s9Ej` |
| F3-D-v2 | `JbXai`（對話框 `wIihe`） | 標題碼 `Dey4O`、內容 `g7PjG8`、步驟 `wJBnM`、說明 `W6snM`、用量 `Gi18m`、SSE 標籤 `j6gR0K`、footer `H2VIe`／提示 `WKtjK`、註記 `b4A6k` |
| F4-D-v2 | `U8rRtv`（對話框 `x6TRyl`） | 錯誤框 `vgChD`／`pjXCe`／`RJos8`、現有字幕 `c6TLrS`、footer `dg5rH`／`RLbWb`／`Bfuec` |
| F5-D-v2 | `f6ZxY` | 面板 `RrbLx`、圖示 `g8IH7`、副標 `dDOH6`、`QGf46`、`iQb2i`、註記 `LMH8J` |
| 母版 | `Component/GenerationProgress-v2` `XkGvG` | 欄距 `Bkbxr`（6）、標籤行高（Label 12／1.5）、連接線 `ITuZl`（26×2）、完成勾 `lG210` |
| 註記 | `CJVC5`、`c4FIoB`、`v16pVI` | 過期內容 |

字階（`DESIGN.md:390-394`）：Label 12／1.5、Body 14／1.625、BodyLg 16／1.625。

---

## Acceptance Criteria

1. **設計稿先改（Pencil MCP；動手前每個節點再 `Get` 一次）。**
   - **F3-D-v2**：`W6snM` 改成**一段** Body 14 `$text-secondary` 文字「正在轉錄音訊」（對應 AC #9 的後端文案）；「45%」`fVBQF` **在母版 `XkGvG` 裡**（`A8AT5n` 之下）——**不准刪母版節點**（會連動 F8-D、F8-M、元件庫樣本 `Thgoj` 三張圖，還會弄壞 F4 `WSt82` 的覆寫）；只在 F3 的步驟 instance `NmhL0` 上覆寫 `descendants: { fVBQF: { enabled: false } }`（F4 `WSt82` 就是這樣做的）；`Gi18m` 本次用量行移除；在畫面旁補一段註記：「進度說明只能顯示 SSE 給得出的內容：階段文字與翻譯百分比。已用／總時間、模型名、本次用量目前沒有資料來源（9R-17）。」
   - **F3-D-v2 footer** `H2VIe`：`justifyContent` `end` → `space_between`（提示靠左、「關閉」靠右，與 AC #5 一致）。
   - **F4-D-v2**：`pjXCe` 改成「翻譯失敗」；錯誤框 `vgChD` 是橫排（圖示＋文字），要把文字包進一個直排 frame，才能在「翻譯失敗」下方加第二行 Mono Label 12 `$error-text`「translate: context deadline exceeded」（示意伺服器原始錯誤，前綴要跟「翻譯」階段一致）；footer `dg5rH` 左側加提示文字 Label 12 `$text-secondary`「已保留轉錄結果，重試只需翻譯」（`justifyContent` → `space_between`，兩顆按鈕包成一組靠右）。
   - **新文字節點一律綁變數**：`fontSize: $Type/Label/Size`、`lineHeight: $Type/Label/Line`、`fontFamily: $Type/Family/Mono`（或 Primary）、`fill: $…`——`check-design-tokens.py:273` 會擋裸數字。
   - **F5-D-v2**：`dDOH6` → 「生成字幕需要雲端語音辨識（ASR）金鑰。請至金鑰設定儲存後即可使用。」；刪 `iQb2i`（「查看部署說明」沒有目標頁）；`LMH8J` 改成「ASR 金鑰熱重載（sub-5-2），儲存即生效。dsr-6a 之後這個面板只在『估價與按下之間金鑰被刪』的競態才出現；平常 J9-D ⑥ 會先停用按鈕。」
   - **F1-D-v2**：`lLRJV` 改成「CTA 可不可按由估價決定（J9-D、dsr-6a）：有金額才可按」；`k4bp6` 的訊號來源改成「估價回應的 translation_configured」；**`dYrRl`「已生成」的顏色 `$accent-text` → `$text-secondary`**（泥金在 DESIGN.md:293/300 只表示「正在跑」，「已生成」是完成的事實）；在軌道列附近補註記：「subtitle_status=found 時分不出是生成還是線上下載，程式碼的來源文字是『字幕引擎』；untranslated 一定是生成的 → 『已生成』。」
   - **`CJVC5`**：改寫成「BE 能力（2026-09-17）：生成管線與階段 SSE、名詞庫 API、批次生成、影集觸發皆已上線。」；**`c4FIoB`**：把「影集不支援」「無 HTTP 成本端點」「gated on active-jobs」三句改成現況（分集可觸發、批次可混電影與分集；單片估價端點已上線；active-jobs 已合併），其餘仍成立的句子保留。
   - 每張改完 `ctx.problems` 掃裁切。🚨 存檔要確認 ` M ux-design.pen` 並 grep 磁碟檔確認新文字落盤（`.claude/memory/feedback_verify_pen_saved_before_commit.md`）。匯出後只 stage `flow-f-subtitle-v2/{f1-d-v2,f3-d-v2,f4-d-v2,f5-d-v2}.png`＋ **`_bmad-output/pen-tokens.json`**，其餘重繪雜訊還原（`CJVC5`／`c4FIoB`／`v16pVI` 在「F · 規格 Spec」群組、不在任何匯出畫面內，改它們不會動到 PNG）。**若 `f8-d-v2`、`f8-m-v2` 或 `design-system/component-library` 也變了，代表動到了母版——回頭檢查。**

2. **對話框外框與標題列（F1／F2／F3／F4／F5 共用）。**
   - 寬度：`sm:max-w-3xl`（768）→ `sm:max-w-[880px]`；加 `sm:border sm:border-[var(--border-subtle)]`（`gD99f`／`wIihe`；用 `sm:` 是因為手機底部抽屜屬 `dsr-6f`）。陰影與圓角不動。
   - 標題與集數間距 `gap-2` → `gap-1.5`（`Nl85C` 6）。
   - 集數（`dialog-title-code`）從膠囊改成純文字：`font-mono text-base font-semibold text-[var(--text-primary)]`（`tO72N`／`Dey4O`：Mono BodyLg 16／600），拿掉 `bg`／`px`／`py`／`rounded`。
   - 內容區 `p-6` → `px-6 py-5`（`pKQCA`／`g7PjG8`：[20, 24]）。
   - ⛔ 關閉 X 不動（🔴 #9）。

3. **「現有字幕」與空狀態（F1／F2）。**
   - 區塊標題 `text-[13px] font-semibold` → `text-sm font-semibold`（`XB5ys` Body／600）。
   - 語言膠囊 `px-2.5 py-0.5` → `px-2.5 py-1`（`gBXbU` [4, 10]），`font-medium`；**字級維持 `text-[11px]`**（🔴 #8）。
   - **`untranslated` 也要有一列**（🔴 #3）：語言取 `subtitleLanguage`（沒有就當 `en`），來源「已生成」、顏色 `text-[var(--text-secondary)]`（與 AC #1 改過的 `dYrRl` 一致——泥金只表示「正在跑」）。`found` 維持「字幕引擎」、`--text-secondary`。
   - **語言標示共用一份**：在 `utils/libraryStatus.ts`（`HANT`／`HANS` 旁）新增 `subtitleLangLabel(lang)` → `{ label, family: 'hant' | 'hans' | 'en' | 'other' }`。**函式內先 `toLowerCase()` 再比對**（對話框傳的是原始值，例如後端寫的 `zh-Hant`，`transcription_service.go:922`；詳情頁的 `trackLangs` 已經是小寫）：`HANT`→繁中、`HANS`→簡中、`en`／`en-*`／`eng`→英文、`''`／`und`→未標示、其他→**原字串（保留原大小寫）**。`DetailTechInfoV2.tsx:42-48` 的 `trackLabel` 與對話框的 `languageDescriptor` 都改用它（對話框用 `family` 決定膠囊顏色與 `isHans`）。**詳情頁輸出不變**（`chi`／`zho` 不在 `HANT`／`HANS`，照舊原字保留）。**對話框刻意改變的只有**：`eng` → 英文（原本顯示 `eng`）、`und` → 未標示（原本 `und`）、`''` → 未標示（原本「未知」）。
   - 空狀態標題 `text-[15px] font-semibold` → `text-base font-semibold`（`mPiuH` BodyLg／600）。

4. **名詞對照表入口與面板 id（F1）。**
   - 入口兩處 `text-[13px]` → `text-sm`（`MlIIH`）；「（{n} 條）」拿掉括號內的空白，數字仍是 `font-mono tabular-nums`（`h43R3`）。
   - **修 🔴 #1**：`<GlossaryPanelV2 mediaId={glossaryMediaId ?? mediaId} …/>`——面板與入口用同一個 query key，新增／刪除後條數立刻更新。spec 的 `GlossaryPanelV2` stub 要記下收到的 `mediaId`，分集模式斷言它是**整部劇的 id**、不是分集 id；電影模式斷言是電影 id。
   - 過期註解一起改：檔頭 `:16-18` 與 spec「RED LINE 1」附近的註解，改成「後端會把分集解析到劇的範圍（sub-7-1），但前端的 query key 必須與入口一致，否則條數不會更新」。

5. **生成進度（F3）。**
   - `GenerationProgressV2` 的說明（`gen-stage-message`）`text-[13px]` → `text-sm`（`W6snM` Body 14）；完成句（`generation-complete-note`）`text-[13px]` → `text-sm`。
   - 「即時更新（SSE）」標籤只在**還在跑**時顯示（`phase` 不是 `complete`／`failed`）——F4 沒有它；字級維持 11px（🔴 #8）。
   - footer（進度與失敗畫面）：`justify-between` → 提示靠左、按鈕靠右（`H2VIe`／`dg5rH`：`gap-3`）；提示文字 `--text-muted` → `--text-secondary`（`WKtjK`）。執行中提示仍是「關閉後生成會在背景繼續」。
   - **母版對齊**（`XkGvG`，桌機欄）：欄距 `sm:gap-1` → `sm:gap-1.5`（`Bkbxr` 6）；標籤 `sm:text-xs` 加 `sm:leading-normal`（Label 12／1.5）；連接線 `sm:w-7` → `sm:w-[26px]`，並刪掉 `w-5`（連接線是 `hidden sm:block`，手機根本不顯示，`w-5` 是死碼）。**手機欄的 `text-[13px]` 不動**（手機屬 `dsr-6f`；`GenerationProgressV2.spec.tsx:169` 也斷言它）。百分比 `text-[11px]` 維持（🔴 #8）。
   - 對話框其餘非 11px 的任意字級一次收乾淨（建單時 12 處任意字級，4 處是 11px 凍結）：`:445` 完成句、`:471` 區塊標題、`:564` 無法開始生成、`:648`／`:649` 名詞入口、`:663`／`:677` 線上字幕搜尋區（預設收合）→ `text-sm`；`:509` 空狀態標題 → `text-base`。改完 `grep -n 'text-\[1[35]px\]' ManageSubtitleDialogV2.tsx` 必須為 0。
   - ⚠️ 母版改動會影響 `GenerationWorkspaceV2`／`GenerationBatchDialogV2` 的夾具（AC #10）。

6. **生成失敗（F4）。**
   - **失敗面板**（`GenerationProgressV2` 的 `gen-failed-panel`）：第一行 `text-sm text-[var(--error-text)]`，文字依失敗的階段：`extracting` → 「提取音訊失敗」、`transcribing` → 「轉錄失敗」、`translating` → 「翻譯失敗」（與步驟名稱同一套詞）。第二行（`data-testid="gen-failed-detail"`）顯示 `error`，規則：
     - `error` 為空、或等於 hook 的保底字「生成失敗」（`useGenerationProgress.ts:156`）→ **不顯示**第二行；
     - `error` 含中日韓文字（例如 D6 家族送的「字幕生成失敗：…」「已略過：…」）→ `text-xs text-[var(--error-text)]`，一般字體；
     - 其他（機器文字，例如 `translate: context deadline exceeded`）→ `font-mono text-xs text-[var(--error-text)] break-all`，不翻譯。
     - 圖示維持。
   - **重試搬到對話框 footer**（🔴 #5）：`GenerationProgressV2` 拿掉 `onRetry`／`retryCost`／`retryNote`／`retryBusy` 與面板內的 `ButtonCost`（`dsr-6a` 加的）。對話框在 `phase === 'failed'` 時 footer 右側依序是「稍後再試」（`bg-tertiary` Secondary 樣式，行為＝關閉對話框，**沿用 `data-testid="dialog-close"`**）與 `ButtonCost`「重試」（`data-testid="gen-retry"`，cost／busy／onClick 同 dsr-6a）。其他狀態 footer 右側仍是「關閉」。
   - **footer 左側提示**（失敗時，`data-testid="gen-retry-note"`，連結沿用 `retry-goto-settings`）：優先順序 ① `costView.retryNote`（dsr-6a 的停用原因或 ≈ 說明，含「前往設定」）② 估價回應 `plan === 'translate_only'` → 「已保留轉錄結果，重試只需翻譯」③ 無（不渲染元素）。`gen-retry` 的 `aria-describedby` 指向這段提示（有內容時）。
   - **失敗時顯示「現有字幕」區**（`c6TLrS`）：與 idle 畫面同一個元件／同一套列，放在失敗面板之下；**沒有任何軌道時整區不顯示**（不要在失敗畫面出現「尚無字幕」）。
   - **失敗時請父層重抓**：新增選用 prop `onGenerationFailed?: () => void`。⚠️ **不能直接放進 dsr-6a 那個 `useEffect` 的依賴陣列**——`SeasonAccordion.tsx:141` 每次 render 都傳新的 inline 函式，重抓 → 父層 re-render → 函式換新 → effect 再跑 → 再重抓，無限迴圈。做法：把 callback 存進 ref（比照 `useGenerationProgress.ts:202-206` 的 `onCompleteRef`），**每次進入 `failed` 只呼叫一次**（記住上一個 phase，只在「非 failed → failed」時觸發）。`LocalDetailV2` 傳入與 `onGenerationComplete` 相同的重抓；`SeasonAccordion` 同理。測試要用**每次 render 都換新的 callback** 驗證只被呼叫一次。
   - **分集要真的看得到保留下來的英文字幕**：`SeasonAccordion.tsx:50` 把整個分集物件存進 `useState`，傳給對話框的 `subtitleStatus`／`subtitleLanguage`（`:137-138`）是**開啟當下的快照**——`refetch()` 更新的是 `data.episodes`，快照不會變，所以失敗後重抓了也看不到新的軌道，dsr-6a 的「狀態一變就重新估價」對分集也永遠觸發不了。改成 state 只存分集 id，每次 render 從最新的 `data.episodes` 找出那一集（找不到時退回快照）。加一條 `SeasonAccordion` 測試：重抓後對話框收到新的 `subtitleStatus`。
   - 「無法開始生成」面板（`generation-trigger-error`，F4 沒有涵蓋）的重試**維持在面板內**；只把 `text-[13px]` → `text-sm`。

7. **F5（尚未設定）。** 圖示 `text-[var(--warning-text)]` → `text-[var(--warning)]`（`g8IH7` fill `$warning`）。文案不變（程式碼本來就對，改的是稿）。

8. **檔頭與過期註解。**
   - `ManageSubtitleDialogV2.tsx:1` 補 `+ Screen F3-D-v2 (JbXai) + Screen F4-D-v2 (U8rRtv) + Screen F5-D-v2 (f6ZxY)`（一行、以 `)` 結尾）。
   - `:36-39` 的註解「no backend endpoint converts an existing local track today」已過期（`POST /api/v1/subtitles/convert` 存在，只收影片旁的外掛檔、只收 movie／series）→ 改寫成現況並指向 AC #11 立的單子；`:161` 「no local-detail source today」→ 電影已由 `LocalDetailV2.tsx:83-84` 接上，影集／分集沒有。

9. **後端：生成進度 SSE 文案改中文（🔴 #2）。** `transcription_service.go`：
   - `:610` → 「正在提取音訊」
   - `:645` → 「正在轉錄音訊」
   - `:864` → 「正在翻譯成繁體中文」
   - `:1301` → `fmt.Sprintf("正在翻譯成繁體中文（%.0f%%）", pct)`
   - **把四句抽成小函式**（例如 `transcriptionStageMessage(phase string, pct float64) string`，比照 `subtitle/progress_sse.go:81` 的 `zhTWStageMessage`），四個廣播點改呼叫它，並對這個函式寫表格測試逐字斷言。原因：`:610`／`:645` 在 `runPipeline` 裡，要 ffmpeg 與 ASR 才跑得到，而 `sseHub` 是具體型別 `*sse.Hub`，現有測試抓不到廣播內容。
   - `phase`、`percentage` 等欄位名與值**不變**（前端靠 `phase` 判斷，不靠文字）。`:1549` 的 `"Transcription failed: " + errMsg` 不動（前端顯示的是 `error` 欄位）。
   - 建單時 grep 過：沒有任何 Go 測試、前端、e2e、TestSprite 或 `docs/sse-event-types*.md` 引用這四句英文。

10. **視覺夾具與基準線。**
    - 會變的：`subtitle-manage-subtitle-dialog-v2`（外框、標題、軌道列、入口字級）、`generation-progress-v2/{提取音訊,轉錄中,翻譯中,完成,失敗,cost-slot}`（母版欄距／行高／連接線、說明字級、失敗面板）、`generation-batch-dialog-v2/running`、`generation-workspace-v2/running`（母版改動外溢——若像素沒變就不要動它們）。
    - `generation-progress-v2/失敗` 夾具拿掉 `retryCost`／`onRetry`（props 已刪）。夾具裡的假說明文字改成 AC #9 的中文實際文案（例如「正在轉錄音訊」），不要再放系統給不出的「12:34 / 45:10」。
    - 新增 `subtitle-manage-subtitle-dialog-v2-untranslated`：同一個對話框、**不同的 media id**（例如 `movie-untranslated-1`——gallery 共用 app 的 query cache，`gallery.tsx:194-197`，用 `movie-1` 會把既有夾具的 $0.42 蓋掉）、`subtitleStatus: 'untranslated'`、`subtitleLanguage: 'en'`、沒有 `subtitleTracks`；`seedQueries` 預塞估價（`plan: 'translate_only'`、`asrAvailable: true`、`translationConfigured: true`、`runtimeSource: 'ffprobe'`、`estimatedUsd: 0.24`，其餘欄位照 `TranscriptionEstimate` 型別補齊）與 `glossaryKeys.list('movie-untranslated-1')`；`penNode`／`statesOnly` 照既有對話框夾具——截圖要看得到「英文・已生成」那一列、「生成字幕 $0.24」與便宜的說明列。
    - 基準線流程照 `.claude/memory/project_visual_baseline_intentional_change.md`：`--update-snapshots=all` 後只留上列、其餘還原；改過的 `-linux` `git rm` 交給 CI bootstrap。⛔ 不要本機產 `-linux.png`。`glossary-panel-v2/seeded` 本機本來就紅（`preexisting-fail-visual-darwin-three-stale-baselines`），不要重生它。

11. **另立的單子（建單時已寫入 sprint-status，不在本張做）。** `disc-2026-09-manage-subtitle-skeleton-unreachable`（F10）、`disc-2026-09-manage-subtitle-running-job-not-attached`（F1 註記 `oTTwd`：打開時已有工作在跑要直接接上進度）、`disc-2026-09-dialog-track-convert-not-wired`（F1 `TtkZH`「轉為繁中」與 `v16pVI`「仍要轉換」）、`disc-2026-09-dialog-close-target-44px`（🔴 #9）、`disc-2026-09-dialog-track-filenames`（F1 `VXof3` 軌道檔名）。

12. **既有測試保留通過**，刻意改的只有：`ManageSubtitleDialogV2.spec.tsx`（標題碼、untranslated 列、名詞面板 id、footer 重試與稍後再試、失敗時現有字幕、onGenerationFailed、RED LINE 1 註解）、`GenerationProgressV2.spec.tsx`（重試相關測試移到對話框 spec；失敗文案改階段名詞；`:165` 的 `toContain('sm:gap-1')` 是子字串比對，改成 `sm:gap-1.5` 仍會過——請改成逐字斷言）、`SeasonAccordion` 的 spec（新增重抓後狀態更新）、`DetailTechInfoV2.spec`（只換成 import 共用函式，輸出斷言不動）、上列夾具基準線。特別守住：`LocalDetailV2.spec.tsx`、`SeasonAccordion`／`EpisodeList` 的 spec、批次與工作區 spec、`tests/e2e/batch-subtitle.spec.ts`。

13. **CI 全綠**：`pnpm run lint:all`、`pnpm nx run web:typecheck --skip-nx-cache`、`python3 scripts/check-design-tokens.py`、`pnpm nx test web`、`pnpm nx test api`、`gofmt`。⛔ 絕不用 `run_in_background` 跑測試；每次跑完 `pnpm run test:cleanup`。⛔ 局部 vitest 綠之後一定要跑 typecheck。

## Tasks / Subtasks

- [x] **Task 1 — 設計稿（AC: #1）**
  - [x] F3（說明改一段、instance 覆寫隱藏 45%、拿掉用量行、footer `space_between`、補註記）、F4（失敗文案、直排包裝＋原始錯誤行、footer 提示）、F5（副標、刪連結、註記）、F1（註記、`dYrRl` 顏色）、`CJVC5`、`c4FIoB`
  - [x] `ctx.problems`；存檔並 grep 落盤；只 stage 改到的 PNG＋`pen-tokens.json`
- [x] **Task 2 — 後端 SSE 中文（AC: #9）**
  - [x] 抽出文案函式＋表格測試（先紅）→ 四個廣播點改呼叫 → `pnpm nx test api`
- [x] **Task 3 — 語言標示共用函式（AC: #3 語言部分）**
  - [x] `subtitleLangLabel` ＋ spec；`DetailTechInfoV2` 改用（既有 spec 不改斷言）
- [x] **Task 4 — 對話框外框、軌道列、名詞入口與面板 id（AC: #2, #3, #4, #7, #8）**
  - [x] 先寫紅測試：標題碼純文字、untranslated 列、分集名詞面板收到劇 id
  - [x] 實作；檔頭與過期註解
- [x] **Task 5 — 進度與失敗（AC: #5, #6）**
  - [x] 先寫紅測試：SSE 標籤只在執行中、失敗文案三種階段＋第二行三種規則、footer「稍後再試」＋「重試」（金額、busy、aria-describedby、提示優先順序）、失敗時現有字幕（沒軌道時不顯示）、`onGenerationFailed` 在不穩定 callback 下只呼叫一次
  - [x] `GenerationProgressV2` 拿掉重試 props、母版對齊；對話框 footer；`LocalDetailV2`／`SeasonAccordion` 接 `onGenerationFailed`
  - [x] `SeasonAccordion` 改成存分集 id、每次從最新資料取那一集（先寫紅測試）
- [x] **Task 6 — 夾具與基準線（AC: #10）**
- [x] **Task 7 — 收尾（AC: #12, #13）**
  - [x] 全套閘門；dev-story Step 9 截圖比對（`f1-d-v2`、`f2-d-v2`、`f3-d-v2`、`f4-d-v2`、`f5-d-v2`）

## Dev Notes

### 這張的重點

- **三個 bug 比對齊更重要**：分集裡加了名詞條數不會變（詞本身沒存錯，後端會解析到劇的範圍）、進度說明是英文、只差翻譯的片說「尚無字幕」。另外分集的對話框收到的是開啟當下的快照，失敗後重抓也看不到新狀態（AC #6）。
- **設計稿畫了做不到的東西時，改稿不改碼**（F3 的時間／模型／用量、F5 的部署說明連結）。這是 dsr-10「12.4 MB/s」以來的慣例。
- **重試搬家要小心 dsr-6a 剛做的東西**：金額、busy、停用原因、`aria-describedby` 全部要跟著搬到 footer，一樣都不能掉。dsr-6a 在 `ManageSubtitleDialogV2.spec.tsx` 的「the failed-run 重試 carries the price, re-fetched when the run fails」要改成找 footer 裡的 `gen-retry`，**語意不准弱化**。「a blocked retry says why under the button」測的是 `generation-trigger-retry`（無法開始生成面板，**不搬**），保持不變。

### SM 建單裁定（2026-09-17，Sally／Alexyu 可在 review 推翻）

1. 生成進度的中文文案：「正在提取音訊」「正在轉錄音訊」「正在翻譯成繁體中文」「正在翻譯成繁體中文（45%）」——與步驟名稱（提取音訊／轉錄中／翻譯中）和原稿 `W6snM` 的「正在轉錄音訊」同一套詞。
2. 失敗文案用「{階段}失敗」三種（提取音訊失敗／轉錄失敗／翻譯失敗），與步驟名稱同一套詞；伺服器原始錯誤另起一行、Mono、不翻譯。
3. `found` 的軌道來源維持「字幕引擎」（分不出是生成還是下載），`untranslated` 顯示「已生成」；「已生成」用 `--text-secondary`，不用泥金（稿的 `dYrRl` 一起改）。
5. 失敗面板的第二行：空的或保底字不顯示、中文照一般字體、機器文字用 Mono；失敗畫面沒有軌道時「現有字幕」整區不顯示。
4. F3 設計稿拿掉已用／總時間、模型名、轉錄百分比、本次用量；F5 拿掉「查看部署說明」。

### 不要做的事

- **不要改 `ui/Dialog.tsx`**（關閉 X、陰影、進場動畫）——共用元件，另有單子。
- **不要把任何 `text-[11px]` 改成 12**。
- **不要做手機版**（F1-M、F3-M 的拖曳把手、滿版按鈕、無 footer）——`dsr-6f`。
- **不要動名詞對照表面板本身**（`GlossaryPanelV2`／`GlossaryRowV2` 的樣式、徽章顏色）——`dsr-6c`；本張只修傳進去的 id。
- **不要做 F10 骨架、打開時接上執行中的工作、轉為繁中按鈕、軌道檔名**（AC #11 的單子）。
- **不要改後端 SSE 的 `phase`、`percentage`、事件名稱**，只改 `message` 文字。
- **不要改「無法開始生成」面板的重試位置**（F4 沒畫這個狀態）。

### 已知陷阱

- **Pencil**：`Copy` 一般節點時 `descendants` 用名稱當 key 會被靜默忽略（`project_pen_schema_gotchas` #5）；`Replace` 會把沒寫的屬性重設、`enabled` 變 false（#9）；`fVBQF` 確定在母版 `XkGvG` 裡——只准在 F3 的 instance `NmhL0` 上用 `descendants` 覆寫 `enabled:false`，不准動母版。改完一定存檔再截圖（#6）。
- **Radix Dialog 走 Portal**：spec 用 `screen` 找。
- **`useGenerationProgress` 只有 `onComplete`**：失敗要靠 `useEffect` 盯 `generation.progress.phase`；新的 `onGenerationFailed` 要用 ref＋「只在進入 failed 時觸發」，不能把 callback 放進依賴陣列（`SeasonAccordion` 的 inline 函式會造成無限重抓）。
- **`toHaveTextContent` 是子字串比對**——文案要逐字斷言。
- **視覺 CI 沒有後端**：新夾具的估價與名詞庫都要 `seedQueries` 預塞，否則截圖時好時壞（dsr-6a CR 教訓）。
- **`--update-snapshots` 預設模式小於門檻不會重寫**（`project_visual_update_snapshots_threshold`）——用 `=all` 再還原。
- **行號以建單時為準**（2026-09-17，main `59e72378`）。

### Source tree

```
apps/api/internal/services/transcription_service.go(+_test)        ← Task 2
apps/web/src/utils/libraryStatus.ts(+spec)                          ← Task 3
apps/web/src/components/media/DetailTechInfoV2.tsx                  ← Task 3
apps/web/src/components/subtitle/ManageSubtitleDialogV2.tsx(+spec)  ← Task 4, 5
apps/web/src/components/subtitle/GenerationProgressV2.tsx(+spec)    ← Task 5
apps/web/src/components/media/LocalDetailV2.tsx、SeasonAccordion.tsx(+spec) ← Task 5（onGenerationFailed、分集改讀最新資料）
apps/api/internal/services/glossary_service.go、glossary_scope_resolver.go ← 只讀（確認分集解析到劇的範圍）
apps/web/src/routes/test/-gallery.fixtures.tsx                      ← Task 6
tests/visual/components.visual.spec.ts-snapshots/components/{subtitle-manage-subtitle-dialog-v2*,generation-progress-v2/*,generation-batch-dialog-v2/running,generation-workspace-v2/running}/** ← Task 6
ux-design.pen、_bmad-output/pen-tokens.json、_bmad-output/screenshots/flow-f-subtitle-v2/{f1-d-v2,f3-d-v2,f4-d-v2,f5-d-v2}.png ← Task 1
```

### Cross-Stack Split Check

後端 task **1 個**（Task 2），前端／設計 task 6 個。後端 ≤ 3 → **本張不拆**。

### Time-dependent visual coverage

- **N/A — no wall-clock-reading components touched.** `GenerationProgressV2` 檔頭已聲明不讀時鐘；新夾具不含日期。

### References

- [Source: `ux-design.pen` `r1EY9`／`S9Rbrq`／`JbXai`／`U8rRtv`／`f6ZxY`／`XkGvG`／`CJVC5`／`c4FIoB`／`v16pVI`] — 三個唯讀稽核代理以 Pencil MCP 讀出（2026-09-17）
- [Source: `apps/web/src/components/subtitle/ManageSubtitleDialogV2.tsx:1, 16-18, 36-39, 83-130, 161, 208, 386-470, 506-600, 648-650, 734-770`]
- [Source: `apps/web/src/components/subtitle/GenerationProgressV2.tsx:32-39, 160-250`、`hooks/useGenerationProgress.ts:120-190, 269`]
- [Source: `apps/web/src/components/media/DetailTechInfoV2.tsx:42-48`、`LocalDetailV2.tsx:114-119, 417-431`、`SeasonAccordion.tsx:50, 127-143`、`queryClient.ts:8`、`apps/api/internal/services/glossary_scope_resolver.go:154-170`、`utils/libraryStatus.ts`]
- [Source: `apps/api/internal/services/transcription_service.go:594, 610, 645, 713, 864, 1301, 1549`]
- [Source: `apps/web/src/components/subtitle/GenerationWorkspaceV2.tsx:187-194`、`GenerationBatchDialogV2.tsx:196-203`]
- [Source: `apps/web/src/components/ui/Dialog.tsx:58-67`]
- [Source: `DESIGN.md:390-394`（字階）、`sprint-status.yaml` → `disc-2026-09-11px-micro-label-not-on-type-scale`／`disc-2026-09-type-scale-even-migration`／`preexisting-fail-visual-darwin-three-stale-baselines`]
- [Source: `_bmad-output/implementation-artifacts/dsr-6a-cost-on-paid-generate-buttons.md`（重試金額、generateCostView）、`dsr-10-flow-k-activity-v2.md` AC #10（13px 收斂先例）]
- [Source: project-context.md#Rule 17 / #Rule 21 / #Rule 23 / #Rule 24]

## Dev Agent Record

### Agent Model Used

Claude Opus 5 (1M context) — `claude-opus-5[1m]`，BMAD dev agent（Amelia）2026-09-17

### Debug Log References

- `pnpm nx test api --skip-nx-cache`：PASS。新測試 `services/transcription_stage_message_test.go`（先紅後綠）。
- `pnpm nx test web --skip-nx-cache`：**3576 / 3576 passed**。「進入 failed 只通知父層一次」另外把 ref 做法換回「callback 放進依賴陣列」跑過一次，確認會紅，再還原。
- `pnpm run lint:all`：**0 errors**（128 warnings，與前兩張相同，改過的檔沒有新增）· `web:typecheck`：PASS · `format:check`：PASS · `check-design-tokens.py`：一致 · Go `gofmt`／`go vet`：乾淨。
- **視覺**：`nx serve web` ＋ `CI=1 VISUAL_BUCKETS=6 --update-snapshots=all`，留下 10 張 darwin（`subtitle-manage-subtitle-dialog-v2`、新的 `-untranslated`、`generation-progress-v2` 六張、`generation-batch-dialog-v2/running`、`generation-workspace-v2/running`），其餘重繪雜訊全還原；9 張舊 `-linux` `git rm` 交給 CI bootstrap。比對模式只剩 `preexisting-fail-visual-darwin-three-stale-baselines` 那三張既有紅。
- **Pencil**：`W6snM` 刪三段只留一段、`NmhL0` instance 覆寫隱藏 `fVBQF`（母版沒動——匯出後 `f8-d-v2`／`f8-m-v2`／元件庫圖都沒變，證明沒碰到母版）、刪 `Gi18m`、`H2VIe` → `space_between`、F4 錯誤框包直排 frame 加 Mono 第二行、footer 提示＋按鈕組；F5／F1 文案與註記；`CJVC5`、`c4FIoB` 改寫。`Insert` 的新 frame 一開始量到 partially clipped，存檔後消失（`project_pen_schema_gotchas` #6）。F1 對話框因為補在內文裡的一行註記長到 904 高、超出 900 的畫面，改把那段註記放到規格群組的 `v16pVI`。AppleScript 存檔後 grep／JSON 讀磁碟檔確認落盤。

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created
- 🔗 **AC Drift: FOUND**
  - `dsr-6a-cost-on-paid-generate-buttons` AC #5 → 生成失敗面板的「重試」從 `GenerationProgressV2` 搬到對話框 footer（旁邊是「稍後再試」），`retryCost`／`retryNote`／`retryBusy` 從元件 props 拿掉；金額、busy、停用原因、`aria-describedby` 全部跟著搬，dsr-6a 的「failed-run 重試帶新價錢」測試原樣通過（改成找 footer 的 `gen-retry`）。
  - `ux3-subtitle-v2`（`GenerationProgressV2` 失敗文案「失敗於{階段}：{錯誤}」）→ 「{階段}失敗」＋另一行原始錯誤。
  - `9R-10c`（名詞對照表：只有「條數查詢」用劇 id）→ 面板也用劇 id。
  - `sub-2-2b`（完成句）→ 字級 13 → 14，文字不變。
- 📎 **Contract Stamps: NONE**（沒有新增或消費 `[@contract-vN]`；SSE 只改 `message` 文字，事件名與欄位不變）。
- 🎭 **A11y Pre-Flight: PASS**（`ManageSubtitleDialogV2`、`GenerationProgressV2`、`SeasonAccordion`；改過的檔 0 個新 jsx-a11y warning。重試的停用原因／續跑提示用 `aria-describedby` 連到按鈕；F5 圖示原本 AC #7 要改成 `--warning`，但 `local/no-base-semantic-as-text` 擋下（`--warning` 當文字色不到 AA），**改成把設計稿 `g8IH7` 對齊程式碼的 `$warning-text`**。）
- ✅ **Pre-existing failures: NONE new**。
- **跟 story 不同的地方（有理由）**：
  - AC #7 反向執行：程式碼維持 `--warning-text`，改稿（上面 A11y 那條）。
  - 名詞入口「（N條）」用 `inline-flex gap-0.5`（對應 `h43R3` 的 2px 間距），文字本身沒有空白。
  - **加做一件小事（lane ①）**：翻譯百分比原本用 Go 的 `%.0f`（四捨六入五成雙），批次 8 句一組時常剛好落在 12.5／37.5／62.5——說明寫 62%、旁邊步驟寫 63%。改成 `math.Round`（五入），與前端 `Math.round` 一致；加兩條測試。
  - `GenerationProgressV2.spec.tsx` 的「server message verbatim」測試仍用舊的假文字（它測的是原樣顯示，不是文案），沒改。

#### 🎨 UX Verification（對照 `flow-f-subtitle-v2/{f1,f2,f3,f4,f5}-d-v2` 與夾具截圖）

| 區域 | 設計稿 | 實作 | 一致？ | 處置 |
| --- | --- | --- | --- | --- |
| 外框 | 880、髮絲框、radius-lg | `sm:max-w-[880px] sm:border` | ✅ | — |
| 標題集數 | Mono BodyLg 16/600 純文字 | 同 | ✅ | — |
| 內容 padding | 20/24 | `px-6 py-5` | ✅ | — |
| 現有字幕標題 | Body 14/600 | `text-sm font-semibold` | ✅ | — |
| 語言膠囊 | [4,10]、Label 12 | `px-2.5 py-1`、**11px** | ⚠️ | 11px 凍結（`disc-2026-09-11px-micro-label-not-on-type-scale`） |
| 已生成 | `$text-secondary`（本張改稿） | 同 | ✅ | — |
| 名詞入口 | Body 14、「（8 條）」2px 間距 | `text-sm`、`gap-0.5` | ✅ | — |
| 空狀態標題 | BodyLg 16/600 | `text-base font-semibold` | ✅ | — |
| F3 說明 | 一段 Body 14「正在轉錄音訊」 | 後端中文 message、`text-sm` | ✅ | — |
| F3 步驟 | 欄距 6、Label 12／1.5、連接線 26 | 同 | ✅ | — |
| F3 footer | 提示靠左、關閉靠右 | 同 | ✅ | — |
| F4 錯誤框 | 「翻譯失敗」＋Mono 原始錯誤 | 同 | ✅ | — |
| F4 footer | 提示＋稍後再試＋重試 $0.39 | 同 | ✅ | — |
| F4 現有字幕 | 失敗時顯示 | 同（沒有軌道時整區不顯示） | ✅ | — |
| F4 SSE 標籤 | 沒有 | 失敗／完成時不顯示 | ✅ | — |
| F5 圖示 | `$warning-text`（本張改稿） | 同 | ✅ | — |
| 關閉 X | 44px、18px、text-secondary | 共用 Dialog 未改 | ⚠️ | `disc-2026-09-dialog-close-target-44px` |

### Discovery Triage

- **Did this story discover any work outside its current scope?** **YES** —— 建單當下處理如下。

- **① expand-scope-in-place**
  - 分集名詞對照表打開的是分集 id，條數不更新（🔴 #1）→ **AC #4**
  - 分集對話框拿的是開啟當下的快照，失敗後看不到新狀態（建單後驗證發現）→ **AC #6**
  - 生成進度說明是英文（🔴 #2）→ **AC #9**
  - `untranslated` 顯示「尚無字幕」（🔴 #3）→ **AC #3**
  - 設計稿畫了做不到的東西、註記過期（🔴 #4, #6）→ **AC #1**
  - 對話框與詳情頁的語言標示兩套（dsr-6 原單子附帶項）→ **AC #3**
  - 程式碼註解過期（轉換端點、陸劇來源）→ **AC #8**

- **② spawn-blocking-story**：無。

- **③ backlog-with-carry-forward-link**（建單時立）
  - `dsr-6c-glossary-panel`、`dsr-6d-batch-and-workspace`、`dsr-6e-consent-flow`、`dsr-6f-flow-f-mobile` — dsr-6 其餘範圍
  - `backlog-consent-policy-implementation` — F21–F27 自動處理政策（功能，非對齊）
  - `disc-2026-09-manage-subtitle-skeleton-unreachable` — F10 到不了
  - `disc-2026-09-manage-subtitle-running-job-not-attached` — 打開時已有工作在跑，要按下去收到 409 才接上
  - `disc-2026-09-dialog-track-convert-not-wired` — 轉換端點已存在，前端沒有按鈕
  - `disc-2026-09-dialog-close-target-44px` — 共用對話框關閉鈕點擊區不足 44px
  - `disc-2026-09-dialog-track-filenames` — 設計稿的軌道檔名

- Reference: `project-context.md` Rule 24

### File List

**新增：**
- `apps/api/internal/services/transcription_stage_message_test.go`
- `tests/visual/components.visual.spec.ts-snapshots/components/subtitle-manage-subtitle-dialog-v2-untranslated/default-visual-darwin.png`
- `_bmad-output/implementation-artifacts/dsr-6b-manage-subtitle-dialog-desktop.md`（本檔）

**修改：**
- `apps/web/src/hooks/useGenerationProgress.ts`（＋spec）— 匯出保底字常數；卸載後 `startTracking` 不再開連線（/ship CR）
- `apps/api/internal/services/transcription_service.go` — 四句 SSE 文案抽成 `transcriptionStageMessage`／`translationProgressMessage`（中文、百分比五入）
- `apps/web/src/utils/libraryStatus.ts`（＋spec）— `subtitleLangLabel`
- `apps/web/src/components/media/DetailTechInfoV2.tsx` — 改用共用函式
- `apps/web/src/components/subtitle/ManageSubtitleDialogV2.tsx`（＋spec）— 外框、標題集數、軌道列（untranslated、共用語言標示）、字級、名詞入口與面板 id、進度／失敗畫面、footer（稍後再試＋重試＋提示）、`onGenerationFailed`、檔頭與註解
- `apps/web/src/components/subtitle/GenerationProgressV2.tsx`（＋spec）— 拿掉重試、失敗文案與原始錯誤行、母版對齊
- `apps/web/src/components/media/SeasonAccordion.tsx`（＋spec）— 存開啟的分集、每次讀最新資料、`onGenerationFailed`
- `apps/web/src/components/media/LocalDetailV2.tsx` — `onGenerationFailed`
- `apps/web/src/routes/test/-gallery.fixtures.tsx` — 進度夾具中文文案、失敗夾具、新 untranslated 夾具
- `tests/visual/components.visual.spec.ts-snapshots/components/{subtitle-manage-subtitle-dialog-v2,generation-progress-v2/*（6 張）,generation-batch-dialog-v2/running,generation-workspace-v2/running}/default-visual-darwin.png`
- `ux-design.pen`、`_bmad-output/pen-tokens.json`、`_bmad-output/screenshots/flow-f-subtitle-v2/{f1-d-v2,f3-d-v2,f4-d-v2,f5-d-v2}.png`
- `_bmad-output/implementation-artifacts/sprint-status.yaml` — 本張 → review

**刪除：**
- 上列 9 個夾具的 `default-visual-linux.png`（交給 CI bootstrap）

**AC drift reference（未修改）：** `dsr-6a-cost-on-paid-generate-buttons.md`、`ux3-subtitle-v2*`、`9R-10c*`、`sub-2-2b-untranslated-badge-frontend.md`

## 對抗式 Code Review（/ship，2026-09-17）

獨立 reviewer（fresh context，只讀；自己跑過 6 支 spec 共 173 條與 Go 測試）回報 **0 HIGH、2 MED、7 LOW**。**修 7、立案 1、不修 1**。每條修正都先寫會紅的測試。修完後相關 spec 934 條全綠。

| # | 等級 | 問題 | 處置 |
| --- | --- | --- | --- |
| M1 | MED | 重試從失敗面板搬到 footer 時，dsr-6a 的「點了會重跑」「送出中不能再按」「停用時有原因」三條測試被刪掉沒補回 | 在對話框 spec 對 footer 的 `gen-retry` 補回三條 |
| M2 | MED | 手機寬度下，失敗畫面的 footer 要在同一列塞提示＋兩顆按鈕，提示會被擠成好幾行 | footer 改 `flex-wrap`，提示在手機上佔整列（`basis-full sm:basis-auto`），按鈕組 `ml-auto` 靠右；測試 |
| L1 | LOW | 「錯誤裡有中文就不用等寬字」會把帶中文檔名的 Go 錯誤（`ffprobe timeout: /media/電影/…`）誤判成給人讀的句子 | 改成「開頭是中日韓字」才算句子；測試 |
| L2 | LOW | 失敗後重新估價還沒回來時，會用失敗前的舊答案說「已保留轉錄結果，重試只需翻譯」 | 估價重抓中不顯示這句；測試 |
| L3 | LOW | 分集面板改傳劇 id 之後，後端「把分集舊詞搬進劇範圍」的路徑不會再被觸發，舊資料可能留在分集範圍（未驗證是否真的有） | 立 `disc-2026-09-episode-glossary-local-scope-orphans`（附 NAS 查詢語句） |
| L4 | LOW | 幾條測試比名稱弱：「執行中也說關閉」只驗了閒置；「只通知一次」沒驗重試後再失敗會再通知；名詞面板 stub 在關著時也記錄 id | 補執行中的斷言；補「重試後再失敗再通知一次、掛載時不通知」；stub 只在打開時記錄，測試先斷言未打開時是 undefined |
| L5 | LOW | 「生成失敗」保底字在 hook 與失敗面板寫了兩份 | hook 匯出 `GENERATION_FAILED_FALLBACK`，面板共用 |
| L6 | LOW | 過期註解：hook 還寫「失敗於{stage}」、`generateCostView` 還寫「retry panels」 | 改寫 |
| L7 | LOW | （既有）按「稍後再試」卸載對話框後，還在飛的重試請求回來會開一條沒人關的 SSE 連線 | `startTracking` 在已卸載時直接返回；hook 測試 |

查過不成立：能按的重試一定有金額、提示元素與 `aria-describedby` 同進同出、重試送出中不會重複送出、`onGenerationFailed` 掛載時不觸發且 ref 更新順序正確、SeasonAccordion 用 id 找得到的那集一定通過 `canManageEpisodeSubtitle`、詳情頁語言標示輸出不變、批次與工作區不會畫失敗面板、`twMerge` 保留 `sm:border`／`sm:max-w-[880px]`、後端四句英文沒有其他引用者、`math.Round` 與 `Math.round` 在 0–100 一致、名詞新增與全部確認用劇 id 解析到同一範圍。

## Change Log

| 日期 | 內容 |
| --- | --- |
| 2026-09-17 | 🔍 **/ship 對抗式 CR**：0 HIGH／2 MED／7 LOW，修 7、立案 1、不修 1。最重要的兩條：① 重試搬到 footer 時，dsr-6a 的三條保護測試被刪掉——已對 footer 的按鈕補回；② 手機上失敗畫面的 footer 會把提示擠成好幾行——提示在手機上改成獨佔一列。另外：帶中文檔名的機器錯誤仍用等寬字、重新估價回來前不說「已保留轉錄結果」、關掉對話框後不會再開出沒人關的 SSE 連線。 |
| 2026-09-17 | ✅ **dev-story 完成 → review**（Amelia）。api PASS、web 3576/3576、lint 0 errors、typecheck、prettier、token 一致；視覺比對只剩三張既有紅。修掉三個真的問題：分集名詞對照表的條數不更新、生成進度下方是英文、只差翻譯的片說「尚無字幕」；另外分集對話框現在看得到失敗後保留的英文字幕。對話框、生成進度、生成失敗對齊設計稿：外框 880、集數純文字、字級收斂（11px 凍結）、重試搬到 footer 配「稍後再試」、失敗時顯示現有字幕與伺服器原始錯誤。設計稿拿掉系統給不了的已用時間／模型名／用量，F4、F5、F1 文案與過期註記更新。順手修翻譯百分比四捨六入五成雙造成的 62%／63% 不一致。 |
| 2026-09-17 | 🔍 **建單後對抗驗證**（fresh-context 驗證代理，只讀；抽查 16 個以上行號、全部 .pen 節點存在）：4 項 CRITICAL、13 項 SHOULD FIX，**全部併入**。最重要的四項：① 🔴 #1 原本說「分集加的詞會消失」——錯的，sub-7-1 之後後端會把分集解析到劇的範圍，真正的問題是**條數不會更新**（改寫描述，修法不變）；② F3 的「45%」在母版裡，照原本「移除」會連動 F8 兩張與元件庫——改成只在 F3 的 instance 上覆寫隱藏；③ 新的 `onGenerationFailed` 若放進 effect 依賴，`SeasonAccordion` 每次 render 傳新函式會無限重抓——改成 ref＋只在進入 failed 時觸發；④ 分集對話框拿的是開啟當下的快照，失敗後重抓也看不到新的軌道——`SeasonAccordion` 改成存 id、每次讀最新資料。另外：F3 footer 稿也要改 `space_between`、「已生成」不能用泥金、`subtitleLangLabel` 要先轉小寫、dsr-6a 測試名稱寫錯一個、手機 13px 不動、兩處線上搜尋區 13px 補進來、後端文案抽成函式才測得到、F4 範例錯誤前綴要是 translate、新文字節點要綁變數、失敗第二行與沒軌道時的規則、新夾具要用不同 media id、外框只加在 `sm:`、`dsr-6d` 條目的字級數量更正。 |
| 2026-09-17 | Story 建立（SM Bob, create-story）。`dsr-6` 拆成 6b–6f 五張＋政策功能另立；本張是管理字幕對話框桌機（F1–F5 與進度母版）。三個唯讀稽核代理用 Pencil MCP 對照程式碼，找到三個真的 bug（分集名詞加錯地方、進度說明英文、只差翻譯卻說尚無字幕）。新立 5 張 disc 單與 5 張 dsr／backlog 單。 |
