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
        
        # -> Open the “設定” page from the app navigation.
        # 設定 link
        elem = page.get_by_test_id("nav-settings")
        await elem.click(timeout=10000)
        
        # -> Open the “媒體庫掃描” settings page.
        # 媒體庫掃描 link
        elem = page.get_by_test_id("settings-tab-scanner")
        await elem.click(timeout=10000)
        
        # -> Open the “新增媒體庫” dialog and inspect its visible fields and initial state.
        # 新增媒體庫 button
        elem = page.get_by_test_id("add-library-button")
        await elem.click(timeout=10000)
        
        # -> Enter 測試庫 as the name and /no/such/folder as the folder path, then check that the form is ready to submit.
        # 我的電影 text field
        elem = page.get_by_test_id("library-name-input")
        await elem.wait_for(state="visible", timeout=10000)
        await elem.fill("\u6e2c\u8a66\u5eab")
        
        # -> Enter 測試庫 as the name and /no/such/folder as the folder path, then check that the form is ready to submit.
        # /media/movies text field
        elem = page.get_by_test_id("library-path-input")
        await elem.wait_for(state="visible", timeout=10000)
        await elem.fill("/no/such/folder")
        
        # -> Submit the 新增媒體庫 dialog and verify the 測試庫 card shows /no/such/folder and 無法存取.
        # 建立 button
        elem = page.get_by_test_id("library-save-button")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> The 測試庫 library is listed with the path /no/such/folder and status 無法存取.
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: The 測試庫 library should show the submitted path and 無法存取 status.
        await expect(page.locator("#root").nth(0)).to_contain_text("\u6e2c\u8a66\u5eab \u96fb\u5f71 /no/such/folder \u7121\u6cd5\u5b58\u53d6", timeout=15000), "The \u6e2c\u8a66\u5eab library should show the submitted path and \u7121\u6cd5\u5b58\u53d6 status."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    