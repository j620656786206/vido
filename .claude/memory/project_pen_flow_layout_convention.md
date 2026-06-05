---
name: project_pen_flow_layout_convention
description: ux-design.pen 畫布整理規範 — 同一使用者流程的桌面+手機畫面合併成一個區塊、語意化命名、標題在上
metadata: 
  node_type: memory
  type: project
  originSessionId: 48d6240e-cb87-4cb9-b08e-16c2e2496939
---

Alex 於 2026-06-05 確認並完成 rollout 的 `ux-design.pen` 畫布 IA 規範（pilot「瀏覽主流程 A」驗收通過後，A–G 全部流程已重組）。

**1. 流程合併（桌面+手機同框）**：不再按版面分（舊 Flow A=瀏覽桌面 / Flow D=瀏覽手機）。改成「以使用者流程為單位」，同一流程的桌面與手機畫面放進「一個區塊」：上排桌面、下排手機，相同步驟左右對齊。某平台獨有的步驟各佔自己的欄。

**2. 命名 = 流程碼＋序號＋語意＋平台**（解決舊的 9a→9b→1→6 跳號難讀）。
- 畫布可見「語意標題」（caption text node，會進截圖）：`B3 · 詳情面板（桌面）` / `B3 · 詳情面板（手機）`。樣式 Noto Sans TC 14/600 `#888888`。
- 「圖框名稱」精簡短碼：`B3-D`、`B3-M`。

**3. 標題在上方、圖框名精簡、不重疊**（caption 放畫面上方會撞 Pencil 圖框名 chrome，要拉開 ~45px）。見 [[feedback_pencil_label_overlap]]。

**區塊座標範本**（origin blockX, blockY；合併欄在 x=17040，各流程往下堆疊、間距 2600）：
- 流程標題 `(blockX, blockY)` DM Sans 24/700 `#222222`「中文 — English」；描述 `(blockX, blockY+34)` Noto Sans TC 14 `#666666`。
- 桌面 frame `y=blockY+120`(h900)、caption `y=blockY+75`；欄 `x=blockX+col*1540`。
- 手機 frame `y=blockY+1130`(h844)、caption `y=blockY+1085`；x 對齊同步驟桌面欄。
- 同步驟桌機/手機共用 seq；遷移時刪掉該流程舊的標題/描述/per-screen caption/步驟箭頭。
- root 畫面用絕對 x/y 定位 → 用 Update 改 x/y/name（非 Move）。絕不改畫面內部 UI、不動 components、不自跑截圖。

**已完成的流程碼與位置**：A=瀏覽主流程(x=17040,y=-90)、B=詳情與互動(2600)、C=搜尋/篩選/設定(5200)、D=下載管理(7800)、E=媒體庫掃描(10400)、F=字幕搜尋/批次(13000)、G=AI 字幕增強(15600)。

注意：`scripts/export-pen-screenshots.py` 的 `SCREENS` dict 與 `flow-a~f` 資料夾仍是舊的分版面結構，重產截圖前需先更新成新 A–G 合併塊。spec 專屬畫面仍依 [[feedback_pencil_spec_standalone_screen]] 獨立成框。
