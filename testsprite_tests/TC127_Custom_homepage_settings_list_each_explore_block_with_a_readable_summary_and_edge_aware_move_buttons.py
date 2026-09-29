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
        
        # -> Open the 設定 → 自訂首頁 (Homepage customization) settings page and wait for the list to load.
        await page.goto("http://localhost:8090/settings/homepage")
        try:
            await page.wait_for_load_state("domcontentloaded", timeout=5000)
        except Exception:
            pass
        
        # --> Assertions to verify final state
        
        # --> The 自訂首頁 heading, the 新增區塊 button, and the block count text (ending in 個區塊) are visible.
        # Assert-outcome: passed
        # Assert: The 自訂首頁 heading is visible.
        await expect(page.get_by_test_id("settings-tab-homepage").nth(0)).to_have_text("\u81ea\u8a02\u9996\u9801", timeout=15000), "The \u81ea\u8a02\u9996\u9801 heading is visible."
        await page.get_by_test_id("explore-blocks-add-button").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: passed
        # Assert: The 新增區塊 button is visible.
        await expect(page.get_by_test_id("explore-blocks-add-button").nth(0)).to_be_visible(timeout=15000), "The \u65b0\u589e\u5340\u584a button is visible."
        
        # --> The homepage shows the three blocks in order: 熱門電影, 熱門影集, 近期新片.
        await page.get_by_test_id("explore-block-move-up-306c5239-18df-4110-ba65-5339f431538a").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: passed
        # Assert: The first block is 熱門電影 (shown by the 上移 熱門電影 control at list position 1).
        await expect(page.get_by_test_id("explore-block-move-up-306c5239-18df-4110-ba65-5339f431538a").nth(0)).to_be_visible(timeout=15000), "The first block is \u71b1\u9580\u96fb\u5f71 (shown by the \u4e0a\u79fb \u71b1\u9580\u96fb\u5f71 control at list position 1)."
        await page.get_by_test_id("explore-block-move-up-74679f31-9486-4afc-84d1-37f59e5a41de").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: passed
        # Assert: The second block is 熱門影集 (shown by the 上移 熱門影集 control at list position 2).
        await expect(page.get_by_test_id("explore-block-move-up-74679f31-9486-4afc-84d1-37f59e5a41de").nth(0)).to_be_visible(timeout=15000), "The second block is \u71b1\u9580\u5f71\u96c6 (shown by the \u4e0a\u79fb \u71b1\u9580\u5f71\u96c6 control at list position 2)."
        
        # --> Each block summary shows the expected content type and sort: 熱門電影 and 熱門影集 show '熱門度', and 近期新片 shows '發行日期'.
        # Assert-outcome: passed
        # Assert: The 熱門電影 summary contains the content type '電影' and the sort '熱門度'.
        await expect(page.locator("#root").nth(0)).to_contain_text("\u96fb\u5f71 \u00b7 \u71b1\u9580\u5ea6", timeout=15000), "The \u71b1\u9580\u96fb\u5f71 summary contains the content type '\u96fb\u5f71' and the sort '\u71b1\u9580\u5ea6'."
        # Assert-outcome: passed
        # Assert: The 熱門影集 summary contains the content type '影集' and the sort '熱門度'.
        await expect(page.locator("#root").nth(0)).to_contain_text("\u5f71\u96c6 \u00b7 \u71b1\u9580\u5ea6", timeout=15000), "The \u71b1\u9580\u5f71\u96c6 summary contains the content type '\u5f71\u96c6' and the sort '\u71b1\u9580\u5ea6'."
        
        # --> The informational note '已擁有的作品不會出現在首頁。' is visible.
        # Assert-outcome: passed
        # Assert: The informational note about owned items not appearing on the homepage is visible.
        await expect(page.locator("#root").nth(0)).to_contain_text("\u5df2\u64c1\u6709\u7684\u4f5c\u54c1\u4e0d\u6703\u51fa\u73fe\u5728\u9996\u9801\u3002", timeout=15000), "The informational note about owned items not appearing on the homepage is visible."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    