# OpenCC 與字典授權盤點

Vido 以 source build 方式打包 BYVoid/OpenCC `ver.1.4.2`，執行檔與 `data/*.json` 字典放在映像內。OpenCC 主專案與其字典採 Apache-2.0；映像不包含原本的 Go binding、`liuzl/da` 或 `liuzl/cedar-go`。

授權證據：

- OpenCC LICENSE：<https://github.com/BYVoid/OpenCC/blob/ver.1.4.2/LICENSE>
- OpenCC NOTICE：<https://github.com/BYVoid/OpenCC/blob/ver.1.4.2/NOTICE>
- OpenCC 字典來源與安裝路徑：<https://github.com/BYVoid/OpenCC/tree/ver.1.4.2/data>
- 每次 release 仍須以 `scripts/audit-licenses.sh` 產生的 Syft SBOM 及 `go list -deps` 結果為準；本文件不是法律意見。

## SBOM 門檻

`syft` 可用時執行 `scripts/audit-licenses.sh vido:local`，輸出 SPDX JSON 到 `artifacts/sbom/`（該目錄不納入版本控制）。腳本也會產生 `opencc-supplemental.spdx.json`，補記 Syft 無法從自編譯 binary 自動辨識的 OpenCC 與 `data/*.json` 字典，並檢查 Dockerfile 的 OpenCC pin、字典設定，以及 Go production dependency graph 不含 GPL binding chain。

注意：這個檢查只證明「OpenCC Go binding chain 已移除」，不代表整個 image 沒有 GPL/LGPL。因為目前 image 仍安裝 FFmpeg 及其 codec/runtime（例如 `ffmpeg`、`x264-libs`、`x265-libs`、`xvidcore`、Alpine `busybox`），Syft 會把它們列在 `artifacts/sbom/copyleft-packages.tsv`。若要宣告 Vido 整體採 MIT 或 Apache-2.0，仍須另外完成 FFmpeg 組態、動態連結與相應 license notice 的法律審核；目前不能把 image 宣告為純 MIT/Apache。

完整 runtime copyleft 分析見 [`runtime-copyleft.md`](runtime-copyleft.md)。
