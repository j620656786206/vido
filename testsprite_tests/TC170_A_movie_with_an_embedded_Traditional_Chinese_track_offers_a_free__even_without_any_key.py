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
        
        # -> Open the 教父 movie page using the specified movie route.
        await page.goto("http://localhost:8090/media/movie/seed-mv-001")
        try:
            await page.wait_for_load_state("domcontentloaded", timeout=5000)
        except Exception:
            pass
        
        # -> Open the '管理字幕' dialog on the 教父 movie page.
        # 管理字幕 button
        elem = page.get_by_test_id("action-manage-subtitle")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> 「管理字幕 — 教父」對話框顯示中。
        await page.locator("xpath=/html/body/div[4]").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: 確認「管理字幕 — 教父」對話框可見。
        await expect(page.locator("xpath=/html/body/div[4]").nth(0)).to_be_visible(timeout=15000), "\u78ba\u8a8d\u300c\u7ba1\u7406\u5b57\u5e55 \u2014 \u6559\u7236\u300d\u5c0d\u8a71\u6846\u53ef\u898b\u3002"
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: 確認對話框標題為「管理字幕 — 教父」。
        await expect(page.locator("xpath=/html/body/div[4]").nth(0)).to_contain_text("\u7ba1\u7406\u5b57\u5e55 \u2014 \u6559\u7236", timeout=15000), "\u78ba\u8a8d\u5c0d\u8a71\u6846\u6a19\u984c\u70ba\u300c\u7ba1\u7406\u5b57\u5e55 \u2014 \u6559\u7236\u300d\u3002"
        
        # --> 「生成字幕」按鈕顯示約 $0.00。
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: 確認「生成字幕」按鈕顯示約 $0.00。
        await expect(page.locator("xpath=/html/body/div[4]/div[3]/section[2]/button").nth(0)).to_contain_text("\u2248 $0.00", timeout=15000), "\u78ba\u8a8d\u300c\u751f\u6210\u5b57\u5e55\u300d\u6309\u9215\u986f\u793a\u7d04 $0.00\u3002"
        
        # --> 對話框說明片內中文字幕可直接套用。
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: 確認說明指出片內中文字幕可直接套用。
        await expect(page.locator("xpath=/html/body/div[4]").nth(0)).to_contain_text("\u7247\u5167\u4e2d\u6587\u5b57\u5e55\u53ef\u76f4\u63a5\u5957\u7528", timeout=15000), "\u78ba\u8a8d\u8aaa\u660e\u6307\u51fa\u7247\u5167\u4e2d\u6587\u5b57\u5e55\u53ef\u76f4\u63a5\u5957\u7528\u3002"
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    