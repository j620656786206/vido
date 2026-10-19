# JetBrains Mono 字型授權盤點

Vido 前端打包 `@fontsource-variable/jetbrains-mono@5.3.0`（JetBrains Mono 可變字型，只用直立體 `wght.css`，不含斜體），用在數字、檔名、路徑等等寬讀數。字型檔在建置時由 Vite 打進前端資產，執行時不連外。

- 授權：SIL Open Font License 1.1（OFL-1.1）
- 版權：Copyright 2020 The JetBrains Mono Project Authors
- 出處：<https://github.com/JetBrains/JetBrainsMono>（npm 套件由 Fontsource 發布：<https://fontsource.org/fonts/jetbrains-mono>）

## 映像內位置

- 字型檔：`/app/public/assets/jetbrains-mono-*-wght-normal-<hash>.woff2`（小於 4 KB 的子集由 Vite 內嵌在 `/app/public/assets/index-<hash>.css`）
- 授權全文：`/app/public/licenses/JetBrainsMono-OFL.txt`（套件 `LICENSE` 原樣複製，來源 `apps/web/public/licenses/`）

## OFL 條件怎麼滿足

- 每一份字型拷貝都附著作權聲明與授權全文：映像裡的授權檔就在字型旁邊（OFL 條件 2）。
- 未修改字型檔（沒有自己裁字、裁軸、改名），所以不涉及保留字型名稱（Reserved Font Name）。
- 不單獨販售字型；字型只隨 Vido 一起散布（OFL 條件 1）。

升級時只改根目錄 `package.json` 的版本號，並重新複製套件 `LICENSE` 到 `apps/web/public/licenses/JetBrainsMono-OFL.txt`。本文件不是法律意見。
