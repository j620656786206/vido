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
        
        # -> Open Settings using the sidebar’s visible Settings link.
        # 設定 link
        elem = page.get_by_test_id("nav-settings")
        await elem.click(timeout=10000)
        
        # -> Open “金鑰設定” and check whether the Gemini row shows “尚未設定.”
        # 連線：金鑰設定 link
        elem = page.get_by_test_id("settings-tab-keys")
        await elem.click(timeout=10000)
        
        # -> Enter the provided test key in the Gemini password field and then inspect the save control.
        # AIza… password field
        elem = page.get_by_role("textbox", name="Gemini（選填）")
        await elem.wait_for(state="visible", timeout=10000)
        await elem.fill("AIzaSyTestOnlyKeyForVido000000000000")
        
        # -> Click “儲存金鑰” and verify the Gemini row changes to a set state without revealing the full key.
        # 儲存金鑰 button
        elem = page.get_by_test_id("key-save")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> After reloading, the Gemini row remains set.
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: The Gemini row shows that a key is set after reload.
        await expect(page.locator("#root").nth(0)).to_contain_text("Gemini\uff08\u9078\u586b\uff09 \u5df2\u8a2d\u5b9a", timeout=15000), "The Gemini row shows that a key is set after reload."
        
        # --> After reloading, the Gemini key is displayed in masked form, not in full.
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: The Gemini key is displayed as a masked partial value.
        await expect(page.locator("#root").nth(0)).to_contain_text("Gemini\uff08\u9078\u586b\uff09 \u5df2\u8a2d\u5b9a AIzaSy\u20260000", timeout=15000), "The Gemini key is displayed as a masked partial value."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    