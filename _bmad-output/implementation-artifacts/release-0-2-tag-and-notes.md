# Release 0.2：tag `v0.2.0`、release notes、GitHub About

Status: done

**Source:** 2026-10-14 party mode，v0.2 發版單 ⑤。Alexyu 選 A：release notes 與 About 文案直接用 founder-growth 的草稿。

## 前提（都要 ✅ 才 tag）

- ① `release-0-2-pipeline-default` #788 ✅
- ②a/②b 文件 #789／#790 ✅、③ README 定位 #791 ✅
- ElevenLabs 正式機驗證 ✅（14:42）、Gemini 整集 ✅（15:08）
- 實測挖出的三個 bug：#792 季別寫法 ✅、#793 人名詞彙表 ✅、#795 字幕最少 1 秒 ✅
- **最後一次 NAS 重聽**（同一段、image `d1149dd7`）：人名「謝拉馬威／巴巴禾斯」、不到 1 秒的句子 ≤5、沒有 0 秒句 → ✅ 16:37：謝拉馬威 ×4、巴巴禾斯 ×5、芭麗絲 ×2；77 句、0 秒 0 句、不到 1 秒 4 句、中位數 1.80 秒；keyterms=85、每段記 0.264/h。

## Acceptance Criteria

1. git tag `v0.2.0` 打在含 #795 的 main 上；CI Docker 發出 image `0.2.0`、`0.2`、`0`、`latest`。
2. GitHub release `v0.2.0`：標題「v0.2.0 內測第二版」、內文＝核可的草稿（升級前三件事、你會看到的差別、還不會的、安裝、回報問題）。
3. GitHub About 改成核可的一句話；topics 不動。
4. sprint-status：①～⑤ 全部 done；`disc-2026-10-asr-cues-too-short-to-read`、`disc-2026-10-scribe-keyterms`、`disc-2026-10-asr-route-season-drawer-skipped` 的正式機驗證結果記上。
5. 停損條件（founder-pm，2026-10-14）寫進 sprint-status：**2026-11-15 前沒有任何一個外部的人裝起來並跑出一支字幕 → 回頭檢視定位，不加功能。**

## Dev Agent Record

### Agent Model Used

Claude Fable 5.1（claude-fable-5-1），2026-10-14。

### Completion Notes List

- 2026-10-14：#796 合併後 `git tag -a v0.2.0` 推上去 → Docker workflow（run 38537264244）建好 image → `gh release create v0.2.0 --latest`（文案＝核可草稿）→ `gh repo edit --description`。
- Release：https://github.com/j620656786206/vido/releases/tag/v0.2.0

### File List

- `_bmad-output/implementation-artifacts/sprint-status.yaml`、本檔
