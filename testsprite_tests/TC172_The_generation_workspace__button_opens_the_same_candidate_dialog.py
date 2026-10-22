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
        
        # -> Open the generation activity page at `/activity?view=generation`.
        await page.goto("http://localhost:8090/activity?view=generation")
        try:
            await page.wait_for_load_state("domcontentloaded", timeout=5000)
        except Exception:
            pass
        
        # -> Click “批次生成字幕” and verify the “產生字幕” dialog and candidate count.
        # 批次生成字幕 button
        elem = page.get_by_test_id("workspace-launch")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> 「產生字幕」對話框已顯示，並列出 8 部候選項目。
        await page.locator("xpath=/html/body/div[4]").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: 確認「產生字幕」對話框可見。
        await expect(page.locator("xpath=/html/body/div[4]").nth(0)).to_be_visible(timeout=15000), "\u78ba\u8a8d\u300c\u7522\u751f\u5b57\u5e55\u300d\u5c0d\u8a71\u6846\u53ef\u898b\u3002"
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: 確認對話框列出 8 部候選項目。
        await expect(page.locator("xpath=/html/body/div[4]").nth(0)).to_contain_text("\u5019\u9078 8 \u90e8", timeout=15000), "\u78ba\u8a8d\u5c0d\u8a71\u6846\u5217\u51fa 8 \u90e8\u5019\u9078\u9805\u76ee\u3002"
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    