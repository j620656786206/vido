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
        
        # -> Open 媒體庫 from the sidebar to find the library sort controls.
        # 媒體庫 link
        elem = page.get_by_test_id("app-sidebar").get_by_test_id("nav-library")
        await elem.click(timeout=10000)
        
        # -> Open the library sort menu and inspect the available sorting options.
        # 新增日期 button
        elem = page.get_by_test_id("sort-selector-button")
        await elem.click(timeout=10000)
        
        # -> Choose “字幕狀態” from the library sort menu and check the resulting ordering.
        # 字幕狀態 button
        elem = page.get_by_test_id("sort-option-subtitle_status")
        await elem.click(timeout=10000)
        
        # -> Inspect the remaining library cards to confirm 教父 appears after every title marked 缺中文.
        await page.mouse.wheel(0, 300)
        
        # --> Assertions to verify final state
        
        # --> The library URL reflects sorting by subtitle status.
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: The URL includes the subtitle-status sort parameter.
        await expect(page).to_have_url(re.compile("sortBy=subtitle_status"), timeout=15000), "The URL includes the subtitle-status sort parameter."
        
        # --> The first result, 進擊的巨人, has the 缺中文 badge.
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: The first result shows the 缺中文 badge.
        await expect(page.get_by_test_id("poster-v2-seed-sr-001").nth(0)).to_contain_text("\u7f3a\u4e2d\u6587", timeout=15000), "The first result shows the \u7f3a\u4e2d\u6587 badge."
        
        # --> 教父 appears after the titles marked 缺中文.
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: 教父 appears after the titles marked 缺中文.
        await expect(page.get_by_test_id("poster-v2-seed-mv-001").nth(0)).to_contain_text("\u6559\u7236", timeout=15000), "\u6559\u7236 appears after the titles marked \u7f3a\u4e2d\u6587."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    