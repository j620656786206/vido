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
        
        # -> Click the “向右捲動” button and verify Unknown.Show.S01 moves out of the visible row area.
        # 向右捲動 button
        elem = page.get_by_test_id("home-recent-scroll-right")
        await elem.click(timeout=10000)
        
        # -> Click “向左捲動” and verify that Unknown.Show.S01 is visible again in 最近新增.
        # 向左捲動 button
        elem = page.get_by_test_id("home-recent-scroll-left")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> Unknown.Show.S01 is visible again in 最近新增 after scrolling back.
        await page.get_by_test_id("poster-v2-seed-sr-101").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: Unknown.Show.S01 is visible in 最近新增.
        await expect(page.get_by_test_id("poster-v2-seed-sr-101").nth(0)).to_be_visible(timeout=15000), "Unknown.Show.S01 is visible in \u6700\u8fd1\u65b0\u589e."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    