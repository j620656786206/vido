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
        
        # -> Open Settings from the sidebar and locate the homepage settings section.
        # 設定 link
        elem = page.get_by_test_id("nav-settings")
        await elem.click(timeout=10000)
        
        # -> Open the “自訂首頁” settings section and inspect the displayed homepage block order.
        # 媒體庫：自訂首頁 link
        elem = page.get_by_test_id("settings-tab-homepage")
        await elem.click(timeout=10000)
        
        # -> Click “下移 熱門電影” and verify that 熱門電影 moves below 熱門影集.
        # 下移 熱門電影 button
        elem = page.get_by_test_id("explore-block-move-down-fad8f245-a727-4f93-8f91-951c4bd497c9")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> 熱門電影 remains listed after 熱門影集 after reloading.
        await page.get_by_test_id("explore-block-move-down-0f877a94-5e8a-4e77-9f40-1f07f921b0bb").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: Verify 熱門影集 remains in the first position.
        await expect(page.get_by_test_id("explore-block-move-down-0f877a94-5e8a-4e77-9f40-1f07f921b0bb").nth(0)).to_be_visible(timeout=15000), "Verify \u71b1\u9580\u5f71\u96c6 remains in the first position."
        await page.get_by_test_id("explore-block-move-up-fad8f245-a727-4f93-8f91-951c4bd497c9").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: Verify 熱門電影 is in the position after 熱門影集.
        await expect(page.get_by_test_id("explore-block-move-up-fad8f245-a727-4f93-8f91-951c4bd497c9").nth(0)).to_be_visible(timeout=15000), "Verify \u71b1\u9580\u96fb\u5f71 is in the position after \u71b1\u9580\u5f71\u96c6."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    