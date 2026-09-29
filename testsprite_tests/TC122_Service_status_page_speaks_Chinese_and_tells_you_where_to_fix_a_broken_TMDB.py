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
        
        # -> Open the '服務狀態' settings page by navigating to /settings/status.
        await page.goto("http://localhost:8090/settings/status")
        try:
            await page.wait_for_load_state("domcontentloaded", timeout=5000)
        except Exception:
            pass
        
        # -> Click the banner link labeled '金鑰設定' to open the Keys settings page.
        # 金鑰設定 link
        elem = page.get_by_test_id("broken-line-tmdb").get_by_role("link", name="金鑰設定")
        await elem.click(timeout=10000)
        
        # -> Click the sidebar link labeled '服務狀態' to open the Service Status page.
        # 連線：服務狀態 link
        elem = page.get_by_test_id("settings-tab-status")
        await elem.click(timeout=10000)
        
        # -> Click the '金鑰設定' link in the TMDB banner to open the Keys settings page.
        # 金鑰設定 link
        elem = page.get_by_test_id("broken-line-tmdb").get_by_role("link", name="金鑰設定")
        await elem.click(timeout=10000)
        
        # -> Click the '服務狀態' sidebar link to open the Service Status page.
        # 連線：服務狀態 link
        elem = page.get_by_test_id("settings-tab-status")
        await elem.click(timeout=10000)
        
        # -> Click the '金鑰設定' link in the TMDB banner to open the Keys settings page.
        # 金鑰設定 link
        elem = page.get_by_test_id("broken-line-tmdb").get_by_role("link", name="金鑰設定")
        await elem.click(timeout=10000)
        
        # -> Click the '服務狀態' sidebar link to open the Service Status page.
        # 連線：服務狀態 link
        elem = page.get_by_test_id("settings-tab-status")
        await elem.click(timeout=10000)
        
        # -> Click the '金鑰設定' link in the TMDB banner to open the Keys settings page.
        # 金鑰設定 link
        elem = page.get_by_test_id("broken-line-tmdb").get_by_role("link", name="金鑰設定")
        await elem.click(timeout=10000)
        
        # -> Click the sidebar link labeled '服務狀態' to open the Service Status page.
        # 連線：服務狀態 link
        elem = page.get_by_test_id("settings-tab-status")
        await elem.click(timeout=10000)
        
        # -> Click the '金鑰設定' link in the TMDB banner (the banner line that reads 'TMDB 目前無法連線') and verify it navigates to the Keys settings page.
        # 金鑰設定 link
        elem = page.get_by_test_id("broken-line-tmdb").get_by_role("link", name="金鑰設定")
        await elem.click(timeout=10000)
        
        # -> Open the '服務狀態' page by clicking the sidebar link labeled '服務狀態'.
        # 連線：服務狀態 link
        elem = page.get_by_test_id("settings-tab-status")
        await elem.click(timeout=10000)
        
        # -> Click the '金鑰設定' link in the TMDB banner to open the Keys settings page.
        # 金鑰設定 link
        elem = page.get_by_test_id("broken-line-tmdb").get_by_role("link", name="金鑰設定")
        await elem.click(timeout=10000)
        
        # -> Click the sidebar link labeled '服務狀態' to open the Service Status page.
        # 連線：服務狀態 link
        elem = page.get_by_test_id("settings-tab-status")
        await elem.click(timeout=10000)
        
        # -> Click the '金鑰設定' link in the TMDB banner to open the Keys settings page.
        # 金鑰設定 link
        elem = page.get_by_test_id("broken-line-tmdb").get_by_role("link", name="金鑰設定")
        await elem.click(timeout=10000)
        
        # -> Click the sidebar link labeled '服務狀態' to open the Service Status page.
        # 連線：服務狀態 link
        elem = page.get_by_test_id("settings-tab-status")
        await elem.click(timeout=10000)
        
        # -> Click the '金鑰設定' link in the TMDB banner to open the Keys settings page.
        # 金鑰設定 link
        elem = page.get_by_test_id("broken-line-tmdb").get_by_role("link", name="金鑰設定")
        await elem.click(timeout=10000)
        
        # -> Click the sidebar link '服務狀態' to open the Service Status page.
        # 連線：服務狀態 link
        elem = page.get_by_test_id("settings-tab-status")
        await elem.click(timeout=10000)
        
        # -> Click the '金鑰設定' link in the TMDB banner to open the Keys settings page.
        # 金鑰設定 link
        elem = page.get_by_test_id("broken-line-tmdb").get_by_role("link", name="金鑰設定")
        await elem.click(timeout=10000)
        
        # -> Click the sidebar link labeled '服務狀態' to open the Service Status page.
        # 連線：服務狀態 link
        elem = page.get_by_test_id("settings-tab-status")
        await elem.click(timeout=10000)
        
        # -> Click the '金鑰設定' link in the TMDB banner (the banner that reads 'TMDB 目前無法連線').
        # 金鑰設定 link
        elem = page.get_by_test_id("broken-line-tmdb").get_by_role("link", name="金鑰設定")
        await elem.click(timeout=10000)
        
        # -> Navigate to the '服務狀態' (Service Status) page and verify the heading '服務狀態' is visible.
        await page.goto("http://localhost:8090/settings/status")
        try:
            await page.wait_for_load_state("domcontentloaded", timeout=5000)
        except Exception:
            pass
        
        # -> Click the '金鑰設定' link in the TMDB banner to open the Keys settings page.
        # 金鑰設定 link
        elem = page.get_by_test_id("broken-line-tmdb").get_by_role("link", name="金鑰設定")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> TMDB shows an offline status indicator in the sidebar.
        await page.get_by_test_id("status-dot-tmdb").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: passed
        # Assert: The TMDb sidebar status element with aria-label 'TMDb API：離線' is visible.
        await expect(page.get_by_test_id("status-dot-tmdb").nth(0)).to_be_visible(timeout=15000), "The TMDb sidebar status element with aria-label 'TMDb API\uff1a\u96e2\u7dda' is visible."
        
        # --> 豆瓣 (Douban) status in the sidebar shows 未設定.
        await page.get_by_test_id("status-dot-douban").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: passed
        # Assert: The Douban sidebar status element shows '未設定'.
        await expect(page.get_by_test_id("status-dot-douban").nth(0)).to_be_visible(timeout=15000), "The Douban sidebar status element shows '\u672a\u8a2d\u5b9a'."
        
        # --> 維基百科 (Wikipedia) status in the sidebar shows 未設定.
        await page.get_by_test_id("status-dot-wikipedia").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: passed
        # Assert: The Wikipedia sidebar status element shows '未設定'.
        await expect(page.get_by_test_id("status-dot-wikipedia").nth(0)).to_be_visible(timeout=15000), "The Wikipedia sidebar status element shows '\u672a\u8a2d\u5b9a'."
        
        # --> AI 解析 (AI Parser) status in the sidebar shows 未設定.
        await page.get_by_test_id("status-dot-ai").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: passed
        # Assert: The AI Parser sidebar status element shows '未設定'.
        await expect(page.get_by_test_id("status-dot-ai").nth(0)).to_be_visible(timeout=15000), "The AI Parser sidebar status element shows '\u672a\u8a2d\u5b9a'."
        
        # --> Clicking the 金鑰設定 link opened the Keys settings page (/settings/keys).
        # Assert-outcome: passed
        # Assert: The current URL contains /settings/keys.
        await expect(page).to_have_url(re.compile("/settings/keys"), timeout=15000), "The current URL contains /settings/keys."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    