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
        
        # -> Open the movie page for the target item and inspect its available controls.
        await page.goto("http://localhost:8090/media/movie/seed-mv-011")
        try:
            await page.wait_for_load_state("domcontentloaded", timeout=5000)
        except Exception:
            pass
        
        # -> Open the '修改資訊' dialog and inspect its text and '重新選片' control.
        # 修改資訊 button
        elem = page.get_by_test_id("action-edit-metadata")
        await elem.click(timeout=10000)
        
        # -> Click ‘重新選片’ and verify that ‘手動選片’ opens with the search box prefilled with ‘Oppenheimer’.
        # 重新選片 button
        elem = page.get_by_role("button", name="重新選片")
        await elem.click(timeout=10000)
        
        # -> Reopen the 修改資訊 dialog and verify that it is visible again.
        # 修改資訊 button
        elem = page.get_by_test_id("action-edit-metadata")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> 修改資訊對話框顯示「認錯片了？」列及「重新選片」按鈕。
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: 修改資訊對話框應顯示「認錯片了？」。
        await expect(page.locator("xpath=/html/body/div[4]").nth(0)).to_contain_text("\u8a8d\u932f\u7247\u4e86\uff1f", timeout=15000), "\u4fee\u6539\u8cc7\u8a0a\u5c0d\u8a71\u6846\u61c9\u986f\u793a\u300c\u8a8d\u932f\u7247\u4e86\uff1f\u300d\u3002"
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: 對話框應顯示「重新選片」按鈕。
        await expect(page.locator("xpath=/html/body/div[4]/div[3]/div/p[1]/button").nth(0)).to_contain_text("\u91cd\u65b0\u9078\u7247", timeout=15000), "\u5c0d\u8a71\u6846\u61c9\u986f\u793a\u300c\u91cd\u65b0\u9078\u7247\u300d\u6309\u9215\u3002"
        
        # --> 修改資訊對話框說明海報、簡介與演員會換成所選影片的內容。
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: 對話框應說明海報、簡介與演員會換成所選影片的內容。
        await expect(page.locator("xpath=/html/body/div[4]").nth(0)).to_contain_text("\u6d77\u5831\u3001\u7c21\u4ecb\u3001\u6f14\u54e1\u6703\u63db\u6210\u4f60\u9078\u7684\u90a3\u4e00\u90e8", timeout=15000), "\u5c0d\u8a71\u6846\u61c9\u8aaa\u660e\u6d77\u5831\u3001\u7c21\u4ecb\u8207\u6f14\u54e1\u6703\u63db\u6210\u6240\u9078\u5f71\u7247\u7684\u5167\u5bb9\u3002"
        
        # --> 修改資訊對話框已重新開啟。
        await page.locator("xpath=/html/body/div[4]").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: 修改資訊對話框應再次開啟。
        await expect(page.locator("xpath=/html/body/div[4]").nth(0)).to_be_visible(timeout=15000), "\u4fee\u6539\u8cc7\u8a0a\u5c0d\u8a71\u6846\u61c9\u518d\u6b21\u958b\u555f\u3002"
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    