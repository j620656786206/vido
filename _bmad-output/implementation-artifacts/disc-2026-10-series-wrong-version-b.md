# disc-2026-10-series-wrong-version-b：已經比對到的電影／影集，也能在「修改資訊」裡「重新選片」

Status: review

<!-- SM Bob create-story 2026-10-23；行號為 main `2dfc100c`；.pen 節點 id 由 Sally 用 Pencil MCP 唯讀讀取。傘狀條目 disc-2026-10-series-wrong-version 的第二張。 -->

**Source:** 將夜被認成 2018 真人版（-a 修了自動比對）。但自動比對總有認錯的時候，現在**認錯了使用者自己改不回來**：「手動選片」只在完全沒比對到時出現。Alexyu 2026-10-23 選 A：入口放在「修改資訊」視窗最上面，不在詳細頁按鈕列多加一顆（手機已經 4 顆）。

## 為什麼要做

- 後端早就能做：`ApplyTMDbMatch`（`enrichment_manual_match.go:73`）收任何一筆電影／影集＋TMDb 編號，寫成 `metadata_source = manual`，自動比對再也不會蓋掉。
- 前端也有選片視窗：`ManualMatchDialogV2`（B12p）。但 `LocalDetailV2.tsx:431` 只在 `noMetadata` 時才放它。
- 所以缺的只是「已經比對到」時的入口。

## Acceptance Criteria

1. **入口**：已經比對到的片（有 TMDb 編號；不是 `noMetadata`）打開「修改資訊」，最上面（標題列下面、海報與欄位上面）有一列：
   - 「認錯片了？」＋連結「重新選片」
   - 下面一行小字：「海報、簡介、演員會換成你選的那一部；這裡還沒儲存的修改不會留下。」
   - 桌機（B13p-D）、手機（B13p-M）都有；手機的連結點擊範圍至少 44px 高。
2. **點了之後**：關掉「修改資訊」（沒儲存的修改直接丟掉，跟按「取消」一樣），打開選片視窗（B12p）。搜尋框預填的字跟現在的規則一樣：電影用檔名，影集用片名。
3. **選好之後**：照現在 B12p 的流程寫入（`metadata_source = manual`），詳細頁重新讀取，顯示新選的那一部（片名、海報、集數）。
4. **沒比對到的片不顯示這一列**：它們已經有 B10p／B11p 的「手動選片」主按鈕，不重複。
5. **測試**：
   - `MetadataEditorDialog`：有傳 `onRematch` 才顯示這一列；點「重新選片」會呼叫它，不會送出儲存。
   - `LocalDetailV2`：已比對的片，從「修改資訊」點「重新選片」→ 選片視窗打開；沒比對到的片，「修改資訊」裡沒有這一列。
   - 畫廊：「修改資訊」夾具更新 `-darwin` 基準，刪掉舊的 `-linux`（CI 會補）。

## 不要做的事

- 不改 B12p 選片視窗本身（標題、列表、按鈕都不動）。
- 不改後端。
- 不在詳細頁按鈕列加按鈕（Alexyu 選 A）。

## Tasks

- [x] **Task 0（設計；2026-10-27 改由 Claude 用 Pencil MCP 執行並存檔）**— 把下方提示詞貼給 Pencil 的 Inline AI Agent；⌘S；`python3 scripts/export-pen-screenshots.py`；只留 `flow-b-detail-v2/b13p-d.png`、`b13p-m.png`；回報完成。**不要切分支、不要 commit，存檔後跟我說。**
- [x] Task 1 — `MetadataEditorDialog` 加 `onRematch`（選填）與這一列（AC 1、4）
- [x] Task 2 — `LocalDetailV2`：已比對時也放 `ManualMatchDialogV2`，從「修改資訊」接過去（AC 2、3）
- [x] Task 3 — 測試、畫廊基準、lint、typecheck（AC 5）

## Sally 的設計裁定

- 一列、兩行：第一行是問句＋連結（跟「改用圖片網址」同一種連結樣式 `$accent-text`），第二行是後果（`$text-muted`、Label 字級）。不用按鈕：這是少數人才需要的動作，不該跟「儲存」搶。
- 放在最上面：認錯片的人一打開就看到，不用捲到底。
- 桌機用下框線跟下面的內容分開（同標題列的 `$border-subtle`）；手機放在 sheet-body 第一個。

## Sally 的 Pencil 提示詞（Task 0，整段貼給 Inline AI Agent）

```
在 ux-design.pen 改兩張設計稿，其他都不要動。

【1】桌機稿「B13p-D」（節點 AFuPx）裡的對話框 dialog（aeP9c，垂直排版：header XFPQk → body XFSXY → footer GZgaT）。
在 header（XFPQk）和 body（XFSXY）中間插入一個新 frame，名稱「rematch-row」：
- 寬度 fill_container，垂直排版，gap 2
- padding：上下 $Space/md，左右 $Space/xl（跟 body 的左右一樣）
- 下框線：stroke $border-subtle，strokeWidth 只有 bottom 1，strokeAlignment inner（跟 header 一樣）
- 裡面兩個子節點：
  a. frame「rematch-line」：水平排版，gap 4，alignItems center
     - text「rematch-q」：內容「認錯片了？」，fill $text-secondary，字型 $Type/Family/Primary，字級 $Type/Body/Size，字重 $Type/Body/Weight，行高 $Type/Body/Line
     - text「rematch-link」：內容「重新選片」，fill $accent-text，其餘字型設定同 rematch-q（跟節點 R9DR4x「改用圖片網址」同一種連結樣式）
  b. text「rematch-hint」：內容「海報、簡介、演員會換成你選的那一部；這裡還沒儲存的修改不會留下。」，fill $text-muted，字級 $Type/Label/Size，字重 $Type/Label/Weight，行高 $Type/Label/Line，textGrowth fixed-width，寬度 fill_container
對話框會變高，這沒關係；dialog 的 x、y 不用動（底部仍在 900 高的畫面內）。

【2】手機稿「B13p-M」（節點 oktn2）裡的 sheet-body（Mpddc，垂直排版）。
在最前面（poster-row wEAPO 之前，index 0）插入一個新 frame，名稱「rematch-row」：
- 寬度 fill_container，垂直排版，gap 0
- 裡面兩個子節點：
  a. frame「rematch-hit」：寬度 fill_container，高度 44，水平排版，gap 4，alignItems center（跟節點 msXEU「link-url-hit」一樣的點擊範圍）
     - text「rematch-q」：內容「認錯片了？」，樣式同【1】的 rematch-q
     - text「rematch-link」：內容「重新選片」，樣式同【1】的 rematch-link
  b. text「rematch-hint」：內容同【1】的 rematch-hint，樣式同【1】
不要加框線。sheet-body 原本就會裁切、可以捲動，下面的欄位往下推沒關係。

【3】兩張稿都檢查：新加的文字沒有被裁切、沒有跟別的東西重疊。

【4】存檔（File ▸ Save）。
```

## Dev Notes

- 「修改資訊」：`apps/web/src/components/metadata-editor/MetadataEditorDialog.tsx`（`Design ref` B13p-D `AFuPx`／B13p-M `oktn2`），測試 `MetadataEditorDialog.spec.tsx`。
- 詳細頁：`apps/web/src/components/media/LocalDetailV2.tsx:213-260`（動作列）、`:421-445`（兩個對話框）。`matchOpen` 狀態已經有了（`:73`）。
- 選片視窗：`ManualMatchDialogV2.tsx`（`mediaType: 'movie' | 'series'`，`onSuccess` 後 refetch）。
- 新節點的 id 要寫進 `Design ref` 註解（Rule 21 會檢查節點存在：`_bmad-output/pen-node-ids.json`，`export-pen-screenshots.py` 會更新）。
- 畫廊：`-gallery.fixtures.tsx` 的 metadata-editor 段（`:1703` 起）。

### Time-dependent visual coverage

- N/A — 靜態文字。

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（claude-opus-5-5），2026-10-27。

### Completion Notes List

- 設計（Task 0）：2026-10-23 第一次用 MCP 改的沒有存檔，Pencil 重開後就不見了；2026-10-27 重做、用選單 File ▸ Save 存檔，確認磁碟檔有新節點。新節點：桌機 `rematch-row` **RhXQl**（line khGfW、hint pFQHx），手機 `rematch-row` **do1eP**（hit CW0T9、hint lr46T）。間距改吃變數（`$Space/2xs`／`$Space/xs`／`$Space/none`），`check-design-tokens.py` 的「裸數字＝0」才會過。存檔後 MCP 的裁切警告消失（之前是沒刷新的舊座標）。
- `MetadataEditorDialog`：選填 `onRematch`；有傳才在標題列下面顯示「認錯片了？重新選片」＋後果小字。按鈕 `type="button"`，不會送出表單。為了讓手機上這一列跟著內容捲動（B13p-M 它是 sheet-body 第一個），把捲動區從 `<form>` 移到外面一層 `div`；桌機另加下框線（B13p-D）。開啟時焦點仍在片名。
- `LocalDetailV2`：`ManualMatchDialogV2` 改成一直掛著（原本只在 noMetadata）；已比對（`!noMetadata`）才把 `onRematch` 傳給修改資訊：關掉修改資訊、打開選片。搜尋字預填規則不變。
- `useApplyMetadata`：影集套用後多讓 `seasons` 與 `season-episodes` 失效（季清單快取 1 小時，換片後會是舊那部的季）。
- **發現（不在本單範圍，另立 `disc-2026-10-rematch-keeps-old-season-rows` P2）**：後端換片時 `seasons` 表不重設。正式機唯讀查詢（2026-10-27）：將夜兩筆都還是 83612（還沒按重新解析），第 1 季 `episode_count=60`。按了重新解析或重新選片之後，季標題會繼續寫「60 集」，直到展開那一季一次。
- 測試：`MetadataEditorDialog.spec`（沒傳就沒有這一列；點了只呼叫 onRematch，不存、不上傳；焦點在片名）、`LocalDetailV2.spec`（已比對電影／影集從修改資訊點重新選片→選片視窗打開、預填正確；沒比對到的不傳 onRematch）、`useManualSearch.spec`（影集才讓季清單失效）。`vitest` 3 檔 104 個全過；`web:typecheck` 過；畫廊兩張 `-darwin` 重拍、`-linux` 刪掉等 CI 補，全畫廊 visual 跑過。E2E 新增一條真實瀏覽器流程（修改資訊 → 重新選片 → 選片視窗打開、Esc 關掉後頁面還能按），`media-detail.spec.ts` 17 條全過。

### File List

- `ux-design.pen`
- `_bmad-output/screenshots/flow-b-detail-v2/b13p-d.png`、`b13p-m.png`
- `_bmad-output/pen-node-ids.json`、`_bmad-output/pen-tokens.json`
- `apps/web/src/components/metadata-editor/MetadataEditorDialog.tsx`、`.spec.tsx`
- `apps/web/src/components/media/LocalDetailV2.tsx`、`.spec.tsx`
- `apps/web/src/hooks/useManualSearch.ts`、`.spec.tsx`
- `apps/web/src/routes/test/-gallery.fixtures.tsx`
- `tests/e2e/media-detail.spec.ts`
- `tests/visual/components.visual.spec.ts-snapshots/components/metadata-editor-metadata-editor-dialog/`（`-darwin` 重拍、`-linux` 刪除）
- `_bmad-output/implementation-artifacts/sprint-status.yaml`、`disc-2026-10-series-wrong-version-c.md`（標 done）
