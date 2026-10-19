# Noto Sans TC 字型授權盤點

Vido 前端打包 `@fontsource-variable/noto-sans-tc@5.3.0`（Noto Sans TC 可變字型，`wght.css`），用在整站內文（含英數字），也是等寬字裡中文的後備字型。字型檔在建置時由 Vite 打進前端資產，執行時不連外。可變字型一組檔案涵蓋全部字重，Vido 實際只用到 400／500／600／700。

- 授權：SIL Open Font License 1.1（OFL-1.1）
- 字型版權（取自字型名稱表）：(c) 2014-2021 Adobe (http://www.adobe.com/), with Reserved Font Name 'Source'
- 套件授權檔開頭：Google Inc.，接 OFL-1.1 全文
- 出處：<https://github.com/notofonts/noto-cjk>、Google Fonts（<https://fonts.google.com/noto/specimen/Noto+Sans+TC>）；npm 套件由 Fontsource 發布：<https://fontsource.org/fonts/noto-sans-tc>

## 映像內位置

- 字型檔：`/app/public/assets/noto-sans-tc-*-wght-normal-<hash>.woff2`（102 個；另有 3 個小於 4 KB 的區塊由 Vite 內嵌在 `/app/public/assets/index-<hash>.css`）
- 授權全文：`/app/public/licenses/NotoSansTC-OFL.txt`（套件 `LICENSE` 原樣複製，來源 `apps/web/public/licenses/`）

## OFL 條件怎麼滿足

- 字型檔本身的授權欄（name ID 13）是空的，所以授權全文一定要另外附：映像裡的授權檔就在字型旁邊（OFL 條件 2）。
- 未修改字型檔（沒有自己裁字、裁字重、改名），所以不涉及保留字型名稱 'Source'。
- 不單獨販售字型；字型只隨 Vido 一起散布（OFL 條件 1）。

升級時只改根目錄 `package.json` 的版本號，並重新複製套件 `LICENSE` 到 `apps/web/public/licenses/NotoSansTC-OFL.txt`。本文件不是法律意見。
