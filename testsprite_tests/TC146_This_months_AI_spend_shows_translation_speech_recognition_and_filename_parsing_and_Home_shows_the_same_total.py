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
        
        # -> Open the “活動” page and check the 本月 AI 花費 breakdown.
        # 活動 link
        elem = page.get_by_test_id("app-sidebar").get_by_test_id("nav-activity")
        await elem.click(timeout=10000)
        
        # -> Open the 首頁 page and verify that its 本月 readout shows $0.6.
        # 首頁 link
        elem = page.get_by_test_id("app-sidebar").get_by_test_id("nav-home")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> The Home monthly-spend readout is visible.
        await page.get_by_test_id("readout-attention").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: The Home monthly-spend readout is visible.
        await expect(page.get_by_test_id("readout-attention").nth(0)).to_be_visible(timeout=15000), "The Home monthly-spend readout is visible."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    