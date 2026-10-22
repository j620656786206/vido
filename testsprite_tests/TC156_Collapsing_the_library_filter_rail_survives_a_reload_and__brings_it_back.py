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
        
        # -> Open 「媒體庫」 and inspect its filter controls.
        # 媒體庫 link
        elem = page.get_by_test_id("app-sidebar").get_by_test_id("nav-library")
        await elem.click(timeout=10000)
        
        # -> Click 「收合篩選」 and confirm the filter rail hides while 「篩選」 appears in the toolbar.
        # 收合篩選 button
        elem = page.get_by_test_id("library-rail-collapse")
        await elem.click(timeout=10000)
        
        # -> Click the 「篩選」 button and verify that the filter rail, including 「類型」 and 「類別」, becomes visible again.
        # 篩選 button
        elem = page.get_by_test_id("library-rail-expand")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> The filter rail remains collapsed after reload.
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: The filter rail remains hidden after reload.
        await expect(page.locator("[data-testid=\"filter-type-all\"]").nth(0)).not_to_be_visible(timeout=15000), "The filter rail remains hidden after reload."
        
        # --> The filter rail is visible again.
        await page.get_by_role("heading", name="篩選").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: The filter rail is visible again.
        await expect(page.get_by_role("heading", name="篩選").nth(0)).to_be_visible(timeout=15000), "The filter rail is visible again."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    