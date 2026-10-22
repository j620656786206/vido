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
        
        # -> Open the Settings page from the visible navigation.
        # 設定 link
        elem = page.get_by_test_id("nav-settings")
        await elem.click(timeout=10000)
        
        # -> Open “金鑰設定” and inspect the available key fields and save controls.
        # 連線：金鑰設定 link
        elem = page.get_by_test_id("settings-tab-keys")
        await elem.click(timeout=10000)
        
        # -> Open the Gemini key editor and inspect its field and save controls.
        # 編輯 button
        elem = page.get_by_test_id("key-edit-gemini")
        await elem.click(timeout=10000)
        
        # -> Enter the provided test Gemini key in the Gemini key field, then save the key.
        # AIza… password field
        elem = page.get_by_role("textbox", name="Gemini（選填）")
        await elem.wait_for(state="visible", timeout=10000)
        await elem.fill("AIzaSyTestOnlyKeyForVido000000000000")
        
        # -> Enter the provided test Gemini key in the Gemini key field, then save the key.
        await page.mouse.wheel(0, 300)
        
        # -> Save the Gemini key using the “儲存金鑰” button.
        # 儲存金鑰 button
        elem = page.get_by_test_id("key-save")
        await elem.click(timeout=10000)
        
        # -> Open the Movies library to locate 侏羅紀公園.
        # 電影 15 link
        elem = page.get_by_test_id("nav-movies")
        await elem.click(timeout=10000)
        
        # -> Open the 侏羅紀公園 movie details and locate its subtitle-management control.
        # 缺中文 8.2 侏羅紀公園 1993 link
        elem = page.get_by_test_id("poster-v2-seed-mv-002")
        await elem.click(timeout=10000)
        
        # -> Open the “管理字幕” dialog for 侏羅紀公園 and inspect its controls and cost estimate.
        # 管理字幕 button
        elem = page.get_by_test_id("action-manage-subtitle")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> The 管理字幕 dialog for 侏羅紀公園 is visible.
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: The subtitle-management dialog should show 管理字幕 — 侏羅紀公園.
        await expect(page.locator("xpath=/html/body/div[4]").nth(0)).to_contain_text("\u7ba1\u7406\u5b57\u5e55 \u2014 \u4f8f\u7f85\u7d00\u516c\u5712", timeout=15000), "The subtitle-management dialog should show \u7ba1\u7406\u5b57\u5e55 \u2014 \u4f8f\u7f85\u7d00\u516c\u5712."
        
        # --> The 生成字幕 button is visible in the dialog.
        await page.locator("xpath=/html/body/div[4]/div[3]/section[2]/button").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: The 生成字幕 button should be visible.
        await expect(page.locator("xpath=/html/body/div[4]/div[3]/section[2]/button").nth(0)).to_be_visible(timeout=15000), "The \u751f\u6210\u5b57\u5e55 button should be visible."
        
        # --> The dialog displays a US-dollar cost estimate.
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: The dialog should display a US-dollar cost estimate.
        await expect(page.locator("xpath=/html/body/div[4]/div[3]/section[2]/button").nth(0)).to_contain_text("\u2248 $", timeout=15000), "The dialog should display a US-dollar cost estimate."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    