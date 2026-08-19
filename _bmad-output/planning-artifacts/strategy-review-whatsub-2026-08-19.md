# 策略重審：What'Sub 事件與商業定位四問（2026-08-19）

**觸發**：YouTuber 壹加壹 2026-08-18 發布 vibe-coded 字幕工具 What'Sub（whatsub.equal2.app），功能表面與 Vido 字幕管線相似，Alexyu 提問「產品功能需要調整嗎」。
**方法**：BMAD party-mode 四輪討論 ＋ 兩個對抗驗證工作流（22 個獨立分析/懷疑代理；每路最高衝擊主張均由對抗性懷疑者實地查證 repo file:line 與網路來源後才採納）。
**結論狀態**：本文件是四個問題的**已查證**白話答案 ＋ 待 Alexyu 裁定事項清單。前三輪的錯誤結論（如「FR17 閘門會擋 ASR 斷句」「Epic 0+2 仍是可選項」）已被第四輪推翻，本文只記存活版本。

---

## Q1 護城河在哪？

**不在字幕管線本身。** 同型態的無人值守整庫 ASR+LLM 管線已被獨立 vibe-coded 復刻（saaak/SubtitlePipeline 2026-04 上線、68 stars，簡中市場；aexachao/nas-submaster 2025-12，早於 What'Sub）；「Bazarr 抽內嵌 → Lingarr LLM 翻譯」是維護者親自文件化的現役組合（Lingarr 1.3.0，2026-08-12）。管線同位是**易腐資產**（對 solo 克隆者半衰期以週計）。

真正站得住的三層（信心由查證支撐）：

1. **值得信任的無人值守閉環**（核心）—— 單一畫面誠實狀態＋花費上限＋同意流程。四容器 DIY 組裝**結構上**做不到跨工具真相；克隆者會跳過信任機制（demo 看不見、工程量是管線的 5–10 倍）。⚠️ **目前是紙上的**：真相層在說謊（bugfix-e）、ASR 多數路徑（抽樣 68.3%）繞過品質閘門（bugfix-j）、入庫自動觸發不存在（9R-10b）。
2. **逐庫累積狀態** —— 詞彙庫（sub-5-5 已閉環）、校正史、TMDb 修正、segment cache。唯一隨 AI 模型變強而**更強**的資產；但它是轉換成本非進入障礙，使用者數 = 1 時現值為零，故戰略順位排在分發之後。.nfo 在地化（ADR 自封 category-level differentiator）目前零前端呼叫者、僅電影（9R-13a backlog）—— 未浮出的狀態不累積任何東西。
3. **法律定位＋真空卡位**（近期最強、前三輪漏掉）—— ChineseSubFinder（中文優先 NAS 字幕現任者）因爬字幕源的法律風險＋burnout 停更；Vido「生成不下載」架構對該死因**結構免疫**（查證：字幕源僅官方 keyed API — OpenSubtitles REST v1 / Assrt token；不加種子，Epic 13 走使用者自有 *arr，同 Overseerr 姿態）。繁中 NAS 字幕位子真空，先行者窗口估 6–18 個月 —— **目前認領率 0%**（無 release、無 Docker image、無合法授權）。

**護城河一句話**：工具需要操作者，Vido 移除操作者 —— 而「移除操作者」要成立，前提是狀態誠實（不然無人值守 = 無人發現它在騙你）。

## Q2 能盈利嗎？

**不能（12 個月內月營收 >US$500 不可行；信心 0.85，經對抗查證）。**

- 價格錨 $0：Bazarr / Lingarr / Jellyseerr / SubtitlePipeline / SmartSub 全免費。
- 市場三重過濾：繁中 NAS 玩家 ∩ Docker 熟手 ∩ 願配 Claude key 者 → 年一裝機量估 100–500；自架圈付費轉換 1–10% → 5–50 買家。對照：SubtitlePipeline 在大 ~20 倍的簡中市場 4 個月 68 stars。
- 收費位置最弱：使用者已直付 Anthropic 計費，軟體費是計費表上的過路費。Plex 終身證漲至 $749.99（2026-07）才撐得起公司；Emby Premiere $119 —— 皆為千萬級用戶產品。
- 各模式天花板：買斷年總額 $100–2,000（一次性）；hosted 翻譯點數代管（唯一差異化模式）約 $150/月，代價是 solo dev 變 24h SaaS 值班。
- 台灣實體收款註記：Stripe 不支援台灣居民實體；若未來收款走 Paddle / Lemon Squeezy（MoR，~5%+$0.50）。

**若堅持親手驗證**（90 天、近零成本）：早鳥支持證 US$19–29 掛 Gumroad/GitHub Sponsors；前 3 週先修真相層＋發 v0.1（OSS 路線本來就要做，測敗不浪費）；預先定死線 —— <200 stars 或 <100 安裝 = 市場太小；≥100 安裝但 <10 付費 = 永久關閉盈利問題；≥10 付費才值得投一週做收款。

**接受的回報定義**：自用價值＋工程作品集＋社群聲譽，現金 $0–150/月。

## Q3 要轉 private 嗎？

**不要。問錯槓桿 —— 正確槓桿是 LICENSE，它把「可見性」與「商業保護」解耦。**

- 現況（已查證）：public repo（2026-01-07 起）、**0 star 0 fork**、根目錄無 LICENSE 檔；README §授權 明文「授權條款尚未確定，目前保留所有權利」，原因是 GPL-2.0 傳遞依賴（opencc → liuzl/da → liuzl/cedar-go）靜態連結 —— **選 MIT/Apache 前須先換掉此鏈**。package.json 的 "MIT" 字串與宣告矛盾 → 已改 "UNLICENSED"（commit 23df656）。
- 無授權的 public = 最糟組態：想法全曝光（含本規劃文件庫）、但外人法律上只能看與 fork（GitHub ToS D.5），不能用 → 沒有社群可能，也沒有保護。
- private 是最弱選項：copyright 不保護 idea（克隆者讀 README 即可）；轉 private 把 solo dev 的工作加倍（授權執行、收款、支援、行銷）換一個 maybe，且刪掉唯一免費複利的資產（公開的工程品質展示）。
- 決策矩陣：接受 OSS 結局 → public + AGPL（防第三方 SaaS 化；先解 GPL dep）；想保留商業選項 → FSL / PolyForm Noncommercial（source 公開、禁商用；唯一作者可隨時改授權，**紅線：沒 CLA 前不收外部 PR**）；private → 不建議。
- `_bmad-output/` 去留：綁定「公開發布 v0」決定（選項 C）—— 屆時要藏就一次做完「移出＋歷史改寫」（0 fork 時仍零後果）；若走 OSS 展示路線，流程紀律本身是資產。若移出：private repo + submodule 掛回原路徑可讓日常 BMAD 流程零改變，代價是遠端 session/CI 需 private 存取權。

## Q4 該儘快上線的功能

**無悔四項**（P 個人用／O 公開 OSS／C 商業 三情境交集；2026-08-19 已全數立案，story 檔經對抗驗證零 blocker）：

| 順位 | Story | 一句話 |
|---|---|---|
| 1 | `bugfix-j-asr-partial-failure-truthful-status` | 部分翻譯失敗不得標 found/zh-Hant —— 68.3% 多數路徑的誠實性 |
| 2 | `bugfix-e-qbt-error-state-as-queued` | 3,068 顆錯誤種子不再偽裝排隊中 |
| 3 | `bugfix-d-ui-visible-data-defects` | 簡體 genre／壞海報／年份全空／標題對調 |
| 4 | `9R-10b-on-add-autotrigger` | 入庫自動觸發（常設同意設計；AC #1 待裁定） |

情境分岔（等身分決定）：P → bugfix-c 資料遷移＋bugfix-f 效能；O → 發布打包（LICENSE＋v0 tag＋Docker image＋中文安裝文件）＋9R-5 VAD；C → Epic 13 完成＋auth＋完整品質棧。注意：M3 wave-1 已完結，wave-2 內容對三情境皆非最優 ——「繼續 M3」迴避不了身分決定。

另立結構性欠賬：`backlog-asr-leg-unify-gated-pipeline`（ASR 路徑併入閘門化 TranslateTrack；9R-5／9R-8／CJK 斷句 prd.md:152 同案待排）。

## Non-goals（五項，寫死）

燒錄字幕（重編碼違 NFR-P1；晚綁定嚴格優於早綁定）· 字幕樣式編輯器 · 自訂字型上傳（創作者品牌需求）· 逐字高亮/淡入特效（需像素渲染）· 字幕校對編輯器（在工具是功能，在自動化是失敗指標 —— 我們的等價物是詞彙庫回饋迴圈）。

## 已執行（2026-08-19）

- 四份 story 立案＋sprint-status 六處更新（commit 380dcf9）
- package.json license → UNLICENSED（commit 23df656）

## 待 Alexyu 裁定

1. **身分**：P 個人工具／O 公開 OSS／C 商業 —— 決定下一 sprint 的分岔內容。
2. **9R-10b AC #1** 常設同意政策表（story 檔內有 authoring 提案）。
3. **LICENSE 正式選擇**（gated on GPL dep 替換）＋ `_bmad-output/` 去留（綁定 v0 發布決定）。
4. （可選）90 天付費意願測試要不要跑。

## 主要證據

repo：`translation_service.go:223-278`、`transcription_service.go:679-693`、`activity_service.go:190`、`cost_consent_test.go`（2026-08-07 裁定全文）、`cmd/api/main.go:913`（.nfo 零前端呼叫者）、`glossary_store.go`、README §授權、`go.mod`（cedar-go GPL-2.0）。
外部：github.com/saaak/SubtitlePipeline · github.com/lingarr-translate/lingarr（discussions/42 維護者工作流）· github.com/ChineseSubFinder/ChineseSubFinder（停更通知）· github.com/buxuku/SmartSub · plex.tv（Lifetime Pass $749.99）· emby.media（Premiere $119）· fsl.software · choosealicense.com/no-permission。
