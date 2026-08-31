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
        
        # -> Click the '下一步' (Next) button on the welcome step to proceed to the next setup step.
        # 下一步 button
        elem = page.get_by_test_id('next-button')
        await elem.click(timeout=10000)
        
        # -> Click the '跳過' (Skip) button on the qBittorrent step to proceed to the next wizard step.
        # 跳過 button
        elem = page.get_by_test_id('skip-button')
        await elem.click(timeout=10000)
        
        # -> Enter '/tmp' into the 資料夾路徑 (folder path) input and click the '+ 新增媒體庫' (Add media library) button.
        # /media/movies text field
        elem = page.get_by_test_id('library-path-0')
        await elem.wait_for(state="visible", timeout=10000)
        await elem.fill("/tmp")
        
        # -> Enter '/tmp' into the 資料夾路徑 (folder path) input and click the '+ 新增媒體庫' (Add media library) button.
        # 新增媒體庫 button
        elem = page.get_by_test_id('add-library-button')
        await elem.click(timeout=10000)
        
        # -> Click the '移除此媒體庫' (Remove this library) button for the '/media/movies' entry to leave only the '/tmp' library and attempt to enable the '下一步' button.
        # 移除此媒體庫 button
        elem = page.get_by_test_id('library-remove-1')
        await elem.click(timeout=10000)
        
        # -> Click the '下一步' (Next) button on the media library wizard card to proceed to the next step.
        # 下一步 button
        elem = page.get_by_test_id('next-button')
        await elem.click(timeout=10000)
        
        # -> Click the '跳過' (Skip) button on the API 金鑰 (API key) step to advance to the next wizard step.
        # 跳過 button
        elem = page.get_by_test_id('skip-button')
        await elem.click(timeout=10000)
        
        # -> Click the '完成設定' (Finish setup) button
        # 完成設定 button
        elem = page.get_by_test_id('finish-button')
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> The application's main navigation is visible (the sidebar contains the '首頁' link).
        # Assert-outcome: passed
        # Assert: The sidebar contains the '首頁' navigation link.
        await expect(page.locator("xpath=/html/body/div/div/div/div[1]/aside/nav/a[1]").nth(0)).to_have_text("\u9996\u9801", timeout=15000), "The sidebar contains the '\u9996\u9801' navigation link."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    