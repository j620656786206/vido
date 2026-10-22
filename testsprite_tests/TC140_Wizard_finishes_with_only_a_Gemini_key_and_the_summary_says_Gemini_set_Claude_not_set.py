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
        
        # -> Advance from the welcome screen to the qBittorrent setup step.
        # 下一步 button
        elem = page.get_by_test_id("next-button")
        await elem.click(timeout=10000)
        
        # -> Skip the optional qBittorrent connection step to continue setup.
        # 跳過 button
        elem = page.get_by_test_id("skip-button")
        await elem.click(timeout=10000)
        
        # -> Add a media library using the visible control so the wizard can continue to API 金鑰.
        # 新增媒體庫 button
        elem = page.get_by_test_id("add-library-button")
        await elem.click(timeout=10000)
        
        # -> Enter `/media/movies` as the path for the first media library so setup can continue.
        # /media/movies text field
        elem = page.get_by_test_id("library-path-0")
        await elem.wait_for(state="visible", timeout=10000)
        await elem.fill("/media/movies")
        
        # -> Remove the unused blank media-library row so the wizard can continue.
        # 移除此媒體庫 button
        elem = page.get_by_test_id("library-remove-1")
        await elem.click(timeout=10000)
        
        # -> Advance from 媒體庫設定 to the API 金鑰 step using `下一步`.
        # 下一步 button
        elem = page.get_by_test_id("next-button")
        await elem.click(timeout=10000)
        
        # -> Replace the invalid media-library path with the mounted movies folder shown in the validation message.
        # /media/movies text field
        elem = page.get_by_test_id("library-path-0")
        await elem.wait_for(state="visible", timeout=10000)
        await elem.fill("/Users/alexyu/projects/personal/vido/.vido-test-env/media/movies")
        
        # -> Continue from 媒體庫設定 using the mounted movies-folder path and check whether the wizard accepts it.
        # 下一步 button
        elem = page.get_by_test_id("next-button")
        await elem.click(timeout=10000)
        
        # -> Enter the provided test key in the Gemini 金鑰 field while leaving Claude 金鑰 blank.
        # 輸入 Gemini API 金鑰... password field
        elem = page.get_by_test_id("gemini-key-input")
        await elem.wait_for(state="visible", timeout=10000)
        await elem.fill("AIzaSyTestOnlyKeyForVido000000000000")
        
        # -> Click “下一步” on the API-key page to reach the final setup summary.
        # 下一步 button
        elem = page.get_by_test_id("next-button")
        await elem.click(timeout=10000)
        
        # -> Continue from the optional anonymous-reporting step to the final setup summary using “下一步”.
        # 下一步 button
        elem = page.get_by_test_id("next-button")
        await elem.click(timeout=10000)
        
        # -> Click “完成設定” and verify that the setup wizard closes and the main app navigation is visible.
        # 完成設定 button
        elem = page.get_by_test_id("finish-button")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> After setup, the wizard closed and the main navigation is visible.
        await page.get_by_test_id("app-sidebar").get_by_test_id("nav-home").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: The main navigation is visible after setup.
        await expect(page.get_by_test_id("app-sidebar").get_by_test_id("nav-home").nth(0)).to_be_visible(timeout=15000), "The main navigation is visible after setup."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    