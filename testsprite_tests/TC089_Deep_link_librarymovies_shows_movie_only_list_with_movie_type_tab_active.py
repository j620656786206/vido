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
        
        # -> Navigate directly to the /library/movies URL to load the 電影 (Movies) library page.
        await page.goto("http://localhost:8090/library/movies")
        try:
            await page.wait_for_load_state("domcontentloaded", timeout=5000)
        except Exception:
            pass
        
        # --> Assertions to verify final state
        
        # --> The media grid displays movie items such as '沙丘:第二部'.
        # Assert-outcome: passed
        # Assert: The grid shows the movie title '沙丘:第二部'.
        await expect(page.locator("xpath=/html/body/div/div/div/div[2]/main/div/div/div[2]/div[2]/a[4]").nth(0)).to_contain_text("\u6c99\u4e18:\u7b2c\u4e8c\u90e8", timeout=15000), "The grid shows the movie title '\u6c99\u4e18:\u7b2c\u4e8c\u90e8'."
        
        # --> Visible media items link to movie detail pages under /media/movie/ (representative items checked).
        # Assert-outcome: passed
        # Assert: A visible item links to a movie detail URL (/media/movie/seed-mv-012).
        await expect(page.locator("xpath=/html/body/div/div/div/div[2]/main/div/div/div[2]/div[2]/a[4]").nth(0)).to_have_attribute("href", "/media/movie/seed-mv-012", timeout=15000), "A visible item links to a movie detail URL (/media/movie/seed-mv-012)."
        # Assert-outcome: passed
        # Assert: Another visible item links to a movie detail URL (/media/movie/seed-mv-001).
        await expect(page.locator("xpath=/html/body/div/div/div/div[2]/main/div/div/div[2]/div[2]/a[15]").nth(0)).to_have_attribute("href", "/media/movie/seed-mv-001", timeout=15000), "Another visible item links to a movie detail URL (/media/movie/seed-mv-001)."
        
        # --> The 電影 filter control is shown in the UI and the sidebar shows the 電影 count (indicating the Movies filter is active).
        await page.locator("xpath=/html/body/div/div/div/div[2]/main/div/div/div[1]/aside/div[2]/div/div[1]/div/button[2]").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: passed
        # Assert: The 電影 filter button is present in the filters column.
        await expect(page.locator("xpath=/html/body/div/div/div/div[2]/main/div/div/div[1]/aside/div[2]/div/div[1]/div/button[2]").nth(0)).to_be_visible(timeout=15000), "The \u96fb\u5f71 filter button is present in the filters column."
        # Assert-outcome: passed
        # Assert: The sidebar shows the 電影 entry with its count ('電影 15').
        await expect(page.locator("xpath=/html/body/div/div/div/div[1]/aside/nav/div[2]/div[2]/a[1]").nth(0)).to_contain_text("\u96fb\u5f71\n15", timeout=15000), "The sidebar shows the \u96fb\u5f71 entry with its count ('\u96fb\u5f71 15')."
        
        # --> The browser remains on the /library/movies route.
        # Assert-outcome: passed
        # Assert: URL contains the /library/movies path.
        await expect(page).to_have_url(re.compile("/library/movies"), timeout=15000), "URL contains the /library/movies path."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    