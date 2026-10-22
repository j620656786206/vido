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
        
        # -> Open the movie library using the sidebar’s 電影 link.
        # 電影 15 link
        elem = page.get_by_test_id("nav-movies")
        await elem.click(timeout=10000)
        
        # -> Select the 1990s decade filter and verify the resulting movie cards.
        # 1990s button
        elem = page.get_by_test_id("filter-decade-1990s")
        await elem.click(timeout=10000)
        
        # -> Select the 動畫 genre filter and check whether the page shows the expected filtered-empty message and a clear-filters control.
        # 動畫 button
        elem = page.get_by_test_id("filter-genre-動畫")
        await elem.click(timeout=10000)
        
        # -> Click「清除全部篩選」and verify that 教父 is visible again.
        # 清除全部篩選 button
        elem = page.get_by_test_id("clear-all-filters")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> 清除篩選後，教父再次顯示。
        await page.get_by_test_id("poster-v2-seed-mv-001").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: 教父應顯示在電影結果中。
        await expect(page.get_by_test_id("poster-v2-seed-mv-001").nth(0)).to_be_visible(timeout=15000), "\u6559\u7236\u61c9\u986f\u793a\u5728\u96fb\u5f71\u7d50\u679c\u4e2d\u3002"
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    