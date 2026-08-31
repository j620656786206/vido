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
        
        # -> Click the 'Unknown.Show.S01' poster card in the 最近新增 row to open its media detail page.
        # U 整理中 Unknown.Show.S01 link
        elem = page.get_by_test_id('poster-v2-seed-sr-101')
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> The media detail page for the selected poster opened and shows the title and the file-path metadata label.
        # Assert-outcome: passed
        # Assert: The page displays the media title 'Unknown.Show.S01'.
        await expect(page.locator("xpath=/html/body/div").nth(0)).to_contain_text("Unknown.Show.S01", timeout=15000), "The page displays the media title 'Unknown.Show.S01'."
        # Assert-outcome: passed
        # Assert: The page displays the file information label '路徑' indicating a file path is shown.
        await expect(page.locator("xpath=/html/body/div").nth(0)).to_contain_text("\u8def\u5f91", timeout=15000), "The page displays the file information label '\u8def\u5f91' indicating a file path is shown."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    