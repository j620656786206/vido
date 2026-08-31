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
        
        # -> Navigate to the '媒體庫' (Library) page by opening http://localhost:8090/library.
        await page.goto("http://localhost:8090/library")
        try:
            await page.wait_for_load_state("domcontentloaded", timeout=5000)
        except Exception:
            pass
        
        # -> Open the poster card context menu for 'Unknown.Show.S01' by clicking the first media poster card.
        # U 失敗 Unknown.Show.S01 link
        elem = page.get_by_test_id('poster-v2-seed-sr-101')
        await elem.click(timeout=10000)
        
        # -> Click the '管理字幕' (Manage Subtitles) button to open the subtitle search dialog.
        # 管理字幕 button
        elem = page.get_by_test_id('action-manage-subtitle')
        await elem.click(timeout=10000)
        
        # -> Click the '搜尋線上字幕（成功率低）' button to open the subtitle search dialog.
        # 搜尋線上字幕（成功率低） button
        elem = page.get_by_test_id('toggle-fetch')
        await elem.click(timeout=10000)
        
        # -> Type 'zzzzz-no-subtitles-exist-99999' into the subtitle search field and click the '搜尋' button
        # 搜尋 button
        elem = page.get_by_test_id('fetch-search')
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> The Manage Subtitles dialog titled '管理字幕 — Unknown.Show.S01' is visible.
        # Assert-outcome: passed
        # Assert: Dialog contains the title '管理字幕 — Unknown.Show.S01'.
        await expect(page.locator("xpath=/html/body/div[3]").nth(0)).to_contain_text("\u7ba1\u7406\u5b57\u5e55 \u2014 Unknown.Show.S01", timeout=15000), "Dialog contains the title '\u7ba1\u7406\u5b57\u5e55 \u2014 Unknown.Show.S01'."
        
        # --> The subtitle search shows the empty-state message indicating no online results.
        # Assert-outcome: passed
        # Assert: Dialog contains the empty-state message for no search results.
        await expect(page.locator("xpath=/html/body/div[3]").nth(0)).to_contain_text("\u5c1a\u7121\u7d50\u679c \u2014 \u7dda\u4e0a\u4f86\u6e90\u6210\u529f\u7387\u4f4e\uff0c\u5efa\u8b70\u6539\u7528\u751f\u6210\u5b57\u5e55", timeout=15000), "Dialog contains the empty-state message for no search results."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    