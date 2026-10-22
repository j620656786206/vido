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
        
        # -> Open 媒體庫 and verify its page route and active sidebar state.
        # 媒體庫 link
        elem = page.get_by_test_id("app-sidebar").get_by_test_id("nav-library")
        await elem.click(timeout=10000)
        
        # -> Open 電影 and verify the resulting page route and whether the 媒體庫 parent is marked as current.
        # 電影 15 link
        elem = page.get_by_test_id("nav-movies")
        await elem.click(timeout=10000)
        
        # -> Open 影集 from the sidebar and verify its page route and 媒體庫 parent current state.
        # 影集（3） link
        elem = page.get_by_test_id("nav-tv")
        await elem.click(timeout=10000)
        
        # -> Open 探索 from the sidebar and verify its route and current-page indication.
        # 探索 link
        elem = page.get_by_test_id("nav-discover")
        await elem.click(timeout=10000)
        
        # -> Open 活動 from the sidebar and verify its route and current-page indication.
        # 活動 link
        elem = page.get_by_test_id("app-sidebar").get_by_test_id("nav-activity")
        await elem.click(timeout=10000)
        
        # -> Open 下載 from the sidebar and verify its page route and current-page indication.
        # 下載 link
        elem = page.get_by_test_id("app-sidebar").get_by_test_id("nav-downloads")
        await elem.click(timeout=10000)
        
        # -> Open 設定 from the sidebar and verify that its page loads with a settings route and current-page indication.
        # 設定 link
        elem = page.get_by_test_id("nav-settings")
        await elem.click(timeout=10000)
        
        # -> Open 首頁 from the sidebar and verify it loads at `/` with 首頁 marked as the current page.
        # 首頁 link
        elem = page.get_by_test_id("app-sidebar").get_by_test_id("nav-home")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> 首頁 opens at `/`.
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: The URL ends at the home route `/`.
        await expect(page).to_have_url(re.compile("/$"), timeout=15000), "The URL ends at the home route `/`."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    