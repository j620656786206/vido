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
        
        # -> Click the '下一步' button to proceed from the welcome step.
        # 下一步 button
        elem = page.get_by_test_id("next-button")
        await elem.click(timeout=10000)
        
        # -> Click the '跳過' (Skip) button to bypass the qBittorrent connection step.
        # 跳過 button
        elem = page.get_by_test_id("skip-button")
        await elem.click(timeout=10000)
        
        # -> Enter '/tmp' into the media folder path input, click the '新增媒體庫' button to add it, then click the '下一步' button to proceed.
        # /media/movies text field
        elem = page.get_by_test_id("library-path-0")
        await elem.wait_for(state="visible", timeout=10000)
        await elem.fill("/tmp")
        
        # -> Enter '/tmp' into the media folder path input, click the '新增媒體庫' button to add it, then click the '下一步' button to proceed.
        # 新增媒體庫 button
        elem = page.get_by_test_id("add-library-button")
        await elem.click(timeout=10000)
        
        # -> Enter '/tmp' into the media folder path input, click the '新增媒體庫' button to add it, then click the '下一步' button to proceed.
        # 下一步 button
        elem = page.get_by_test_id("next-button")
        await elem.click(timeout=10000)
        
        # -> Click the '移除此媒體庫' (remove) button for the empty media library entry, then click the '下一步' button to proceed.
        # 移除此媒體庫 button
        elem = page.get_by_test_id("library-remove-1")
        await elem.click(timeout=10000)
        
        # -> Click the '移除此媒體庫' (remove) button for the empty media library entry, then click the '下一步' button to proceed.
        # 下一步 button
        elem = page.get_by_test_id("next-button")
        await elem.click(timeout=10000)
        
        # -> Click the '跳過' button to skip API key setup
        # 跳過 button
        elem = page.get_by_test_id("skip-button")
        await elem.click(timeout=10000)
        
        # -> Click the '完成設定' button on the final summary page to finish the setup wizard.
        # 完成設定 button
        elem = page.get_by_test_id("finish-button")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> The setup wizard is no longer shown and the main navigation (首頁, 媒體庫, 設定) is visible.
        await page.get_by_test_id("app-sidebar").get_by_test_id("nav-library").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: passed
        # Assert: The main navigation link '媒體庫' is visible on the page.
        await expect(page.get_by_test_id("app-sidebar").get_by_test_id("nav-library").nth(0)).to_be_visible(timeout=15000), "The main navigation link '\u5a92\u9ad4\u5eab' is visible on the page."
        # Assert-outcome: passed
        # Assert: The browser address is on the application root, indicating the wizard was exited.
        await expect(page).to_have_url(re.compile("localhost:8090/"), timeout=15000), "The browser address is on the application root, indicating the wizard was exited."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    