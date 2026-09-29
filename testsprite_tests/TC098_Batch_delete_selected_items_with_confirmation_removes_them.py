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
        
        # -> Click the '媒體庫' (Media Library) link in the left sidebar to open the library page.
        # 媒體庫 link
        elem = page.get_by_test_id("app-sidebar").get_by_test_id("nav-library")
        await elem.click(timeout=10000)
        
        # -> Click the '選取' (Select) button to enter selection mode
        # 選取 button
        elem = page.get_by_test_id("enter-selection-btn")
        await elem.click(timeout=10000)
        
        # -> Select the first two visible media items ('Unknown.Show.S01' and '怪奇物語') by clicking their cards to mark them as selected.
        # U 整理中 Unknown.Show.S01 link
        elem = page.get_by_test_id("poster-v2-seed-sr-101")
        await elem.click(timeout=10000)
        
        # -> Select the first two visible media items ('Unknown.Show.S01' and '怪奇物語') by clicking their cards to mark them as selected.
        # 缺字幕 怪奇物語 2016 link
        elem = page.get_by_test_id("poster-v2-seed-sr-002")
        await elem.click(timeout=10000)
        
        # -> Click the '刪除選取項目' (Delete selected items) button to open the deletion confirmation dialog.
        # 刪除選取項目 button
        elem = page.get_by_test_id("batch-delete-btn")
        await elem.click(timeout=10000)
        
        # -> Click the '刪除' button in the confirmation dialog to confirm deletion of the 2 selected items.
        # 刪除 button
        elem = page.get_by_test_id("confirm-action-btn")
        await elem.click(timeout=10000)
        
        # -> Confirm that the items 'Unknown.Show.S01' and '怪奇物語' are no longer present on the library page and that the library count shows '媒體庫 16 部', then close the completion dialog using the '關閉' button.
        # 關閉 button
        elem = page.get_by_test_id("progress-close-btn")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> The library header shows 16 items after the batch delete.
        # Assert-outcome: passed
        # Assert: Library header contains the updated item count '16'.
        await expect(page.get_by_test_id("library-result-count").nth(0)).to_contain_text("16", timeout=15000), "Library header contains the updated item count '16'."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    