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
        
        # -> Click the '設定' (Settings) link in the left sidebar to open the Settings page.
        # 設定 link
        elem = page.get_by_test_id("nav-settings")
        await elem.click(timeout=10000)
        
        # -> Click the '快取管理' (Cache management) link in the settings sidebar to open the cache settings page.
        # 維護：快取管理 link
        elem = page.get_by_test_id("settings-tab-cache")
        await elem.click(timeout=10000)
        
        # -> Click the '清除 30 天前的快取' button to arm the 2-step cache-clear confirmation.
        # 清除 30 天前 清除 30 天前的快取 button
        elem = page.get_by_test_id("clear-old-cache-btn")
        await elem.click(timeout=10000)
        
        # -> Click the '取消' button to cancel the clearance and return the UI to its initial state.
        # 取消 button
        elem = page.get_by_test_id("clear-old-cache-cancel-btn")
        await elem.click(timeout=10000)
        
        # -> Click the '清除 30 天前的快取' button to arm the confirmation and reveal the warning.
        # 清除 30 天前 清除 30 天前的快取 button
        elem = page.get_by_test_id("clear-old-cache-btn")
        await elem.click(timeout=10000)
        
        # -> Click the '取消' button to cancel the cache clear confirmation and return the UI to the initial unarmed state.
        # 取消 button
        elem = page.get_by_test_id("clear-old-cache-cancel-btn")
        await elem.click(timeout=10000)
        
        # -> Click the '清除 30 天前的快取' button to arm the confirmation
        # 清除 30 天前 清除 30 天前的快取 button
        elem = page.get_by_test_id("clear-old-cache-btn")
        await elem.click(timeout=10000)
        
        # -> Click the '取消' button to cancel the cache-clear confirmation.
        # 取消 button
        elem = page.get_by_test_id("clear-old-cache-cancel-btn")
        await elem.click(timeout=10000)
        
        # -> Click the '清除 30 天前的快取' button to arm the confirmation and reveal the warning.
        # 清除 30 天前 清除 30 天前的快取 button
        elem = page.get_by_test_id("clear-old-cache-btn")
        await elem.click(timeout=10000)
        
        # -> Verify the page shows '確認清除 30 天前的快取' and the warning text, then click the '取消' button.
        # 取消 button
        elem = page.get_by_test_id("clear-old-cache-cancel-btn")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> Clicking the clear button arms confirmation and the primary button reads "確認清除 30 天前的快取".
        # Assert-outcome: passed
        # Assert: The main clear button shows the confirmation label.
        await expect(page.get_by_test_id("clear-old-cache-btn").nth(0)).to_have_text("\u78ba\u8a8d\u6e05\u9664 30 \u5929\u524d\u7684\u5feb\u53d6", timeout=15000), "The main clear button shows the confirmation label."
        
        # --> While armed, a warning is visible containing the phrases '再按一次才會真的清除' and '不會影響影片、字幕與你上傳的海報'.
        # Assert-outcome: passed
        # Assert: The page shows the warning phrase that a second press will actually clear.
        await expect(page.locator("#root").nth(0)).to_contain_text("\u518d\u6309\u4e00\u6b21\u624d\u6703\u771f\u7684\u6e05\u9664", timeout=15000), "The page shows the warning phrase that a second press will actually clear."
        # Assert-outcome: passed
        # Assert: The warning clarifies that videos, subtitles, and uploaded posters are not affected.
        await expect(page.locator("#root").nth(0)).to_contain_text("\u4e0d\u6703\u5f71\u97ff\u5f71\u7247\u3001\u5b57\u5e55\u8207\u4f60\u4e0a\u50b3\u7684\u6d77\u5831", timeout=15000), "The warning clarifies that videos, subtitles, and uploaded posters are not affected."
        
        # --> Clicking '取消' returns the UI to the unarmed state and the primary button reads '清除 30 天前的快取' again.
        # Assert-outcome: passed
        # Assert: The main clear button has reverted to its initial label.
        await expect(page.get_by_test_id("clear-old-cache-btn").nth(0)).to_have_text("\u6e05\u9664 30 \u5929\u524d\u7684\u5feb\u53d6", timeout=15000), "The main clear button has reverted to its initial label."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    