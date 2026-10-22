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
        
        # -> Open the “媒體庫” section from the sidebar and inspect its title and subtitle-count control.
        # 媒體庫 link
        elem = page.get_by_test_id("app-sidebar").get_by_test_id("nav-library")
        await elem.click(timeout=10000)
        
        # -> Click the control for the seven titles missing Chinese subtitles and verify the filtered results.
        # · 7 部缺中文 button
        elem = page.get_by_test_id("library-missing-zh-link")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> The missing-Chinese-subtitle filter is active on the library.
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: The missing-Chinese-subtitle filter should be active.
        await expect(page).to_have_url(re.compile("chineseSubtitle=missing"), timeout=15000), "The missing-Chinese-subtitle filter should be active."
        
        # --> The filtered results include 侏羅紀公園 and 進擊的巨人.
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: The filtered results should include 侏羅紀公園.
        await expect(page.get_by_test_id("library-results").nth(0)).to_contain_text("\u4f8f\u7f85\u7d00\u516c\u5712", timeout=15000), "The filtered results should include \u4f8f\u7f85\u7d00\u516c\u5712."
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: The filtered results should include 進擊的巨人.
        await expect(page.get_by_test_id("library-results").nth(0)).to_contain_text("\u9032\u64ca\u7684\u5de8\u4eba", timeout=15000), "The filtered results should include \u9032\u64ca\u7684\u5de8\u4eba."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    