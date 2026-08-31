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
        
        # -> Open the '媒體庫' (Media Library) page by clicking the '媒體庫' link in the left sidebar.
        # 媒體庫 link
        elem = page.get_by_text('內容', exact=True).locator("xpath=ancestor-or-self::*[.//a][1]").get_by_role('link', name='媒體庫', exact=True)
        await elem.click(timeout=10000)
        
        # -> Click the '選取' button to enter selection mode on the Library page.
        # 選取 button
        elem = page.get_by_test_id('enter-selection-btn')
        await elem.click(timeout=10000)
        
        # -> Click the 'Unknown.Show.S01' media card to select it and observe the selection toolbar update.
        # U 失敗 Unknown.Show.S01 link
        elem = page.get_by_test_id('poster-v2-seed-sr-101')
        await elem.click(timeout=10000)
        
        # -> Click the '取消' (Cancel) button in the selection toolbar to exit selection mode.
        # 取消 button
        elem = page.get_by_test_id('batch-cancel-btn')
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> Selection toolbar is hidden and no items remain selected after exiting selection mode.
        await page.locator("xpath=/html/body/div/div/div/div[2]/main/div/div/div[2]/div[1]/button[2]").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: passed
        # Assert: The '選取' button is visible, indicating selection mode is not active.
        await expect(page.locator("xpath=/html/body/div/div/div/div[2]/main/div/div/div[2]/div[1]/button[2]").nth(0)).to_be_visible(timeout=15000), "The '\u9078\u53d6' button is visible, indicating selection mode is not active."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    