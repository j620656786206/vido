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
        
        # -> Click the '媒體庫' (Media Library) link in the left sidebar to open the library page.
        # 媒體庫 link
        elem = page.get_by_text('內容', exact=True).locator("xpath=ancestor-or-self::*[.//a][1]").get_by_role('link', name='媒體庫', exact=True)
        await elem.click(timeout=10000)
        
        # -> Open the '排序方式' control labeled '新增日期' to reveal sort options.
        # 新增日期 button
        elem = page.get_by_test_id('sort-selector-button')
        await elem.click(timeout=10000)
        
        # -> Select the '標題' option from the sort menu to sort the library by title.
        # 標題 button
        elem = page.get_by_test_id('sort-option-title')
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> After selecting the '標題' sort option, the library grid's first card is now 'Home.Video.Collection.Vol1'.
        # Assert-outcome: passed
        # Assert: The page URL includes the selected sort parameters (sortBy=title & sortOrder=asc).
        await expect(page).to_have_url(re.compile("sortBy=title\\&sortOrder=asc"), timeout=15000), "The page URL includes the selected sort parameters (sortBy=title & sortOrder=asc)."
        await page.locator("xpath=/html/body/div[1]/div/div/div[2]/main/div/div/div[2]/div[2]/a[1]").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: passed
        # Assert: The first grid card element is visible after sorting.
        await expect(page.locator("xpath=/html/body/div[1]/div/div/div[2]/main/div/div/div[2]/div[2]/a[1]").nth(0)).to_be_visible(timeout=15000), "The first grid card element is visible after sorting."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    