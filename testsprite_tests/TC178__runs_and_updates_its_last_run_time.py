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
        
        # -> Open the Settings page using the visible sidebar navigation.
        # 設定 link
        elem = page.get_by_test_id("nav-settings")
        await elem.click(timeout=10000)
        
        # -> Open “字幕設定” and inspect the official-subtitle learning control and its current status.
        # 媒體庫：字幕設定 link
        elem = page.get_by_test_id("settings-tab-subtitle")
        await elem.click(timeout=10000)
        
        # -> Click “重新從官方字幕學習” and wait for the page to show the run result.
        # 重新從官方字幕學習 button
        elem = page.get_by_test_id("official-subtitle-mining-start")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> The official-subtitle learning status shows a last-run entry.
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: Verify the page shows the official-subtitle learning last-run status.
        await expect(page.locator("#root").nth(0)).to_contain_text("\u4e0a\u6b21", timeout=15000), "Verify the page shows the official-subtitle learning last-run status."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    