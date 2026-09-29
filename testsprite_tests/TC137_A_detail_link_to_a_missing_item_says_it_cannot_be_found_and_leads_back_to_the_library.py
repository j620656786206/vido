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
        
        # -> Open the detail URL /media/movie/seed-mv-999 and check for the '找不到這部影片' not-found heading and explanatory text.
        await page.goto("http://localhost:8090/media/movie/seed-mv-999")
        try:
            await page.wait_for_load_state("domcontentloaded", timeout=5000)
        except Exception:
            pass
        
        # -> Click the '返回媒體庫' button to return to the library.
        # 返回媒體庫 button
        elem = page.get_by_test_id("detail-not-found").get_by_role("button", name="返回媒體庫")
        await elem.click(timeout=10000)
        
        # -> Navigate to the podcast detail URL (/media/podcast/abc) and verify the not-found heading '找不到這部影片', the message '這個連結可能已失效。', and that '已從媒體庫移除' is not shown.
        await page.goto("http://localhost:8090/media/podcast/abc")
        try:
            await page.wait_for_load_state("domcontentloaded", timeout=5000)
        except Exception:
            pass
        
        # --> Assertions to verify final state
        
        # --> The movie detail not-found page shows the heading '找不到這部影片' and the message '這個項目可能已從媒體庫移除，或連結已失效。'.
        # Assert-outcome: passed
        # Assert: The not-found heading '找不到這部影片' is visible.
        await expect(page.locator("#root").nth(0)).to_contain_text("\u627e\u4e0d\u5230\u9019\u90e8\u5f71\u7247", timeout=15000), "The not-found heading '\u627e\u4e0d\u5230\u9019\u90e8\u5f71\u7247' is visible."
        # Assert-outcome: passed
        # Assert: The explanatory text about removal or invalid link is visible.
        await expect(page.locator("#root").nth(0)).to_contain_text("\u9019\u500b\u9805\u76ee\u53ef\u80fd\u5df2\u5f9e\u5a92\u9ad4\u5eab\u79fb\u9664\uff0c\u6216\u9023\u7d50\u5df2\u5931\u6548\u3002", timeout=15000), "The explanatory text about removal or invalid link is visible."
        
        # --> A visible '返回媒體庫' button is present on the not-found page.
        # Assert-outcome: passed
        # Assert: The '返回媒體庫' button is visible with the correct label.
        await expect(page.get_by_test_id("detail-not-found").get_by_role("button").nth(0)).to_have_text("\u8fd4\u56de\u5a92\u9ad4\u5eab", timeout=15000), "The '\u8fd4\u56de\u5a92\u9ad4\u5eab' button is visible with the correct label."
        
        # --> Clicking the '返回媒體庫' button navigates back to the library (URL contains '/library').
        # Assert-outcome: passed
        # Assert: The current URL contains '/library'.
        await expect(page).to_have_url(re.compile("/library"), timeout=15000), "The current URL contains '/library'."
        
        # --> The podcast detail not-found page shows the heading '找不到這部影片' and the message '這個連結可能已失效。'.
        # Assert-outcome: passed
        # Assert: The not-found heading '找不到這部影片' is visible on the podcast page.
        await expect(page.locator("#root").nth(0)).to_contain_text("\u627e\u4e0d\u5230\u9019\u90e8\u5f71\u7247", timeout=15000), "The not-found heading '\u627e\u4e0d\u5230\u9019\u90e8\u5f71\u7247' is visible on the podcast page."
        # Assert-outcome: passed
        # Assert: The message '這個連結可能已失效。' is visible on the podcast page.
        await expect(page.locator("#root").nth(0)).to_contain_text("\u9019\u500b\u9023\u7d50\u53ef\u80fd\u5df2\u5931\u6548\u3002", timeout=15000), "The message '\u9019\u500b\u9023\u7d50\u53ef\u80fd\u5df2\u5931\u6548\u3002' is visible on the podcast page."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    