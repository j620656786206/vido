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
        
        # -> Search the homepage for the text 'degraded' (and then for the Chinese status '離線') to locate the connection health indicator.
        # 媒體庫 link
        elem = page.get_by_test_id("app-sidebar").get_by_test_id("nav-library")
        await elem.click(timeout=10000)
        
        # -> Click the '首頁' link to open the homepage and check for a degraded connection indicator (look for 'degraded' or the zh-TW equivalent such as '離線').
        # 首頁 link
        elem = page.get_by_test_id("app-sidebar").get_by_test_id("nav-home")
        await elem.click(timeout=10000)
        
        # -> Click the '媒體庫' link to open the Media Library and verify the poster grid is visible
        # 媒體庫 link
        elem = page.get_by_test_id("app-sidebar").get_by_test_id("nav-library")
        await elem.click(timeout=10000)
        
        # -> Click the '首頁' link to open the homepage and check for the degraded (離線) indicator.
        # 首頁 link
        elem = page.get_by_test_id("app-sidebar").get_by_test_id("nav-home")
        await elem.click(timeout=10000)
        
        # -> Open the '媒體庫' (Media Library) page and verify the poster grid is visible.
        await page.goto("http://localhost:8090/library")
        try:
            await page.wait_for_load_state("domcontentloaded", timeout=5000)
        except Exception:
            pass
        
        # --> Assertions to verify final state
        
        # --> The degraded connection indicator is visible in the sidebar and shows '離線'.
        await page.get_by_test_id("status-dot-tmdb").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: passed
        # Assert: Degraded indicator element is visible in the sidebar.
        await expect(page.get_by_test_id("status-dot-tmdb").nth(0)).to_be_visible(timeout=15000), "Degraded indicator element is visible in the sidebar."
        # Assert-outcome: passed
        # Assert: Degraded indicator's aria-label equals 'TMDb API：離線'.
        await expect(page.get_by_test_id("status-dot-tmdb").nth(0)).to_have_attribute("aria-label", "TMDb API\uff1a\u96e2\u7dda", timeout=15000), "Degraded indicator's aria-label equals 'TMDb API\uff1a\u96e2\u7dda'."
        
        # --> The Media Library page displays poster cards (the media grid is visible).
        await page.get_by_test_id("poster-v2-seed-sr-002").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: passed
        # Assert: At least one poster card is visible in the media grid (media library).
        await expect(page.get_by_test_id("poster-v2-seed-sr-002").nth(0)).to_be_visible(timeout=15000), "At least one poster card is visible in the media grid (media library)."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    