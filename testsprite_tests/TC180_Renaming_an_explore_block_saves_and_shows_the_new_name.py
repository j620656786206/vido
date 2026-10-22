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
        
        # -> Open the settings page using the visible “設定” navigation link.
        # 設定 link
        elem = page.get_by_test_id("nav-settings")
        await elem.click(timeout=10000)
        
        # -> Open the 自訂首頁 settings section to locate the 近期新片 list item.
        # 媒體庫：自訂首頁 link
        elem = page.get_by_test_id("settings-tab-homepage")
        await elem.click(timeout=10000)
        
        # -> Open the “近期新片” section editor and inspect the “編輯探索區塊” dialog.
        # 編輯 近期新片 button
        elem = page.get_by_test_id("explore-block-edit-eb9035f4-130f-4ca5-a61d-14577013da4d")
        await elem.click(timeout=10000)
        
        # -> Change the “區塊名稱” to “近期新片（測試）” and save the section.
        # 例如：熱門台劇 text field
        elem = page.get_by_test_id("explore-block-name-input")
        await elem.wait_for(state="visible", timeout=10000)
        await elem.fill("\u8fd1\u671f\u65b0\u7247\uff08\u6e2c\u8a66\uff09")
        
        # -> Change the “區塊名稱” to “近期新片（測試）” and save the section.
        # 儲存變更 button
        elem = page.get_by_test_id("explore-block-save-button")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> 「近期新片（測試）」在重新載入後仍列於區塊清單中。
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: 區塊清單中應仍有「近期新片（測試）」。
        await expect(page.get_by_test_id("explore-block-edit-eb9035f4-130f-4ca5-a61d-14577013da4d").nth(0)).to_have_attribute("aria-label", "\u7de8\u8f2f \u8fd1\u671f\u65b0\u7247\uff08\u6e2c\u8a66\uff09", timeout=15000), "\u5340\u584a\u6e05\u55ae\u4e2d\u61c9\u4ecd\u6709\u300c\u8fd1\u671f\u65b0\u7247\uff08\u6e2c\u8a66\uff09\u300d\u3002"
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    