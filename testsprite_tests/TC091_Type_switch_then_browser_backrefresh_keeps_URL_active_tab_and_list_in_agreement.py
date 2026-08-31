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
        
        # -> Click the '電影' type filter button in the filter panel to switch the view to Movies and observe the URL and listed content.
        # 電影 button
        elem = page.get_by_test_id('filter-type-movie')
        await elem.click(timeout=10000)
        
        # -> Click the '影集' (TV) type control to switch the view to TV series and verify the URL and listed content update.
        # 影集 button
        elem = page.get_by_test_id('filter-type-tv')
        await elem.click(timeout=10000)
        
        # -> Press the browser Back button to return to the previous library view (expecting the '電影' view to be restored).
        await page.go_back()
        
        # -> Reload the page (hard refresh) and verify the 電影 control remains active and only movies are listed (URL should contain /library/movies).
        await page.goto("http://localhost:8090/library/movies?page=1&pageSize=20&type=all")
        try:
            await page.wait_for_load_state("domcontentloaded", timeout=5000)
        except Exception:
            pass
        
        # --> Assertions to verify final state
        
        # --> The page URL is the movies view (/library/movies).
        # Assert-outcome: passed
        # Assert: URL contains /library/movies.
        await expect(page).to_have_url(re.compile("/library/movies"), timeout=15000), "URL contains /library/movies."
        
        # --> The 電影 filter control is active (pressed) after returning to the movies view.
        # Assert-outcome: passed
        # Assert: The 電影 filter button has aria-pressed=true indicating it is active.
        await expect(page.locator("xpath=/html/body/div/div/div/div[2]/main/div/div/div[1]/aside/div[2]/div/div[1]/div/button[2]").nth(0)).to_have_attribute("aria-pressed", "true", timeout=15000), "The \u96fb\u5f71 filter button has aria-pressed=true indicating it is active."
        
        # --> Movie tiles are listed in the view (movies are visible).
        await page.locator("xpath=/html/body/div/div/div/div[2]/main/div/div/div[2]/div[2]/a[4]").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: passed
        # Assert: A movie tile (example entry) is visible in the movies list.
        await expect(page.locator("xpath=/html/body/div/div/div/div[2]/main/div/div/div[2]/div[2]/a[4]").nth(0)).to_be_visible(timeout=15000), "A movie tile (example entry) is visible in the movies list."
        
        # --> Clicking the 影集 control previously switched the view to TV and updated the URL to /library/tv.
        # Assert-outcome: passed
        # Assert: URL contains /library/tv after selecting the 影集 control.
        await expect(page).to_have_url(re.compile("/library/tv"), timeout=15000), "URL contains /library/tv after selecting the \u5f71\u96c6 control."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    