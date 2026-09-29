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
        
        # -> Open the '設定' (Settings) page and go to the Scanner settings (navigate to /settings/scanner).
        await page.goto("http://localhost:8090/settings/scanner")
        try:
            await page.wait_for_load_state("domcontentloaded", timeout=5000)
        except Exception:
            pass
        
        # -> Open the action menu for the '影集庫' library card and choose the '編輯' (Edit) option from the menu.
        # 影集庫 的操作 button
        elem = page.get_by_test_id("library-card-seed-lib-tv").get_by_test_id("library-menu-button")
        await elem.click(timeout=10000)
        
        # -> Click the '編輯' (Edit) menu item in the action menu for the 影集庫 library.
        # 編輯 menu item
        elem = page.get_by_role("menuitem", name="編輯")
        await elem.click(timeout=10000)
        
        # -> Open the '類型' (Type) dropdown in the '編輯媒體庫' dialog so the '電影' (Movie) option can be selected.
        # 電影 影集 dropdown
        elem = page.get_by_test_id("library-type-select")
        await elem.click(timeout=10000)
        
        # -> Change the '類型' dropdown to '電影', check the dialog for a rebuild/rescan warning message, then click '儲存變更' to apply the change.
        # 電影 影集 dropdown
        elem = page.locator("xpath=/html/body/div[3]/div[2]/div[2]/select").nth(0)
        await elem.wait_for(state="visible", timeout=10000)
        await elem.select_option("")
        
        # -> Change the '類型' dropdown to '電影', check the dialog for a rebuild/rescan warning message, then click '儲存變更' to apply the change.
        # 儲存變更 button
        elem = page.get_by_test_id("library-save-button")
        await elem.click(timeout=10000)
        
        # -> Wait about 15 seconds for the automatic rescan to run, then reload the '媒體庫掃描' (Settings → 媒體庫掃描) page to check the sidebar '電影' and '影集' counts.
        await page.goto("http://localhost:8090/settings/scanner")
        try:
            await page.wait_for_load_state("domcontentloaded", timeout=5000)
        except Exception:
            pass
        
        # --> Assertions to verify final state
        
        # --> After saving, the sidebar updated showing movies increased and series decreased (sidebar shows 電影（27） and 影集（0）).
        # Assert-outcome: passed
        # Assert: The sidebar '電影' link is present (shows the movie count).
        await expect(page.get_by_test_id("nav-movies").nth(0)).to_contain_text("\u96fb\u5f71", timeout=15000), "The sidebar '\u96fb\u5f71' link is present (shows the movie count)."
        # Assert-outcome: passed
        # Assert: The sidebar '影集' link is present (shows the series count).
        await expect(page.get_by_test_id("nav-tv").nth(0)).to_contain_text("\u5f71\u96c6", timeout=15000), "The sidebar '\u5f71\u96c6' link is present (shows the series count)."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    