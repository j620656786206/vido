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
        
        # -> Click the '活動' navigation item in the sidebar to open the Activity page.
        # 活動 link
        elem = page.get_by_test_id("app-sidebar").get_by_test_id("nav-activity")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> The Activity page opens: the browser navigates to /activity and the '批次生成字幕' button is visible.
        # Assert-outcome: passed
        # Assert: The URL path contains /activity.
        await expect(page).to_have_url(re.compile("/activity"), timeout=15000), "The URL path contains /activity."
        # Assert-outcome: passed
        # Assert: The Activity page shows the '批次生成字幕' button.
        await expect(page.get_by_test_id("activity-generation-batch-cta").nth(0)).to_have_text("\u6279\u6b21\u751f\u6210\u5b57\u5e55", timeout=15000), "The Activity page shows the '\u6279\u6b21\u751f\u6210\u5b57\u5e55' button."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    