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
        
        # -> Click the '選取' button to enter selection mode.
        # 選取 button
        elem = page.get_by_test_id("enter-selection-btn")
        await elem.click(timeout=10000)
        
        # -> Select the first two library items ("Unknown.Show.S01" and "怪奇物語") and click the '重新解析' (Reparse) button.
        # U 整理中 Unknown.Show.S01 link
        elem = page.get_by_test_id("poster-v2-seed-sr-101")
        await elem.click(timeout=10000)
        
        # -> Select the first two library items ("Unknown.Show.S01" and "怪奇物語") and click the '重新解析' (Reparse) button.
        # 缺字幕 怪奇物語 2016 link
        elem = page.get_by_test_id("poster-v2-seed-sr-002")
        await elem.click(timeout=10000)
        
        # -> Select the first two library items ("Unknown.Show.S01" and "怪奇物語") and click the '重新解析' (Reparse) button.
        # 重新解析 button
        elem = page.get_by_test_id("batch-reparse-btn")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> A confirmation dialog titled '確認重新解析' asks to reparse the two selected items.
        # Assert-outcome: passed
        # Assert: The confirmation dialog asks '確定要重新解析 2 個項目嗎？'.
        await expect(page.get_by_test_id("batch-confirm-backdrop").nth(0)).to_contain_text("\u78ba\u5b9a\u8981\u91cd\u65b0\u89e3\u6790 2 \u500b\u9805\u76ee\u55ce\uff1f", timeout=15000), "The confirmation dialog asks '\u78ba\u5b9a\u8981\u91cd\u65b0\u89e3\u6790 2 \u500b\u9805\u76ee\u55ce\uff1f'."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    