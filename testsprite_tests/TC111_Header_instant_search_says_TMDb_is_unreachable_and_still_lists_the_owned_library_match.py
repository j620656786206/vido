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
        
        # -> Click the '搜尋媒體庫...' search input and type '駭客', then wait for the dropdown to settle.
        # 搜尋 text field
        elem = page.get_by_test_id("instant-search-input")
        await elem.wait_for(state="visible", timeout=10000)
        await elem.fill("\u99ed\u5ba2")
        
        # --> Assertions to verify final state
        
        # --> The header search dropdown shows the TMDb-unavailable notice: "TMDb 暫時無法連線，只顯示媒體庫結果".
        # Assert-outcome: passed
        # Assert: The dropdown displays the TMDb-down notice.
        await expect(page.get_by_test_id("search-suggestions-tmdb-down").nth(0)).to_have_text("TMDb \u66ab\u6642\u7121\u6cd5\u9023\u7dda\uff0c\u53ea\u986f\u793a\u5a92\u9ad4\u5eab\u7d50\u679c", timeout=15000), "The dropdown displays the TMDb-down notice."
        
        # --> The dropdown lists a 媒體庫 result titled "駭客任務" with its subtext and ownership tag.
        # Assert-outcome: passed
        # Assert: A library result titled '駭客任務' appears in the suggestions.
        await expect(page.get_by_test_id("search-suggestion-item").nth(0)).to_contain_text("\u99ed\u5ba2\u4efb\u52d9", timeout=15000), "A library result titled '\u99ed\u5ba2\u4efb\u52d9' appears in the suggestions."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    