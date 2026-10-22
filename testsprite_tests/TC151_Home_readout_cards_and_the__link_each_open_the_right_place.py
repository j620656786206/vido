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
        
        # -> Open the 繁中字幕 readout and verify that the library shows the 產生字幕 dialog.
        # 繁中字幕 · 產生字幕 4/18 link
        elem = page.get_by_test_id("readout-coverage")
        await elem.click(timeout=10000)
        
        # -> Return to Home to check whether「今天處理」opens the activity hub.
        await page.goto("http://localhost:8090/")
        try:
            await page.wait_for_load_state("domcontentloaded", timeout=5000)
        except Exception:
            pass
        
        # -> Click the 今天處理 readout and verify that the activity hub opens.
        # 今天處理 2 部 link
        elem = page.get_by_test_id("readout-processed")
        await elem.click(timeout=10000)
        
        # -> Return to Home using the「首頁」navigation link to check the「需要注意」readout.
        # 首頁 link
        elem = page.get_by_test_id("app-sidebar").get_by_test_id("nav-home")
        await elem.click(timeout=10000)
        
        # -> Click the「需要注意」readout card and verify that the activity hub opens.
        # 需要注意 3 部失敗 · 本月 $0.6 link
        elem = page.get_by_test_id("readout-attention")
        await elem.click(timeout=10000)
        
        # -> Return to Home using the「首頁」navigation link to check the「進行中」readout.
        # 首頁 link
        elem = page.get_by_test_id("app-sidebar").get_by_test_id("nav-home")
        await elem.click(timeout=10000)
        
        # -> Click the「進行中」readout and verify that it opens the activity hub.
        # 進行中 0 個任務 link
        elem = page.get_by_test_id("readout-inflight")
        await elem.click(timeout=10000)
        
        # -> Return Home and verify that the「整理中 · 3」link next to 最近新增 opens the activity hub.
        # 首頁 link
        elem = page.get_by_test_id("app-sidebar").get_by_test_id("nav-home")
        await elem.click(timeout=10000)
        
        # -> Click the 「整理中」 count link beside 最近新增 and verify that it opens the activity hub.
        # 整理中 · 3 link
        elem = page.get_by_test_id("home-recent-progress")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> The 繁中字幕 readout opened the library.
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: The 繁中字幕 readout should open the library.
        await expect(page).to_have_url(re.compile("/library"), timeout=15000), "The \u7e41\u4e2d\u5b57\u5e55 readout should open the library."
        
        # --> The 今天處理 readout opened the activity hub.
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: The 今天處理 readout should open the activity hub.
        await expect(page).to_have_url(re.compile("/activity"), timeout=15000), "The \u4eca\u5929\u8655\u7406 readout should open the activity hub."
        
        # --> The 需要注意 readout opened the activity hub.
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: The 需要注意 readout should open the activity hub.
        await expect(page).to_have_url(re.compile("/activity"), timeout=15000), "The \u9700\u8981\u6ce8\u610f readout should open the activity hub."
        
        # --> The 進行中 readout opened the activity hub.
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: The 進行中 readout should open the activity hub.
        await expect(page).to_have_url(re.compile("/activity"), timeout=15000), "The \u9032\u884c\u4e2d readout should open the activity hub."
        
        # --> The 整理中 count link opened the activity hub.
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: The 整理中 count link should open the activity hub.
        await expect(page).to_have_url(re.compile("/activity"), timeout=15000), "The \u6574\u7406\u4e2d count link should open the activity hub."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    