# Vido 字幕管線 — 統一架構規格(unified)

**版本:** v1.0(2026-07-23)— 合併 Alexyu 硬體現實 spec(骨幹)+ party-mode 本地模型研究(Tier-2 選配層)
**整合:** 本地生成 + 模型管理已折入本文件 §6 Tier-2(初版 local-ai-subtitle-generation-spec 草稿已併入本檔,不另存)
**驗證環境:** DS920+(J4125 / 4→20GB)標準目標機;Unraid 自組 + M5 MacBook worker 為高階/外部層

---

## 1. 核心價值鏈(依 Alexyu 2026-07-23 定案)

```
掃描 → ffprobe 偵測字幕軌
  ①「抽取內嵌字幕」(優先,I/O bound、零 CPU、最準)
  ② ASR 辨識(fallback,重運算 → 一律外包)
  ③ 線上搜尋(最後,OpenSubtitles 成功率低)
→ 語言判斷 → (需要時)LLM 翻譯成繁中 → 軟字幕 .zh-TW.srt → 播放器直接載入
```

**優先序 = 抽內嵌 > ASR > 線上搜尋**(Alexyu 定案)。理由:抽取免運算又自帶正確斷句/時間軸;ASR 品質雖不如人工字幕但可控;線上搜尋命中率低,列為最後備援。

---

## 2. 硬體現實與算力歸屬(關鍵約束)

| 階段 | 跑在哪 | 負載 |
|---|---|---|
| scan / probe / **extract** / deliver(soft srt) | **NAS 本地** | 極輕 / I/O bound,主流 NAS 可勝任 |
| **asr** | 雲端 API / 外部 worker / **NAS-本地(僅 FunASR 輕量 SenseVoice·Paraformer)**;Whisper-large 絕不在主流 NAS | Whisper 重 / FunASR 輕(CPU 17× 實時) |
| **translate** | **雲端 Provider(Claude 預設)** 或本地 Ollama(選配) | NAS 僅發 HTTP |
| burn(選配) | NAS,僅在偵測到 Quick Sync 時開放(h264_qsv) | CPU/GPU 密集 |

**本地生成可行性(實測推導 2026-07-23):分兩個模型家族,結論完全不同。**

- **Whisper 家族(含 Breeze ASR 25)= autoregressive,CPU 重。** q5_1 檔案/RAM:tiny 30MB/0.4GB · base 60MB/0.5GB · small 250MB/1GB · medium 500MB/2.6GB · large-v3 1.2GB/**4.7GB**。無 AVX2(J4125/N5095)再慢 2-3×。→ 主流 NAS 只有 tiny/base 實用,準度對字幕不夠。
- **FunASR 家族(SenseVoice-Small / Paraformer)= non-autoregressive,CPU 超輕。** SenseVoice-Small:中文 **CER 8%**(Whisper 同場 22-31%)、純 CPU **17× 實時**、~470MB;J4125 再慢 3× 仍約 5-6× 實時(2hr 電影 ~20 分鐘)。原生 OpenAI 相容 serving,直接接 §3 ASR 介面。

| 硬體 | Whisper / Breeze | **FunASR SenseVoice / Paraformer** |
|---|---|---|
| ARM 2GB(DS223) | ❌ | 🟡 SenseVoice-small 勉強 |
| J4125/N5095 4-8GB(DS920+) | ❌(只 tiny/base,差) | ✅ **實用**(17× 實時、CER 8%,中文更好) |
| 高階自組 Unraid 32GB(+GPU) | ✅(英文/繁中混講最佳) | ✅ |
| 外部 worker(M5 MacBook Metal / GPU) | ✅ 最佳 | ✅ |

→ **修正先前「主流 NAS 不能本地 ASR」:用 Whisper 不行,用 SenseVoice/Paraformer 可以,且中文更準。** 主流 NAS 本地 ASR 首選 = **FunASR SenseVoice-Small**;英文+繁中混講最佳品質仍是 Whisper-large/Breeze(需 worker/GPU)。**算力感知**:ARM/低 RAM → 只給 SenseVoice-small 或雲端;隱藏 Whisper-large 本地選項、burn 停用、佇列並發=1。但書:M1 英文→繁中,英文 ASR 品質 Whisper 仍是標竿,SenseVoice 英文「夠用」;惟英文片多半自帶內嵌英文字幕(走抽取免 ASR),影響小。

---

## 3. 系統架構(重用既有,補新路徑)

```
偵測字幕軌(ffprobe_service.go ✅ 已有)
├─ 文字型內嵌(subrip/ass/mov_text) → 抽取(🆕 ffmpeg -map 0:s -c copy)──┐
├─ sidecar .srt(DetectExternalSubtitles ✅ 已有) ─────────────────────┤
├─ (fallback)ASR → 轉錄(Route C ✅ 已有,重定位為 fallback) ──────────┤→ 語言判斷
└─ (最後)線上搜尋(engine.Process ✅ 已有,重排到最後) ────────────────┘
                                                                        │
                          繁中? → 完成 ┃ 簡中? → OpenCC s2twp(✅ converter.go)┃ 英文? → 翻譯 Provider
                                                                        │
                          Translate Provider:claude(預設,雲端) | ollama(選配,本地)
                                        │
                          軟字幕 .zh-TW.srt(placer.go ✅ 已有)
```

**Provider 抽象(兩 spec 共識,ASR 走 OpenAI 相容 `/v1/audio/transcriptions`):**
- ASR:`cloud-openai | cloud-groq | local-worker(worker_url) | disabled`
- Translate:`claude(sonnet 打底 / opus 重翻) | openai | deepl | ollama`

---

## 4. 現有 vs 新增(避免重造)

| 能力 | 現況 | 動作 |
|---|---|---|
| 偵測字幕軌 + sidecar | ✅ `ffprobe_service.go` | 沿用 |
| OpenCC 簡→繁 | ✅ `converter.go` s2twp | 沿用(簡中 edge case 用它) |
| AI 術語校正 | ✅ engine Stage4.5 | 沿用 |
| 放檔 `.srt` sidecar | ✅ `placer.go` | 沿用 |
| 線上搜尋(Assrt/OpenSub) | ✅ `engine.Process` search-first | **重排到最後** |
| ASR 生成(Route C) | ✅ transcription/generation_batch | **重定位為 fallback**,ASR 改走 Provider(雲/worker) |
| **抽取內嵌字幕** | ❌ 只偵測不抽 | **🆕 新增** `ffmpeg -map 0:s -c copy`(SDH 過濾、多軌一次抽) |
| **外語→繁中 LLM 翻譯(非 ASR 路徑)** | ❌ 只有 OpenCC 轉換 | **🆕 新增** Translate Provider |
| 算力感知自動預設 | ❌ | 🆕 新增 |
| **金鑰設定 UI** | ❌(連 TMDB key 都只在 setup 精靈、事後不可改) | **🆕 必做**(§5) |

---

## 5. 金鑰設定 UI(必做,Alexyu 定案第 4 點)

不論走哪條路都需要 key:抽內嵌+雲端翻譯 → Claude key;雲端 ASR → OpenAI/Groq key;連 TMDB 都沒地方填。新設定頁 **`/settings/keys`**(或併入 `/settings/models`):
- 可**事後編輯**(非只在 setup 精靈):TMDB · Claude(翻譯)· 雲端 ASR(OpenAI/Groq)· (選配)本地 worker URL。
- 全部寫進既有加密 **secrets 服務**(不進 code)。
- 修好 `ManageSubtitleDialogV2` 的「前往設定」死迴圈 → 指到這頁。
- 真正本地 worker(自架 whisper)= **免 key**;雲端路徑才要 key。

---

## 6. 本地模型層(經 §3 `local-worker` + `ollama` 介面接入,免 key)

分兩個算力層:

**NAS-本地層(主流 NAS 可跑,CPU 輕量):**
- **ASR:** **FunASR SenseVoice-Small⭐**(中文 CER 8%、CPU 17× 實時、~470MB、OpenAI 相容 serving)/ Paraformer-zh(中文串流+時間軸)。ARM 入門機也可能跑 SenseVoice-small。

**高階/外部-worker 層(需 GPU 或桌面級 CPU):**
- **ASR:** Breeze ASR 25⭐(國語+中英混講,Whisper-large-v2 底)/ Whisper-large-v3 / Voxtral;跑 faster-whisper 或 whisper.cpp(使用者選引擎)。英文品質標竿。
- **翻譯:** Qwen3⭐(CJK 最強)/ Breeze2 / Gemma3;跑在 Ollama。

**模型管理 UI**(`/settings/models`):選用 / 下載 / **刪除省空間**(Ollama 原生 `/api/delete`)/ 磁碟用量。**依偵測到的算力顯示對應層的模型**(主流 NAS 只列 FunASR + 雲端;高階機才列 Whisper-large/Breeze 本地選項)。

---

## 6.5 名詞庫自動擴充 — Glossary Auto-Harvest(Alexyu 2026-07-23)

**閉合迴圈:** 目前「名詞庫 → 影響翻譯」已存在(`9R-6`/`9R-8`);補上反向「翻譯 → 回填名詞庫」,讓同劇其他集數免手動重輸專有名詞。

**⚠️ 順序無關(Alexyu 2026-07-23 提醒):使用者不一定從第 1 集開始翻。** 名詞庫是 **per-show 累積、與集數順序無關**——從**任何一集**(可能是第 5 集)翻起都會回填該劇名詞庫,並套用到**所有其他集(含更早的集)**。因此設計上:
- 不假設「第 1 集 → 之後集數」;框架是「任一集 → 全劇名詞庫 → 全部集數(任意順序)」。
- 使用者**先翻的那一集**(不管第幾集)名詞庫最空、收成最多;之後各集受惠。
- **回頭補譯較早集數**時,名詞庫已長大 → 早集也一致。可選:對「名詞庫還很空時就譯好的集數」提供**用擴充後名詞庫重譯**的入口(向前補一致性)。

**復用既有基建(不重造):** `show_glossary` 表(per-series, migration 028)、`/api/v1/media/:id/glossary` 全套(List/Add/**Confirm**/**ConfirmAll**/Edit/Delete, `9R-15` done)、`GlossaryPanelV2`、以及既有的 glossary→translation 餵入。

**流程:**
1. 翻譯 LLM 呼叫時,額外要求結構化回傳一份 **term map**(人名/地名/術語:原文→採用譯名)——剛翻完的模型最清楚它挑了什麼,近乎零額外成本。
2. **去重** vs 該劇現有名詞庫(順序無關,merge 累積);新詞以 **未確認/建議** 狀態寫入(沿用既有 Confirm/ConfirmAll 審核流,不默默全信——避免抽錯污染)。
3. 翻**任一集**時 prompt 自動帶入該劇已(確認的)名詞庫 → 譯名跨集一致、免手動再輸入。

**範圍:** series 為主(名詞庫本就 per-show);電影做片內一致性。M2/M3 交付(見 §8)。

---

## 7. 翻譯範圍與 edge cases(Alexyu 定案第 5 點)

- **繁體中文內容** → 偵測到即**跳過翻譯**(直接完成)。
- **簡體中文內容** → **不 LLM 翻譯**,只走 OpenCC s2twp 轉繁(既有,便宜)。
- **英文內容** → 翻譯成繁中(**M1 首要範圍**)。
- **日文及其他語言發音/字幕** → **先跳過翻譯**(未來再擴)。M1 只處理英文 → 繁中(影集與電影皆同)。
- 斷句:每行 ≤ N 字、雙行規則、SDH 過濾;ASR 產出的長句交翻譯階段 LLM 一併重斷。

---

## 8. 里程碑(調整後)

**M1 — 抽取 + 翻譯(不碰 ASR,DS920+ 100% 可跑)**
- [ ] `ffmpeg -map 0:s -c copy` 抽內嵌文字型字幕(多軌一次抽、SDH 過濾)
- [ ] 語言偵測分流:繁中→完成 / 簡中→OpenCC / 英文→翻譯 / 其他→跳過
- [ ] Claude 翻譯 Provider(斷句 prompt、保留時間軸)
- [ ] deliver `.zh-TW.srt`;既有 UI + SSE 任務狀態
- [ ] **金鑰設定 UI**(§5)+ 修死迴圈
- [ ] Docker multi-arch + Container Manager 部署文件

**M2 — ASR fallback + Provider 完備 + 重排優先序**
- [ ] OpenAI 相容 ASR client(雲端 OpenAI/Groq + 外部 worker via Tailscale)
- [ ] 把管線串成 **抽 > ASR > 搜尋**(既有 search 引擎重排到最後)
- [ ] 算力偵測與自動預設;翻譯模型階梯 + 成本估算顯示

**M3 — 產品化 + Tier-2 本地**
- [ ] 批次整季、失敗重試、增量掃描
- [ ] burn(僅 Quick Sync,h264_qsv)
- [ ] Tier-2 本地模型 + 模型管理 UI(§6);(選配)PGS OCR 評估

---

## 9. 風險 / 待決

| 項目 | 傾向 |
|---|---|
| 內嵌只有 PGS 圖片字幕 → 走 ASR 時人名可能不如原字幕準 | M3 再評估 OCR;M1/M2 先接受 |
| Claude 整季翻譯成本 | Sonnet 打底 + 快取已翻片段 + UI 顯示預估 |
| 既有 Route C 生成 UI(ux3-ai-subtitle)與新管線的關係 | Route C 保留,ASR 改走 Provider 介面;UI 收斂待設計 |
| 線上搜尋既有程式碼 | 保留,僅重排優先序到最後 |
