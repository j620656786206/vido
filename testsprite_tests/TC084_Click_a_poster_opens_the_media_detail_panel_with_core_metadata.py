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
        
        # -> Click the '媒體庫' link in the left sidebar to open the media library view.
        # 媒體庫 link
        elem = page.get_by_text('內容', exact=True).locator("xpath=ancestor-or-self::*[.//a][1]").get_by_role('link', name='媒體庫', exact=True)
        await elem.click(timeout=10000)
        
        # -> Click the '駭客任務' poster card to open its media detail side panel.
        # 缺字幕 8.7 駭客任務 1999 link
        elem = page.get_by_test_id('poster-v2-seed-mv-003')
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> Verify the media detail view shows the title 駭客任務
        # Assert: The media detail view shows the title 駭客任務.
        await expect(page.locator("xpath=/html/body/div[1]").nth(0)).to_contain_text("\u99ed\u5ba2\u4efb\u52d9", timeout=15000), "The media detail view shows the title \u99ed\u5ba2\u4efb\u52d9."
        
        # --> Verify the year 1999 is visible in the detail metadata
        # Assert: Detail metadata displays the year 1999.
        await expect(page.locator("xpath=/html/body/div[1]/div/div/div[2]/main/div/section/div[2]/div/div[2]/div[2]/span[1]").nth(0)).to_have_text("1999", timeout=15000), "Detail metadata displays the year 1999."
        
        # --> Verify a rating value (e.g. 8.7 or a star score) is visible
        # Assert: Rating '8.7' is visible in the media detail panel.
        await expect(page.locator("xpath=/html/body/div[1]/div/div/div[2]/main/div/section/div[2]/div/div[2]/div[2]/div/div/span[2]").nth(0)).to_have_text("8.7", timeout=15000), "Rating '8.7' is visible in the media detail panel."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    