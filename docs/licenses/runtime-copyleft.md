# Runtime copyleft inventory

最新本機 SBOM（`vido:opencc-test`，2026-08-28）顯示 51 個 GPL/LGPL 宣告的 Alpine runtime package。最重要的直接來源是 Alpine `ffmpeg` 6.1.2-r1；其 build configuration 明確包含 `--enable-gpl`，並連帶安裝 `x264-libs`、`x265-libs`、`xvidcore` 與 `vidstab`。另外還有 Alpine 基礎套件（例如 `busybox`）與 LGPL 系統/多媒體函式庫。

這些套件與 OpenCC Go binding 是不同問題：OpenCC binding chain 已從 Go production graph 清除，但 FFmpeg image 仍有 copyleft。若要選 MIT/Apache 作為 Vido 原始碼授權，必須保留對 runtime image 的第三方 notices 與相應散布義務；不能只依賴根目錄的 MIT/Apache 宣告。

每次 image release 請執行：

```bash
scripts/audit-licenses.sh <image>
```

並人工審核 `artifacts/sbom/copyleft-packages.tsv`。若產品要求 image 也避免 GPL，下一個工程決策是改用 LGPL-only 的 FFmpeg build（確認功能需求與 codec 取捨），不是再修改 OpenCC。
