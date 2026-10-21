# Disc（文件）：README 補上「沒字幕一集」的整集實測價錢

Status: done

**Source：** 從 `disc-2026-10-readme-sidecar-asr-cost`（#883）拆出。那次在正式機跑一集 56 分鐘、沒字幕的影集，聽到 34 分鐘時 ElevenLabs 免費額度用完，README 只寫了聽寫單價。2026-10-21 Alexyu 升級 ElevenLabs Starter 方案後，接著跑完。

**為什麼要做：** README 的價錢表要放實測數字，不是推算。上一版把原本的「約 US$0.8（推算）」拿掉，改成「整集總價補量中」。

## Acceptance Criteria

1. 正式機把同一集跑完（Vido 會續跑：前 34 分鐘不重聽），兩次帳本的 `spent_usd` 加起來就是整集總價。
2. README「影集一集，沒有字幕、要先聽聲音」那一列換成整集實測的數字，寫明是對白多的影集，並提供對白少的對照。不寫片名。
3. `prettier --check README.md` 通過。

## Dev Agent Record

### Completion Notes

- 第一次（10-21 09:35）：聽寫 17 段、2,040 秒，`spent_usd` US$0.1496，第 18／29 段遇到 ElevenLabs `quota_exceeded` 停下。
- 第二次（10-21 09:55，Alexyu 升級 Starter 後）：估價顯示 `resumed_asr_minutes: 34`，只聽剩下的 12 段、1,357 秒；翻譯 938 句、19 次請求、輸入 87,703／輸出 17,334 tokens（claude-sonnet-5）。`spent_usd` US$0.6226。
- **整集總價 US$0.77**（聽寫約 US$0.25＋翻譯約 US$0.52），估價 US$0.89，沒有少算。
- 對照：《Tears of Steel》12 分鐘、對白少，聽寫＋翻譯實付 US$0.068（約每小時 US$0.33）。
- ⚠️ 2026-10-21 FB 留言回覆寫「沒字幕一集約 US$0.3～0.4」，是從 Tears of Steel 推算的，對白多的影集實際是兩倍；已提醒 Alexyu 補更正留言。
- 小問題：產出的第一句是 0～7 秒的「可以」，那段沒有對白，看起來是聽寫多聽出來的。沒有立案，先記在這裡。
- README 改成：「約 US$0.8（對白多的影集實測：56 分鐘 US$0.77，聽寫約 0.25＋翻譯約 0.52；對白少的片會便宜很多）」。
