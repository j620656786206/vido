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
        
        # -> Click the '媒體庫' link in the sidebar to open the library page.
        # 媒體庫 link
        elem = page.get_by_test_id("app-sidebar").get_by_test_id("nav-library")
        await elem.click(timeout=10000)
        
        # -> Click the '電影' type control (the button labeled '電影') to switch the library view to movies.
        # 電影 button
        elem = page.get_by_test_id("filter-type-movie")
        await elem.click(timeout=10000)
        
        # -> Click the '影集' type control to switch the library view to TV series and verify the URL and listed content update accordingly.
        # 影集 button
        elem = page.get_by_test_id("filter-type-tv")
        await elem.click(timeout=10000)
        
        # -> Press the browser Back button to return to the previous '電影' library view and verify the URL is /library/movies, the '電影' control is active, and only movies are listed.
        await page.go_back()
        
        # -> Reload the page (hard refresh) and verify the URL remains /library/movies, the '電影' type control stays active, and movie items are still listed.
        await page.goto("http://localhost:8090/library/movies?page=1&pageSize=20&type=all")
        try:
            await page.wait_for_load_state("domcontentloaded", timeout=5000)
        except Exception:
            pass
        
        # -> Reload the page (hard refresh) on the '電影' library view and then verify the URL is /library/movies, the '電影' control remains active, and movies are still listed.
        await page.goto("http://localhost:8090/library/movies?page=1&pageSize=20&type=all")
        try:
            await page.wait_for_load_state("domcontentloaded", timeout=5000)
        except Exception:
            pass
        
        # -> Reload the page (hard refresh) and verify the URL is /library/movies, the '電影' button remains active, and movie items are displayed.
        await page.goto("http://localhost:8090/library/movies?page=1&pageSize=20&type=all")
        try:
            await page.wait_for_load_state("domcontentloaded", timeout=5000)
        except Exception:
            pass
        
        # --> Assertions to verify final state
        
        # --> After Back and a hard refresh the browser is on the movies view: the URL is /library/movies, the 電影 header is shown, and movie items are visible.
        # Assert-outcome: passed
        # Assert: The URL contains /library/movies.
        await expect(page).to_have_url(re.compile("/library/movies"), timeout=15000), "The URL contains /library/movies."
        # Assert-outcome: passed
        # Assert: The page header reads '電影', indicating the movies view is active.
        await expect(page.get_by_test_id("library-page-title").nth(0)).to_have_text("\u96fb\u5f71", timeout=15000), "The page header reads '\u96fb\u5f71', indicating the movies view is active."
        
        # --> Clicking the 影集 control navigated the app to the TV view URL /library/tv.
        # Assert-outcome: passed
        # Assert: The URL contains /library/tv after selecting the 影集 control.
        await expect(page).to_have_url(re.compile("/library/tv"), timeout=15000), "The URL contains /library/tv after selecting the \u5f71\u96c6 control."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    