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
        
        # -> Click the '設定' (Settings) link in the left navigation to open the Settings page.
        # 設定 link
        elem = page.get_by_test_id("nav-settings")
        await elem.click(timeout=10000)
        
        # -> Click the '備份與還原' (Backup and Restore) link in the Settings navigation to open the Backup page.
        # 備份與還原 link
        elem = page.get_by_test_id("settings-tab-backup")
        await elem.click(timeout=10000)
        
        # -> Click the '建立備份' button and wait for a backup entry showing the status '完成' to appear on the page.
        # 建立備份 button
        elem = page.get_by_test_id("create-backup-btn")
        await elem.click(timeout=10000)
        
        # -> Click the '還原' (Restore) button for the completed backup to open the 確認還原 confirmation dialog.
        # 還原 vido-backup-20260929-202450-v39.tar.gz button
        elem = page.get_by_test_id("restore-btn-12dc8eb1-248b-4e1b-a795-e2d9539f377a")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> The backup 'vido-backup-20260929-202450-v39.tar.gz' remains listed on the Backup page.
        await page.get_by_test_id("restore-btn-12dc8eb1-248b-4e1b-a795-e2d9539f377a").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: passed
        # Assert: The backup's restore button is visible, proving the backup entry is still listed.
        await expect(page.get_by_test_id("restore-btn-12dc8eb1-248b-4e1b-a795-e2d9539f377a").nth(0)).to_be_visible(timeout=15000), "The backup's restore button is visible, proving the backup entry is still listed."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    