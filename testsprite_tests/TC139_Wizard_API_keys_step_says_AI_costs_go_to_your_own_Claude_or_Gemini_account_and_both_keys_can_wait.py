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
        
        # -> Click “下一步” to continue from the welcome step.
        # 下一步 button
        elem = page.get_by_test_id("next-button")
        await elem.click(timeout=10000)
        
        # -> Click “跳過” on the qBittorrent connection step to continue the setup wizard.
        # 跳過 button
        elem = page.get_by_test_id("skip-button")
        await elem.click(timeout=10000)
        
        # -> Enter the displayed media folder path so the wizard can proceed to the API keys step.
        # /media/movies text field
        elem = page.get_by_test_id("library-path-0")
        await elem.wait_for(state="visible", timeout=10000)
        await elem.fill("/media/movies")
        
        # -> Click “下一步” on the media library step to open the API keys page.
        # 下一步 button
        elem = page.get_by_test_id("next-button")
        await elem.click(timeout=10000)
        
        # -> Replace the invalid media folder path with the example path shown in the validation message, then check whether the wizard accepts it.
        # /media/movies text field
        elem = page.get_by_test_id("library-path-0")
        await elem.wait_for(state="visible", timeout=10000)
        await elem.fill("/Users/alexyu/projects/personal/vido/.vido-test-env/media/movies")
        
        # -> Continue from the media-library step using the example folder path and check whether the API keys step appears.
        # 下一步 button
        elem = page.get_by_test_id("next-button")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> The API keys page says Vido does not charge and Claude or Gemini costs are billed to the user's account.
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: The page explains that Vido does not charge and Claude or Gemini costs go to the user's account.
        await expect(page.locator("#root").nth(0)).to_contain_text("Vido \u4e0d\u6536\u9322\u3002AI \u5b57\u5e55\u7528\u4f60\u81ea\u5df1\u7684\u91d1\u9470\uff0c\u8cbb\u7528\u76f4\u63a5\u8a18\u5728\u4f60\u7684 Claude \u6216 Gemini \u5e33\u865f", timeout=15000), "The page explains that Vido does not charge and Claude or Gemini costs go to the user's account."
        
        # --> The Gemini API key is labeled optional.
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: The Gemini key field is labeled optional.
        await expect(page.locator("#root").nth(0)).to_contain_text("Gemini \u91d1\u9470\uff08\u9078\u586b\uff09", timeout=15000), "The Gemini key field is labeled optional."
        
        # --> With the key fields left empty, the page warns that AI subtitle translation will not be available.
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: The empty-key warning explains that AI subtitle translation will not be available.
        await expect(page.locator("#root").nth(0)).to_contain_text("\u5c31\u4e0d\u80fd\u505a AI \u5b57\u5e55\u7ffb\u8b6f", timeout=15000), "The empty-key warning explains that AI subtitle translation will not be available."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    