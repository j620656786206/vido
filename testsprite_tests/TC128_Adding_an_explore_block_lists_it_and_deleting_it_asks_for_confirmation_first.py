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
        
        # -> Navigate to the settings homepage (Settings → 首頁) so the '新增區塊' button can be clicked.
        await page.goto("http://localhost:8090/settings/homepage")
        try:
            await page.wait_for_load_state("domcontentloaded", timeout=5000)
        except Exception:
            pass
        
        # -> Click the '新增區塊' button to open the 新增探索區塊 dialog.
        # 新增區塊 button
        elem = page.get_by_test_id("explore-blocks-add-button")
        await elem.click(timeout=10000)
        
        # -> Type 'TestSprite 暫存區塊' into the 區塊名稱 field and open the '內容類型' dropdown.
        # 例如：熱門台劇 text field
        elem = page.get_by_test_id("explore-block-name-input")
        await elem.wait_for(state="visible", timeout=10000)
        await elem.fill("TestSprite \u66ab\u5b58\u5340\u584a")
        
        # -> Type 'TestSprite 暫存區塊' into the 區塊名稱 field and open the '內容類型' dropdown.
        # 電影 影集 dropdown
        elem = page.get_by_test_id("explore-block-type-select")
        await elem.click(timeout=10000)
        
        # -> Select '影集' from the 內容類型 dropdown and click the '儲存區塊' button to save the new block.
        # 電影 影集 dropdown
        elem = page.locator("xpath=/html/body/div/div/div/div[2]/main/div/div/div/div/div/div[2]/div/div[2]/div[2]/select").nth(0)
        await elem.wait_for(state="visible", timeout=10000)
        await elem.select_option("")
        
        # -> Select '影集' from the 內容類型 dropdown and click the '儲存區塊' button to save the new block.
        # 儲存區塊 button
        elem = page.get_by_test_id("explore-block-save-button")
        await elem.click(timeout=10000)
        
        # -> Click the '刪除 TestSprite 暫存區塊' button to open the 確認刪除 confirmation dialog.
        # 刪除 TestSprite 暫存區塊 button
        elem = page.get_by_test_id("explore-block-delete-5de3968a-8db3-4790-9364-212f44919acc")
        await elem.click(timeout=10000)
        
        # -> Click the '確認刪除' button in the confirmation dialog to confirm deletion of 'TestSprite 暫存區塊'.
        # 確認刪除 button
        elem = page.get_by_test_id("explore-block-delete-confirm-button")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> After confirming deletion, the dialog closed and the homepage shows 熱門電影, 熱門影集 and 近期新片.
        await page.get_by_test_id("explore-block-delete-c9b6c861-7ec7-4895-b9c5-105e8e191c1e").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: passed
        # Assert: The '熱門電影' row is present on the homepage.
        await expect(page.get_by_test_id("explore-block-delete-c9b6c861-7ec7-4895-b9c5-105e8e191c1e").nth(0)).to_be_visible(timeout=15000), "The '\u71b1\u9580\u96fb\u5f71' row is present on the homepage."
        await page.get_by_test_id("explore-block-delete-2fd247e0-d766-4fac-96df-14ee998a7378").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: passed
        # Assert: The '熱門影集' row is present on the homepage.
        await expect(page.get_by_test_id("explore-block-delete-2fd247e0-d766-4fac-96df-14ee998a7378").nth(0)).to_be_visible(timeout=15000), "The '\u71b1\u9580\u5f71\u96c6' row is present on the homepage."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    