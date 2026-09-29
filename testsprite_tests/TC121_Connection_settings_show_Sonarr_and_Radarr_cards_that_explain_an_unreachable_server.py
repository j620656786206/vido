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
        
        # -> Open the '設定' (Settings) → '連線' (Connection) page and load the connection cards.
        await page.goto("http://localhost:8090/settings/connection")
        try:
            await page.wait_for_load_state("domcontentloaded", timeout=5000)
        except Exception:
            pass
        
        # -> Fill the Sonarr '網址' field with 'http://127.0.0.1:1' and the Sonarr 'API 金鑰' field with 'test-api-key', then click the Sonarr '測試連線' button.
        # http://192.168.1.100:8989 text field
        elem = page.get_by_test_id("arr-form-sonarr").get_by_role("textbox", name="網址")
        await elem.wait_for(state="visible", timeout=10000)
        await elem.fill("http://127.0.0.1:1")
        
        # -> Fill the Sonarr '網址' field with 'http://127.0.0.1:1' and the Sonarr 'API 金鑰' field with 'test-api-key', then click the Sonarr '測試連線' button.
        # 貼上 Sonarr 的 API 金鑰 password field
        elem = page.get_by_test_id("arr-form-sonarr").get_by_role("textbox", name="API 金鑰")
        await elem.wait_for(state="visible", timeout=10000)
        await elem.fill("test-api-key")
        
        # -> Fill the Sonarr '網址' field with 'http://127.0.0.1:1' and the Sonarr 'API 金鑰' field with 'test-api-key', then click the Sonarr '測試連線' button.
        # 測試連線 button
        elem = page.get_by_test_id("arr-form-sonarr").get_by_role("button", name="測試連線")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> The Sonarr and Radarr connection cards are present with subtitles '影集的搜尋與匯入' and '電影的搜尋與匯入'.
        # Assert-outcome: passed
        # Assert: The Sonarr card subtitle '影集的搜尋與匯入' is visible.
        await expect(page.locator("#root").nth(0)).to_contain_text("\u5f71\u96c6\u7684\u641c\u5c0b\u8207\u532f\u5165", timeout=15000), "The Sonarr card subtitle '\u5f71\u96c6\u7684\u641c\u5c0b\u8207\u532f\u5165' is visible."
        # Assert-outcome: passed
        # Assert: The Radarr card subtitle '電影的搜尋與匯入' is visible.
        await expect(page.locator("#root").nth(0)).to_contain_text("\u96fb\u5f71\u7684\u641c\u5c0b\u8207\u532f\u5165", timeout=15000), "The Radarr card subtitle '\u96fb\u5f71\u7684\u641c\u5c0b\u8207\u532f\u5165' is visible."
        
        # --> Both the Sonarr and Radarr cards show the status text '未設定'.
        # Assert-outcome: passed
        # Assert: The page displays the status text '未設定' for unconfigured connection cards.
        await expect(page.locator("#root").nth(0)).to_contain_text("\u672a\u8a2d\u5b9a", timeout=15000), "The page displays the status text '\u672a\u8a2d\u5b9a' for unconfigured connection cards."
        
        # --> Testing the configured Sonarr address produced a Chinese failure banner stating the address could not be reached.
        # Assert-outcome: passed
        # Assert: A Sonarr connection failure message explaining the address could not be reached is shown.
        await expect(page.locator("#root").nth(0)).to_contain_text("\u9023\u4e0d\u5230\u9019\u500b\u7db2\u5740\u3002\u78ba\u8a8d Sonarr \u6709\u5728\u57f7\u884c\uff0c\u7db2\u5740\u548c\u9023\u63a5\u57e0\u90fd\u6b63\u78ba\u3002", timeout=15000), "A Sonarr connection failure message explaining the address could not be reached is shown."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    