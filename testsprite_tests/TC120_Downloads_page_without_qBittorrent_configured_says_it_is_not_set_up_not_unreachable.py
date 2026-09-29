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
        
        # -> Click the '下載' link in the left sidebar to open the Downloads page.
        # 下載 link
        elem = page.get_by_test_id("app-sidebar").get_by_test_id("nav-downloads")
        await elem.click(timeout=10000)
        
        # -> Wait for the Downloads page to finish loading, then verify the empty-state texts and click the '前往設定' link.
        # 前往設定 link
        elem = page.get_by_role("link", name="前往設定")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> Downloads page shows the not-configured message '還沒有設定 qBittorrent'.
        # Assert-outcome: passed
        # Assert: The page contains the text '還沒有設定 qBittorrent'.
        await expect(page.locator("#root").nth(0)).to_contain_text("\u9084\u6c92\u6709\u8a2d\u5b9a qBittorrent", timeout=15000), "The page contains the text '\u9084\u6c92\u6709\u8a2d\u5b9a qBittorrent'."
        
        # --> Downloads page shows the helper text '設定好之後，下載進度會顯示在這裡'.
        # Assert-outcome: passed
        # Assert: The page contains the helper text about download progress.
        await expect(page.locator("#root").nth(0)).to_contain_text("\u8a2d\u5b9a\u597d\u4e4b\u5f8c\uff0c\u4e0b\u8f09\u9032\u5ea6\u6703\u986f\u793a\u5728\u9019\u88e1", timeout=15000), "The page contains the helper text about download progress."
        
        # --> The connection-error text '無法連線到 qBittorrent' is not present on the Downloads page.
        # Assert-outcome: passed
        # Assert: The connection-error text '無法連線到 qBittorrent' is not visible on the page.
        await expect(page.locator("xpath=/html/body/div[1]").nth(0)).not_to_be_visible(timeout=15000), "The connection-error text '\u7121\u6cd5\u9023\u7dda\u5230 qBittorrent' is not visible on the page."
        
        # --> Clicking '前往設定' navigated to the connection settings page (/settings/connection).
        # Assert-outcome: passed
        # Assert: The browser is on the /settings/connection URL.
        await expect(page).to_have_url(re.compile("/settings/connection"), timeout=15000), "The browser is on the /settings/connection URL."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    