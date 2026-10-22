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
        
        # -> Open 探索 and inspect the Discover page for the 動作 and 台灣 filters.
        # 探索 link
        elem = page.get_by_test_id("nav-discover")
        await elem.click(timeout=10000)
        
        # -> Select the visible 動作 genre filter and inspect the resulting selection state and URL.
        # 動作 – button
        elem = page.get_by_test_id("filter-genre-28")
        await elem.click(timeout=10000)
        
        # -> Select the 台灣 region filter and check that both filters are active and represented in the URL.
        # 🇹🇼 台灣 – button
        elem = page.get_by_test_id("filter-region-TW")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> After reloading, 動作 and 台灣 remained selected and their filters were present in the URL.
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: The active filter chips show 動作 and 台灣.
        await expect(page.locator("#root").nth(0)).to_contain_text("\u985e\u578b\uff1a\u52d5\u4f5c \u5730\u5340\uff1a\u53f0\u7063", timeout=15000), "The active filter chips show \u52d5\u4f5c and \u53f0\u7063."
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: The URL includes the 動作 and 台灣 filter parameters.
        await expect(page).to_have_url(re.compile(".*genre=%2228%22.*region=TW.*"), timeout=15000), "The URL includes the \u52d5\u4f5c and \u53f0\u7063 filter parameters."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    