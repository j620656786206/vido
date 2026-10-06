# Disc：給語音辨識的人名提示只放角色名、只放拉丁字母、最多 12 個——40 個名字反而害它把名單念出來

Status: review

**Source:** 2026-10-07 第五次實測（`eval-see-s01e02-asr-vs-official.md` 第五節）：演員表補上後提示變成 41 個名字（含 zh-TW 演員名），Whisper 把名單念進字幕 5 句、對白剩三分之一、Jerlamarel 還是沒聽對。

## Story

身為 Vido 的使用者，我希望人名提示只幫忙、不添亂：字幕裡不會出現演員名單，對白不會因此變少。

## 設計

- `asrPromptNames`：只收**角色名**（billing order）＋可信名詞表；演員名不送（劇裡沒人會喊演員的名字）；含漢字／假名／韓文的名字跳過（`isLatinName`）。
- `BuildASRPrompt` 上限 40／700 → **12 個／200 字元**。
- `filterPromptEcho` 加 `isNameList`：段落用逗號／頓號／分號切開 ≥2 項、其中 ≥2 項且 ≥60% 是名單裡的名字 → `prompt_echo`。單一人名永遠不算名單。

## Acceptance Criteria

1. See 的提示 ≤12 個名字、全是拉丁字母的角色名（log `names=` ≤12）。
2. 「The Bank, Lord Diego, …, The Bank, Lord」這類亂序名單被記 `prompt_echo`；「Kofun, come here, Haniwa, now, please」保留。
3. `go test ./...`、staticcheck、lint:all 全綠。
4. **實測（2 次，約 $0.2）：** 句數回到不帶提示的水準（≥65）、字幕裡 0 句名單；Jerlamarel 拼法有無改善另記。沒改善就把提示預設關掉（另立單）。

## Dev Agent Record

- **Adversarial CR（2026-10-07，fresh agent）2H／2M／1L，全部修掉：**
  - H 逗號名單規則 60% 門檻會砍掉「Baba Voss, Maghra, come.」這種叫人名＋短句的真對白（7 條測試句全砍）→ 改成「每一項都得是名單裡的名字、或被切斷的名字開頭」，出現任何自由詞就保留；補 4 條對白負向案例。
  - H 存起來的演員表是 TMDb zh-TW 版，主角名可能被翻成中文而被「拉丁字母」規則跳過、剩下的反而是沒翻譯的小角色（正是實測看到的 The Bank／Lord Diego）；名詞表又按字母序把演員名混回來 → 用 metadata 名詞表的 zh→en 對照把角色名找回英文、保住 billing order；TermZh 等於演員名的名詞表列（演員對）跳過；補 zh-credits 測試。
  - M 多重角色「Baba Voss / Young Baba」整串進提示 → 取第一個。
  - M 200 字元上限時一條長片語會 `break` 把後面短名字全擋掉 → 改 `continue`。
  - L `isLatinName` 實作是「非 CJK」不是「拉丁」 → 改用 `unicode.Latin`（Zoë／Ægir 仍通過，Иван 不通過）。
- Claude Fable 5.1。🔗 AC Drift: FOUND — `disc-2026-10-asr-proper-names-inconsistent` 的「角色→演員→名詞表」改成「角色→名詞表」。📎 Contract Stamps: NONE。🎭 A11y: N/A。🎨 UX: SKIPPED。

### File List

- `apps/api/internal/ai/asr_prompt.go`、`asr_prompt_test.go`、`whisper_segments.go`（改）
- `apps/api/internal/services/transcription_asr_prompt.go`、`transcription_asr_prompt_test.go`（改）

## Change Log

| 日期 | 內容 |
|---|---|
| 2026-10-07 | CR 2H/2M/1L 修掉：名單規則改「全是名字才算」、zh 角色反查英文、多角色取第一、長片語 continue、Latin script。 |
| 2026-10-07 | create-story＋dev 同日；全綠，狀態 review，待 2 次實測。 |
