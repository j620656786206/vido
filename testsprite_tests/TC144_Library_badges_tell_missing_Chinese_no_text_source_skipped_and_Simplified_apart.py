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
        
        # -> Open the visible 媒體庫 section to view the movie cards and their subtitle badges.
        # 媒體庫 link
        elem = page.get_by_test_id("app-sidebar").get_by_test_id("nav-library")
        await elem.click(timeout=10000)
        
        # -> Verify the subtitle badges on the 侏羅紀公園 and 教父 movie cards in the media library.
        # [internal] extract_content: 
        
        # --> Assertions to verify final state
        
        # --> The 沙丘:第二部 card shows 已略過.
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: 沙丘:第二部 should show 已略過.
        await expect(page.get_by_test_id("poster-v2-seed-mv-012").nth(0)).to_contain_text("\u5df2\u7565\u904e", timeout=15000), "\u6c99\u4e18:\u7b2c\u4e8c\u90e8 should show \u5df2\u7565\u904e."
        
        # --> The 全面啟動 card shows 無字幕源.
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: 全面啟動 should show 無字幕源.
        await expect(page.get_by_test_id("poster-v2-seed-mv-006").nth(0)).to_contain_text("\u7121\u5b57\u5e55\u6e90", timeout=15000), "\u5168\u9762\u555f\u52d5 should show \u7121\u5b57\u5e55\u6e90."
        
        # --> The 駭客任務 card shows 簡中.
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: 駭客任務 should show 簡中.
        await expect(page.get_by_test_id("poster-v2-seed-mv-003").nth(0)).to_contain_text("\u7c21\u4e2d", timeout=15000), "\u99ed\u5ba2\u4efb\u52d9 should show \u7c21\u4e2d."
        
        # --> The 侏羅紀公園 card shows 缺中文.
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: 侏羅紀公園 should show 缺中文.
        await expect(page.get_by_test_id("poster-v2-seed-mv-002").nth(0)).to_contain_text("\u7f3a\u4e2d\u6587", timeout=15000), "\u4f8f\u7f85\u7d00\u516c\u5712 should show \u7f3a\u4e2d\u6587."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    