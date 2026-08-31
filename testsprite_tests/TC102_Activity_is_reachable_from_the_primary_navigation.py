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
        
        # -> Click the '活動' (Activity) navigation item in the sidebar.
        # 活動 link
        elem = page.get_by_text('內容', exact=True).locator("xpath=ancestor-or-self::*[.//a][1]").get_by_role('link', name='活動', exact=True)
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> The browser navigated to the Activity route and the Activity UI is visible.
        # Assert-outcome: passed
        # Assert: URL contains '/activity'.
        await expect(page).to_have_url(re.compile("/activity"), timeout=15000), "URL contains '/activity'."
        await page.locator("xpath=/html/body/div[1]/div/div/div[2]/main/div/header/button").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: passed
        # Assert: The Activity header button '批次生成字幕' is visible.
        await expect(page.locator("xpath=/html/body/div[1]/div/div/div[2]/main/div/header/button").nth(0)).to_be_visible(timeout=15000), "The Activity header button '\u6279\u6b21\u751f\u6210\u5b57\u5e55' is visible."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    