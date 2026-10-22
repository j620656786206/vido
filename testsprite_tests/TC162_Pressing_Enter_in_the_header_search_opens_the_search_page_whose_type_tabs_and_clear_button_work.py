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
        
        # -> Enter 教父 in the header search box and inspect whether search suggestions appear.
        # 搜尋 text field
        elem = page.get_by_test_id("instant-search-input")
        await elem.wait_for(state="visible", timeout=10000)
        await elem.fill("\u6559\u7236")
        
        # -> Select the “影集” tab and verify the search URL uses type=tv.
        # 影集 button
        elem = page.get_by_role("tab", name="影集")
        await elem.click(timeout=10000)
        
        # -> Click the page search field’s clear button and verify that the field becomes empty.
        # 清除搜尋 button
        elem = page.locator("#base-ui-_r_m_")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> The search URL is on /search with an empty q parameter.
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: The search URL is on /search with an empty q parameter.
        await expect(page).to_have_url(re.compile("/search\\?q="), timeout=15000), "The search URL is on /search with an empty q parameter."
        
        # --> The search URL uses the 影集 filter.
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: The search URL uses the 影集 filter.
        await expect(page).to_have_url(re.compile("type=tv"), timeout=15000), "The search URL uses the \u5f71\u96c6 filter."
        
        # --> The page search field is empty after clearing the search.
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: The page search field is empty.
        await expect(page.get_by_role("textbox", name="搜尋").nth(0)).to_have_value("", timeout=15000), "The page search field is empty."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    