# Release 0.2：README 第一段說清楚「給誰用、跟 Bazarr／Lingarr 差在哪」

Status: done

**Source:** 2026-10-14 party mode。Mary：FB 招募文第一則留言就是對手（Toby Wong 推 Lingarr）；Alexyu 在留言裡回的三個差異（台灣用語、角色名統一、沒字幕可以聽）只在留言裡，README 第一段沒有。founder-growth 寫初稿，Alexyu 2026-10-14 選 **A：直接用這版**。v0.2 發版單 ③。

## 查到的事（2026-10-14，main `66de7398`）

- `README.md:3-4` 是一句定位＋一句流程，`:6` 英文版，`:8` 分隔線；接著 `:10` 「為什麼做這個」。沒有任何一句提到 Bazarr／Lingarr，也沒有「給誰用」。
- 草稿每一句對應的功能：台灣用語＝`sub-7-4`（README `:32`）；陸劇只轉字＝2026-10-05 裁定（README `:32`）；角色名＝`sub-7-1`／`sub-7-5`（README `:33`）；沒字幕能聽＝ASR（README `:31`）；報價、上限、沒錢停＝README `:34`；Claude 或 Gemini 擇一＝#786（README `:56`）；只做英文對白＝README `:55`。
- 品牌硬邊界（`ai-context/core/brand.md`）：不說「免費」——寫的是「Gemini 可以用你的免費額度」「Vido 本身不收錢」，都是事實不是宣稱。

## Acceptance Criteria

1. README 在英文簡介（`:6`）之後、分隔線之前插入「給誰用」＋「跟 Bazarr／Lingarr 差在哪」＋「目前只做英文對白」三段，內容＝Alexyu 核可的草稿，逐字。
2. 每一句都對得到上面列的功能，不新增任何程式裡沒有的承諾。
3. 不動 README 其他段落；prettier 綠。

## Tasks / Subtasks

- [x] T1 插入段落（AC 1–2）
- [x] T2 prettier（AC 3）

## Dev Notes

- 這段是 Alexyu 的聲音；之後要改措辭由他改，不由 agent 潤。
- 英文版 README 段落（`:6`）沒有同步加英文的差異清單——README 是 zh-TW 為主、英文只有一句簡介，維持。

### Time-dependent visual coverage

N/A — 純文件。

## Dev Agent Record

### Agent Model Used

Claude Fable 5.1（claude-fable-5-1），2026-10-14，founder-growth 初稿、Alexyu 選 A。

### Completion Notes List

- 插在 `README.md` 英文簡介之後、`---` 之前。逐字用核可的草稿。

### Discovery Triage

N/A — no out-of-scope work discovered.

### File List

- `README.md`
