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
        
        # -> Open the media library using the visible “媒體庫” navigation link.
        # 媒體庫 link
        elem = page.get_by_test_id("app-sidebar").get_by_test_id("nav-library")
        await elem.click(timeout=10000)
        
        # -> Select the “有中文字幕” subtitle filter and verify the matching titles in the results.
        # 有中文字幕 button
        elem = page.get_by_test_id("filter-chinese-has")
        await elem.click(timeout=10000)
        
        # -> Clear the selected “有中文字幕” filter and verify that the filter is no longer active.
        # 有中文字幕 button
        elem = page.get_by_test_id("filter-chinese-has")
        await elem.click(timeout=10000)
        
        # -> Select the “不知道” subtitle filter and verify that 媽的多重宇宙 appears while 教父 does not.
        # 不知道 button
        elem = page.get_by_test_id("filter-chinese-unknown")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> With the “不知道” filter selected, 媽的多重宇宙 is visible.
        await page.get_by_test_id("poster-v2-seed-mv-010").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: The “不知道” results include 媽的多重宇宙.
        await expect(page.get_by_test_id("poster-v2-seed-mv-010").nth(0)).to_be_visible(timeout=15000), "The \u201c\u4e0d\u77e5\u9053\u201d results include \u5abd\u7684\u591a\u91cd\u5b87\u5b99."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    