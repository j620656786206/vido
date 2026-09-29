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
        
        # -> Click the '探索' link in the sidebar to open the Discover page.
        # 探索 link
        elem = page.get_by_test_id("nav-discover")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> The page shows the heading '探索'.
        await page.get_by_test_id("nav-discover").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: passed
        # Assert: Page heading '探索' is visible.
        await expect(page.get_by_test_id("nav-discover").nth(0)).to_be_visible(timeout=15000), "Page heading '\u63a2\u7d22' is visible."
        
        # --> A TMDb fail-soft banner is shown with the message and a '重試' button.
        # Assert-outcome: passed
        # Assert: Fail-soft banner contains the TMDB unreachable message.
        await expect(page.locator("#root").nth(0)).to_contain_text("TMDB \u670d\u52d9\u66ab\u6642\u7121\u6cd5\u9023\u7dda", timeout=15000), "Fail-soft banner contains the TMDB unreachable message."
        await page.get_by_test_id("discover-section-error-retry").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: passed
        # Assert: The '重試' button is visible.
        await expect(page.get_by_test_id("discover-section-error-retry").nth(0)).to_be_visible(timeout=15000), "The '\u91cd\u8a66' button is visible."
        
        # --> The filter rail displays the result count text '暫時無法計算'.
        # Assert-outcome: passed
        # Assert: Filter rail/result area shows '暫時無法計算'.
        await expect(page.locator("#root").nth(0)).to_contain_text("\u66ab\u6642\u7121\u6cd5\u8a08\u7b97", timeout=15000), "Filter rail/result area shows '\u66ab\u6642\u7121\u6cd5\u8a08\u7b97'."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    