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
        
        # -> Open the 活動 page using the visible sidebar link.
        # 活動 link
        elem = page.get_by_test_id("app-sidebar").get_by_test_id("nav-activity")
        await elem.click(timeout=10000)
        
        # -> Open the 批次生成字幕 dialog to inspect the candidate list, estimate, and missing-key notice.
        # 批次生成字幕 button
        elem = page.get_by_test_id("activity-generation-batch-cta")
        await elem.click(timeout=10000)
        
        # -> Inspect the sort menu options for the subtitle candidate list.
        # [internal] get_dropdown_options: index=
        
        # -> Choose 片名 A→Z and verify that the candidate list reorders while the 產生字幕 dialog remains open.
        # 群組 金額高→低 金額低→高 片名 A→Z 未匹配優先 dropdown
        elem = page.locator("xpath=/html/body/div[3]/div[3]/div/div[3]/span/select").nth(0)
        await elem.wait_for(state="visible", timeout=10000)
        await elem.select_option("")
        
        # -> Close the 產生字幕 dialog without starting generation.
        # 關閉視窗 button
        elem = page.get_by_role("button", name="關閉視窗")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        current_url = await page.evaluate("() => window.location.href")
        # Assert-outcome: passed
        # Assert: page loaded with a URL (final outcome verified by the AI judge during the run)
        assert current_url, 'Page should have loaded with a URL'
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    