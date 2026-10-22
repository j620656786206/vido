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
        
        # -> Open Settings from the sidebar to locate the logs page.
        # 設定 link
        elem = page.get_by_test_id("nav-settings")
        await elem.click(timeout=10000)
        
        # -> Open the “系統日誌” settings page to inspect the log list and its controls.
        # 維護：系統日誌 link
        elem = page.get_by_test_id("settings-tab-logs")
        await elem.click(timeout=10000)
        
        # -> Click the “ERROR” filter and verify the resulting count line and visible log levels.
        # ERROR button
        elem = page.get_by_test_id("log-filter-error")
        await elem.click(timeout=10000)
        
        # -> Expand the first ERROR log entry and check that additional details appear.
        # 展開 button
        elem = page.locator(".-m-1\\.5").first
        await elem.click(timeout=10000)
        
        # -> Select “全部” to display all log levels before testing pagination.
        # 全部 button
        elem = page.get_by_test_id("log-filter-all")
        await elem.click(timeout=10000)
        
        # -> Scroll to the bottom of the system log list to find the “下一頁” and “上一頁” pagination controls.
        await page.mouse.wheel(0, 300)
        
        # -> Click “下一頁” and verify the list advances and “上一頁” becomes available.
        # 下一頁 button
        elem = page.get_by_test_id("logs-next-btn")
        await elem.click(timeout=10000)
        
        # -> Click “上一頁” and verify the list returns to page 1 with the previous-page control disabled.
        # 上一頁 button
        elem = page.get_by_test_id("logs-prev-btn")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> 上一頁 is disabled after returning to page 1.
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: 上一頁 should be disabled on page 1.
        await expect(page.get_by_test_id("logs-prev-btn").nth(0)).to_have_attribute("disabled", "", timeout=15000), "\u4e0a\u4e00\u9801 should be disabled on page 1."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    