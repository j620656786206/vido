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
        
        # --> Assertions to verify final state
        
        # --> The 最近新增 (Recently Added) row is visible on the Home page.
        await page.locator("xpath=/html/body/div/div/div/div[2]/main/div/section/section/div[1]/a").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: passed
        # Assert: The 最近新增 row header (整理中 · 3) is visible.
        await expect(page.locator("xpath=/html/body/div/div/div/div[2]/main/div/section/section/div[1]/a").nth(0)).to_be_visible(timeout=15000), "The \u6700\u8fd1\u65b0\u589e row header (\u6574\u7406\u4e2d \u00b7 3) is visible."
        
        # --> The Explore blocks area is rendered below the 最近新增 row and shows the TMDb connection error.
        await page.locator("xpath=/html/body/div/div/div/div[2]/main/div/div[2]/div/p/a").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: passed
        # Assert: The Explore area shows the '前往連線設定' link (connection error) and is visible.
        await expect(page.locator("xpath=/html/body/div/div/div/div[2]/main/div/div[2]/div/p/a").nth(0)).to_be_visible(timeout=15000), "The Explore area shows the '\u524d\u5f80\u9023\u7dda\u8a2d\u5b9a' link (connection error) and is visible."
        
        # --> Poster cards from the seeded library are rendered in the 最近新增 row.
        await page.locator("xpath=/html/body/div/div/div/div[2]/main/div/section/section/div[2]/div[3]/div[1]/a").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: passed
        # Assert: At least one poster card is visible in the 最近新增 row.
        await expect(page.locator("xpath=/html/body/div/div/div/div[2]/main/div/section/section/div[2]/div[3]/div[1]/a").nth(0)).to_be_visible(timeout=15000), "At least one poster card is visible in the \u6700\u8fd1\u65b0\u589e row."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    