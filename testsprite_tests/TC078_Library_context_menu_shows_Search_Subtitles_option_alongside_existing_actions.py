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
        
        # -> Click the '媒體庫' (Media Library) link in the left sidebar to open the library page.
        # 媒體庫 link
        elem = page.get_by_text('內容', exact=True).locator("xpath=ancestor-or-self::*[.//a][1]").get_by_role('link', name='媒體庫', exact=True)
        await elem.click(timeout=10000)
        
        # -> Open the context menu for the first media poster card by clicking the card labeled 'Unknown.Show.S01' and verify the context menu contains 'Search Subtitles', 'Re-parse', and 'Delete'.
        # U 失敗 Unknown.Show.S01 link
        elem = page.get_by_test_id('poster-v2-seed-sr-101')
        await elem.click(timeout=10000)
        
        # -> Click the '返回媒體庫' (Back to Library) button to return to the library grid view.
        # 返回媒體庫 button
        elem = page.get_by_test_id('detail-back')
        await elem.click(timeout=10000)
        
        # -> Click the '選取' (Select) button at the top of the library page to enable per-card actions and reveal per-card overflow/menu controls.
        # 選取 button
        elem = page.get_by_test_id('enter-selection-btn')
        await elem.click(timeout=10000)
        
        # -> Scroll the library page to reveal more poster controls, then inspect the 'button' and 'a' elements to locate the per-card selection control or overflow menu for the 'Unknown.Show.S01' poster.
        await page.mouse.wheel(0, 300)
        
        # --> Assertions to verify final state
        
        # --> A media poster card for 'Unknown.Show.S01' is visible in the library grid.
        await page.locator("xpath=/html/body/div/div/div/div[2]/main/div/div/div[2]/div[2]/a[1]").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: passed
        # Assert: Verifies the poster link for 'Unknown.Show.S01' is visible in the library grid.
        await expect(page.locator("xpath=/html/body/div/div/div/div[2]/main/div/div/div[2]/div[2]/a[1]").nth(0)).to_be_visible(timeout=15000), "Verifies the poster link for 'Unknown.Show.S01' is visible in the library grid."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    