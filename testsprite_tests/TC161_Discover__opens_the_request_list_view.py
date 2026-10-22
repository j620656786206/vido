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
        
        # -> Open the 探索 page using the visible navigation link.
        # 探索 link
        elem = page.get_by_test_id("nav-discover")
        await elem.click(timeout=10000)
        
        # -> Click the 想要清單 button and verify that the request list opens with an empty state and no error.
        # 想要清單 button
        elem = page.get_by_test_id("discover-requests-entry")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> The URL identifies the request-list view with `view=requests`.
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: The URL contains `view=requests`.
        await expect(page).to_have_url(re.compile("view=requests"), timeout=15000), "The URL contains `view=requests`."
        
        # --> The request list displays the empty-state message「尚無請求」.
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: The request list shows the empty-state message「尚無請求」.
        await expect(page.locator("#root").nth(0)).to_contain_text("\u5c1a\u7121\u8acb\u6c42", timeout=15000), "The request list shows the empty-state message\u300c\u5c1a\u7121\u8acb\u6c42\u300d."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    