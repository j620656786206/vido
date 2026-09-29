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
        
        # -> Click the '媒體庫' (Library) link in the left navigation to open the library page.
        # 媒體庫 link
        elem = page.get_by_test_id("app-sidebar").get_by_test_id("nav-library")
        await elem.click(timeout=10000)
        
        # -> Click the first poster card (the card titled '怪奇物語') to open the media detail panel
        # 缺字幕 怪奇物語 2016 link
        elem = page.get_by_test_id("poster-v2-seed-sr-002")
        await elem.click(timeout=10000)
        
        # -> Scroll the media detail panel and reveal the '檔案資訊' (File info) section, then look for technical badges (e.g., resolution or codec labels such as 1080, 720, HEVC, H.264) within the media detail panel.
        await page.mouse.wheel(0, 300)
        
        # --> Assertions to verify final state
        
        # --> The media detail panel for the selected item is visible.
        await page.get_by_test_id("detail-back").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: failed
        # Assert: Expected the media detail panel to be visible.
        await expect(page.get_by_test_id("detail-back").nth(0)).to_be_visible(timeout=15000), "Expected the media detail panel to be visible."
        
        # --> The '檔案資訊' (File info) section is shown in the detail panel.
        # Assert-outcome: failed
        # Assert: Expected the page to contain the '檔案資訊' (File info) heading.
        await expect(page.locator("#root").nth(0)).to_contain_text("\u6a94\u6848\u8cc7\u8a0a", timeout=15000), "Expected the page to contain the '\u6a94\u6848\u8cc7\u8a0a' (File info) heading."
        
        # --> Technical badges (resolution/codec) are not present in the media detail panel.
        # Assert-outcome: failed
        # Assert: Expected a technical badge such as '1080' to be visible in the detail panel.
        await expect(page.locator("#root").nth(0)).to_contain_text("1080", timeout=15000), "Expected a technical badge such as '1080' to be visible in the detail panel."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    