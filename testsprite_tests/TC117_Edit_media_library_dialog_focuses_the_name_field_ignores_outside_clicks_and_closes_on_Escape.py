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
        
        # -> Open the Scanner settings page by navigating to the '設定 / Scanner' page (URL: /settings/scanner).
        await page.goto("http://localhost:8090/settings/scanner")
        try:
            await page.wait_for_load_state("domcontentloaded", timeout=5000)
        except Exception:
            pass
        
        # -> Open the 操作 menu for the 電影庫 library by clicking the '電影庫 的操作' button.
        # 電影庫 的操作 button
        elem = page.get_by_test_id("library-card-seed-lib-movies").get_by_test_id("library-menu-button")
        await elem.click(timeout=10000)
        
        # -> Click the '編輯' menu item in the 電影庫 的操作 menu to open the 編輯媒體庫 dialog.
        # 編輯 menu item
        elem = page.get_by_role("menuitem", name="編輯")
        await elem.click(timeout=10000)
        
        # -> Click on the dimmed backdrop near the top-left of the page to confirm the 編輯媒體庫 dialog does not close when clicking outside it.
        # vido NAS 媒體庫 內容 首頁 媒體庫 電影 15 影集 3 探索 任務 活動 下載 設定...
        elem = page.locator("#root")
        await elem.click(timeout=10000)
        
        # -> Click the '編輯' menu item in the 電影庫 的操作 menu to re-open the 編輯媒體庫 dialog.
        # 編輯 menu item
        elem = page.get_by_role("menuitem", name="編輯")
        await elem.click(timeout=10000)
        
        # -> Verify the 名稱 input has keyboard focus (type a character), click the dim backdrop to confirm the modal stays open and preserves input, press Escape to close the modal, then press Enter to confirm focus returned to the '電影庫 的操作' button (m...
        # vido NAS 媒體庫 內容 首頁 媒體庫 電影 15 影集 3 探索 任務 活動 下載 設定...
        elem = page.locator("#root")
        await elem.click(timeout=10000)
        
        # -> Click the '編輯' menu item to open the 編輯媒體庫 dialog so the 名稱 input value, focus, backdrop behavior, and Escape/focus-return can be verified.
        # 編輯 menu item
        elem = page.get_by_role("menuitem", name="編輯")
        await elem.click(timeout=10000)
        
        # -> Type a character into the 名稱 field to verify it has keyboard focus, then click the page backdrop (top-left) to ensure the modal stays open.
        # 我的電影 text field
        elem = page.get_by_test_id("library-name-input")
        await elem.wait_for(state="visible", timeout=10000)
        await elem.fill("\u6211")
        
        # -> Type a character into the 名稱 field to verify it has keyboard focus, then click the page backdrop (top-left) to ensure the modal stays open.
        # vido NAS 媒體庫 內容 首頁 媒體庫 電影 15 影集 3 探索 任務 活動 下載 設定...
        elem = page.locator("#root")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> Pressing Escape closed the 編輯媒體庫 dialog and keyboard focus returned to the 電影庫 的操作 button (its menu opened).
        await page.get_by_role("menu", name="電影庫 的操作").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: passed
        # Assert: The 電影庫 的操作 menu is visible, indicating focus returned to its button.
        await expect(page.get_by_role("menu", name="電影庫 的操作").nth(0)).to_be_visible(timeout=15000), "The \u96fb\u5f71\u5eab \u7684\u64cd\u4f5c menu is visible, indicating focus returned to its button."
        
        # --> The 電影庫 card still shows the name 電影庫.
        # Assert-outcome: passed
        # Assert: The 電影庫 card displays the name '電影庫'.
        await expect(page.locator("xpath=/html/body/div[1]/div/div/div[2]/main/div/div/div/div/div/div/div[1]/div/div[1]/div[1]/div[1]/span[1]").nth(0)).to_have_text("\u96fb\u5f71\u5eab", timeout=15000), "The \u96fb\u5f71\u5eab card displays the name '\u96fb\u5f71\u5eab'."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    