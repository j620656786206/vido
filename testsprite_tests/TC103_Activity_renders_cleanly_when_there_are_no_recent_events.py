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
        
        # -> Click the '活動' (Activity) link in the sidebar to open the Activity page.
        # 活動 link
        elem = page.get_by_test_id("app-sidebar").get_by_test_id("nav-activity")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> The Activity page loaded and shows a per-section fail-soft message ('無法載入，請稍後再試') with a visible '重試' button.
        # Assert-outcome: passed
        # Assert: The page shows the Activity header '活動'.
        await expect(page.locator("#root").nth(0)).to_contain_text("\u6d3b\u52d5", timeout=15000), "The page shows the Activity header '\u6d3b\u52d5'."
        # Assert-outcome: passed
        # Assert: The Downloads section shows a visible Retry button labeled '重試'.
        await expect(page.get_by_test_id("activity-section-retry").nth(0)).to_have_text("\u91cd\u8a66", timeout=15000), "The Downloads section shows a visible Retry button labeled '\u91cd\u8a66'."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    