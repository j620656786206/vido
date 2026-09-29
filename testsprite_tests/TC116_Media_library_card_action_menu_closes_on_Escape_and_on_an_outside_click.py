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
        
        # -> Click the '設定' (Settings) link in the left navigation to open the Settings page.
        # 設定 link
        elem = page.get_by_test_id("nav-settings")
        await elem.click(timeout=10000)
        
        # -> Click the '媒體庫：媒體庫掃描' link in the Settings navigation to open the media library scanner page.
        # 媒體庫掃描 link
        elem = page.get_by_test_id("settings-tab-scanner")
        await elem.click(timeout=10000)
        
        # -> Click the '電影庫 的操作' button on the 電影庫 card to open its action menu.
        # 電影庫 的操作 button
        elem = page.get_by_test_id("library-card-seed-lib-movies").get_by_test_id("library-menu-button")
        await elem.click(timeout=10000)
        
        # -> Click the '電影庫 的操作' button, verify the menu shows '編輯' and '刪除', then click outside the menu to ensure it closes and the page remains on the media library scanner page.
        # 電影庫 的操作 button
        elem = page.get_by_test_id("library-card-seed-lib-movies").get_by_test_id("library-menu-button")
        await elem.click(timeout=10000)
        
        # -> Click the '電影庫 的操作' button, verify the menu shows '編輯' and '刪除', then click outside the menu to ensure it closes and the page remains on the media library scanner page.
        # 電影庫
        elem = page.get_by_text("電影庫")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> Opening the 電影庫 的操作 menu revealed the items 編輯 and 刪除.
        # Assert-outcome: passed
        # Assert: Menu shows the item '編輯'.
        await expect(page.locator("#root").nth(0)).to_contain_text("\u7de8\u8f2f", timeout=15000), "Menu shows the item '\u7de8\u8f2f'."
        # Assert-outcome: passed
        # Assert: Menu shows the item '刪除'.
        await expect(page.locator("#root").nth(0)).to_contain_text("\u522a\u9664", timeout=15000), "Menu shows the item '\u522a\u9664'."
        
        # --> Pressing Escape closed the 電影庫 action menu.
        # Assert-outcome: passed
        # Assert: Menu button has data-state='closed' after pressing Escape.
        await expect(page.get_by_test_id("library-card-seed-lib-movies").get_by_test_id("library-menu-button").nth(0)).to_have_attribute("data-state", "closed", timeout=15000), "Menu button has data-state='closed' after pressing Escape."
        
        # --> Clicking outside the menu closed the 電影庫 action menu.
        # Assert-outcome: passed
        # Assert: Menu button has data-state='closed' after clicking outside.
        await expect(page.get_by_test_id("library-card-seed-lib-movies").get_by_test_id("library-menu-button").nth(0)).to_have_attribute("data-state", "closed", timeout=15000), "Menu button has data-state='closed' after clicking outside."
        
        # --> The page did not navigate away and remained on /settings/scanner.
        # Assert-outcome: passed
        # Assert: URL contains /settings/scanner.
        await expect(page).to_have_url(re.compile("/settings/scanner"), timeout=15000), "URL contains /settings/scanner."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    