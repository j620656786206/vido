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
        
        # --> The 最近新增 (Recently Added) row is visible on the Home page (poster card 'Unknown.Show.S01' is shown).
        await page.get_by_test_id("poster-v2-seed-sr-101").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: passed
        # Assert: A poster card (Unknown.Show.S01) in the 最近新增 row is visible.
        await expect(page.get_by_test_id("poster-v2-seed-sr-101").nth(0)).to_be_visible(timeout=15000), "A poster card (Unknown.Show.S01) in the \u6700\u8fd1\u65b0\u589e row is visible."
        
        # --> The Explore/Hero area (fail‑soft banner and '前往連線設定' link) is rendered below the 最近新增 row.
        await page.get_by_test_id("poster-v2-seed-sr-101").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: passed
        # Assert: A poster card in the 最近新增 row is visible (establishes the own-content row's presence above).
        await expect(page.get_by_test_id("poster-v2-seed-sr-101").nth(0)).to_be_visible(timeout=15000), "A poster card in the \u6700\u8fd1\u65b0\u589e row is visible (establishes the own-content row's presence above)."
        await page.get_by_role("link", name="前往連線設定").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: passed
        # Assert: The Explore-area fail‑soft banner's '前往連線設定' link is visible (establishes Explore content below).
        await expect(page.get_by_role("link", name="前往連線設定").nth(0)).to_be_visible(timeout=15000), "The Explore-area fail\u2011soft banner's '\u524d\u5f80\u9023\u7dda\u8a2d\u5b9a' link is visible (establishes Explore content below)."
        
        # --> Seeded poster cards from the library render in the 最近新增 row (example: '怪奇物語' is shown).
        await page.get_by_test_id("poster-v2-seed-sr-002").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: passed
        # Assert: Seeded poster card '怪奇物語' is visible in the 最近新增 row.
        await expect(page.get_by_test_id("poster-v2-seed-sr-002").nth(0)).to_be_visible(timeout=15000), "Seeded poster card '\u602a\u5947\u7269\u8a9e' is visible in the \u6700\u8fd1\u65b0\u589e row."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    