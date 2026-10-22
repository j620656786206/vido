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
        
        # -> Open the media library from the sidebar and note the theme button label and page colors there.
        # 媒體庫 link
        elem = page.get_by_test_id("app-sidebar").get_by_test_id("nav-library")
        await elem.click(timeout=10000)
        
        # -> Click “切換到夜行” and verify that the theme button label and page colors change.
        # 切換到夜行 button
        elem = page.get_by_test_id("sidebar-footer-status").get_by_test_id("theme-toggle")
        await elem.click(timeout=10000)
        
        # -> Reload the media library page and confirm that the dark colors and “切換到日巡” label remain.
        await page.goto("http://localhost:8090/library")
        try:
            await page.wait_for_load_state("domcontentloaded", timeout=5000)
        except Exception:
            pass
        
        # --> Assertions to verify final state
        
        # --> After reloading, the theme button still reads “切換到日巡”.
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: The theme button shows “切換到日巡” after reload.
        await expect(page.get_by_test_id("sidebar-footer-status").get_by_test_id("theme-toggle").nth(0)).to_have_text("\u5207\u63db\u5230\u65e5\u5de1", timeout=15000), "The theme button shows \u201c\u5207\u63db\u5230\u65e5\u5de1\u201d after reload."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    