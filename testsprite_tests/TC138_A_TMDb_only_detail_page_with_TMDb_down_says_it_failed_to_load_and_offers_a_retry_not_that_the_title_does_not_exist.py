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
        
        # -> Open the media detail page for the movie and verify the heading '無法載入這部影片' and the message '詳情資料暫時無法取得，請稍後再試。' are visible.
        await page.goto("http://localhost:8090/media/movie/238")
        try:
            await page.wait_for_load_state("domcontentloaded", timeout=5000)
        except Exception:
            pass
        
        # -> Click the '重試' button to trigger a retry of loading the movie details.
        # 重試 button
        elem = page.get_by_test_id("detail-load-error-retry")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> The TMDb-load-error heading and message are visible on the movie detail page.
        # Assert-outcome: passed
        # Assert: Heading '無法載入這部影片' is visible.
        await expect(page.locator("#root").nth(0)).to_contain_text("\u7121\u6cd5\u8f09\u5165\u9019\u90e8\u5f71\u7247", timeout=15000), "Heading '\u7121\u6cd5\u8f09\u5165\u9019\u90e8\u5f71\u7247' is visible."
        # Assert-outcome: passed
        # Assert: Message '詳情資料暫時無法取得，請稍後再試。' is visible.
        await expect(page.locator("#root").nth(0)).to_contain_text("\u8a73\u60c5\u8cc7\u6599\u66ab\u6642\u7121\u6cd5\u53d6\u5f97\uff0c\u8acb\u7a0d\u5f8c\u518d\u8a66\u3002", timeout=15000), "Message '\u8a73\u60c5\u8cc7\u6599\u66ab\u6642\u7121\u6cd5\u53d6\u5f97\uff0c\u8acb\u7a0d\u5f8c\u518d\u8a66\u3002' is visible."
        
        # --> An error code label containing TMDB_UNAUTHORIZED is visible.
        # Assert-outcome: passed
        # Assert: The page shows the error label 'TMDB_UNAUTHORIZED'.
        await expect(page.locator("#root").nth(0)).to_contain_text("TMDB_UNAUTHORIZED", timeout=15000), "The page shows the error label 'TMDB_UNAUTHORIZED'."
        
        # --> The '重試' and '返回媒體庫' buttons are visible and interactable.
        await page.get_by_test_id("detail-load-error-retry").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: passed
        # Assert: The '重試' button is visible.
        await expect(page.get_by_test_id("detail-load-error-retry").nth(0)).to_be_visible(timeout=15000), "The '\u91cd\u8a66' button is visible."
        await page.get_by_test_id("detail-load-error").get_by_role("button", name="返回媒體庫").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: passed
        # Assert: The '返回媒體庫' button is visible.
        await expect(page.get_by_test_id("detail-load-error").get_by_role("button", name="返回媒體庫").nth(0)).to_be_visible(timeout=15000), "The '\u8fd4\u56de\u5a92\u9ad4\u5eab' button is visible."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    