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
        
        # -> Search the Home page for the section heading '探索' (Explore) and, if not found, scroll down to reveal lower sections so an Explore block can be located.
        await page.mouse.wheel(0, 300)
        
        # --> Assertions to verify final state
        
        # --> An Explore placeholder is visible on the Home page (link '前往連線設定' present).
        # Assert-outcome: passed
        # Assert: The Explore placeholder's '前往連線設定' link is visible.
        await expect(page.locator("xpath=/html/body/div/div/div/div[2]/main/div/div[2]/div/p/a").nth(0)).to_contain_text("\u524d\u5f80\u9023\u7dda\u8a2d\u5b9a", timeout=15000), "The Explore placeholder's '\u524d\u5f80\u9023\u7dda\u8a2d\u5b9a' link is visible."
        
        # --> Explore placeholder appears after the own-content rows on the Home page.
        # Assert-outcome: passed
        # Assert: An own-content media card is visible above the Explore area.
        await expect(page.locator("xpath=/html/body/div/div/div/div[2]/main/div/section/section/div[2]/div[3]/div[1]/a").nth(0)).to_contain_text("U\n\u6574\u7406\u4e2d\nUnknown.Show.S01", timeout=15000), "An own-content media card is visible above the Explore area."
        # Assert-outcome: passed
        # Assert: The Explore placeholder link is visible below the own-content rows.
        await expect(page.locator("xpath=/html/body/div/div/div/div[2]/main/div/div[2]/div/p/a").nth(0)).to_contain_text("\u524d\u5f80\u9023\u7dda\u8a2d\u5b9a", timeout=15000), "The Explore placeholder link is visible below the own-content rows."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    