# Story DSR.3b：連線設定與金鑰設定對齊設計稿——金鑰的「測試」鈕放進輸入框裡、手機上每把金鑰一張卡、兩條警告講清楚「為什麼不能存」

Status: review

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As the person who sets up qBittorrent, Sonarr, Radarr and API keys on Vido,
I want 連線設定的每張卡長得一樣、金鑰頁一眼看得出哪把已經設好、哪把可以測，手機上每把金鑰各自一張卡，而且「沒有加密金鑰」「連線沒加密」兩種警告告訴我發生了什麼、要做什麼,
so that 我不用在同一個分頁裡看到兩種表單寫法，也不會被一句混在內文裡的警告帶過。

## Context

`dsr-3`（Flow C 設定）拆出來的第二張：**「連線」分組的前兩個分頁**（連線設定、金鑰設定）。**依賴 `dsr-3a` 先合併**（用 `SettingsPageHeader`、`SettingsErrorState`）。與 `-3c`〜`-3f` 互不相依。拆單理由與全域裁定見 `dsr-3a` Context。

| 稿 | 節點 | 元件 | 方向（大宗） |
| --- | --- | --- | --- |
| C4-D／C4-M 連線設定 | `6UCtX`／`2H4OM` | `routes/settings/connection.tsx`、`QBittorrentForm.tsx` | 稿→碼（間距、按鈕）；碼→稿（手機字級、圖示） |
| C23-D／C23-M Sonarr／Radarr | `Qva0y`／`p37q9` | `ArrConnectionForm.tsx` | 文案**全部逐字相符**；稿補漏畫的 Radarr 提示 |
| C7-D／C7-M 金鑰設定 | `PWvEX`／`f8Fda` | `routes/settings/keys.tsx`、`ApiKeysForm.tsx` | 稿→碼（列結構、手機卡片）；碼→稿（已儲存狀態、環境變數說明、TMDB 標示） |
| C21-D 沒有加密金鑰 | `AVUg2` | `ApiKeysForm.tsx:249-268` | 稿→碼（硃砂、兩行）；稿刪掉到不了的遮罩值 |
| C22-D HTTP 未加密 | `t6FA4` | `ApiKeysForm.tsx:273-300` | 稿→碼（赭色標題＋說明、勾選框顏色） |

### 🔴 建單時查到的事（main `b041ea09`；行號皆為現況）

**連線設定（C4、C23）**
1. 標題與說明逐字相符（`connection.tsx:18-20`），由 `-3a` 換成 `SettingsPageHeader`。
2. 卡片內距：碼 `p-4 md:p-8`（桌機 32，`connection.tsx:25`、`ArrConnectionForm.tsx:273`），稿桌機 24（`pHZw1`）／手機 16 → `p-4 md:p-6`。卡片間距碼 `gap-6`（`connection.tsx:22`），稿 16（`fQ1bM`／`M3dam`）→ `gap-4`。
3. 欄位間距：QB `space-y-5`（`QBittorrentForm.tsx:154`）、Arr `space-y-6`（`ArrConnectionForm.tsx:338`），稿 16 → 兩份都 `space-y-4`。標籤到輸入框：QB `mb-1.5`（6），稿 8 → `mb-2`（Arr 已是）。
4. 卡片標題：碼 `text-base md:text-lg`（手機 16／桌機 18，`connection.tsx:30`，Arr 同），稿桌機 16（`afZEH`）／手機 14（`x1aWVi`）。⚖️ **一律 16（`text-base`）**：桌機稿→碼；手機稿改 BodyLg 16（手機內文不縮，DESIGN.md `:615`）。
5. Base Path 提示：碼 `text-sm`（`QBittorrentForm.tsx:214`），稿 12（`o0Rt4t`）→ `text-xs`。手機稿寫「（選填）」、碼「（選填，反向代理用）」→ 稿改成碼的字。
6. 按鈕：QB `font-medium px-4`（`:250`、`:268`），Arr 已是 `font-semibold px-5` → QB 對齊 Arr。手機：QB `flex-col`（`:244`），稿兩顆並排各 151（`NO9va`），Arr 已是 `grid-cols-2` → QB 改 `grid grid-cols-2 md:flex`（照 Arr 那一行）。按鈕間距：Arr `md:gap-4`（`:559`），稿 12（`z8Cyb`）→ `md:gap-3`（兩份一致）。
7. 碼→稿：按鈕上的 Plug／Save 圖示稿沒畫（兩份表單都有）→ 稿補；QB 輸入框稿用 JetBrains Mono（`gGHGg`／`Y7MRw`／`efCGP`）、Arr 輸入框稿用一般字、碼全部一般字 → 稿改一般字；手機稿輸入框與標籤 12 → 14；Radarr 清單提示「送出電影請求時，Vido 用這兩個設定把它加進 Radarr。」碼一定顯示（`ArrConnectionForm.tsx:504-508`）、稿 `LfM8B` 漏畫 → 稿補；「未設定」小標 `o82CE` 是手畫的 → 稿照碼的樣式畫（dev 讀 `ArrConnectionForm` 的未設定 badge class 後對應 token）。

**金鑰設定（C7、C21、C22）**
8. 列名：碼 `text-sm font-medium text-secondary`（`ApiKeysForm.tsx:330,335`），稿 Body 14／600／`$text-primary` → 稿→碼。
9. 狀態 pill：碼 `text-[11px] font-medium`（`:342`，**全 settings 三個舊字級之一**），稿 Label 12／600 → `text-xs font-semibold`。
10. 每列的結構（稿→碼）：稿是「標籤＋pill → 44 高 `$bg-tertiary` 輸入框（**等寬字**），『測試』是嵌在框內右側的小鈕（`j6UfaO`：28 高、Label 12、`$radius-sm`、有框線）→ 說明文字在**輸入框下方**」。碼：說明在上方（`:361`）、輸入框 `bg-secondary py-2`（`:105`）、測試鈕是框外另一列的次要按鈕帶 Plug 16（`:475-498`）。
11. 不能測試的列：碼在每一列都寫「目前僅支援 Claude 金鑰測試」＋停用的測試鈕（`:499-503`）；稿只在卡片下方一句（`pegsz`）「僅 Claude 金鑰支援連線測試；其餘欄位儲存後由服務自行驗證。」。⚖️ **稿→碼，但拿掉後半句**：「由服務自行驗證」沒有查證依據（`backlog-asr-key-test-probe` 記錄的是「測試只探 Claude」），寫進 UI 等於一個沒人保證的承諾 → 只寫「僅 Claude 金鑰支援連線測試。」；非 Claude 列**不放**測試鈕。稿同步改字。
12. 碼→稿（稿要補畫，碼不動）：已儲存的金鑰碼**不畫輸入框**，改顯示遮罩值（等寬）＋「編輯／清除」（稿 C7 的 Claude 列寫「已設定」卻畫空輸入框）；環境變數覆蓋說明「在此儲存的金鑰會覆蓋環境變數…」（`:367-374`）；TMDB 列的 `TmdbAttribution`（TMDB 條款要求，sub-6-9）；沒輸入任何東西時儲存鈕是停用的（稿畫成可按的金色——不合理）。
13. **TMDB 的拼法**：碼 `KEY_ROWS` 寫「TMDB」（`:57`），`TmdbAttribution` 依 TMDB API Terms 也寫「TMDB」；稿寫「TMDb」。⚖️ **碼→稿，一律「TMDB」**（C7／C21／C22 所有出現處）。後端 displayName 的「TMDb API」不在本張（服務狀態頁由 `-3c` 做中文名稱對照，不會露出後端字串）。
14. 儲存鈕：碼「儲存」＋Save icon、`py-2`（`:566-568`），稿「儲存金鑰」44 高 → ⚖️ **稿→碼**（「儲存金鑰」比「儲存」清楚：這頁的儲存只存金鑰）。
15. 列之間：碼 `divide-y`＋`py-5`，稿無分隔線、間距 `Space/lg` 16 → 稿→碼。卡片底 `bg-[var(--bg-secondary)]/50`（`:304`）→ 實色 `bg-secondary`；圓角用 `rounded-[var(--radius-lg)]`（Tailwind 的 `rounded-lg` 是 8px，token 是 12）。
16. **手機（C7-M `f8Fda`）**：稿把每把金鑰畫成獨立卡片、儲存鈕滿版 → 稿→碼（`max-sm:` 下每列自成一卡、儲存鈕 `max-sm:w-full`）。但稿同時**拿掉了說明文字與測試鈕**、縮短了頁面說明（少「並優先於環境變數」）、pill 縮成「環境變數」→ ⚖️ **碼→稿**：手機不砍功能、只有一套文案，稿補回。
17. **C21 沒有加密金鑰**（`:249-268`）：碼用**赭**（`warning-tint`＋`AlertTriangle`、文字 `text-primary`、一整句）；稿用**硃砂**（`O9YinR` `$error-tint`＋`lock`＋`$error-text`），兩行：標題 `oU3JS`「未設定加密金鑰，無法安全儲存 API 金鑰」（Body 600）＋說明 `c9MFeI`「請設定 ENCRYPTION_KEY 後重啟容器。在那之前，這一頁只能檢視，不能儲存。」（Label）。⚖️ **稿→碼（硃砂）**：狀態詞彙「赭＝你要求了但沒發生、硃砂＝壞了」——缺少加密金鑰是設定壞了、存不了，不是「要求了沒發生」。停用時的輸入框：稿 `$text-disabled` 字、碼 `opacity-50` → 稿→碼。
    - 稿 `e9OjVD` 在這個狀態顯示已存的遮罩值「sk-ant-••••1f4a」——**到不了**：後端沒有加密金鑰時沒有密鑰儲存庫（`key_settings_service.go:52-58`），遮罩格式也不是這樣（`maskKey`，`:139`：前 6＋「…」＋後 4）→ 稿刪。
    - 稿 `nAPD6` 把「（由環境變數提供）」寫進輸入框 → 碼是徽章＋說明，稿改成碼的畫法。
    - `u9e2Mf`「儲存停用中：…原因寫在上面。」—— dev 用 `Get` 確認它在不在畫面 frame 裡：若是 frame 外的設計註記就不動；若在 frame 裡，**碼不加**、稿刪。
18. **C22 HTTP 未加密**（`:273-300`）：稿 `b6rD2b` `$warning-tint`，第一行 `shield-alert` 18＋標題 `f3XOnd`「目前連線未加密（HTTP）」Body 600 **`$warning-text`**，第二行說明 `G4jG9P`「API 金鑰會以明文傳送到 NAS。建議先設定 HTTPS 反向代理。」Label `$warning-text`；勾選框 `xYITT` 18、勾選時 `$warning-text` 填色、文字 `MZpch` Label 600 `$warning-text`。碼是一整句 `text-primary` 14（`:286`）、勾選框 `accent-[var(--accent-primary)]`（**泥金＝正在跑**，用在這裡是錯的詞）、文字 `text-secondary` 14（`:295`）→ 全部稿→碼。**勾選框的 label 文字**以碼為準（dev 讀 `:295` 附近），稿同步。
19. 載入失敗 `:218-224` 直接印錯誤 → 換 `SettingsErrorState`（`title`「無法載入金鑰設定」、`description`「與後端的連線中斷了。已存的金鑰不受影響。」、`onRetry` = 查詢的 `refetch`）。⚠️ 文案是 SM 擬的、比照 C16，**待 Sally 確認**。

### 設計稿節點

| 代號 | 節點 | 說明 |
| --- | --- | --- |
| C4-D／C4-M | `6UCtX`／`2H4OM` | `pHZw1` 卡內距、`fQ1bM`／`M3dam` 卡間距、`afZEH`／`x1aWVi` 卡標題、`o0Rt4t`／`sCizF` Base Path 提示、`gGHGg`／`Y7MRw`／`efCGP` QB 輸入框、`BU92u`／`s2VSNs` 手機標籤、`NO9va` 手機按鈕列 |
| C23-D／C23-M | `Qva0y`／`p37q9` | `LfM8B` Radarr 清單區、`o82CE` 未設定小標、`z8Cyb` 按鈕列 |
| C7-D／C7-M | `PWvEX`／`f8Fda` | `j6UfaO` 框內測試鈕、`pegsz` 卡底說明 |
| C21-D | `AVUg2` | `O9YinR` 警告框、`oU3JS`／`c9MFeI`、`y97vhg` 停用輸入框、`e9OjVD`（刪）、`nAPD6`、`y5BCp` 儲存、`u9e2Mf`（查） |
| C22-D | `t6FA4` | `b6rD2b`、`f3XOnd`、`G4jG9P`、`xYITT`、`MZpch`、`vy0ff`、`or01e`、`v2FGa` |

---

## Acceptance Criteria

1. **設計稿先改（Pencil MCP；動手前再 `Get`）。** 🔴 #7、#11 的字、#12、#13（全部「TMDb」→「TMDB」）、#16 手機補回說明與測試鈕與完整文案、#17 刪 `e9OjVD`／`nAPD6` 改畫法、#4 手機卡標題 16。規格註記 `spec-note-dsr-3b`：
   > 「連線／金鑰：卡片內距桌機 24／手機 16、欄位間距 16。金鑰每列：標籤＋狀態 → 輸入框（等寬，測試鈕嵌在框內右側，只有 Claude 有）→ 說明。已儲存的金鑰顯示遮罩值＋編輯／清除，不顯示空輸入框。沒有加密金鑰＝硃砂（壞了、存不了）；連線沒加密＝赭（建議你處理）。手機每把金鑰一張卡，功能與文案和桌機相同。」
   收尾同 `dsr-3a` AC #1（`problems` 不增、選單 Save、grep 磁碟、只 stage 有改的 `c4-*`／`c7-*`／`c21-d`／`c22-d`／`c23-*`＋`pen-tokens.json`）。

2. **連線設定（`connection.tsx`、`QBittorrentForm.tsx`、`ArrConnectionForm.tsx`）**：🔴 #2–#6 全部。⛔ 文案、欄位、驗證、儲存／測試行為**不動**；⛔ 不改 Arr 表單的狀態機。

3. **金鑰每列重排（`ApiKeysForm.tsx`）**：🔴 #8–#11、#14、#15。
   - 輸入框：`min-h-11`、`bg-[var(--bg-tertiary)]`、`font-mono`、右側內距留給框內測試鈕；測試鈕在框內右側（絕對定位或 flex 包一層），高 28、`text-xs`、`rounded-[var(--radius-sm)]`、有框線，**保留**原本的 accessible name、`data-testid` 與測試中／成功／失敗三種回饋。
   - 說明文字移到輸入框下方。
   - 非 Claude 列不渲染測試鈕；卡片底下一行「僅 Claude 金鑰支援連線測試。」。
   - 已儲存狀態（遮罩＋編輯／清除）、環境變數徽章與說明、`TmdbAttribution`、停用的儲存鈕：**行為不動**，只套新的列結構。

4. **手機（<640）**：每一列 `max-sm:` 自成一張卡（`bg-secondary`、`radius-lg`、`p-4`、卡間 12）；儲存鈕 `max-sm:w-full`、`min-h-11`。桌機 ≥640 仍是一張大卡。

5. **兩條警告（C21、C22）**：🔴 #17、#18。C21 容器 `role="alert"`；C22 維持現在的語意（非 alert——它是建議不是錯誤；dev 讀現況，若已是 `role="status"` 就不動）。

6. **載入失敗**：🔴 #19。

7. **既有的行為不准回歸。**
   - 金鑰的儲存、清除、編輯、測試、env 覆蓋、加密金鑰缺失時停用、HTTP 勾選框必須勾才能存——**全部行為照舊**。
   - `ApiKeysForm.spec.tsx`、`QBittorrentForm.spec.tsx`、`ArrConnectionForm.spec.tsx`、`e2e/arr-settings.spec.ts` 的行為斷言不改。**例外**：斷言「每列都有停用的測試鈕／每列都有『目前僅支援 Claude 金鑰測試』」的條目是本張刻意退役的行為 → 改成新行為並在 Completion Notes 逐條列出（改了哪條、為什麼）。
   - ⛔ 不改後端、`TmdbAttribution`、`SettingsLayout`。

8. **測試。** 紅／守（Rule 16）。
   - `ApiKeysForm.spec.tsx`：（紅）測試鈕在輸入框的同一個容器內（`within(inputWrapper)`）；非 Claude 列沒有測試鈕、卡片底有那一句；說明文字在 DOM 順序上位於輸入框之後；pill 帶 `text-xs` 不帶 `text-[11px]`；儲存鈕文字「儲存金鑰」；C21 容器帶 `--error-tint` 與兩段文字逐字；C22 標題與勾選框用 `--warning-text`、勾選框**不帶** `accent-primary`；載入失敗顯示 `SettingsErrorState` 且**不含** mock 的英文錯誤字串、按重試呼叫 refetch。（守）儲存／清除／編輯／測試／env／HTTP 勾選既有斷言。
   - `QBittorrentForm.spec.tsx`／`ArrConnectionForm.spec.tsx`：（紅）欄位容器 `space-y-4`、QB 按鈕 `font-semibold`、QB 按鈕列 `grid-cols-2`；（守）既有。
   - **視覺夾具**：既有 `settings-qbittorrent-form`、`settings-arr-connection-form/*` 的基準**會變**（間距）——寫進 Completion Notes。新增 `settings-api-keys-form`（`width: 1200`、`penNode: 'PWvEX'`；states：預設／已儲存 Claude＋env TMDB）、`settings-api-keys-form/mobile`（`width: 390`、`penNode: 'f8Fda'`）、`settings-api-keys-form/no-encryption-key`（`penNode: 'AVUg2'`）、`settings-api-keys-form/http-warning`（`penNode: 't6FA4'`）。`penNode` 同時把兩個既有連線夾具從 `'screen-section'` 改成 `'6UCtX'`／`'Qva0y'`。
   - **e2e**（追加到 `tests/e2e/settings-shell.spec.ts`，`-3a` 建立）：390 開 `/settings/keys` → 每把金鑰各自一個卡片容器、儲存鈕寬度 ≥ 視窗寬 − 32；1440 → 所有金鑰在同一個卡片容器。
   - **每一項修法做 mutation check**，結果寫進 Completion Notes。

9. **CI 全綠**：同 `dsr-3a` AC #9。

## Tasks / Subtasks

- [x] **Task 1 — 設計稿：C4／C23／C7／C21／C22 的碼→稿項目、TMDB、規格註記（AC: #1）**
- [x] **Task 2 — 連線設定兩份表單的間距、標題、按鈕（AC: #2, #8）**
- [x] **Task 3 — 金鑰每列重排：框內測試鈕、說明下移、只剩一句不能測的說明、儲存金鑰（AC: #3, #8）**
- [x] **Task 4 — 金鑰手機卡片（AC: #4, #8）**
- [x] **Task 5 — C21 硃砂兩行、C22 赭色標題與勾選框、載入失敗換共用元件（AC: #5, #6, #8）**
- [x] **Task 6 — 夾具、e2e、mutation check、收尾（AC: #7, #8, #9）**
  - [x] dev-story Step 9：`c4-d`／`c4-m`／`c7-d`／`c7-m`／`c21-d`／`c22-d`／`c23-d`／`c23-m`

## Dev Notes

### 這張的重點

- **連線設定幾乎全是間距**；真正的工作量在金鑰頁的列結構。
- **行為一個都不改**。金鑰頁有一長串既有的狀態（已存、env、無加密金鑰、HTTP、測試中/成功/失敗），全部只換外殼。先讀完 `ApiKeysForm.tsx` 的 `KEY_ROWS` 與渲染分支再動手。
- **手機只換排法不砍功能**——稿上被砍掉的東西補回稿上。

### 上游契約（Rule 20）

- 本張不定義也不消費任何 `[@contract-v*]`。

### 建單裁定（2026-09-23，SM；Sally／Alexyu 可在 review 推翻）

1. ⚖️ C23 納入 dsr-3（見 `dsr-3a` Context）。
2. ⚖️ 卡片標題一律 16；手機輸入框與標籤維持 14（手機不縮內文）。
3. ⚖️ 不能測試的說明只寫「僅 Claude 金鑰支援連線測試。」——後半句沒有依據。
4. ⚖️ 「TMDB」（碼→稿）；「儲存金鑰」（稿→碼）。
5. ⚖️ 沒有加密金鑰用硃砂（稿→碼）。
6. ⚖️ 手機金鑰卡保留說明與測試鈕（碼→稿）。
7. ⚖️ 載入失敗文案 SM 擬、待 Sally。

### 不要做的事

- 不要改任何金鑰的儲存／測試／清除邏輯、API 呼叫、欄位名。
- 不要讓非 Claude 列的測試鈕「看起來可按」。
- 不要在 UI 寫「由服務自行驗證」。
- 不要改 `TmdbAttribution`。

### 已知陷阱

- **框內測試鈕 + 密碼輸入框的瀏覽器顯示鈕**：Edge／Chrome 的 `type="password"` 可能有自帶的顯示圖示佔右側，右內距要留夠；在 e2e 390 截一張確認沒有重疊。
- **`rounded-lg` 是 8px**，token `--radius-lg` 是 12 → 一律 `rounded-[var(--radius-lg)]`。
- **Pencil／匯出／gh**：同 `dsr-3a` 已知陷阱。

### Source tree

```
ux-design.pen、pen-tokens.json、screenshots/flow-c-search-settings/{c4,c7,c23}-{d,m}.png、c21-d.png、c22-d.png  ← Task 1
apps/web/src/routes/settings/connection.tsx；components/settings/QBittorrentForm.tsx、ArrConnectionForm.tsx（+spec） ← Task 2
apps/web/src/components/settings/ApiKeysForm.tsx（+spec）                                                          ← Task 3-5
apps/web/src/routes/test/-gallery.fixtures.tsx；tests/e2e/settings-shell.spec.ts（追加）                            ← Task 6
```

### Cross-Stack Split Check

後端 task **0**、前端／設計／測試 task 6 → 不觸發。規模：一個大元件重排（`ApiKeysForm`）＋兩個表單的 class 調整——與 `dsr-4b-2` 同級。

### Time-dependent visual coverage

- **N/A — no wall-clock-reading components touched.**

### References

- [Source: `routes/settings/connection.tsx:18-30`；`QBittorrentForm.tsx:154, 214, 244, 250, 268`；`ArrConnectionForm.tsx:186, 273, 338, 504-508, 559`；`ApiKeysForm.tsx:57, 105, 218-224, 249-268, 273-300, 304, 330-374, 475-503, 529, 566-568`]
- [Source: `apps/api/internal/services/key_settings_service.go:52-58, 139`；`components/ui/TmdbAttribution.tsx:1-8`]
- [Source: `ux-design.pen` 節點見上表 —— SM 建單時以 Pencil MCP 讀出（2026-09-23）]
- [Source: `sprint-status.yaml` → `dsr-3-flow-c-settings`、`backlog-asr-key-test-probe`、`sub-2-1b-key-config-page`、`13-6`（C23 的來源）、`sub-6-9-tmdb-attribution`]
- [Source: `dsr-3a-settings-shell-page-header-and-states.md`（拆單裁定、共用元件 API）；DESIGN.md `:364, 615`；`project_pen_design_token_system`（狀態詞彙）]

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5 — `claude-opus-5-5`（dev-story，Amelia，2026-09-23）

### Debug Log References

- Pencil：`problems` 67 → **67**。中途一度到 74：C4-M／C7-M／C23-M／C23-D 的內容變長（手機字級 12→14、C7-M 補回說明與測試鈕）超出畫面高度 → 四張畫面各加高（C4-M 844→868、C7-M 844→980、C23-M 1540→1604、C23-D 1533→1553，比照 C23 本來就是長捲動稿）；C7-D `pegsz` 改 `fill_container` 後歸零。存檔走選單 Save、磁碟 grep 到 `spec-note-dsr-3b`（`ZCZLf`）。匯出 196/196，只保留 8 張：`c4-d`、`c4-m`、`c7-d`、`c7-m`、`c21-d`、`c22-d`、`c23-d`、`c23-m`。
- 本機 e2e／visual 要自己起後端（`VIDO_DATA_DIR=./vido-data VIDO_PORT=8080 go run ./cmd/api`）與前端（`NX_DAEMON=false npx nx serve web`）；全套 unit 跑完會連帶收掉這兩個程序，visual 前要重起。

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created
- 🔗 AC Drift: **FOUND — see below**（checked: `僅支援 Claude|api-keys-load-error|無法讀取金鑰設定|compliance is not conditional|flex-col` across `_bmad-output/implementation-artifacts/*.md`）
  - 🔗 AC Drift: `sub-2-1b-key-config-page`（其 spec 註解「The other rows still SHOW 測試, disabled with a reason」）— 非 Claude 列「停用的測試鈕＋每列一句說明」→「不畫測試鈕、卡片底下說一次」。本張 🔴 #11 的刻意變更；對應 spec 條目已改寫。
  - 🔗 AC Drift: `sub-2-1b-key-config-page` CR（fail-soft：讀不到時表單照樣畫、狀態顯示「無法確認」）與 `sub-6-9-tmdb-attribution`（TMDB 標示不因讀取失敗而消失）→ **REUSE，所以本張 🔴 #19 沒有照字面做**（見偏離 1）。
  - 🔗 AC Drift: `feat-settings-*`／QBittorrentForm 既有 spec「button container uses flex-col on mobile」→ 手機兩欄並排（🔴 #6 的刻意變更，與 Arr 卡片一致）。
- 📎 Contract Stamps: NONE（no [@contract-v*] stamps in this story or upstream refs — 純前端外觀，讀的是既有 `/settings/keys`、`/settings/qbittorrent`、`/settings/{sonarr,radarr}` implicit v0）
- 🎭 A11y Pre-Flight: PASS（3 components — ApiKeysForm／QBittorrentForm／ArrConnectionForm；0 jsx-a11y warnings on these files；0 introduced。框內「測試」鈕仍是 `<button>`、`data-testid` 與測試中／結果的 `role="status"` 回饋不變；輸入框 `aria-describedby` 仍指向說明（說明移到框下方不影響關聯）；C21 改 `role="alert"`；讀取失敗的重試用 `aria-disabled` 不丟焦點；手機所有按鈕 `max-sm:min-h-11`）
- ⚠️ **偏離 1（載入失敗，🔴 #19／AC #6）**：story 說換成 `SettingsErrorState`（整頁）。但金鑰頁的讀取失敗有兩個先前刻意的決定：sub-2-1b CR「表單照樣畫，頁面不能是死路」、sub-6-9「TMDB 標示不因讀取失敗而消失（合規不是有條件的）」——整頁錯誤會同時違反兩者。所以**保留 fail-soft 的橫幅＋表單**，只改該改的：拿掉後端原文（`error.message`）、改成 SM 擬的「無法讀取金鑰設定／與後端的連線中斷了。已存的金鑰不受影響。」、加「重試」（`refetch`，`aria-disabled` 防焦點掉落）。
- ⚠️ **偏離 2（C22 的視覺夾具）**：AC #8 列了 `settings-api-keys-form/http-warning`。gallery 跑在 localhost＝secure context，C22 的警告根本不會出現（這正是那段程式碼的設計：看 `window.isSecureContext`，不看 protocol），沒有注入點就拍不到 → 沒做；C22 由 unit spec 守（標題／說明／勾選框色票、泥金不得出現）。
- ⚠️ **稿與碼的一致化順帶改了一處碼**：TMDB 列的 placeholder「TMDb API Key」→「TMDB API Key」（品牌拼法 🔴 #13 一律 TMDB；稿同步）。
- **Task 1（稿）**：C4-D／C4-M／C23-D／C23-M 四張的 8 顆按鈕補 Plug／Save 圖示（碼→稿）；QB 輸入框 8 個等寬字改一般字；手機 C4-M／C23-M 29 個欄位標籤、輸入值改 Body 14、卡標題改 BodyLg 16；C4-M Base Path 提示改「（選填，反向代理用）」；C23-D／C23-M Radarr 根資料夾補「送出電影請求時…」提示；「未設定」小標 `o82CE` 已經是碼的 NEUTRAL 樣式（`$bg-tertiary`／`$text-secondary`／circle-dashed／600）→ 不用改。C7-D：Claude 列改成已儲存（遮罩＋編輯／清除／測試、原輸入框 `enabled:false`）、TMDB／ASR 刪框內測試鈕、TMDB 補環境變數覆蓋說明與 TMDB 標示、「僅 Claude 金鑰支援連線測試。」（拿掉「由服務自行驗證」）移進卡片、儲存鈕畫成停用；C7-M 同樣＋三列補回說明、完整頁面說明、pill「目前由環境變數提供」；C21-D 刪掉到不了的遮罩值、`nAPD6` 改 placeholder、刪畫面裡的設計註記 `u9e2Mf`；全部「TMDb」→「TMDB」（剩 0）。`spec-note-dsr-3b`。
- **Task 2**：QB 欄位 `space-y-5`→`space-y-4`、標籤 `mb-1.5`→`mb-2`、按鈕 `px-4 font-medium`→`px-5 font-semibold`、按鈕列 `flex-col`→`grid grid-cols-2 … md:gap-3`、Base Path 提示 `text-xs`；Arr 卡 `md:p-8`→`md:p-6`、標題固定 `text-base`、欄位 `space-y-6`→`space-y-4`、按鈕間距 `md:gap-4`→`md:gap-3`；connection route 卡片間距 `gap-6`→`gap-4`、QB 卡同樣的內距與標題。
- **Task 3–5**：`ApiKeysForm` 每列重排為「標籤＋狀態（＋遮罩）→ 輸入框（44 高、`bg-tertiary`、等寬，Claude 的測試鈕 `absolute` 嵌在框內右側、`pr-24` 留位）或已儲存時的『編輯／清除／測試』→ 說明 → 環境變數覆蓋說明 → 測試結果 → TMDB 標示」；非 Claude 列不渲染測試鈕，卡片底下一句；列標籤 600／`text-primary`（唯讀時 `text-muted`）、pill 12／600；列間 `gap-4`（無分隔線）、卡片實色 `radius-lg`；「儲存金鑰」44 高 600、手機滿寬；手機外層卡 `max-sm:bg-transparent`、每列自成一卡；C21 硃砂兩行＋`Lock`＋`role="alert"`、停用輸入框 `disabled:text-[var(--text-disabled)]`；C22 赭色標題＋說明、18px `ShieldAlert`、勾選框 `accent-[var(--warning-text)]`、標籤 12／600 赭。
- **測試**：新／改 unit — ApiKeysForm 7 條新（C7 列版面）＋ C21 2 條、C22、非 Claude、讀取失敗改寫；QB 4 條、Arr 3 條。`nx test web` **284 files／4178 tests 全綠**；typecheck、lint 0 error。e2e `settings-shell.spec.ts` 追加 3 條（390 每把金鑰一張卡且外層卡透明、儲存鈕寬＝欄寬；1440 一張卡；框內測試鈕在框內且文字不壓到它）＋既有 `arr-settings.spec.ts`，chromium `--repeat-each=3` **36／36**。
- **Mutation：unit 15／15 紅、e2e 2／2 紅**（QB 欄距／按鈕列／按鈕字重／提示字級、Arr 內距／欄距、pill 11px、拿掉框內測試鈕、C21 改回赭、C22 勾選框改回泥金、TMDB 列可測、重試不呼叫 refetch、「儲存」、手機外層卡不透明、錯誤原文回來；e2e：拿掉 `pr-24`、外層卡不透明）。
- **視覺基準**：新 darwin 3 張（`settings-api-keys-form`、`/mobile`（viewport 390）、`/no-encryption-key`）；`settings-qbittorrent-form` default／hover／focus 與 `settings-arr-connection-form/{sonarr-connected,radarr-unconfigured}` 對稿**預期變動**，darwin 更新、5 張 stale `-linux` 刪除待 CI bootstrap。兩個連線夾具的 `penNode` 從 `'screen-section'` 改成 `'6UCtX'`／`'Qva0y'`。`retry-retry-notifications` 仍是本機環境差異（CI 綠，見 dsr-3a）。
- 🔍 **/ship 對抗式 CR（2026-09-23，獨立 context）0 HIGH／1 MEDIUM／3 LOW／2 NIT，全部吸收**：① 🟠 按「重試」時，TanStack 會把沒有快取的失敗查詢退回 loading，整頁換成轉圈、表單與 TMDB 標示一起消失——正好違反偏離 1 的兩條理由 → 本地 `retrying` 狀態讓轉圈只在第一次載入出現、重試期間橫幅／表單／標示都留著；新增 `ApiKeysForm.retry.spec.tsx`（真的 QueryClient，mock 服務層不 mock hook），拿掉修法 → 紅。② 讀取失敗時標籤不再變灰（只有 `writable:false` 才灰）。③ 手機上編輯已存金鑰時，輸入框與「取消」改上下疊，不再只剩 ~140px 可見。④ e2e「文字不壓到測試鈕」改在測試鈕最寬（轉圈中）時量，拿掉沒作用的 `fill`。⑤ 清除確認不再依賴 `showInput`（來源翻轉時不會殘留）。⑥ 重試仍失敗時 alert 以 `errorUpdatedAt` 為 key 重新宣讀。之後 unit 285 files／4180 綠、e2e `--repeat-each=3` 36／36。
- 🎨 UX Verification: PASS（`c4-d`／`c4-m`／`c7-d`／`c7-m`／`c21-d`／`c22-d`／`c23-d`／`c23-m`：稿已改成碼的樣子的部分逐項對過；碼改的部分——卡內距 24／16、欄距 16、標籤到框 8、QB 按鈕 600／20、手機並排、框內測試鈕 28 高 12px、說明在框下、儲存金鑰 44、手機一把一卡、C21 硃砂兩行、C22 赭——與稿一致）

### File List

- `ux-design.pen`
- `_bmad-output/pen-tokens.json`
- `_bmad-output/screenshots/flow-c-search-settings/{c4-d,c4-m,c7-d,c7-m,c21-d,c22-d,c23-d,c23-m}.png`
- `apps/web/src/components/settings/ApiKeysForm.tsx`、`ApiKeysForm.spec.tsx`
- `apps/web/src/components/settings/QBittorrentForm.tsx`、`QBittorrentForm.spec.tsx`
- `apps/web/src/components/settings/ArrConnectionForm.tsx`、`ArrConnectionForm.spec.tsx`
- `apps/web/src/routes/settings/connection.tsx`
- `apps/web/src/routes/test/-gallery.fixtures.tsx`
- `tests/e2e/settings-shell.spec.ts`
- `tests/visual/components.visual.spec.ts-snapshots/components/settings-api-keys-form/{default,mobile/default,no-encryption-key/default}-visual-darwin.png`（新）
- `tests/visual/components.visual.spec.ts-snapshots/components/settings-qbittorrent-form/{default,hover,focus}-visual-darwin.png`（改）、`-linux.png`（刪）
- `tests/visual/components.visual.spec.ts-snapshots/components/settings-arr-connection-form/{sonarr-connected,radarr-unconfigured}/default-visual-darwin.png`（改）、`-linux.png`（刪）
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
- `_bmad-output/implementation-artifacts/dsr-3b-settings-connection-and-keys.md`

### Change Log

| Date | Change |
| --- | --- |
| 2026-09-23 | Task 1：設計稿——C4／C23 按鈕圖示、字級、提示；C7-D／C7-M 已儲存狀態、非 Claude 無測試、環境變數說明、TMDB 標示、停用儲存鈕；C21 刪遮罩與註記；TMDb→TMDB；四張加高；`spec-note-dsr-3b` |
| 2026-09-23 | Task 2：連線設定兩份表單的間距、按鈕、卡片 |
| 2026-09-23 | Task 3–5：金鑰每列重排、框內測試鈕、手機一把一卡、C21 硃砂、C22 赭、讀取失敗改人話＋重試（偏離 1） |
| 2026-09-23 | Task 6：3 個新夾具、e2e 3 條、mutation 17／17、全套 web 4178 綠 |
