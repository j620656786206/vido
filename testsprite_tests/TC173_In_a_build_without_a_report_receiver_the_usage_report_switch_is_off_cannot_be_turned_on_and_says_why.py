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
        
        # -> Open Settings from the sidebar and locate the connection settings page.
        # 設定 link
        elem = page.get_by_test_id("nav-settings")
        await elem.click(timeout=10000)
        
        # -> Scroll down the connection settings page to locate the 匿名使用回報 card.
        await page.mouse.wheel(0, 300)
        
        # --> Assertions to verify final state
        
        # --> The 匿名使用回報 switch is off and disabled.
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: The switch is off.
        await expect(page.get_by_test_id("usage-report-switch").nth(0)).to_have_attribute("aria-checked", "false", timeout=15000), "The switch is off."
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: The switch is disabled.
        await expect(page.get_by_test_id("usage-report-switch").nth(0)).to_have_attribute("disabled", "", timeout=15000), "The switch is disabled."
        
        # --> The card says 這個版本沒有設定接收端，無法開啟。
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: The card displays the no-receiver message.
        await expect(page.locator("#root").nth(0)).to_contain_text("\u9019\u500b\u7248\u672c\u6c92\u6709\u8a2d\u5b9a\u63a5\u6536\u7aef\uff0c\u7121\u6cd5\u958b\u555f\u3002", timeout=15000), "The card displays the no-receiver message."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    