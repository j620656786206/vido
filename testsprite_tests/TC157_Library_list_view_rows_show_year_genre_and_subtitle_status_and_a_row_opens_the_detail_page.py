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
        
        # -> Open the media library in list view to inspect the movie rows.
        await page.goto("http://localhost:8090/library?view=list")
        try:
            await page.wait_for_load_state("domcontentloaded", timeout=5000)
        except Exception:
            pass
        
        # -> Scroll down the media list to find and verify the rows for 教父 and 侏羅紀公園.
        await page.mouse.wheel(0, 300)
        
        # -> Click the 教父 row and verify that the movie page opens with the 教父 heading.
        # 教父 1972 · 犯罪 繁中 8.7 — link
        elem = page.get_by_test_id("list-row-v2-seed-mv-001")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> Clicking 教父 opens its movie page with the 教父 heading visible.
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: The movie page URL contains /media/movie/seed-mv-001.
        await expect(page).to_have_url(re.compile("/media/movie/seed\\-mv\\-001"), timeout=15000), "The movie page URL contains /media/movie/seed-mv-001."
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: The movie page displays 教父.
        await expect(page.locator("#root").nth(0)).to_contain_text("\u6559\u7236", timeout=15000), "The movie page displays \u6559\u7236."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    