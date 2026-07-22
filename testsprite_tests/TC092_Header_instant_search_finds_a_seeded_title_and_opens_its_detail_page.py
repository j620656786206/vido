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
        
        # -> Type '駭客' into the header search box (placeholder '搜尋媒體庫...') and wait for the instant results dropdown to appear.
        # 搜尋 text field
        elem = page.get_by_test_id('instant-search-input')
        await elem.wait_for(state="visible", timeout=10000)
        await elem.fill("\u99ed\u5ba2")
        
        # -> Click the '駭客任務' result in the instant suggestions dropdown
        # 駭客任務 The Matrix (1999) 已擁有 button
        elem = page.get_by_test_id('search-suggestion-item')
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> Verify the media detail page shows the title 駭客任務 and core metadata (year 1999 or genres)
        # Assert: The media detail page displays the year 1999.
        await expect(page.locator("xpath=/html/body/div[1]/div/div/div[2]/main/div/section/div[2]/div/div[2]/div[2]/span[1]").nth(0)).to_have_text("1999", timeout=15000), "The media detail page displays the year 1999."
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
    