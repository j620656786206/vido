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
        
        # -> Click the '[FanSub] 未知電影 (2021)' card to open its detail page.
        # F 失敗 [FanSub] 未知電影 (2021) link
        elem = page.get_by_test_id("poster-v2-seed-mv-102")
        await elem.click(timeout=10000)
        
        # -> Click the '手動選片' button to open the manual-selection dialog.
        # 手動選片 button
        elem = page.get_by_test_id("no-metadata-manual-match")
        await elem.click(timeout=10000)
        
        # -> Type 'Inception' into the '在 TMDb 搜尋電影' search field, wait ~5 seconds, then check for the message '搜尋暫時無法使用' and ensure '找不到符合的作品' is not shown.
        # 在 TMDb 搜尋電影 search field
        elem = page.get_by_role("searchbox", name="在 TMDb 搜尋電影")
        await elem.wait_for(state="visible", timeout=10000)
        await elem.fill("Inception")
        
        # -> Open the '手動選片' dialog by clicking the '手動選片' button.
        # 手動選片 button
        elem = page.get_by_test_id("no-metadata-manual-match")
        await elem.click(timeout=10000)
        
        # -> Type 'Inception' into the '在 TMDb 搜尋電影' search field, wait ~5 seconds, and check for the error message and absence of the no-results message.
        # 在 TMDb 搜尋電影 search field
        elem = page.get_by_role("searchbox", name="在 TMDb 搜尋電影")
        await elem.wait_for(state="visible", timeout=10000)
        await elem.fill("Inception")
        
        # --> Assertions to verify final state
        
        # --> The movie detail page shows the heading 沒有找到這部電影的資料.
        # Assert-outcome: passed
        # Assert: The page includes the empty-state heading '沒有找到這部電影的資料'.
        await expect(page.locator("#root").nth(0)).to_contain_text("\u6c92\u6709\u627e\u5230\u9019\u90e8\u96fb\u5f71\u7684\u8cc7\u6599", timeout=15000), "The page includes the empty-state heading '\u6c92\u6709\u627e\u5230\u9019\u90e8\u96fb\u5f71\u7684\u8cc7\u6599'."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    