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
        
        # -> Open Settings from the sidebar to reach the service-status page.
        # 設定 link
        elem = page.get_by_test_id("nav-settings")
        await elem.click(timeout=10000)
        
        # -> Open the “服務狀態” page from the Settings navigation.
        # 連線：服務狀態 link
        elem = page.get_by_test_id("settings-tab-status")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> The 服務狀態 heading is visible.
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: The service-status heading is visible.
        await expect(page.locator("#root").nth(0)).to_contain_text("\u670d\u52d9\u72c0\u614b", timeout=15000), "The service-status heading is visible."
        
        # --> The service-status cards for TMDB, 豆瓣, 維基百科, AI 解析, and qBittorrent are visible.
        await page.get_by_test_id("service-cards-list").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: The service-status cards are visible.
        await expect(page.get_by_test_id("service-cards-list").nth(0)).to_be_visible(timeout=15000), "The service-status cards are visible."
        
        # --> The TMDB, 豆瓣, 維基百科, and AI 解析 cards show 未設定.
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: The service-status cards show the unconfigured status.
        await expect(page.get_by_test_id("service-cards-list").nth(0)).to_contain_text("\u672a\u8a2d\u5b9a", timeout=15000), "The service-status cards show the unconfigured status."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    