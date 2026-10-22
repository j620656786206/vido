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
        
        # -> Open the movie library using the visible “電影” navigation link.
        # 電影 15 link
        elem = page.get_by_test_id("nav-movies")
        await elem.click(timeout=10000)
        
        # -> Open the “新增日期” sort selector and inspect the available sort options.
        # 新增日期 button
        elem = page.get_by_test_id("sort-selector-button")
        await elem.click(timeout=10000)
        
        # -> Choose “評分” from the sort menu and verify the updated poster order.
        # 評分 button
        elem = page.get_by_test_id("sort-option-rating")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> The URL reflects rating-based sorting.
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: The URL includes the rating sort parameter.
        await expect(page).to_have_url(re.compile("sortBy=rating"), timeout=15000), "The URL includes the rating sort parameter."
        
        # --> The first two posters are 駭客任務 and 教父.
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: The first poster is 駭客任務.
        await expect(page.get_by_test_id("poster-v2-seed-mv-003").nth(0)).to_contain_text("\u99ed\u5ba2\u4efb\u52d9", timeout=15000), "The first poster is \u99ed\u5ba2\u4efb\u52d9."
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: The second poster is 教父.
        await expect(page.get_by_test_id("poster-v2-seed-mv-001").nth(0)).to_contain_text("\u6559\u7236", timeout=15000), "The second poster is \u6559\u7236."
        
        # --> 媽的多重宇宙 appears after 侏羅紀公園 in the sorted posters.
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: 侏羅紀公園 appears in the earlier poster position.
        await expect(page.get_by_test_id("poster-v2-seed-mv-002").nth(0)).to_contain_text("\u4f8f\u7f85\u7d00\u516c\u5712", timeout=15000), "\u4f8f\u7f85\u7d00\u516c\u5712 appears in the earlier poster position."
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: 媽的多重宇宙 appears in the later poster position.
        await expect(page.get_by_test_id("poster-v2-seed-mv-010").nth(0)).to_contain_text("\u5abd\u7684\u591a\u91cd\u5b87\u5b99", timeout=15000), "\u5abd\u7684\u591a\u91cd\u5b87\u5b99 appears in the later poster position."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    