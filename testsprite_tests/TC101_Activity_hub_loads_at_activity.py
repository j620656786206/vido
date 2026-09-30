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
        
        # -> Open the '活動' (Activity) page by navigating to /activity and verify the Activity page renders its events feed or a defined empty/error state.
        await page.goto("http://localhost:8090/activity")
        try:
            await page.wait_for_load_state("domcontentloaded", timeout=5000)
        except Exception:
            pass
        
        # --> Assertions to verify final state
        
        # --> The Activity page renders its main region. The test env has no
        # qBittorrent, which is "not set up", not a failure: since
        # disc-activity-downloads-unconfigured-copy (⚖️ 2026-09-30, A) the 下載
        # section is hidden and no 無法載入／重試 appears.
        await expect(page.get_by_test_id("activity-root").nth(0)).to_be_visible(timeout=15000), "The Activity page is shown."
        await expect(page.get_by_test_id("activity-section-retry")).to_have_count(0, timeout=15000), "No section shows a load failure with a Retry button."
        await expect(page.get_by_role("heading", name="\u4e0b\u8f09")).to_have_count(0, timeout=15000), "The downloads section is hidden while qBittorrent is not set up."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    