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
        
        # -> Click the '設定' link in the sidebar to open the Settings page.
        # 設定 link
        elem = page.get_by_test_id("nav-settings")
        await elem.click(timeout=10000)
        
        # -> Click the '維護：系統日誌' link in the Settings sidebar to open the System Logs page.
        # 維護：系統日誌 link
        elem = page.get_by_test_id("settings-tab-logs")
        await elem.click(timeout=10000)
        
        # -> Type 'zzqxnolog987' into the 搜尋關鍵字 input and press Enter to apply the filter, then wait for the list to refresh.
        # 搜尋關鍵字 text field
        elem = page.get_by_test_id("log-keyword-input")
        await elem.wait_for(state="visible", timeout=10000)
        await elem.fill("zzqxnolog987")
        
        # -> Click the '清除篩選' button to remove the active filter and clear the search input.
        # 清除篩選 button
        elem = page.get_by_test_id("logs-clear-filters")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> When filtering by the keyword, the page shows the empty-state message and the active-filter advice with the typed keyword.
        # Assert-outcome: passed
        # Assert: The empty-state text '沒有符合條件的日誌記錄' is shown.
        await expect(page.get_by_test_id("log-keyword-search").nth(0)).to_contain_text("\u6c92\u6709\u7b26\u5408\u689d\u4ef6\u7684\u65e5\u8a8c\u8a18\u9304", timeout=15000), "The empty-state text '\u6c92\u6709\u7b26\u5408\u689d\u4ef6\u7684\u65e5\u8a8c\u8a18\u9304' is shown."
        # Assert-outcome: passed
        # Assert: The active-filter advice with 'zzqxnolog987' is shown.
        await expect(page.get_by_test_id("log-keyword-search").nth(0)).to_contain_text("\u95dc\u9375\u5b57\u300czzqxnolog987\u300d\u3002\u6e05\u9664\u95dc\u9375\u5b57\u518d\u8a66\u4e00\u6b21\u3002", timeout=15000), "The active-filter advice with 'zzqxnolog987' is shown."
        
        # --> Clicking 清除篩選 clears the 搜尋關鍵字 input and returns the logs view (empty-state message disappears).
        # Assert-outcome: passed
        # Assert: The 搜尋關鍵字 input is empty after clearing filters.
        await expect(page.get_by_test_id("log-keyword-input").nth(0)).to_have_value("", timeout=15000), "The \u641c\u5c0b\u95dc\u9375\u5b57 input is empty after clearing filters."
        await page.locator(".-m-1\\.5").first.nth(0).scroll_into_view_if_needed()
        # Assert-outcome: passed
        # Assert: A log entry expand button is visible, indicating the logs list is restored.
        await expect(page.locator(".-m-1\\.5").first.nth(0)).to_be_visible(timeout=15000), "A log entry expand button is visible, indicating the logs list is restored."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    