# Disc：聽聲音前先在本機找出「有人在說話」的段落，只送那些去聽寫

Status: backlog

**Source:** 2026-10-07 第三次實測；Alexyu 選 A 的第 3 步。See S01E02 前 10 分鐘有 3 分半配樂，送去聽寫既花錢又把 Whisper 帶崩。

## 2026-10-07 可行性筆記（Explore agent，唯讀）

- 現況：`audio_extractor_service.go:195` 用 ffmpeg 抽 16 kHz 單聲道 WAV（約 1.92 MB／分鐘）；Go 端沒有任何 VAD／silencedetect／能量判斷；計費在 `whisper.go:268` 按**上傳的 WAV 長度**記，所以少送就少算，不用改計費；估價（`transcription_estimate.go:211`）仍以全片長報價，會高估（偏安全）。
- 執行環境：Dockerfile 最終層 `alpine:3.21` 的 `apk add ffmpeg`，核心濾鏡（silencedetect／silenceremove／volumedetect／astats）都在；`arnndn` 要另帶模型；ffmpeg 8 的 whisper 濾鏡不在。API 是 `CGO_ENABLED=0`，但有 OpenCC 這種「另建一層、複製一個 CLI 進來」的先例。
- **方案 A（只用 silencedetect）：** 零相依；但它只分「有聲／無聲」，配樂是有聲——See 那 210 秒配樂一秒都省不掉，♪♪ 觸發也還在。它的價值是切點（已被 `disc-2026-10-asr-chunk-at-silence` 用掉）。
- **方案 B（真正的語音 VAD，Silero v5）：** Go 內嵌 ONNX 要 cgo、又是 musl，不行；**走 sidecar**：builder 層編 whisper.cpp 的 VAD 範例＋Silero ggml 模型（約 1 MB），像 OpenCC 一樣放進最終層，Go 用 exec 叫、讀出「說話區間（ms）」。NAS CPU 估 60 分鐘約 10～40 秒。要做的事：把說話區間打包成 ≤120 秒的上傳（區間之間留 0.5 秒空白）、記「打包時間 → 原片時間」的分段對照表，合併時每句起訖各自換算；快取身分要加區間清單或升 v3。
- **先量再做：** 用 NAS 片段跑一次 VAD 看「說話比例」。一般影集若配樂 <10%，B 的價值主要是準確度不是省錢。

## 建議順序

chunk-at-silence 實測後若崩的問題已被壓到可接受，這張降 P3；若還是常崩或費用在意，做 B（sidecar）。
