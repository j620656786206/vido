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
        
        # -> Type 'zzqxnomatch' into the search box labeled '搜尋媒體庫...' and wait for the dropdown to settle.
        # 搜尋 text field
        elem = page.get_by_test_id("instant-search-input")
        await elem.wait_for(state="visible", timeout=10000)
        await elem.fill("zzqxnomatch")
        
        # --> Assertions to verify final state
        
        # --> The search dropdown shows the TMDb unreachable notice containing ‘只顯示媒體庫結果’.
        # Assert-outcome: passed
        # Assert: The dropdown shows the TMDb unreachable notice text.
        await expect(page.get_by_test_id("search-suggestions-tmdb-down").nth(0)).to_have_text("TMDb \u66ab\u6642\u7121\u6cd5\u9023\u7dda\uff0c\u53ea\u986f\u793a\u5a92\u9ad4\u5eab\u7d50\u679c", timeout=15000), "The dropdown shows the TMDb unreachable notice text."
        
        # --> The search dropdown shows the local empty-state message 媒體庫裡沒有「zzqxnomatch」 for the query.
        # Assert-outcome: passed
        # Assert: The dropdown shows the library-empty message for the query.
        await expect(page.get_by_test_id("search-suggestions-empty").nth(0)).to_have_text("\u5a92\u9ad4\u5eab\u88e1\u6c92\u6709\u300czzqxnomatch\u300d", timeout=15000), "The dropdown shows the library-empty message for the query."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    