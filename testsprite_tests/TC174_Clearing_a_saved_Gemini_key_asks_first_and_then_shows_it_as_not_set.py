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
        
        # -> Open 「設定」 and inspect the available key-management controls.
        # 設定 link
        elem = page.get_by_test_id("nav-settings")
        await elem.click(timeout=10000)
        
        # -> Open 「金鑰設定」 to inspect the Gemini key row and its current state.
        # 連線：金鑰設定 link
        elem = page.get_by_test_id("settings-tab-keys")
        await elem.click(timeout=10000)
        
        # -> Enter the provided test key in the Gemini key field and inspect the page before saving.
        # AIza… password field
        elem = page.get_by_role("textbox", name="Gemini（選填）")
        await elem.wait_for(state="visible", timeout=10000)
        await elem.fill("AIzaSyTestOnlyKeyForVido000000000000")
        
        # -> Save the entered Gemini key and verify that the Gemini row changes to 「已設定」.
        # 儲存金鑰 button
        elem = page.get_by_test_id("key-save")
        await elem.click(timeout=10000)
        
        # -> Click the Gemini row’s 「清除」 control and inspect the confirmation message.
        # 清除 button
        elem = page.get_by_test_id("key-clear-gemini")
        await elem.click(timeout=10000)
        
        # -> Confirm clearing the Gemini key and verify that the Gemini row changes to 「尚未設定」.
        # 確認清除 button
        elem = page.get_by_test_id("key-clear-confirm-yes-gemini")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> The Gemini row shows 已設定 after the supplied key is saved.
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: The Gemini row shows 已設定 after saving the key.
        await expect(page.locator("#root").nth(0)).to_contain_text("Gemini\uff08\u9078\u586b\uff09 \u5df2\u8a2d\u5b9a", timeout=15000), "The Gemini row shows \u5df2\u8a2d\u5b9a after saving the key."
        
        # --> Clearing the Gemini key displays the environment-variable fallback confirmation.
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: The clear confirmation explains that the environment-variable key will be used.
        await expect(page.locator("#root").nth(0)).to_contain_text("\u6e05\u9664\u5f8c\u5c07\u6539\u7528\u74b0\u5883\u8b8a\u6578\u7684\u91d1\u9470", timeout=15000), "The clear confirmation explains that the environment-variable key will be used."
        
        # --> After confirming the clear, the Gemini row shows 尚未設定.
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: The Gemini row shows 尚未設定 after the clear is confirmed.
        await expect(page.locator("#root").nth(0)).to_contain_text("Gemini\uff08\u9078\u586b\uff09 \u5c1a\u672a\u8a2d\u5b9a", timeout=15000), "The Gemini row shows \u5c1a\u672a\u8a2d\u5b9a after the clear is confirmed."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    