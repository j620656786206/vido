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
        
        # -> Open the 媒體庫 view using its visible navigation link.
        # 媒體庫 link
        elem = page.get_by_test_id("app-sidebar").get_by_test_id("nav-library")
        await elem.click(timeout=10000)
        
        # -> Enable selection mode using the visible 選取 control.
        # 選取 button
        elem = page.get_by_test_id("enter-selection-btn")
        await elem.click(timeout=10000)
        
        # -> Select the [FanSub] 未知電影 (2021) card and verify the selected count changes.
        # F 失敗 [FanSub] 未知電影 (2021) link
        elem = page.get_by_test_id("poster-v2-seed-mv-102")
        await elem.click(timeout=10000)
        
        # -> Select the Unknown.Show.S01 card and verify the toolbar reports two selected items.
        # U 失敗 Unknown.Show.S01 link
        elem = page.get_by_test_id("poster-v2-seed-sr-101")
        await elem.click(timeout=10000)
        
        # -> Start 重新解析 for the two selected media items and review the confirmation dialog.
        # 重新解析 button
        elem = page.get_by_test_id("batch-reparse-btn")
        await elem.click(timeout=10000)
        
        # -> Confirm re-parsing from the “確認重新解析” dialog and inspect the resulting status.
        # 重新解析 button
        elem = page.get_by_test_id("confirm-action-btn")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> The selection toolbar shows 已選取 2 項.
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: The selection toolbar shows two selected items.
        await expect(page.locator("#root").nth(0)).to_contain_text("\u5df2\u9078\u53d6 2 \u9805", timeout=15000), "The selection toolbar shows two selected items."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    