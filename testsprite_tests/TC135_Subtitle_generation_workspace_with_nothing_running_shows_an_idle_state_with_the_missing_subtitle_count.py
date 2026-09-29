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
        
        # -> Navigate to the 活動 → 生成字幕 page (open /activity?view=generation) and wait for it to load.
        await page.goto("http://localhost:8090/activity?view=generation")
        try:
            await page.wait_for_load_state("domcontentloaded", timeout=5000)
        except Exception:
            pass
        
        # --> Assertions to verify final state
        
        # --> The page shows the heading "生成工作區".
        # Assert-outcome: passed
        # Assert: Heading 生成工作區 is visible.
        await expect(page.locator("#root").nth(0)).to_contain_text("\u751f\u6210\u5de5\u4f5c\u5340", timeout=15000), "Heading \u751f\u6210\u5de5\u4f5c\u5340 is visible."
        
        # --> The breadcrumb shows "活動 ／ 生成字幕".
        # Assert-outcome: passed
        # Assert: Breadcrumb contains 活動 ／ 生成字幕.
        await expect(page.locator("#root").nth(0)).to_contain_text("\u6d3b\u52d5 \uff0f \u751f\u6210\u5b57\u5e55", timeout=15000), "Breadcrumb contains \u6d3b\u52d5 \uff0f \u751f\u6210\u5b57\u5e55."
        
        # --> The idle message "目前沒有進行中的生成" is visible.
        # Assert-outcome: passed
        # Assert: Idle message 目前沒有進行中的生成 is visible.
        await expect(page.locator("#root").nth(0)).to_contain_text("\u76ee\u524d\u6c92\u6709\u9032\u884c\u4e2d\u7684\u751f\u6210", timeout=15000), "Idle message \u76ee\u524d\u6c92\u6709\u9032\u884c\u4e2d\u7684\u751f\u6210 is visible."
        
        # --> A line indicating missing Traditional Chinese subtitles (starts with "缺繁中字幕：") is visible and the "批次生成字幕" button is present.
        # Assert-outcome: passed
        # Assert: The page shows a line starting with 缺繁中字幕：.
        await expect(page.locator("#root").nth(0)).to_contain_text("\u7f3a\u7e41\u4e2d\u5b57\u5e55\uff1a", timeout=15000), "The page shows a line starting with \u7f3a\u7e41\u4e2d\u5b57\u5e55\uff1a."
        await page.get_by_test_id("workspace-launch").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: passed
        # Assert: The 批次生成字幕 button is visible.
        await expect(page.get_by_test_id("workspace-launch").nth(0)).to_be_visible(timeout=15000), "The \u6279\u6b21\u751f\u6210\u5b57\u5e55 button is visible."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    