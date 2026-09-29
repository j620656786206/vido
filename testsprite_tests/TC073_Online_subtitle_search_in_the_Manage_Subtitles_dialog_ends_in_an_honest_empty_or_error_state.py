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
        
        # -> Click the '媒體庫' link in the left navigation to open the Library page.
        # 媒體庫 link
        elem = page.get_by_test_id("app-sidebar").get_by_test_id("nav-library")
        await elem.click(timeout=10000)
        
        # -> Open the context menu for the first poster (Unknown.Show.S01) and choose '管理字幕' (Manage Subtitles).
        # U 整理中 Unknown.Show.S01 link
        elem = page.get_by_test_id("poster-v2-seed-sr-101")
        await elem.click(timeout=10000)
        
        # -> Click the '管理字幕' button to open the Manage Subtitles dialog.
        # 管理字幕 button
        elem = page.get_by_test_id("action-manage-subtitle")
        await elem.click(timeout=10000)
        
        # -> Click the '搜尋線上字幕（成功率低）' button to reveal the 線上字幕搜尋 (online subtitle search) section.
        # 搜尋線上字幕（成功率低） button
        elem = page.get_by_test_id("toggle-fetch")
        await elem.click(timeout=10000)
        
        # -> Click the '搜尋' button in the 線上字幕搜尋 section and wait up to 30 seconds for the result message to appear.
        # 搜尋 button
        elem = page.get_by_test_id("fetch-search")
        await elem.click(timeout=10000)
        
        # -> Click the '搜尋' button in the 線上字幕搜尋 section to start the online subtitle search and then verify the resulting message.
        # 搜尋 button
        elem = page.get_by_test_id("fetch-search")
        await elem.click(timeout=10000)
        
        # -> Click the '搜尋' button in the 線上字幕搜尋 section and verify the section shows either '尚無結果' (no results) or '搜尋失敗' (search failed) and that the button is not showing '搜尋中'.
        # 搜尋 button
        elem = page.get_by_test_id("fetch-search")
        await elem.click(timeout=10000)
        
        # -> Click the '搜尋' button in the 線上字幕搜尋 section and verify the section shows either '尚無結果' or '搜尋失敗' and that the button is not showing '搜尋中'.
        # 搜尋 button
        elem = page.get_by_test_id("fetch-search")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> 管理字幕 對話框已開啟並顯示標題 '管理字幕 — Unknown.Show.S01'.
        await page.get_by_test_id("manage-subtitle-dialog-v2").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: passed
        # Assert: Manage Subtitles dialog is visible.
        await expect(page.get_by_test_id("manage-subtitle-dialog-v2").nth(0)).to_be_visible(timeout=15000), "Manage Subtitles dialog is visible."
        
        # --> 線上字幕搜尋區塊已展開且可見 搜尋 按鈕。
        await page.get_by_test_id("fetch-search").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: passed
        # Assert: The 線上字幕搜尋 section's 搜尋 button is visible.
        await expect(page.get_by_test_id("fetch-search").nth(0)).to_be_visible(timeout=15000), "The \u7dda\u4e0a\u5b57\u5e55\u641c\u5c0b section's \u641c\u5c0b button is visible."
        
        # --> 點選搜尋後，區塊顯示以 '尚無結果' 開頭的空結果提示，且搜尋按鈕未顯示為 '搜尋中'。
        # Assert-outcome: passed
        # Assert: The search section shows a hint starting with '尚無結果'.
        await expect(page.get_by_test_id("manage-subtitle-dialog-v2").nth(0)).to_contain_text("\u5c1a\u7121\u7d50\u679c", timeout=15000), "The search section shows a hint starting with '\u5c1a\u7121\u7d50\u679c'."
        # Assert-outcome: passed
        # Assert: The search button text is '搜尋' (not '搜尋中').
        await expect(page.get_by_test_id("fetch-search").nth(0)).to_have_text("\u641c\u5c0b", timeout=15000), "The search button text is '\u641c\u5c0b' (not '\u641c\u5c0b\u4e2d')."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    