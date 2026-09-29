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
        
        # -> Click the '媒體庫' link in the left sidebar to open the library page.
        # 媒體庫 link
        elem = page.get_by_test_id("app-sidebar").get_by_test_id("nav-library")
        await elem.click(timeout=10000)
        
        # -> Click the '未匹配' status filter in the left filter panel to activate it and filter the results.
        # 未匹配 (1) button
        elem = page.get_by_test_id("filter-unmatched")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> The '未匹配' status filter is active and its label reads '未匹配 (1)', and the page header shows '媒體庫 1 部'.
        # Assert-outcome: passed
        # Assert: The 未匹配 filter label shows the count (1).
        await expect(page.get_by_test_id("filter-unmatched").nth(0)).to_contain_text("\u672a\u5339\u914d (1)", timeout=15000), "The \u672a\u5339\u914d filter label shows the count (1)."
        # Assert-outcome: passed
        # Assert: The page header shows 1 item.
        await expect(page.get_by_test_id("library-result-count").nth(0)).to_contain_text("1 \u90e8", timeout=15000), "The page header shows 1 item."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    