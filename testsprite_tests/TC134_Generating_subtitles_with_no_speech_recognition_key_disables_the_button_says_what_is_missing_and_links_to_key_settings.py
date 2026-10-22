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
        
        # -> Open the 電影 library and locate 全面啟動.
        # 電影 15 link
        elem = page.get_by_test_id("nav-movies")
        await elem.click(timeout=10000)
        
        # -> Open the 全面啟動 movie page from the visible movie library result.
        # 無字幕源 8.4 全面啟動 2010 link
        elem = page.get_by_test_id("poster-v2-seed-mv-006")
        await elem.click(timeout=10000)
        
        # -> Open the 管理字幕 dialog for 全面啟動 and inspect the generation control and ASR-key explanation.
        # 管理字幕 button
        elem = page.get_by_test_id("action-manage-subtitle")
        await elem.click(timeout=10000)
        
        # -> Click “前往設定” and verify the 金鑰設定 page opens at /settings/keys.
        # 前往設定 button
        elem = page.get_by_test_id("helper-goto-settings")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> 金鑰設定頁面已於 /settings/keys 開啟。
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: The URL is /settings/keys.
        await expect(page).to_have_url(re.compile("/settings/keys"), timeout=15000), "The URL is /settings/keys."
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: The page displays 金鑰設定.
        await expect(page.locator("#root").nth(0)).to_contain_text("\u91d1\u9470\u8a2d\u5b9a", timeout=15000), "The page displays \u91d1\u9470\u8a2d\u5b9a."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    