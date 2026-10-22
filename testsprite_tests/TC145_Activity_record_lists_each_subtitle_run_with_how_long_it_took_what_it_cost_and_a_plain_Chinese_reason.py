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
        
        # -> Open the “活動” page to inspect the seeded subtitle run records.
        # 活動 link
        elem = page.get_by_test_id("app-sidebar").get_by_test_id("nav-activity")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> 奧本海默的活動記錄顯示字幕完成。
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: 確認活動記錄列出奧本海默字幕完成。
        await expect(page.locator("#root").nth(0)).to_contain_text("\u5b57\u5e55\u5b8c\u6210 \u5967\u672c\u6d77\u9ed8", timeout=15000), "\u78ba\u8a8d\u6d3b\u52d5\u8a18\u9304\u5217\u51fa\u5967\u672c\u6d77\u9ed8\u5b57\u5e55\u5b8c\u6210\u3002"
        
        # --> 怪奇物語 S01E01 的活動記錄顯示字幕完成。
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: 確認活動記錄列出怪奇物語 S01E01 字幕完成。
        await expect(page.locator("#root").nth(0)).to_contain_text("\u5b57\u5e55\u5b8c\u6210 \u602a\u5947\u7269\u8a9e S01E01", timeout=15000), "\u78ba\u8a8d\u6d3b\u52d5\u8a18\u9304\u5217\u51fa\u602a\u5947\u7269\u8a9e S01E01 \u5b57\u5e55\u5b8c\u6210\u3002"
        
        # --> 侏羅紀公園的活動記錄顯示字幕沒做成，原因是影片所在的資料夾寫不進字幕。
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: 確認侏羅紀公園記錄顯示字幕失敗及其原因。
        await expect(page.locator("#root").nth(0)).to_contain_text("\u5b57\u5e55\u6c92\u505a\u6210 \u4f8f\u7f85\u7d00\u516c\u5712 \u00b7 \u5f71\u7247\u6240\u5728\u7684\u8cc7\u6599\u593e\u5beb\u4e0d\u9032\u5b57\u5e55", timeout=15000), "\u78ba\u8a8d\u4f8f\u7f85\u7d00\u516c\u5712\u8a18\u9304\u986f\u793a\u5b57\u5e55\u5931\u6557\u53ca\u5176\u539f\u56e0\u3002"
        
        # --> 星際效應的活動記錄顯示字幕沒做成，原因是處理逾時。
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: 確認星際效應記錄顯示字幕失敗及處理逾時。
        await expect(page.locator("#root").nth(0)).to_contain_text("\u5b57\u5e55\u6c92\u505a\u6210 \u661f\u969b\u6548\u61c9 \u00b7 \u8655\u7406\u903e\u6642", timeout=15000), "\u78ba\u8a8d\u661f\u969b\u6548\u61c9\u8a18\u9304\u986f\u793a\u5b57\u5e55\u5931\u6557\u53ca\u8655\u7406\u903e\u6642\u3002"
        
        # --> 神隱少女的活動記錄顯示字幕沒做成，原因是 AI 帳號沒有額度。
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: 確認神隱少女記錄顯示字幕失敗及其原因。
        await expect(page.locator("#root").nth(0)).to_contain_text("\u5b57\u5e55\u6c92\u505a\u6210 \u795e\u96b1\u5c11\u5973 \u00b7 AI \u5e33\u865f\u6c92\u6709\u984d\u5ea6", timeout=15000), "\u78ba\u8a8d\u795e\u96b1\u5c11\u5973\u8a18\u9304\u986f\u793a\u5b57\u5e55\u5931\u6557\u53ca\u5176\u539f\u56e0\u3002"
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    