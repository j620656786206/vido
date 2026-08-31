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
        
        # -> Click the '媒體庫' link in the left sidebar to open the library view.
        # 媒體庫 link
        elem = page.get_by_text('內容', exact=True).locator("xpath=ancestor-or-self::*[.//a][1]").get_by_role('link', name='媒體庫', exact=True)
        await elem.click(timeout=10000)
        
        # -> Click the '選取' button to enter selection mode.
        # 選取 button
        elem = page.get_by_test_id('enter-selection-btn')
        await elem.click(timeout=10000)
        
        # -> Select two media items by clicking the first two item cards so the header updates to show the selected count (e.g., '已選取 2 項').
        # U 失敗 Unknown.Show.S01 link
        elem = page.get_by_test_id('poster-v2-seed-sr-101')
        await elem.click(timeout=10000)
        
        # -> Select two media items by clicking the first two item cards so the header updates to show the selected count (e.g., '已選取 2 項').
        # 缺字幕 怪奇物語 2016 link
        elem = page.get_by_test_id('poster-v2-seed-sr-002')
        await elem.click(timeout=10000)
        
        # -> Click the '刪除選取項目' button to open the deletion confirmation dialog.
        # 刪除選取項目 button
        elem = page.get_by_test_id('batch-delete-btn')
        await elem.click(timeout=10000)
        
        # -> Click the '刪除' button in the confirmation dialog to confirm deletion of the 2 selected items.
        # 刪除 button
        elem = page.get_by_test_id('confirm-action-btn')
        await elem.click(timeout=10000)
        
        # -> Click the '關閉' button on the '操作完成' dialog to close it, then search the page for the text '已選取' to verify selection is cleared and the item count updated.
        # 關閉 button
        elem = page.get_by_test_id('progress-close-btn')
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> The batch delete completed and the completion dialog showed '已完成 2 / 2'.
        # Assert-outcome: passed
        # Assert: Checks the page contains the completion dialog text '已完成 2 / 2'.
        await expect(page.locator("xpath=/html/body/div").nth(0)).to_contain_text("\u5df2\u5b8c\u6210 2 / 2", timeout=15000), "Checks the page contains the completion dialog text '\u5df2\u5b8c\u6210 2 / 2'."
        
        # --> The two targeted items ('Unknown.Show.S01' and '怪奇物語') are no longer present and the library total shows 16 項.
        # Assert-outcome: passed
        # Assert: Verifies the library total element displays '16 項'.
        await expect(page.locator("xpath=/html/body/div/div/div/div[2]/main/div/div/div[2]/div[1]/span").nth(0)).to_have_text("16\n \u9805", timeout=15000), "Verifies the library total element displays '16 \u9805'."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    