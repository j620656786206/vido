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
        
        # -> Click the '下一步' button on the welcome step to proceed to the qBittorrent connection step.
        # 下一步 button
        elem = page.get_by_test_id("next-button")
        await elem.click(timeout=10000)
        
        # -> Click the '跳過' button on the qBittorrent connection step to proceed to the next wizard step.
        # 跳過 button
        elem = page.get_by_test_id("skip-button")
        await elem.click(timeout=10000)
        
        # -> Enter '/tmp' into the media folder '資料夾路徑' input and click the '上一步' (Back) button.
        # /media/movies text field
        elem = page.get_by_test_id("library-path-0")
        await elem.wait_for(state="visible", timeout=10000)
        await elem.fill("/tmp")
        
        # -> Enter '/tmp' into the media folder '資料夾路徑' input and click the '上一步' (Back) button.
        # 上一步 button
        elem = page.get_by_test_id("back-button")
        await elem.click(timeout=10000)
        
        # -> Click the '跳過' (Skip) button on the qBittorrent step to navigate to the Media Library step so the media folder input can be verified.
        # 跳過 button
        elem = page.get_by_test_id("skip-button")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> Media library folder path input retains the value '/tmp' after navigating back and forward.
        # Assert-outcome: passed
        # Assert: Media library folder path input value equals '/tmp'.
        await expect(page.get_by_test_id("library-path-0").nth(0)).to_have_value("/tmp", timeout=15000), "Media library folder path input value equals '/tmp'."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    