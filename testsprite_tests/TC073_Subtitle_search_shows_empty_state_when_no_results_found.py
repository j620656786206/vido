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
        
        # -> Click the '媒體庫' sidebar link to open the Media Library page
        # 媒體庫 link
        elem = page.get_by_text('內容', exact=True).locator("xpath=ancestor-or-self::*[.//a][1]").get_by_role('link', name='媒體庫', exact=True)
        await elem.click(timeout=10000)
        
        # -> Open the first media item by clicking the poster labeled 'Unknown.Show.S01' to view its detail page.
        # U 失敗 Unknown.Show.S01 link
        elem = page.get_by_test_id('poster-v2-seed-sr-101')
        await elem.click(timeout=10000)
        
        # -> Click the '管理字幕' (Manage Subtitles) button to open the Manage Subtitles dialog and trigger the auto-search.
        # 管理字幕 button
        elem = page.get_by_test_id('action-manage-subtitle')
        await elem.click(timeout=10000)
        
        # -> Click the '搜尋線上字幕（成功率低）' button to trigger the online subtitles search.
        # 搜尋線上字幕（成功率低） button
        elem = page.get_by_test_id('toggle-fetch')
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> Verify element with data-testid "subtitle-search-dialog" is visible
        await page.locator("xpath=/html/body/div[3]").nth(0).scroll_into_view_if_needed()
        # Assert: The subtitle search dialog is visible.
        await expect(page.locator("xpath=/html/body/div[3]").nth(0)).to_be_visible(timeout=15000), "The subtitle search dialog is visible."
        
        # --> Verify element with data-testid "subtitle-empty-state" is visible
        # Assert: Verify the subtitle empty-state message is visible in the Manage Subtitles dialog.
        await expect(page.locator("xpath=/html/body/div[3]").nth(0)).to_contain_text("\u5c1a\u7121\u7d50\u679c \u2014 \u7dda\u4e0a\u4f86\u6e90\u6210\u529f\u7387\u4f4e\uff0c\u5efa\u8b70\u6539\u7528\u751f\u6210\u5b57\u5e55", timeout=15000), "Verify the subtitle empty-state message is visible in the Manage Subtitles dialog."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    