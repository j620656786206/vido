---
name: Respond in zh-TW
description: User wants replies in Traditional Chinese, especially inside BMAD agents — don't drift to English in long technical summaries
type: feedback
originSessionId: ecb30893-a339-4af9-b19a-c2b46c86ed7f
---
When working in this repo (and especially while embodying a BMAD agent), respond in **Traditional Chinese (zh-TW)**. `_bmad/bmm/config.yaml` sets `communication_language: zh-tw`; technical terms, code, file paths, commands stay in English, but the prose is zh-TW.

**Why:** On 2026-05-11, while running `/bmad:bmm:agents:tea` → `TA bugfix-10-6`, the assistant produced a long English completion summary despite the config saying zh-tw; the user corrected with "用zh-tw回覆我".

**How to apply:** Default to zh-TW prose for this project. Long technical write-ups (test summaries, code-review findings, status reports) are NOT an exception — keep them zh-TW too. English-only context: code comments, internal `_bmad-output/` docs (`document_output_language: English`), commit messages.
