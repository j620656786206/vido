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
        
        # -> Click “下一步” on the welcome screen to continue the setup wizard.
        # 下一步 button
        elem = page.get_by_test_id("next-button")
        await elem.click(timeout=10000)
        
        # -> Skip the optional qBittorrent connection step to continue through setup.
        # 跳過 button
        elem = page.get_by_test_id("skip-button")
        await elem.click(timeout=10000)
        
        # -> Enter the example media folder path so the setup wizard can continue.
        # /media/movies text field
        elem = page.get_by_test_id("library-path-0")
        await elem.wait_for(state="visible", timeout=10000)
        await elem.fill("/media/movies")
        
        # -> Click “下一步” to continue the setup wizard toward the anonymous usage-report step.
        # 下一步 button
        elem = page.get_by_test_id("next-button")
        await elem.click(timeout=10000)
        
        # -> Replace the invalid media folder path with the displayed movies-path example and check whether the wizard accepts it.
        # /media/movies text field
        elem = page.get_by_test_id("library-path-0")
        await elem.wait_for(state="visible", timeout=10000)
        await elem.fill("/Users/alexyu/projects/personal/vido/.vido-test-env/media/movies")
        
        # -> Submit the entered media-folder path by selecting “下一步” and inspect the wizard’s response.
        # 下一步 button
        elem = page.get_by_test_id("next-button")
        await elem.click(timeout=10000)
        
        # -> Skip the optional API key step to continue toward 匿名使用回報.
        # 跳過 button
        elem = page.get_by_test_id("skip-button")
        await elem.click(timeout=10000)
        
        # -> Turn on the anonymous usage report switch and verify that it is enabled.
        # button
        elem = page.get_by_test_id("usage-report-step-switch")
        await elem.click(timeout=10000)
        
        # -> Select “下一步” and verify the final setup summary shows 匿名使用回報 as 開啟.
        # 下一步 button
        elem = page.get_by_test_id("next-button")
        await elem.click(timeout=10000)
        
        # -> Click「完成設定」and then verify the anonymous usage-report setting in connection settings.
        # 完成設定 button
        elem = page.get_by_test_id("finish-button")
        await elem.click(timeout=10000)
        
        # -> Open「設定」from the sidebar and verify the anonymous usage-report setting is on in connection settings.
        # 設定 link
        elem = page.get_by_test_id("nav-settings")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> 匿名使用回報在設定摘要中顯示「開啟」，且連線設定中的開關已啟用。
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: 匿名使用回報開關應保持啟用。
        await expect(page.get_by_test_id("usage-report-switch").nth(0)).to_have_attribute("aria-checked", "true", timeout=15000), "\u533f\u540d\u4f7f\u7528\u56de\u5831\u958b\u95dc\u61c9\u4fdd\u6301\u555f\u7528\u3002"
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    