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
        
        # -> Open the /setup page (visit http://localhost:8090/setup) and verify that the setup wizard does not re-open and the main navigation/content remains visible.
        await page.goto("http://localhost:8090/setup")
        try:
            await page.wait_for_load_state("domcontentloaded", timeout=5000)
        except Exception:
            pass
        
        # -> Open the '/setup' page and confirm that the main app navigation or 首頁 content is visible (the setup wizard should not appear).
        await page.goto("http://localhost:8090/setup")
        try:
            await page.wait_for_load_state("domcontentloaded", timeout=5000)
        except Exception:
            pass
        
        # -> Open the '/setup' page and verify that the main navigation/home content (for example the '首頁' dashboard) is visible and the setup wizard does not appear.
        await page.goto("http://localhost:8090/setup")
        try:
            await page.wait_for_load_state("domcontentloaded", timeout=5000)
        except Exception:
            pass
        
        # -> Visit the URL '/setup' (http://localhost:8090/setup) and check whether the setup wizard appears or the main dashboard ('首頁' and '最近新增') remains visible.
        await page.goto("http://localhost:8090/setup")
        try:
            await page.wait_for_load_state("domcontentloaded", timeout=5000)
        except Exception:
            pass
        
        # -> Navigate to the /setup page and verify whether the setup wizard appears or the main dashboard ('首頁' and '最近新增') remains visible.
        await page.goto("http://localhost:8090/setup")
        try:
            await page.wait_for_load_state("domcontentloaded", timeout=5000)
        except Exception:
            pass
        
        # -> Navigate to the '/setup' page and confirm the setup wizard does not appear — the main dashboard ('首頁' and '最近新增') should remain visible.
        await page.goto("http://localhost:8090/setup")
        try:
            await page.wait_for_load_state("domcontentloaded", timeout=5000)
        except Exception:
            pass
        
        # --> Assertions to verify final state
        
        # --> Revisiting /setup did not reopen the setup wizard — the main dashboard is visible with the sidebar '首頁' link.
        await page.get_by_test_id("app-sidebar").get_by_test_id("nav-home").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: passed
        # Assert: The sidebar '首頁' navigation link is visible, showing the main app UI is displayed.
        await expect(page.get_by_test_id("app-sidebar").get_by_test_id("nav-home").nth(0)).to_be_visible(timeout=15000), "The sidebar '\u9996\u9801' navigation link is visible, showing the main app UI is displayed."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    