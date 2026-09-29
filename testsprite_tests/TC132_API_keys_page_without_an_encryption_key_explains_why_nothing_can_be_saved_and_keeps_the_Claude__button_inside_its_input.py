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
        
        # -> Open the '金鑰設定' settings page by navigating to /settings/keys and then verify the page loads.
        await page.goto("http://localhost:8090/settings/keys")
        try:
            await page.wait_for_load_state("domcontentloaded", timeout=5000)
        except Exception:
            pass
        
        # --> Assertions to verify final state
        
        # --> Settings page shows the heading 金鑰設定 and a red alert about the missing ENCRYPTION_KEY.
        # Assert-outcome: passed
        # Assert: Page shows the heading '金鑰設定'.
        await expect(page.locator("#root").nth(0)).to_contain_text("\u91d1\u9470\u8a2d\u5b9a", timeout=15000), "Page shows the heading '\u91d1\u9470\u8a2d\u5b9a'."
        # Assert-outcome: passed
        # Assert: Alert shows the missing ENCRYPTION_KEY message.
        await expect(page.locator("#root").nth(0)).to_contain_text("\u672a\u8a2d\u5b9a\u52a0\u5bc6\u91d1\u9470\uff0c\u7121\u6cd5\u5b89\u5168\u5132\u5b58 API \u91d1\u9470 \u8acb\u8a2d\u5b9a ENCRYPTION_KEY \u5f8c\u91cd\u555f\u5bb9\u5668", timeout=15000), "Alert shows the missing ENCRYPTION_KEY message."
        
        # --> The three key rows Claude（翻譯）, TMDB and 雲端 ASR（選配） are present and each shows the status 尚未設定.
        # Assert-outcome: passed
        # Assert: Each of the three key rows shows the status '尚未設定'.
        await expect(page.locator("#root").nth(0)).to_contain_text("Claude\uff08\u7ffb\u8b6f\uff09 \u5c1a\u672a\u8a2d\u5b9a \u6e2c\u8a66 \u7528\u65bc\u5b57\u5e55\u7ffb\u8b6f\u8207 AI \u6a94\u540d\u89e3\u6790\u3002\u5132\u5b58\u5f8c\u7acb\u5373\u751f\u6548\uff0c\u7121\u9700\u91cd\u555f\u4f3a\u670d\u5668\u3002 TMDB \u5c1a\u672a\u8a2d\u5b9a  \u7528\u65bc\u4e2d\u7e7c\u8cc7\u6599\u8207\u6d77\u5831\u3002\u5132\u5b58\u5f8c\u9700\u91cd\u555f\u4f3a\u670d\u5668\u624d\u6703\u751f\u6548\u3002 \u96f2\u7aef ASR\uff08\u9078\u914d\uff09 \u5c1a\u672a\u8a2d\u5b9a  \u9078\u914d\uff1a\u96f2\u7aef\u8a9e\u97f3\u8fa8\u8b58\u3002\u5132\u5b58\u5f8c\u7acb\u5373\u751f\u6548\uff0c\u7121\u9700\u91cd\u555f\u4f3a\u670d\u5668\u3002\u672a\u8a2d\u5b9a\u6642\u4ecd\u53ef\u4f7f\u7528\u5167\u5efa\u7684\u5b57\u5e55\u4f86\u6e90\u3002", timeout=15000), "Each of the three key rows shows the status '\u5c1a\u672a\u8a2d\u5b9a'."
        
        # --> Only the Claude key has a 測試 button and the page states that only Claude supports connection testing.
        await page.get_by_test_id("key-test-claude").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: passed
        # Assert: The Claude row's 測試 button is visible.
        await expect(page.get_by_test_id("key-test-claude").nth(0)).to_be_visible(timeout=15000), "The Claude row's \u6e2c\u8a66 button is visible."
        # Assert-outcome: passed
        # Assert: The page states only the Claude key supports connection testing.
        await expect(page.locator("#root").nth(0)).to_contain_text("\u50c5 Claude \u91d1\u9470\u652f\u63f4\u9023\u7dda\u6e2c\u8a66\u3002", timeout=15000), "The page states only the Claude key supports connection testing."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    