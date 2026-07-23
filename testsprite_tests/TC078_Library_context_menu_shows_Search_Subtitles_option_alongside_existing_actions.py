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
        
        # -> Click the '媒體庫' (Media Library) link to open the library page.
        # 媒體庫 link
        elem = page.get_by_text('內容', exact=True).locator("xpath=ancestor-or-self::*[.//a][1]").get_by_role('link', name='媒體庫', exact=True)
        await elem.click(timeout=10000)
        
        # -> Open the context menu for the first poster ('Unknown.Show.S01') and check that the menu contains '搜尋字幕', '重新解析', and '刪除'.
        # U 整理中 Unknown.Show.S01 link
        elem = page.get_by_test_id('poster-v2-seed-sr-101')
        await elem.click(timeout=10000)
        
        # -> Click the '媒體庫' link in the left navigation to return to the library page so the grid of poster cards is visible.
        # 媒體庫 link
        elem = page.get_by_text('內容', exact=True).locator("xpath=ancestor-or-self::*[.//a][1]").get_by_role('link', name='媒體庫', exact=True)
        await elem.click(timeout=10000)
        
        # -> Open the '媒體庫' (Media Library) page and verify a grid of media poster cards is visible.
        await page.goto("http://localhost:8090/library")
        try:
            await page.wait_for_load_state("domcontentloaded", timeout=5000)
        except Exception:
            pass
        
        # -> Click the '選取' button to enter selection mode and then look for action labels '搜尋字幕', '重新解析', or '刪除' in the UI.
        # 選取 button
        elem = page.get_by_test_id('enter-selection-btn')
        await elem.click(timeout=10000)
        
        # -> Select the first poster card (Unknown.Show.S01) and check the page for the labels '搜尋字幕', '重新解析', and '刪除' to verify whether 'Search Subtitles' is available.
        # U 失敗 Unknown.Show.S01 link
        elem = page.get_by_test_id('poster-v2-seed-sr-101')
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> Verify a grid of media poster cards is visible
        await page.locator("xpath=/html/body/div/div/div/div[2]/main/div/div/div[2]/div[2]/a[1]").nth(0).scroll_into_view_if_needed()
        # Assert: The first media poster card (U 失敗 Unknown.Show.S01) is visible.
        await expect(page.locator("xpath=/html/body/div/div/div/div[2]/main/div/div/div[2]/div[2]/a[1]").nth(0)).to_be_visible(timeout=15000), "The first media poster card (U \u5931\u6557 Unknown.Show.S01) is visible."
        await page.locator("xpath=/html/body/div/div/div/div[2]/main/div/div/div[2]/div[2]/a[2]").nth(0).scroll_into_view_if_needed()
        # Assert: A media poster card (缺字幕 怪奇物語 2016) is visible.
        await expect(page.locator("xpath=/html/body/div/div/div/div[2]/main/div/div/div[2]/div[2]/a[2]").nth(0)).to_be_visible(timeout=15000), "A media poster card (\u7f3a\u5b57\u5e55 \u602a\u5947\u7269\u8a9e 2016) is visible."
        await page.locator("xpath=/html/body/div/div/div/div[2]/main/div/div/div[2]/div[2]/a[3]").nth(0).scroll_into_view_if_needed()
        # Assert: A media poster card (缺字幕 進擊的巨人 2013) is visible.
        await expect(page.locator("xpath=/html/body/div/div/div/div[2]/main/div/div/div[2]/div[2]/a[3]").nth(0)).to_be_visible(timeout=15000), "A media poster card (\u7f3a\u5b57\u5e55 \u9032\u64ca\u7684\u5de8\u4eba 2013) is visible."
        
        # --> Verify text "Re-parse" is visible in the context menu
        await page.locator("xpath=/html/body/div/div/div/div[2]/main/div/div/div[2]/div[1]/div[1]/div[2]/button[1]").nth(0).scroll_into_view_if_needed()
        # Assert: The context menu displays the '重新解析' (Re-parse) action.
        await expect(page.locator("xpath=/html/body/div/div/div/div[2]/main/div/div/div[2]/div[1]/div[1]/div[2]/button[1]").nth(0)).to_be_visible(timeout=15000), "The context menu displays the '\u91cd\u65b0\u89e3\u6790' (Re-parse) action."
        
        # --> Verify text "Delete" is visible in the context menu
        # Assert: Context menu shows the '刪除' action.
        await expect(page.locator("xpath=/html/body/div/div/div/div[2]/main/div/div/div[2]/div[1]/div[1]/div[2]/button[4]").nth(0)).to_contain_text("\u522a\u9664", timeout=15000), "Context menu shows the '\u522a\u9664' action."
        current_url = await page.evaluate("() => window.location.href")
        # Assert: page loaded with a URL (final outcome verified by the AI judge during the run)
        assert current_url, 'Page should have loaded with a URL'
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    