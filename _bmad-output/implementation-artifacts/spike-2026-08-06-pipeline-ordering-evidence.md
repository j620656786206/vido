# Spike — 字幕管線順序裁定證據 (2026-08-06)

**目的:** 為 M2「把管線串成三層 fallback」裁定 ②③ 的順序。
**執行:** Amelia (dev),Alexyu 指示,實測於 Unraid NAS + 本機。
**狀態:** 證據蒐集完成,待 Alexyu 裁定。

---

## 0. 一句話結論

> **當初選 A 順序(抽 > ASR > 搜尋)的理由「線上搜尋命中率低」,實測不成立 —— 對真正需要翻譯的外語內容,Assrt 命中率是 85%,其中 75% 直接就有繁體。建議改 B(抽 > 搜尋 > ASR),但必須先修好 Assrt provider(目前完全壞掉),並解決同步驗證問題。**

---

## 1. 被推翻的前提

`vido-subtitle-pipeline-spec.md:21` 的定案:

> 優先序 = 抽內嵌 > ASR > 線上搜尋(Alexyu 定案)。理由:…線上搜尋命中率低,列為最後備援。

`sprint-status.yaml:374` 的墓誌銘:

> Route A fetch confirmed non-viable for 繁中 (Assrt token unobtainable, …)

**兩者都已失效:** token 現在申請得到(2026-08-06 實測可用);命中率經實測為 85%(外語內容)。
當年沒有量到真實命中率,是因為(a)拿不到 token,(b)就算有 token,Vido 的解析程式也是壞的(見 §4)。

---

## 2. ASR 成本(已量化)

Unraid NAS,`scripts/whisper-nas-benchmark.sh`,10 分鐘真實對白音軌:

| 引擎 | wall | xRT | 每 45 分鐘一集 |
|---|---|---|---|
| faster-whisper **base** int8 | 9s / 600s | **66.7×** | **0.7 分鐘** |
| faster-whisper **small** int8 | 27s / 600s | **22.2×** | **2.0 分鐘** |

硬體:`12th Gen Intel Core i5-12400` · 12 threads · 31 GiB · `/dev/dri` 存在。

**⚠️ 不可外推:** 這台遠強於 spec 假設的「主流 NAS」(J4125/N5095,4 核無 AVX2)。
結論只適用於此部署;spec 對低階機的判斷仍然成立,產品化需要算力感知(spec §6)。

**OpenVINO iGPU:未測。** NAS 的 `docker.img` 只剩 3.4G(40G 已用 36G,37 個 image / 38 個容器),
pip install 失敗 `No space left on device`。CPU 已足夠快,不值得為此動使用者的 docker.img。

### benchmark kit 修掉的 4 個 bug

kit 自 2026-07-05 建立以來從未成功執行過,原因是四個彼此遮蔽的問題:

1. `linuxserver/ffmpeg` 跑 s6-overlay,容器內降權到 `nobody`,寫不進 root 擁有的工作目錄
   → `--user 0:0` **無效**,正解是 `chmod 0777 "$WORK"`
2. 新版 Speaches **不再自動下載模型**,需先 `POST /v1/models/{model_id}`(`WHISPER__MODEL` 只預載已在磁碟的模型)
3. `-ss 300` 取樣點在片頭,常是配樂無對白 → 轉錄回空字串,量到的是 VAD 跳過而非轉錄
   → 改為可調參數,預設 `1200`
4. WhisperLive image 7.48GB,`docker run -d` 後只 `sleep 25` 就連線 → 無限等待
   → 需就緒輪詢;另 Unraid 主機**沒有 python3**,腳本原本用 python3 取 text head

修正已 commit 進 `scripts/whisper-nas-benchmark.sh`,含成因註解。

---

## 3. 片庫組成(決定各層的服務範圍)

取樣 125 檔,母體 2494(已排除 `adults/` 目錄 2817 檔 —— 首次取樣未排除,數字失真,已重做):

| 內嵌字幕軌 | 數量 | 佔比 |
|---|---:|---:|
| 有中文文字軌 | 21 | 16.8% |
| 有文字軌但非中文(可翻譯) | 20 | 16.0% |
| 只有圖片型(PGS/VobSub) | 6 | 4.8% |
| **完全無字幕軌** | **78** | **62.4%** |

- **第 ① 層(抽內嵌)服務 32.8%**
- **67.2% 掉到 ②③** —— 順序決定三分之二片庫的命運

片庫幾乎全是影集(取樣 122 tv / 3 movies;電影母體僅 60 檔)。

---

## 4. Assrt provider 目前完全壞掉 🔴

### 4.1 兩個型別宣告錯誤(`providers/assrt.go`)

以真實 API 回應餵入實際結構,驗證於 `scratchpad/assrtprobe`:

```
有結果 → UNMARSHAL FAILED: cannot unmarshal object into
         Go struct field assrtSearchItem.sub.subs.lang of type string
無結果 → UNMARSHAL FAILED: cannot unmarshal object into
         Go struct field assrtSearchSub.sub.subs of type []assrtSearchItem
```

| 欄位 | Vido 宣告 | 實際 |
|---|---|---|
| `lang` (`:87`) | `string` | 物件 `{"langlist":{...},"desc":"简"}` |
| `subs` (`:79`) | `[]assrtSearchItem` | 有結果=陣列,**無結果=`{}`** |

`engine.go:249` 對 provider 錯誤只 `slog.Warn` 後繼續 → **Assrt 永遠貢獻 0 筆,且無人察覺。**

### 4.2 回應有兩套 schema,官方文件只描述其中一套

同一個 `/sub/search` 端點會回傳兩種不同形狀的項目:

| | schema A(文件版) | schema B(未文件化) |
|---|---|---|
| 語言 | `lang.langlist` + `lang.desc` | `m_langn: [...]` + `m_lang: "英&nbsp;简&nbsp;双语"` |
| 識別 | `id` / `native_name` / `videoname` | `fileid` / `sub_name` / `m_version` |

實測 `q=The Boys S05E06` 的 14 筆結果 **全部是 schema B**;`q=The Matrix` 則回 schema A。

### 4.3 現成的相容性對照表

第三方實作 [`dyphire/mpv-sub-assrt`](https://github.com/dyphire/mpv-sub-assrt) 獨立遇到並繞過同一問題,
`sub-assrt.lua:514` 對三個欄位都做 fallback —— 可直接抄:

| 用途 | fallback 鏈 |
|---|---|
| 名稱 | `video_chinese_name` → `native_name` → `videoname` |
| 語言 | `lang.desc` → `m_lang`(需 `gsub("&nbsp;", " ")`) |
| 識別碼 | `id` → `fileid` |

其查詢形式為 `q=<短查詢>&no_muxer=1&pos=N`,**不使用 `is_file`** —— 與本 spike 的實測最佳解一致。

### 4.4 損壞範圍是侷限的

| 元件 | 狀態 |
|---|---|
| `Search` 回應解析 | 🔴 壞 |
| 查詢構造 | 🟡 可用,但缺 `no_muxer=1` |
| `/sub/detail` 解析 | ✅ `URL`/`Filename` 與 `Filelist` 兩形狀都已處理 |
| 下載/解壓 | ✅ |
| 速率限制 | ✅ 13s 間隔正確 |

### 4.5 文件不可信

官方文件寫配額 **20/min**,實測為 **5/min**(`/v1/user/quota` 打一次即從 5 掉到 4)。
`assrt.go:23-27` 的註解(2026-07-31 實測)是對的。**未來勿依文件調速率。**

---

## 5. Assrt 命中率(實測)

方法:繞過壞掉的 provider 直接打 API;短查詢(`Show SxxExx` / `Title Year`),雙 schema 判讀;
以使用者瀏覽器驗證過的 `The Boys S05E06`(14 筆 / 13 中文 / 3 繁)作自檢閘門,不通過則拒絕跑批次。

| 樣本 | n | 有中文 | 含繁體 |
|---|---:|---:|---:|
| **外語作品** | 20 | **17 (85%)** | **15 (75%)** |
| 混合片庫(含華語原生) | 20 | 5 (25%) | 5 (25%) |
| └ 其中華語原生內容 | 16 | 1 (6%) | — |

**分裂的原因:** 射手網是中文字幕站,索引的是**給中文觀眾看的外語作品**。
華語劇/國漫沒有中文字幕組上傳 —— 而它們本來就不需要「外語→中文翻譯」,
拿它們拉低命中率對 ②③ 順序的決策是誤導。**外語作品那一欄才是決策依據。**

3 筆未命中中有 2 筆疑似查詢構造問題(`House of Cards US`、`Overlord S02E03` —— 同劇 S04E01 命中),
真實命中率可能高於 85%。

### ⚠️ 方法論教訓(v1 量測全數作廢)

第一版探針量到 8%,**完全錯誤**,兩個 bug 疊加:

1. 用完整發行檔名 + `is_file=1` 查詢 —— 實測會把 `The Boys S05E06` 從 14 筆砍到 4 筆,多數標題砍到 0
2. 分類器只認 schema A,對 schema B 一律判為「非中文」

使用者以瀏覽器搜尋同一部片得到 14 筆結果,才揭露此錯誤。
**教訓:任何外部 API 的批次量測,必須先在已知案例上通過自檢再開跑。** v2 已內建此閘門。

---

## 6. 尚未解決:同步驗證

「搜到中文字幕」≠「這份字幕對得上這支影片的音軌」。這是 B 順序**唯一未解的風險**。

已測的兩個自製方法,**在測試素材上無法驗證**(LibriVox 朗讀是連續講話,cue mask 幾乎鋪滿時間軸,沒有對比訊號):

| 方法 | 成本 | 鑑別力 |
|---|---|---|
| `ffmpeg silencedetect` | 1595× 實時 | ❌ 素材不適用 |
| 能量包絡互相關 | 1771× 實時(抽包絡) | ⚠️ 6 組偏移只對 3 組,r≈0.07 |

**成本沒問題,鑑別力未證實。** 需以真實對白內容重測(NAS 上有 23 部片同時具備 `.mkv` 與 sidecar `.srt`,可作 ground truth)。

**建議不要自己實作。** `alass` 2.0.0(Homebrew 有 bottle)專門做這件事,而且**不只驗證還能自動校正對齊** ——
對 B 路線是加分:搜到的字幕即使有偏移也能修好,不必丟棄。
下一步應評估 vendoring `alass`,而非手刻互相關。

---

## 7. 裁定建議

### 對外語內容(真正需要翻譯的那部分)

| | 線上搜尋 | ASR + 翻譯 |
|---|---|---|
| 成功率 | **85%** | ~100% |
| 成本 | 一次 API 呼叫 | 2 分鐘 ASR + LLM 翻譯費用 |
| 品質 | **人工字幕** | 機器聽寫,專有名詞易錯 |
| 繁體 | **75% 直接就有**(免 OpenCC) | 需翻譯產生 |
| 風險 | 可能不同步(待解) | 無 |

**→ B 順序(抽 > 搜尋 > ASR)對外語內容明顯較優。** 85% 的情況下完全免除 AI 成本,且品質更好。
搜尋失敗的成本極低(一次 API 呼叫),落到 ASR 只慢幾秒。

### 對華語原生內容

搜尋命中率僅 6% → 幾乎必然落到 ASR。這是正確的,且 ASR 在此硬體上便宜。
B 順序對這類內容的代價是「多一次註定失敗的 API 呼叫」—— 可接受。

### 但 B 有前置條件

1. **修好 Assrt provider**(§4)—— 這是獨立的一張 story,可先做且不阻塞其他工作
2. **解決同步驗證**(§6)—— 評估 `alass`;在此之前,搜尋回來的字幕不應無條件採信

### 建議的推進順序

```
1. story: 修復 Assrt search 解析(§4,有現成對照表,範圍侷限)
2. spike: alass 同步驗證/校正評估(§6,NAS 上有 23 組 ground truth)
3. story: M2 串管線 —— 順序依 2 的結果定案
```

**若 2 的結論是「同步驗證可行」→ B;若不可行 → 維持 A**(不同步的人工字幕比機器字幕更糟)。

---

## 8. 連帶待辦

- `sprint-status.yaml:374` 的 Assrt 墓誌銘已過時,應更新
- `9R-S2-nas-whisper-benchmark-spike` 可據 §2 收斂(但需註明硬體不具代表性)
- NAS `docker.img` 使用率 92%,建議擴大或清理(與本 spike 無關的既有風險)
- spec §5「算力感知」的必要性由 §2 的硬體落差進一步佐證
