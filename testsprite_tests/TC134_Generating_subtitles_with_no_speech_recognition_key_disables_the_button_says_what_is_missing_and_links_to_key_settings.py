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
        
        # -> Open the movie detail page for 教父 by navigating to /media/movie/seed-mv-001
        await page.goto("http://localhost:8090/media/movie/seed-mv-001")
        try:
            await page.wait_for_load_state("domcontentloaded", timeout=5000)
        except Exception:
            pass
        
        # -> Click the '管理字幕' button to open the 管理字幕 dialog.
        # 管理字幕 button
        elem = page.get_by_test_id("action-manage-subtitle")
        await elem.click(timeout=10000)
        
        # -> Click the '前往設定' button next to the ASR key message to open the keys settings page
        # 前往設定 button
        elem = page.get_by_test_id("helper-goto-settings")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> 點擊「前往設定」後已導向金鑰設定頁面 /settings/keys，且頁面顯示「金鑰設定」
        # Assert-outcome: passed
        # Assert: The browser navigated to a URL containing /settings/keys.
        await expect(page).to_have_url(re.compile("/settings/keys"), timeout=15000), "The browser navigated to a URL containing /settings/keys."
        # Assert-outcome: passed
        # Assert: The settings page shows the '金鑰設定' heading/link.
        await expect(page.get_by_test_id("settings-tab-keys").nth(0)).to_have_text("\u91d1\u9470\u8a2d\u5b9a", timeout=15000), "The settings page shows the '\u91d1\u9470\u8a2d\u5b9a' heading/link."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    