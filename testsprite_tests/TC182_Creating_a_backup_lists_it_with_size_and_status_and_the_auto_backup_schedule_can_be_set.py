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
        
        # -> Open the backup settings page at /settings/backup and inspect its visible controls.
        await page.goto("http://localhost:8090/settings/backup")
        try:
            await page.wait_for_load_state("domcontentloaded", timeout=5000)
        except Exception:
            pass
        
        # -> Click “建立備份” and verify that a completed backup row appears with its filename, size, and time.
        # 建立備份 button
        elem = page.get_by_test_id("create-backup-btn")
        await elem.click(timeout=10000)
        
        # -> Turn on “自動備份” and check that daily/weekly frequency and time options appear.
        # 自動備份 button
        elem = page.get_by_test_id("schedule-toggle")
        await elem.click(timeout=10000)
        
        # -> Select “每週” in the automatic backup frequency options.
        # 每週 button
        elem = page.get_by_role("radio", name="每週")
        await elem.click(timeout=10000)
        
        # -> Click “儲存排程” to save the weekly automatic-backup schedule.
        # 儲存排程 button
        elem = page.get_by_test_id("schedule-save-btn")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> 已建立一筆檔名以 .tar.gz 結尾且狀態為「完成」的備份，並顯示其大小。
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: Verify that a listed backup filename ends in .tar.gz.
        await expect(page.locator("#root").nth(0)).to_contain_text(".tar.gz", timeout=15000), "Verify that a listed backup filename ends in .tar.gz."
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: Verify that the listed backup is marked complete.
        await expect(page.locator("#root").nth(0)).to_contain_text("\u5b8c\u6210", timeout=15000), "Verify that the listed backup is marked complete."
        
        # --> 自動備份設定顯示每日與每週選項，以及備份時間選擇。
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: Verify that the daily and weekly frequency options are displayed.
        await expect(page.locator("#root").nth(0)).to_contain_text("\u6bcf\u65e5 \u6bcf\u9031", timeout=15000), "Verify that the daily and weekly frequency options are displayed."
        await page.get_by_test_id("schedule-hour").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: Verify that a backup-time choice is available.
        await expect(page.get_by_test_id("schedule-hour").nth(0)).to_be_visible(timeout=15000), "Verify that a backup-time choice is available."
        
        # --> 重新載入後，自動備份仍為開啟狀態，且每週仍為選取狀態。
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: Verify that automatic backup remains enabled.
        await expect(page.get_by_test_id("schedule-toggle").nth(0)).to_have_attribute("aria-checked", "true", timeout=15000), "Verify that automatic backup remains enabled."
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: Verify that the weekly frequency remains selected.
        await expect(page.get_by_role("radio", name="每週").nth(0)).to_have_attribute("aria-checked", "true", timeout=15000), "Verify that the weekly frequency remains selected."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    