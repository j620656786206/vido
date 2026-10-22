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
        
        # -> Open the 媒體庫 page from the sidebar to access the media list and genre filters.
        # 媒體庫 link
        elem = page.get_by_test_id("app-sidebar").get_by_test_id("nav-library")
        await elem.click(timeout=10000)
        
        # -> Select the「動畫」genre filter and verify the two expected titles and a genres query in the URL.
        # 動畫 button
        elem = page.get_by_test_id("filter-genre-動畫")
        await elem.click(timeout=10000)
        
        # -> Click the selected 動畫 genre filter again and verify 教父 and 侏羅紀公園 are visible in the restored list.
        # 動畫 button
        elem = page.get_by_test_id("filter-genre-動畫")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> The restored library shows both 教父 and 侏羅紀公園.
        await page.get_by_test_id("poster-v2-seed-mv-002").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: Confirm 侏羅紀公園 is visible in the restored library.
        await expect(page.get_by_test_id("poster-v2-seed-mv-002").nth(0)).to_be_visible(timeout=15000), "Confirm \u4f8f\u7f85\u7d00\u516c\u5712 is visible in the restored library."
        await page.get_by_test_id("poster-v2-seed-mv-001").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: Confirm 教父 is visible in the restored library.
        await expect(page.get_by_test_id("poster-v2-seed-mv-001").nth(0)).to_be_visible(timeout=15000), "Confirm \u6559\u7236 is visible in the restored library."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    