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
        
        # -> Navigate to the '媒體庫' (Library) page by opening /library so the item list can be inspected.
        await page.goto("http://localhost:8090/library")
        try:
            await page.wait_for_load_state("domcontentloaded", timeout=5000)
        except Exception:
            pass
        
        # -> Open the detail panel for the item labeled '[FanSub] 未知電影 (2021)' (the card showing the '失敗' badge).
        # F 失敗 [FanSub] 未知電影 (2021) link
        elem = page.get_by_test_id("poster-v2-seed-mv-102")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> A recovery action is available: the '手動選片' (manual select) button is visible.
        await page.get_by_test_id("no-metadata-manual-match").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: passed
        # Assert: The manual-select recovery button '手動選片' is visible.
        await expect(page.get_by_test_id("no-metadata-manual-match").nth(0)).to_be_visible(timeout=15000), "The manual-select recovery button '\u624b\u52d5\u9078\u7247' is visible."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    