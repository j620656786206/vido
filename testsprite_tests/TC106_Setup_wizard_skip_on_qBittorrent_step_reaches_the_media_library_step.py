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
        
        # -> Click the '下一步' (Next) button on the welcome wizard card to advance to the qBittorrent step.
        # 下一步 button
        elem = page.get_by_test_id('next-button')
        await elem.click(timeout=10000)
        
        # -> Click the '跳過' (Skip) button on the qBittorrent connection step to advance to the media library step.
        # 跳過 button
        elem = page.get_by_test_id('skip-button')
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> Setup wizard is visible and shows the step indicator '步驟 3 / 5'.
        # Assert-outcome: passed
        # Assert: The wizard header displays the step indicator '步驟 3 / 5'.
        await expect(page.locator("xpath=/html/body/div[1]").nth(0)).to_contain_text("\u6b65\u9a5f 3 / 5", timeout=15000), "The wizard header displays the step indicator '\u6b65\u9a5f 3 / 5'."
        
        # --> Media library step rendered: the folder path input and the '新增媒體庫' button are visible.
        await page.locator("xpath=/html/body/div[1]/div/div/div/div[3]/div[1]/div/div[1]/input").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: passed
        # Assert: The media folder path input is visible.
        await expect(page.locator("xpath=/html/body/div[1]/div/div/div/div[3]/div[1]/div/div[1]/input").nth(0)).to_be_visible(timeout=15000), "The media folder path input is visible."
        await page.locator("xpath=/html/body/div[1]/div/div/div/div[3]/button").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: passed
        # Assert: The '新增媒體庫' (Add library) button is visible.
        await expect(page.locator("xpath=/html/body/div[1]/div/div/div/div[3]/button").nth(0)).to_be_visible(timeout=15000), "The '\u65b0\u589e\u5a92\u9ad4\u5eab' (Add library) button is visible."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    