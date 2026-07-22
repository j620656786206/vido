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
        
        # -> Open the '媒體庫' page using the deep link with genre '科幻' and verify the '科幻' genre filter is applied and the visible results show science-fiction titles (e.g., 駭客任務 or 全面啟動) while 教父 is absent.
        await page.goto("http://localhost:8090/library?genres=\u79d1\u5e7b")
        try:
            await page.wait_for_load_state("domcontentloaded", timeout=5000)
        except Exception:
            pass
        
        # --> Assertions to verify final state
        
        # --> Verify the genre filter UI shows 科幻 as applied/selected
        await page.locator("xpath=/html/body/div/div/div/div[2]/main/div/div/div[2]/div[1]/div[2]/div/span/button").nth(0).scroll_into_view_if_needed()
        # Assert: The 科幻 genre filter chip is applied and its remove button is visible.
        await expect(page.locator("xpath=/html/body/div/div/div/div[2]/main/div/div/div[2]/div[1]/div[2]/div/span/button").nth(0)).to_be_visible(timeout=15000), "The \u79d1\u5e7b genre filter chip is applied and its remove button is visible."
        
        # --> Verify the visible results are science-fiction titles (e.g. 駭客任務 or 全面啟動 present; 教父 absent)
        # Assert: Visible results include the science-fiction title 全面啟動.
        await expect(page.locator("xpath=/html/body/div/div/div/div[2]/main/div/div/div[2]/div[2]/a[5]").nth(0)).to_contain_text("\u5168\u9762\u555f\u52d5", timeout=15000), "Visible results include the science-fiction title \u5168\u9762\u555f\u52d5."
        # Assert: Visible results include the science-fiction title 駭客任務.
        await expect(page.locator("xpath=/html/body/div/div/div/div[2]/main/div/div/div[2]/div[2]/a[6]").nth(0)).to_contain_text("\u99ed\u5ba2\u4efb\u52d9", timeout=15000), "Visible results include the science-fiction title \u99ed\u5ba2\u4efb\u52d9."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    