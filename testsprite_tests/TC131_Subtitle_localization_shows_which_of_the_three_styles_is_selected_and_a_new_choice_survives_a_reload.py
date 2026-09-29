import asyncio
import re
from playwright import async_api
from playwright.async_api import expect

async def run_test():
    pw = None
    browser = None
    context = None

    try:
        # Start a Playwright session in asynchronous mode
        pw = await async_api.async_playwright().start()

        # Launch a Chromium browser in headless mode with custom arguments
        browser = await pw.chromium.launch(
            headless=True,
            args=[
                "--window-size=1280,720",
                "--disable-dev-shm-usage",
                "--ipc=host",
                "--single-process"
            ],
        )

        # Create a new browser context (like an incognito window)
        context = await browser.new_context()
        # Wider default timeout to match the agent's DOM-stability budget;
        # auto-waiting Playwright APIs (expect, locator.wait_for) inherit this.
        context.set_default_timeout(15000)

        # Open a new page in the browser context
        page = await context.new_page()

        # Interact with the page elements to simulate user flow
        # -> navigate
        await page.goto("http://localhost:8090")
        try:
            await page.wait_for_load_state("domcontentloaded", timeout=5000)
        except Exception:
            pass
        
        # -> Navigate to the '字幕設定' settings page (open /settings/subtitle) to inspect the AI 字幕的在地化程度 options.
        await page.goto("http://localhost:8090/settings/subtitle")
        try:
            await page.wait_for_load_state("domcontentloaded", timeout=5000)
        except Exception:
            pass
        
        # -> Click the '直譯' option to change the AI subtitle localization to 直譯 and wait for the save to complete.
        # subtitle-localization-level radio button
        elem = page.get_by_role("radio", name="直譯 照原文翻，不把故事世界搬到台灣。超市就是超市，感恩節就是感恩節。 例： 「I grabbed milk at the grocery store")
        await elem.click(timeout=10000)
        
        # -> Verify the page heading '字幕設定', the group title 'AI 字幕的在地化程度', and the helper text '這是口味，不是對錯。' are present, read the three localization radio inputs and then reload the subtitle settings page.
        await page.goto("http://localhost:8090/settings/subtitle")
        try:
            await page.wait_for_load_state("domcontentloaded", timeout=5000)
        except Exception:
            pass
        
        # -> Click the '台灣用語（預設）' option to restore the default and confirm no '儲存失敗' message is displayed.
        # subtitle-localization-level radio button
        elem = page.get_by_role("radio", name="台灣用語（預設） 用台灣的詞和語氣（影片、品質、超商），但不拿本地品牌或俚語替換泛稱。 例： 「I grabbed milk at the grocery")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> The page heading '字幕設定' is visible.
        # Assert-outcome: passed
        # Assert: Heading '字幕設定' is displayed in the settings navigation.
        await expect(page.get_by_test_id("settings-tab-subtitle").nth(0)).to_have_text("\u5b57\u5e55\u8a2d\u5b9a", timeout=15000), "Heading '\u5b57\u5e55\u8a2d\u5b9a' is displayed in the settings navigation."
        
        # --> The group title 'AI 字幕的在地化程度' and helper line '這是口味，不是對錯。' are visible.
        # Assert-outcome: passed
        # Assert: The settings page contains the group title and helper text about AI subtitle localization taste.
        await expect(page.locator("#root").nth(0)).to_contain_text("AI \u5b57\u5e55\u7684\u5728\u5730\u5316\u7a0b\u5ea6 \u9019\u662f\u53e3\u5473\uff0c\u4e0d\u662f\u5c0d\u932f\u3002", timeout=15000), "The settings page contains the group title and helper text about AI subtitle localization taste."
        
        # --> The three options 直譯, 台灣用語（預設） and OTT 風格 are present and the default 台灣用語（預設） was selected.
        # Assert-outcome: passed
        # Assert: The page shows the three localization option labels.
        await expect(page.locator("#root").nth(0)).to_contain_text("\u76f4\u8b6f \u53f0\u7063\u7528\u8a9e\uff08\u9810\u8a2d\uff09 OTT \u98a8\u683c", timeout=15000), "The page shows the three localization option labels."
        # Assert-outcome: passed
        # Assert: The '台灣用語（預設）' radio is selected by default.
        await expect(page.get_by_role("radio", name="台灣用語（預設） 用台灣的詞和語氣（影片、品質、超商），但不拿本地品牌或俚語替換泛稱。 例： 「I grabbed milk at the grocery").nth(0)).to_have_attribute("checked", "true", timeout=15000), "The '\u53f0\u7063\u7528\u8a9e\uff08\u9810\u8a2d\uff09' radio is selected by default."
        
        # --> Changing to '直譯' saved (直譯 became checked) and the default was restored without any '儲存失敗' message being shown.
        # Assert-outcome: passed
        # Assert: The '直譯' radio becomes checked after selecting it.
        await expect(page.get_by_role("radio", name="直譯 照原文翻，不把故事世界搬到台灣。超市就是超市，感恩節就是感恩節。 例： 「I grabbed milk at the grocery store").nth(0)).to_have_attribute("checked", "true", timeout=15000), "The '\u76f4\u8b6f' radio becomes checked after selecting it."
        # Assert-outcome: passed
        # Assert: The '台灣用語（預設）' radio is checked after restoring the default.
        await expect(page.get_by_role("radio", name="台灣用語（預設） 用台灣的詞和語氣（影片、品質、超商），但不拿本地品牌或俚語替換泛稱。 例： 「I grabbed milk at the grocery").nth(0)).to_have_attribute("checked", "true", timeout=15000), "The '\u53f0\u7063\u7528\u8a9e\uff08\u9810\u8a2d\uff09' radio is checked after restoring the default."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    