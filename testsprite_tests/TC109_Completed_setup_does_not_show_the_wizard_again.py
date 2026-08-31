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
        
        # -> Click the '下一步' button to advance the setup wizard from the welcome step.
        # 下一步 button
        elem = page.get_by_test_id('next-button')
        await elem.click(timeout=10000)
        
        # -> Click the '跳過' (Skip) button to skip the qBittorrent step and advance to the media library step.
        # 跳過 button
        elem = page.get_by_test_id('skip-button')
        await elem.click(timeout=10000)
        
        # -> Click the '新增媒體庫' button to add the media library, then click the '下一步' button to advance to the API 金鑰 step.
        # 新增媒體庫 button
        elem = page.get_by_test_id('add-library-button')
        await elem.click(timeout=10000)
        
        # -> Click the '新增媒體庫' button to add the media library, then click the '下一步' button to advance to the API 金鑰 step.
        # 下一步 button
        elem = page.get_by_test_id('next-button')
        await elem.click(timeout=10000)
        
        # -> Enter '/media/movies' into the first media library path field, then click the '下一步' (Next) button to advance to the API 金鑰 step.
        # /media/movies text field
        elem = page.get_by_test_id('library-path-0')
        await elem.wait_for(state="visible", timeout=10000)
        await elem.fill("/media/movies")
        
        # -> Enter '/media/movies' into the first media library path field, then click the '下一步' (Next) button to advance to the API 金鑰 step.
        # 下一步 button
        elem = page.get_by_test_id('next-button')
        await elem.click(timeout=10000)
        
        # -> Click the '移除此媒體庫' (Remove this media library) button on the second media library card to leave a single valid library and allow the '下一步' button to enable.
        # 移除此媒體庫 button
        elem = page.get_by_test_id('library-remove-1')
        await elem.click(timeout=10000)
        
        # -> Click the '下一步' (Next) button to advance to the API 金鑰 step
        # 下一步 button
        elem = page.get_by_test_id('next-button')
        await elem.click(timeout=10000)
        
        # -> Fill the '資料夾路徑' (folder path) field with '/tmp' and click the '下一步' (Next) button to attempt to advance to the API 金鑰 step.
        # /media/movies text field
        elem = page.get_by_test_id('library-path-0')
        await elem.wait_for(state="visible", timeout=10000)
        await elem.fill("/tmp")
        
        # -> Fill the '資料夾路徑' (folder path) field with '/tmp' and click the '下一步' (Next) button to attempt to advance to the API 金鑰 step.
        # 下一步 button
        elem = page.get_by_test_id('next-button')
        await elem.click(timeout=10000)
        
        # -> Click the '下一步' button on the API 金鑰 step to advance to the final 完成 (Finish) step.
        # 下一步 button
        elem = page.get_by_test_id('next-button')
        await elem.click(timeout=10000)
        
        # -> Navigate to the '/setup' page to simulate revisiting the setup flow and verify the wizard does not restart and the main application UI (e.g., dashboard or main navigation) is shown instead.
        await page.goto("http://localhost:8090/setup")
        try:
            await page.wait_for_load_state("domcontentloaded", timeout=5000)
        except Exception:
            pass
        
        # --> Assertions to verify final state
        
        # --> Revisiting /setup after completing setup shows the main application UI instead of restarting the setup wizard.
        await page.locator("xpath=/html/body/div/div/div/div[1]/aside/nav/a[1]").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: passed
        # Assert: The main sidebar navigation item '首頁' is visible, indicating the app UI is shown.
        await expect(page.locator("xpath=/html/body/div/div/div/div[1]/aside/nav/a[1]").nth(0)).to_be_visible(timeout=15000), "The main sidebar navigation item '\u9996\u9801' is visible, indicating the app UI is shown."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    