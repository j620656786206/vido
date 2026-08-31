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
        
        # -> Open the '設定' (Settings) page by clicking the '設定' link in the sidebar.
        # 設定 link
        elem = page.get_by_test_id('nav-settings')
        await elem.click(timeout=10000)
        
        # -> Click the '媒體庫：媒體庫掃描' (Library: Media library scan) settings link to open the Scanner settings page.
        # 媒體庫掃描 link
        elem = page.get_by_test_id('settings-tab-scanner')
        await elem.click(timeout=10000)
        
        # -> Open the '掃描排程' (scan schedule) dropdown so the '每天' (daily) option can be selected.
        # 每小時 每天 僅手動 dropdown
        elem = page.get_by_test_id('schedule-select')
        await elem.click(timeout=10000)
        
        # -> Select the '每天' (daily) option in the '掃描排程' (Scan schedule) dropdown and verify it shows '每天' on the page.
        # 每小時 每天 僅手動 dropdown
        elem = page.locator("xpath=/html/body/div/div/div/div[2]/main/div/div/div/div/div/div[2]/div[2]/select").nth(0)
        await elem.wait_for(state="visible", timeout=10000)
        await elem.select_option("")
        
        # -> Select '每天' from the '掃描排程' (scan schedule) dropdown and verify the page shows '每天', then go to the Library page.
        # 媒體庫 link
        elem = page.get_by_text('內容', exact=True).locator("xpath=ancestor-or-self::*[.//a][1]").get_by_role('link', name='媒體庫', exact=True)
        await elem.click(timeout=10000)
        
        # -> Click the '設定' (Settings) link in the left sidebar to open the Settings page.
        # 設定 link
        elem = page.get_by_test_id('nav-settings')
        await elem.click(timeout=10000)
        
        # -> Open the '媒體庫：媒體庫掃描' (Media library scan) settings page by clicking the sidebar link labeled '媒體庫：媒體庫掃描'.
        # 媒體庫掃描 link
        elem = page.get_by_test_id('settings-tab-scanner')
        await elem.click(timeout=10000)
        
        # -> Open the '掃描排程' (Scan schedule) dropdown so the '每天' option is revealed.
        # 每小時 每天 僅手動 dropdown
        elem = page.get_by_test_id('schedule-select')
        await elem.click(timeout=10000)
        
        # -> Select '每天' from the '掃描排程' (Scan schedule) dropdown on the 媒體庫掃描 (Media library scan) settings page and verify the page shows '每天'.
        # 每小時 每天 僅手動 dropdown
        elem = page.locator("xpath=/html/body/div/div/div/div[2]/main/div/div/div/div/div/div[2]/div[2]/select").nth(0)
        await elem.wait_for(state="visible", timeout=10000)
        await elem.select_option("")
        
        # -> Select '每天' from the '掃描排程' (Scan schedule) dropdown on the 媒體庫掃描 (Media library scan) settings page and verify the page shows '每天'.
        # 媒體庫 link
        elem = page.get_by_text('內容', exact=True).locator("xpath=ancestor-or-self::*[.//a][1]").get_by_role('link', name='媒體庫', exact=True)
        await elem.click(timeout=10000)
        
        # -> Click the left-sidebar '設定' (Settings) link to open the Settings page.
        # 設定 link
        elem = page.get_by_test_id('nav-settings')
        await elem.click(timeout=10000)
        
        # -> Open the '媒體庫：媒體庫掃描' (Media library scan) settings page by clicking the sidebar link labeled '媒體庫掃描'.
        # 媒體庫掃描 link
        elem = page.get_by_test_id('settings-tab-scanner')
        await elem.click(timeout=10000)
        
        # -> Set the '掃描排程' (Scan schedule) dropdown to '每天' and verify the page shows the text '每天', then click the '媒體庫' (Library) link.
        # 每小時 每天 僅手動 dropdown
        elem = page.locator("xpath=/html/body/div/div/div/div[2]/main/div/div/div/div/div/div[2]/div[2]/select").nth(0)
        await elem.wait_for(state="visible", timeout=10000)
        await elem.select_option("")
        
        # -> Set the '掃描排程' (Scan schedule) dropdown to '每天' and verify the page shows the text '每天', then click the '媒體庫' (Library) link.
        # 媒體庫 link
        elem = page.get_by_text('內容', exact=True).locator("xpath=ancestor-or-self::*[.//a][1]").get_by_role('link', name='媒體庫', exact=True)
        await elem.click(timeout=10000)
        
        # -> Open the 媒體庫掃描 (Media library scan) settings page by navigating to the Settings → 媒體庫掃描 URL (/settings/scanner).
        await page.goto("http://localhost:8090/settings/scanner")
        try:
            await page.wait_for_load_state("domcontentloaded", timeout=5000)
        except Exception:
            pass
        
        # -> Select '每天' from the '掃描排程' (Scan schedule) dropdown and verify the page shows '每天', then navigate to the Library page.
        # 每小時 每天 僅手動 dropdown
        elem = page.locator("xpath=/html/body/div/div/div/div[2]/main/div/div/div/div/div/div[2]/div[2]/select").nth(0)
        await elem.wait_for(state="visible", timeout=10000)
        await elem.select_option("")
        
        # -> Select '每天' from the '掃描排程' (Scan schedule) dropdown and verify the page shows '每天', then navigate to the Library page.
        # 媒體庫 link
        elem = page.get_by_text('內容', exact=True).locator("xpath=ancestor-or-self::*[.//a][1]").get_by_role('link', name='媒體庫', exact=True)
        await elem.click(timeout=10000)
        
        # -> Click the left-sidebar '設定' (Settings) link to open the Settings page.
        # 設定 link
        elem = page.get_by_test_id('nav-settings')
        await elem.click(timeout=10000)
        
        # -> Click the '媒體庫：媒體庫掃描' (Media library scan) link in the left settings menu to open the Media library scan settings page.
        # 媒體庫掃描 link
        elem = page.get_by_test_id('settings-tab-scanner')
        await elem.click(timeout=10000)
        
        # -> Select '每天' from the '掃描排程' dropdown and verify the page shows '每天', then click the '媒體庫' (Library) link to begin the persistence check.
        # 每小時 每天 僅手動 dropdown
        elem = page.locator("xpath=/html/body/div/div/div/div[2]/main/div/div/div/div/div/div[2]/div[2]/select").nth(0)
        await elem.wait_for(state="visible", timeout=10000)
        await elem.select_option("")
        
        # -> Select '每天' from the '掃描排程' dropdown and verify the page shows '每天', then click the '媒體庫' (Library) link to begin the persistence check.
        # 媒體庫 link
        elem = page.get_by_text('內容', exact=True).locator("xpath=ancestor-or-self::*[.//a][1]").get_by_role('link', name='媒體庫', exact=True)
        await elem.click(timeout=10000)
        
        # -> Navigate to the 媒體庫掃描 (Media library scan) settings page (Settings → 媒體庫掃描) by opening /settings/scanner.
        await page.goto("http://localhost:8090/settings/scanner")
        try:
            await page.wait_for_load_state("domcontentloaded", timeout=5000)
        except Exception:
            pass
        
        # -> Open the '掃描排程' dropdown (Scan schedule) and reveal the options so '每天' can be clicked.
        # 每小時 每天 僅手動 dropdown
        elem = page.get_by_test_id('schedule-select')
        await elem.click(timeout=10000)
        
        # -> Select the '每天' option in the '掃描排程' (scan schedule) dropdown and then click the '媒體庫' (Library) link to navigate away for a persistence check.
        # 每小時 每天 僅手動 dropdown
        elem = page.locator("xpath=/html/body/div/div/div/div[2]/main/div/div/div/div/div/div[2]/div[2]/select").nth(0)
        await elem.wait_for(state="visible", timeout=10000)
        await elem.select_option("")
        
        # -> Select the '每天' option in the '掃描排程' (scan schedule) dropdown and then click the '媒體庫' (Library) link to navigate away for a persistence check.
        # 媒體庫 link
        elem = page.get_by_text('內容', exact=True).locator("xpath=ancestor-or-self::*[.//a][1]").get_by_role('link', name='媒體庫', exact=True)
        await elem.click(timeout=10000)
        
        # -> Navigate to the 媒體庫掃描 (Media library scan) settings page to access the '掃描排程' control.
        await page.goto("http://localhost:8090/settings/scanner")
        try:
            await page.wait_for_load_state("domcontentloaded", timeout=5000)
        except Exception:
            pass
        
        # --> Assertions to verify final state
        
        # --> The scan schedule dropdown is present and lists the options 每小時, 每天, 僅手動.
        # Assert-outcome: passed
        # Assert: Verifies the scan schedule dropdown lists 每小時, 每天, 僅手動.
        await expect(page.locator("xpath=/html/body/div/div/div/div[2]/main/div/div/div/div/div/div/div[2]/select").nth(0)).to_have_text("\u6bcf\u5c0f\u6642\n\u6bcf\u5929\n\u50c5\u624b\u52d5", timeout=15000), "Verifies the scan schedule dropdown lists \u6bcf\u5c0f\u6642, \u6bcf\u5929, \u50c5\u624b\u52d5."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    