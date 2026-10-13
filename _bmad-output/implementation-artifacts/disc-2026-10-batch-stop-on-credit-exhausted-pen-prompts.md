# Pencil 提示詞：AI 帳號沒錢（Sally，2026-10-13）

給 `disc-2026-10-batch-stop-on-credit-exhausted-b` 用。兩段各自貼給 Pencil 的 Inline AI Agent，一次一段。跑完按 ⌘S。

節點 ID 是 2026-10-13 用 MCP 唯讀查的（`ux-design.pen`，main `4e1834d6`）。

---

## 提示詞 1：F11-SPEC-LOG 加兩種列

```
請只修改畫面 F11-SPEC-LOG（節點 tT3JX），不要動其他畫面。

這張是「生成工作區右欄的紀錄列」規格。每個 spec-entry 是一列樣本（row-box）加一段說明（spec-note）。

1. 新增 ⑤b：複製 spec-entry「xPTom」（⑤ 已停止），放在它的正下方（在 qwV2R 之前）。複製出來的那份改這些字，其他樣式都不動：
   - 樣本裡寫「芭比」的 evt-item（原節點 bVg7L）→「奧本海默」
   - 樣本裡寫「已達預算上限」的 evt-item（原節點 XCzdu）→「AI 帳號沒有額度」
   - spec-note（原節點 I4V8GW）→「⑤b 已停止・帳號沒錢（單部）｜OpenAI 或 Anthropic 帳號沒有額度了。跟預算上限一樣用赭色：片子沒壞，儲值後再按一次。後端原字串不上畫面。」
   圖示、「已停止」字樣和顏色（$warning-text）保持不變。

2. 新增 ⑪：複製 spec-entry「o8b2v5」（⑩ 已達預算上限），放在它的正下方（在 spec-omitted 文字 MOyan 之前）。複製出來的那份改這些字：
   - evt-stage「已達預算上限」（原節點 g8A0G）→「AI 帳號沒有額度」
   - evt-item「本批次」（原節點 EETWE）不變
   - evt-trail「$5.00」（原節點 ee6sr）→「OpenAI」，顏色保持 $text-primary
   - spec-note（原節點 UajNg）→「⑪ AI 帳號沒有額度｜批次層，赭色，跟⑩同一類：整批停下，剩下的片子是「已暫停」，不是失敗。尾端寫是哪一家（OpenAI／Anthropic），用主要文字色。」

3. 改 ④ 的說明（節點 I2WeeC），在最後面加一句：「帳號沒錢不算失敗，見⑤b。」

4. 改開頭說明（節點 VMUYs）：把「十一種列」改成「十三種列」，其他字不動。

5. 改播報說明（節點 HEgN0）：把「③–⑩」改成「③–⑪」，其他字不動。

6. 檢查：新的兩列 row-box 寬度跟其他列一樣是 400，沒有文字被切掉或溢出；整個 F11-SPEC-LOG 框會自動變高，下面的文字不要跟其他東西重疊。
```

---

## 提示詞 2：新畫面 F9b-D-v2（批次停在帳號沒錢）

```
請新增一個畫面，不要修改原本的 F9-D-v2。

1. 複製畫面「F9-D-v2 · 批次生成 — 預算上限」（節點 JMqPg），放在同一個群組「F · 桌面 Desktop」（節點 wJQM8）裡，找 F9-D-v2 附近的空位，不要跟任何畫面重疊。如果 F9-D-v2 上方有說明文字，比照放一份。
   新畫面命名為「F9b-D-v2 · 批次生成 — AI 帳號沒有額度」。

2. 在新畫面裡改黃色橫幅 budget-banner（原節點 GNV6A）裡的 bn-text 文字，數字樣式保持原本數字的樣式：
   - bn-t1（原「已達本次預算上限（」）→「OpenAI 帳號沒有額度了 — 已完成」
   - bn-n1（原「$5.00」）→「3」
   - bn-t2（原「）— 已完成」）→「部，剩餘」
   - bn-n2（原「3」）→「2」
   - bn-t3（原「部，剩餘」）→「部已暫停，儲值後按「下次繼續」」
   - bn-n3、bn-t4 設成不顯示（enabled: false）
   圖示（bn-ic）和底色（$warning-tint）不動。

3. 在同一個橫幅裡、文字後面加一個連結文字「去 OpenAI 儲值 ↗」：
   - 字型、字級跟 bn-t1 一樣，字重 500，顏色 $accent-text，加底線。
   - 跟橫幅文字同一列，靠右或緊接在文字後面都可以，但不能換行擠壞橫幅。
   - 節點命名 bn-billing-link。

4. 底部的花費列 cost-row（原節點 elv26）：cost-used（原「$5.00」）→「$1.20」；cost-cap「$5.00」不變。意思是錢還沒用到上限，是帳號先沒錢。

5. 清單（item-list）、總進度「3 / 5」、按鈕「關閉」「下次繼續」都不動。

6. 在畫面上方（比照其他畫面的說明文字位置，不要蓋到畫面內容）加一段設計註記，顏色 $text-muted，字級 13：
   「設計註記：帳號沒錢＝已停止（赭色），不是失敗。版型跟 F9 預算上限一樣，只換橫幅文字、加儲值連結（新分頁開帳號的儲值頁）。剩下的片子照舊是『已暫停 — 下次繼續』。生成工作區（F11）的橫幅照抄這一句，狀態小標寫『沒有額度』。Anthropic 帳號沒錢時，把 OpenAI 換成 Anthropic。」

7. 檢查：橫幅沒有文字被切掉或溢出；新畫面跟其他畫面沒有重疊。
```

---

## 跑完之後

1. 在 Pencil 按 ⌘S。
2. 跟 Claude 說「跑完了」。Sally 會用 MCP 唯讀檢查，然後：
   - 把 F9b 的節點 ID 加進 `scripts/export-pen-screenshots.py` 的 `SCREENS`（`flow-f-subtitle-v2`、`f9b-d-v2`）。
   - 跑 `python3 scripts/export-pen-screenshots.py`，只留 `f9b-d-v2.png` 和 `f11-spec-log.png` 這兩張的變更。
   - 接著做 b 的程式。
