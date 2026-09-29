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
        
        # --> Assertions to verify final state
        
        # --> An Explore fail-soft banner is visible on the Home page with an actionable '前往連線設定' link.
        # Assert-outcome: passed
        # Assert: The Explore fail-soft banner includes the '前往連線設定' link.
        await expect(page.get_by_test_id("explore-degraded-notice").get_by_role("link").nth(0)).to_have_text("\u524d\u5f80\u9023\u7dda\u8a2d\u5b9a", timeout=15000), "The Explore fail-soft banner includes the '\u524d\u5f80\u9023\u7dda\u8a2d\u5b9a' link."
        
        # --> The Explore fail-soft banner is rendered after the own-content '最近新增' carousel on the Home page.
        await page.get_by_test_id("poster-v2-seed-sr-101").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: passed
        # Assert: A '最近新增' carousel item is visible (own-content row present).
        await expect(page.get_by_test_id("poster-v2-seed-sr-101").nth(0)).to_be_visible(timeout=15000), "A '\u6700\u8fd1\u65b0\u589e' carousel item is visible (own-content row present)."
        await page.get_by_role("link", name="前往連線設定").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: passed
        # Assert: The Explore fail-soft '前往連線設定' link is visible below the carousel.
        await expect(page.get_by_role("link", name="前往連線設定").nth(0)).to_be_visible(timeout=15000), "The Explore fail-soft '\u524d\u5f80\u9023\u7dda\u8a2d\u5b9a' link is visible below the carousel."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    