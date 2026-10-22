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
        
        # -> Open the “媒體庫” section from the sidebar.
        # 媒體庫 link
        elem = page.get_by_test_id("app-sidebar").get_by_test_id("nav-library")
        await elem.click(timeout=10000)
        
        # -> Search the media library for 教父 and inspect the matching result.
        # 搜尋 text field
        elem = page.get_by_test_id("instant-search-input")
        await elem.wait_for(state="visible", timeout=10000)
        await elem.fill("\u6559\u7236")
        
        # -> Select the 教父 (The Godfather, 1972) library suggestion to open its detail page.
        # 教父 The Godfather (1972) 已擁有 button
        elem = page.get_by_test_id("search-suggestion-item")
        await elem.click(timeout=10000)
        
        # -> Click “返回媒體庫” and verify that the media-library poster grid appears at `/library`.
        # 返回媒體庫 button
        elem = page.get_by_test_id("detail-back")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> Returned to /library with the media-library results grid visible.
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: The page is at /library.
        await expect(page).to_have_url(re.compile("/library"), timeout=15000), "The page is at /library."
        await page.get_by_test_id("library-results").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: The media-library results grid is visible.
        await expect(page.get_by_test_id("library-results").nth(0)).to_be_visible(timeout=15000), "The media-library results grid is visible."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    