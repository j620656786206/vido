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
        
        # -> Click the '媒體庫' link in the sidebar to open the Library page.
        # 媒體庫 link
        elem = page.get_by_text('內容', exact=True).locator("xpath=ancestor-or-self::*[.//a][1]").get_by_role('link', name='媒體庫', exact=True)
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> Verify element "degraded state indicator" is visible
        await page.locator("xpath=/html/body/div/div/div/div[1]/aside/div[2]/div[2]/span/span[1]").nth(0).scroll_into_view_if_needed()
        # Assert: Degraded TMDb API indicator (aria-label "TMDb API：離線") is visible in the sidebar footer.
        await expect(page.locator("xpath=/html/body/div/div/div/div[1]/aside/div[2]/div[2]/span/span[1]").nth(0)).to_be_visible(timeout=15000), "Degraded TMDb API indicator (aria-label \"TMDb API\uff1a\u96e2\u7dda\") is visible in the sidebar footer."
        
        # --> Verify a grid of media poster cards is still visible (core feature unaffected by degradation)
        await page.locator("xpath=/html/body/div/div/div/div[2]/main/div/div/div[2]/div[2]/a[16]").nth(0).scroll_into_view_if_needed()
        # Assert: Media poster card 駭客任務 (1999) is visible in the library grid.
        await expect(page.locator("xpath=/html/body/div/div/div/div[2]/main/div/div/div[2]/div[2]/a[16]").nth(0)).to_be_visible(timeout=15000), "Media poster card \u99ed\u5ba2\u4efb\u52d9 (1999) is visible in the library grid."
        await page.locator("xpath=/html/body/div/div/div/div[2]/main/div/div/div[2]/div[2]/a[18]").nth(0).scroll_into_view_if_needed()
        # Assert: Media poster card 教父 (1972) is visible in the library grid.
        await expect(page.locator("xpath=/html/body/div/div/div/div[2]/main/div/div/div[2]/div[2]/a[18]").nth(0)).to_be_visible(timeout=15000), "Media poster card \u6559\u7236 (1972) is visible in the library grid."
        await page.locator("xpath=/html/body/div/div/div/div[2]/main/div/div/div[2]/div[2]/a[1]").nth(0).scroll_into_view_if_needed()
        # Assert: At least one poster card (Unknown.Show.S01) is visible in the library grid.
        await expect(page.locator("xpath=/html/body/div/div/div/div[2]/main/div/div/div[2]/div[2]/a[1]").nth(0)).to_be_visible(timeout=15000), "At least one poster card (Unknown.Show.S01) is visible in the library grid."
        current_url = await page.evaluate("() => window.location.href")
        # Assert: page loaded with a URL (final outcome verified by the AI judge during the run)
        assert current_url, 'Page should have loaded with a URL'
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    