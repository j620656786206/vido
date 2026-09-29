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
        
        # -> Navigate to the Library page using the URL path '/library?genres=科幻' so the genre filter '科幻' should be applied on load.
        await page.goto("http://localhost:8090/library?genres=\u79d1\u5e7b")
        try:
            await page.wait_for_load_state("domcontentloaded", timeout=5000)
        except Exception:
            pass
        
        # --> Assertions to verify final state
        
        # --> The genre filter '科幻' is applied and visible in the active filters area.
        await page.get_by_role("button", name="移除 科幻 篩選").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: passed
        # Assert: The '科幻' filter chip (remove button) is visible, indicating the genre is applied.
        await expect(page.get_by_role("button", name="移除 科幻 篩選").nth(0)).to_be_visible(timeout=15000), "The '\u79d1\u5e7b' filter chip (remove button) is visible, indicating the genre is applied."
        
        # --> Visible results include science-fiction titles such as '全面啟動' and '駭客任務'.
        # Assert-outcome: passed
        # Assert: The page shows the '全面啟動' result tile.
        await expect(page.get_by_test_id("poster-v2-seed-mv-006").nth(0)).to_contain_text("\u5168\u9762\u555f\u52d5", timeout=15000), "The page shows the '\u5168\u9762\u555f\u52d5' result tile."
        # Assert-outcome: passed
        # Assert: The page shows the '駭客任務' result tile.
        await expect(page.get_by_test_id("poster-v2-seed-mv-003").nth(0)).to_contain_text("\u99ed\u5ba2\u4efb\u52d9", timeout=15000), "The page shows the '\u99ed\u5ba2\u4efb\u52d9' result tile."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    