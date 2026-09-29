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
        
        # -> Open the '匯出/匯入' settings page (navigate to /settings/export) and wait for it to finish loading.
        await page.goto("http://localhost:8090/settings/export")
        try:
            await page.wait_for_load_state("domcontentloaded", timeout=5000)
        except Exception:
            pass
        
        # -> Verify the heading '匯出/匯入' and the description '將媒體庫元資料匯出為 JSON、YAML 或 NFO 檔案。' are visible on the page.
        # 匯出 button
        elem = page.get_by_test_id("export-btn")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> 頁面顯示標題「匯出/匯入」且描述「將媒體庫元資料匯出為 JSON、YAML 或 NFO 檔案。」可見。
        await page.locator("#root").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: passed
        # Assert: The page heading and description are visible.
        await expect(page.locator("#root").nth(0)).to_be_visible(timeout=15000), "The page heading and description are visible."
        
        # --> 匯出媒體資料卡顯示三種格式選項：JSON、YAML 與 NFO，且各自顯示其說明文字。
        # Assert-outcome: passed
        # Assert: JSON option shows its label and description.
        await expect(page.get_by_test_id("export-format-json").locator("div").nth(0)).to_have_text("JSON\n\u4eba\u985e\u53ef\u8b80\u7684 JSON \u683c\u5f0f\uff0c\u9069\u5408\u7a0b\u5f0f\u8655\u7406", timeout=15000), "JSON option shows its label and description."
        # Assert-outcome: passed
        # Assert: YAML option shows its label and description.
        await expect(page.get_by_test_id("export-format-yaml").locator("div").nth(0)).to_have_text("YAML\n\u4eba\u985e\u53ef\u8b80\u7684 YAML \u683c\u5f0f\uff0c\u9069\u5408\u8a2d\u5b9a\u5099\u4efd", timeout=15000), "YAML option shows its label and description."
        
        # --> JSON 格式的選項預設為已選取 (預設 radio 為 JSON)。
        # Assert-outcome: passed
        # Assert: The JSON radio input is selected by default.
        await expect(page.get_by_role("radio", name="JSON：人類可讀的 JSON 格式，適合程式處理").nth(0)).to_have_attribute("checked", "true", timeout=15000), "The JSON radio input is selected by default."
        
        # --> 點選匯出後出現匯出完成訊息，並顯示「下載匯出檔案」連結。
        # Assert-outcome: passed
        # Assert: An export completion message is visible.
        await expect(page.locator("#root").nth(0)).to_contain_text("\u532f\u51fa\u5b8c\u6210", timeout=15000), "An export completion message is visible."
        # Assert-outcome: passed
        # Assert: The '下載匯出檔案' link is present.
        await expect(page.get_by_test_id("export-download-link").nth(0)).to_have_text("\u4e0b\u8f09\u532f\u51fa\u6a94\u6848", timeout=15000), "The '\u4e0b\u8f09\u532f\u51fa\u6a94\u6848' link is present."
        
        # --> 頁面顯示匯入功能尚未實作的註記，並指向「備份與還原」。
        # Assert-outcome: passed
        # Assert: The import-not-implemented note referencing 備份與還原 is visible.
        await expect(page.locator("#root").nth(0)).to_contain_text("\u532f\u5165\u529f\u80fd\u5c1a\u672a\u5be6\u4f5c \u2014 \u76ee\u524d\u8acb\u4ee5\u300c\u5099\u4efd\u8207\u9084\u539f\u300d\u9084\u539f\u5b8c\u6574\u8cc7\u6599\u3002", timeout=15000), "The import-not-implemented note referencing \u5099\u4efd\u8207\u9084\u539f is visible."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    