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
        
        # -> Open the 影集 page from the media-library navigation and verify the displayed series and absence of movies.
        # 影集 link
        elem = page.get_by_test_id("nav-tv")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> The 影集 library is active.
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: The current page is the 影集 library.
        await expect(page).to_have_url(re.compile("/library/tv"), timeout=15000), "The current page is the \u5f71\u96c6 library."
        
        # --> 怪奇物語 and 進擊的巨人 are listed in the TV library.
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: The TV results include 怪奇物語.
        await expect(page.get_by_test_id("library-results").nth(0)).to_contain_text("\u602a\u5947\u7269\u8a9e", timeout=15000), "The TV results include \u602a\u5947\u7269\u8a9e."
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: The TV results include 進擊的巨人.
        await expect(page.get_by_test_id("library-results").nth(0)).to_contain_text("\u9032\u64ca\u7684\u5de8\u4eba", timeout=15000), "The TV results include \u9032\u64ca\u7684\u5de8\u4eba."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    