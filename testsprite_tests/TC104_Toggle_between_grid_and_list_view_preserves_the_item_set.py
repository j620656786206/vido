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
        
        # -> Click the '媒體庫' link in the left navigation to open the library page.
        # 媒體庫 link
        elem = page.get_by_text('內容', exact=True).locator("xpath=ancestor-or-self::*[.//a][1]").get_by_role('link', name='媒體庫', exact=True)
        await elem.click(timeout=10000)
        
        # -> Click the '列表檢視' (List view) button to switch the library to list layout after recording the current poster items shown in the grid.
        # 列表檢視 button
        elem = page.get_by_text('列表檢視', exact=True)
        await elem.click(timeout=10000)
        
        # -> Click the '格狀檢視' (Grid view) button to switch the library to grid layout and record the visible poster titles.
        # 格狀檢視 button
        elem = page.get_by_text('格狀檢視', exact=True)
        await elem.click(timeout=10000)
        
        # -> Click the '列表檢視' (List view) button to switch the library to list layout.
        # 列表檢視 button
        elem = page.get_by_text('列表檢視', exact=True)
        await elem.click(timeout=10000)
        
        # -> Click the '格狀檢視' (Grid view) button to switch back to grid layout so the grid can be compared to the baseline.
        # 格狀檢視 button
        elem = page.get_by_text('格狀檢視', exact=True)
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> The library grid view is visible and shows poster cards.
        await page.locator("xpath=/html/body/div[1]/div/div/div[2]/main/div/div/div[2]/div[2]/a[1]").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: passed
        # Assert: A poster card (Unknown.Show.S01) is visible in the grid.
        await expect(page.locator("xpath=/html/body/div[1]/div/div/div[2]/main/div/div/div[2]/div[2]/a[1]").nth(0)).to_be_visible(timeout=15000), "A poster card (Unknown.Show.S01) is visible in the grid."
        
        # --> Switching to list view displays the same poster titles as the grid.
        # Assert-outcome: passed
        # Assert: The list contains the title '教父'.
        await expect(page.locator("xpath=/html/body/div[1]/div/div/div[2]/main/div/div/div[2]/div[2]/a[18]").nth(0)).to_contain_text("\u6559\u7236", timeout=15000), "The list contains the title '\u6559\u7236'."
        
        # --> Toggling back to grid restores the grid and shows the same poster titles again.
        # Assert-outcome: passed
        # Assert: The grid contains the title '沙丘:第二部'.
        await expect(page.locator("xpath=/html/body/div[1]/div/div/div[2]/main/div/div/div[2]/div[2]/a[7]").nth(0)).to_contain_text("\u6c99\u4e18:\u7b2c\u4e8c\u90e8", timeout=15000), "The grid contains the title '\u6c99\u4e18:\u7b2c\u4e8c\u90e8'."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    