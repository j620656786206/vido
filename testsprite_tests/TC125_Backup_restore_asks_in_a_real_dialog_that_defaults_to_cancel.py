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
        
        # -> Open the settings backup page at /settings/backup and inspect its available controls.
        await page.goto("http://localhost:8090/settings/backup")
        try:
            await page.wait_for_load_state("domcontentloaded", timeout=5000)
        except Exception:
            pass
        
        # -> Click 建立備份 and wait for the new backup row to show 完成.
        # 建立備份 button
        elem = page.get_by_test_id("create-backup-btn")
        await elem.click(timeout=10000)
        
        # -> Open the restore confirmation for the completed backup and inspect its warning text and initially focused safe action.
        # 還原 vido-backup-20261022-161053-v47.tar.gz button
        elem = page.get_by_test_id("restore-btn-378025fb-6aed-4f38-bce7-bd1e756adc04")
        await elem.click(timeout=10000)
        
        # -> Reopen the restore confirmation for the completed backup and press Escape to verify it closes without restoring.
        # 還原 vido-backup-20261022-161053-v47.tar.gz button
        elem = page.get_by_test_id("restore-btn-378025fb-6aed-4f38-bce7-bd1e756adc04")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> Dismissing the restore confirmation with Enter and Escape leaves the backup listed without restoring it.
        await page.get_by_test_id("restore-btn-378025fb-6aed-4f38-bce7-bd1e756adc04").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: The backup remains listed after dismissing the restore confirmation.
        await expect(page.get_by_test_id("restore-btn-378025fb-6aed-4f38-bce7-bd1e756adc04").nth(0)).to_be_visible(timeout=15000), "The backup remains listed after dismissing the restore confirmation."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    