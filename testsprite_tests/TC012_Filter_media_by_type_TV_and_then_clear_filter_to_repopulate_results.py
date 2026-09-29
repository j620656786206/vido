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
        
        # -> Click the '媒體庫' (Media Library) link in the left sidebar to open the library page.
        # 媒體庫 link
        elem = page.get_by_test_id("app-sidebar").get_by_test_id("nav-library")
        await elem.click(timeout=10000)
        
        # -> Click the '影集' (TV) type filter button to apply the TV type filter.
        # 影集 button
        elem = page.get_by_test_id("filter-type-tv")
        await elem.click(timeout=10000)
        
        # -> Click the '1990s' year filter to try to produce an empty result set while the '影集' (TV) type filter is active.
        # 1990s button
        elem = page.get_by_test_id("filter-decade-1990s")
        await elem.click(timeout=10000)
        
        # -> Click the '影集' (TV) type filter button to apply the TV filter while the 1990s year filter remains selected.
        # 影集 button
        elem = page.get_by_test_id("filter-type-tv")
        await elem.click(timeout=10000)
        
        # -> Click the '清除全部篩選' (Clear all filters) button to remove current filters.
        # 清除全部篩選 button
        elem = page.get_by_test_id("clear-all-filters")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> The type filter control is visible with buttons labeled '全部', '電影', and '影集'.
        await page.get_by_test_id("filter-type-tv").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: passed
        # Assert: The '影集' (TV) type filter button is visible.
        await expect(page.get_by_test_id("filter-type-tv").nth(0)).to_be_visible(timeout=15000), "The '\u5f71\u96c6' (TV) type filter button is visible."
        
        # --> Selecting '影集' + '1990s' produced the empty-state message '找不到符合的結果', and clicking '清除全部篩選' removed it so media cards are shown again.
        # Assert-outcome: passed
        # Assert: The URL shows type=all after clearing filters, indicating filters were cleared.
        await expect(page).to_have_url(re.compile("type=all"), timeout=15000), "The URL shows type=all after clearing filters, indicating filters were cleared."
        await page.get_by_test_id("poster-v2-seed-sr-001").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: passed
        # Assert: A media card ('進擊的巨人') is visible, showing results are present after clearing filters.
        await expect(page.get_by_test_id("poster-v2-seed-sr-001").nth(0)).to_be_visible(timeout=15000), "A media card ('\u9032\u64ca\u7684\u5de8\u4eba') is visible, showing results are present after clearing filters."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    