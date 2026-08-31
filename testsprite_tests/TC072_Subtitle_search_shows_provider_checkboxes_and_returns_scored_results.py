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
        # -> Navigate to the application's Library page by opening /library and wait for the Library UI to load.
        await page.goto("http://localhost:8090/library")
        try:
            await page.wait_for_load_state("domcontentloaded", timeout=5000)
        except Exception:
            pass
        
        # -> Open the context menu for the first media card 'Unknown.Show.S01'.
        # U 整理中 Unknown.Show.S01 link
        elem = page.get_by_test_id('poster-v2-seed-sr-101')
        await elem.click(timeout=10000)
        
        # -> Click the '管理字幕' (Manage Subtitles) button to open the subtitle search / management dialog.
        # 管理字幕 button
        elem = page.get_by_test_id('action-manage-subtitle')
        await elem.click(timeout=10000)
        
        # -> Click the '搜尋線上字幕（成功率低）' (Search online subtitles) button in the Manage Subtitles dialog.
        # 搜尋線上字幕（成功率低） button
        elem = page.get_by_test_id('toggle-fetch')
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> The Manage Subtitles dialog does not show the 'Assrt' provider option.
        # Assert-outcome: failed
        # Assert: Expected the Manage Subtitles dialog to show the 'Assrt' provider label.
        await expect(page.locator("xpath=/html/body/div[3]").nth(0)).to_contain_text("Assrt", timeout=15000), "Expected the Manage Subtitles dialog to show the 'Assrt' provider label."
        
        # --> The Manage Subtitles dialog does not show the 'OpenSubtitles' provider option.
        # Assert-outcome: failed
        # Assert: Expected the Manage Subtitles dialog to show the 'OpenSubtitles' provider label.
        await expect(page.locator("xpath=/html/body/div[3]").nth(0)).to_contain_text("OpenSubtitles", timeout=15000), "Expected the Manage Subtitles dialog to show the 'OpenSubtitles' provider label."
        
        # --> The Manage Subtitles dialog does not show the 'Zimuku' provider option.
        # Assert-outcome: failed
        # Assert: Expected the Manage Subtitles dialog to show the 'Zimuku' provider label.
        await expect(page.locator("xpath=/html/body/div[3]").nth(0)).to_contain_text("Zimuku", timeout=15000), "Expected the Manage Subtitles dialog to show the 'Zimuku' provider label."
        
        # --> Searching did not produce a subtitle results table; the dialog shows a no-results message instead.
        # Assert-outcome: failed
        # Assert: Expected the Manage Subtitles dialog to display the subtitle results table after searching.
        await expect(page.locator("xpath=/html/body/div[3]").nth(0)).to_contain_text("\u5c1a\u7121\u7d50\u679c \u2014 \u7dda\u4e0a\u4f86\u6e90\u6210\u529f\u7387\u4f4e\uff0c\u5efa\u8b70\u6539\u7528\u751f\u6210\u5b57\u5e55", timeout=15000), "Expected the Manage Subtitles dialog to display the subtitle results table after searching."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    