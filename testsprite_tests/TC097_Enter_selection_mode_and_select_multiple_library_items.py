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
        
        # -> Click the '媒體庫' link in the left sidebar to open the Library page.
        # 媒體庫 link
        elem = page.get_by_test_id("app-sidebar").get_by_test_id("nav-library")
        await elem.click(timeout=10000)
        
        # -> Click the '選取' button to enter selection mode.
        # 選取 button
        elem = page.get_by_test_id("enter-selection-btn")
        await elem.click(timeout=10000)
        
        # -> Click the poster cards labeled '沙丘:第二部' and '奧本海默' to select two items and observe the selection toolbar count.
        # 缺字幕 8.2 沙丘:第二部 2024 link
        elem = page.get_by_test_id("poster-v2-seed-mv-012")
        await elem.click(timeout=10000)
        
        # -> Click the poster cards labeled '沙丘:第二部' and '奧本海默' to select two items and observe the selection toolbar count.
        # 缺字幕 8.1 奧本海默 2023 link
        elem = page.get_by_test_id("poster-v2-seed-mv-011")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> The selection toolbar is visible showing action buttons such as '重新解析'.
        await page.get_by_test_id("batch-reparse-btn").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: passed
        # Assert: The selection toolbar's '重新解析' button is visible.
        await expect(page.get_by_test_id("batch-reparse-btn").nth(0)).to_be_visible(timeout=15000), "The selection toolbar's '\u91cd\u65b0\u89e3\u6790' button is visible."
        
        # --> The selection toolbar shows '已選取 2 項', indicating two items are selected.
        # Assert-outcome: passed
        # Assert: The page displays the selection count text '已選取 2 項'.
        await expect(page.locator("#root").nth(0)).to_contain_text("\u5df2\u9078\u53d6 2 \u9805", timeout=15000), "The page displays the selection count text '\u5df2\u9078\u53d6 2 \u9805'."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    