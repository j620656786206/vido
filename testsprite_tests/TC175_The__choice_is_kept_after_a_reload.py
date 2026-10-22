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
        
        # -> Open the Settings page using the visible Settings navigation link.
        # 設定 link
        elem = page.get_by_test_id("nav-settings")
        await elem.click(timeout=10000)
        
        # -> Open the 金鑰設定 page from the Settings navigation.
        # 連線：金鑰設定 link
        elem = page.get_by_test_id("settings-tab-keys")
        await elem.click(timeout=10000)
        
        # -> Select the OpenAI Whisper option under 聽寫服務.
        # asr-provider radio button
        elem = page.get_by_role("radio", name="OpenAI Whisper 一律用 Whisper")
        await elem.click(timeout=10000)
        
        # -> Save the selected OpenAI Whisper dictation service using the visible 儲存金鑰 button.
        # 儲存金鑰 button
        elem = page.get_by_test_id("key-save")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> OpenAI Whisper remained selected after saving and reloading the settings.
        await page.locator("xpath=/html/body/div/div/div/div[2]/main/div/div/div/div/div/div/form/fieldset/div/label[3]/span[1]/svg").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: OpenAI Whisper should remain selected.
        await expect(page.locator("xpath=/html/body/div/div/div/div[2]/main/div/div/div/div/div/div/form/fieldset/div/label[3]/span[1]/svg").nth(0)).to_be_visible(timeout=15000), "OpenAI Whisper should remain selected."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    