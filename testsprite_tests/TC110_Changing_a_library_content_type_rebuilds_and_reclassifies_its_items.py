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
        
        # -> Open the Scanner settings page (navigate to /settings/scanner).
        await page.goto("http://localhost:8090/settings/scanner")
        try:
            await page.wait_for_load_state("domcontentloaded", timeout=5000)
        except Exception:
            pass
        
        # -> Open the action menu for the library card labeled '影集庫' (the TV library) by clicking its three-dot/action button.
        # button
        elem = page.locator('xpath=/html/body/div/div/div/div[2]/main/div/div/div/div/div/div/div/div/div[2]/div/div[2]/button')
        await elem.click(timeout=10000)
        
        # -> Click the '編輯' button in the 影集庫 action menu to open the library edit dialog.
        # 編輯 button
        elem = page.get_by_role('button', name='編輯', exact=True)
        await elem.click(timeout=10000)
        
        # -> Open the library '類型' (content type) dropdown in the edit dialog so the '電影' option can be selected.
        # 電影 影集 dropdown
        elem = page.get_by_test_id('library-type-select')
        await elem.click(timeout=10000)
        
        # -> Change the library '類型' dropdown to '電影' in the edit dialog and wait for the UI to update.
        # 電影 影集 dropdown
        elem = page.locator("xpath=/html/body/div/div/div/div[2]/main/div/div/div/div/div/div/div/div[2]/div/div[2]/div[2]/select").nth(0)
        await elem.wait_for(state="visible", timeout=10000)
        await elem.select_option("")
        
        # -> Click the '儲存變更' button to save the library change (this should trigger the rebuild/purge and automatic rescan), then wait ~15 seconds and reload the settings page.
        # 儲存變更 button
        elem = page.get_by_test_id('library-save-button')
        await elem.click(timeout=10000)
        
        # -> Click the '儲存變更' button to save the library change (this should trigger the rebuild/purge and automatic rescan), then wait ~15 seconds and reload the settings page.
        await page.goto("http://localhost:8090/settings/scanner")
        try:
            await page.wait_for_load_state("domcontentloaded", timeout=5000)
        except Exception:
            pass
        
        # --> Assertions to verify final state
        
        # --> After saving the library change and reloading, the sidebar shows the movie count increased and the series count decreased (電影 = 27, 影集 = 0).
        # Assert-outcome: passed
        # Assert: Sidebar shows the movie link with the updated movie count (27).
        await expect(page.locator("xpath=/html/body/div/div/div/div[1]/aside/nav/div[2]/div[2]/a[1]").nth(0)).to_have_text("\u96fb\u5f71\n27", timeout=15000), "Sidebar shows the movie link with the updated movie count (27)."
        # Assert-outcome: passed
        # Assert: Sidebar shows the series link with the updated series count (0).
        await expect(page.locator("xpath=/html/body/div/div/div/div[1]/aside/nav/div[2]/div[2]/a[2]").nth(0)).to_have_text("\u5f71\u96c6\n0", timeout=15000), "Sidebar shows the series link with the updated series count (0)."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    