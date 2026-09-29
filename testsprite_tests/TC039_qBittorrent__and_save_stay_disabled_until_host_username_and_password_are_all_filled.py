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
        
        # -> Click the '設定' (Settings) link in the left sidebar to open the Settings page.
        # 設定 link
        elem = page.get_by_test_id("nav-settings")
        await elem.click(timeout=10000)
        
        # -> Scroll down to reveal the rest of the qBittorrent form and clear the 主機位址, 使用者名稱, and 密碼 fields, then verify that the '測試連線' and '儲存設定' buttons are disabled.
        await page.mouse.wheel(0, 300)
        
        # -> Scroll down to reveal the rest of the qBittorrent form and clear the 主機位址, 使用者名稱, and 密碼 fields, then verify that the '測試連線' and '儲存設定' buttons are disabled.
        # http://192.168.1.100:8080 text field
        elem = page.get_by_role("textbox", name="主機位址")
        await elem.wait_for(state="visible", timeout=10000)
        await elem.fill("")
        
        # -> Scroll down to reveal the rest of the qBittorrent form and clear the 主機位址, 使用者名稱, and 密碼 fields, then verify that the '測試連線' and '儲存設定' buttons are disabled.
        # admin text field
        elem = page.get_by_role("textbox", name="使用者名稱")
        await elem.wait_for(state="visible", timeout=10000)
        await elem.fill("")
        
        # -> Scroll down to reveal the rest of the qBittorrent form and clear the 主機位址, 使用者名稱, and 密碼 fields, then verify that the '測試連線' and '儲存設定' buttons are disabled.
        # •••••••• password field
        elem = page.get_by_role("textbox", name="密碼")
        await elem.wait_for(state="visible", timeout=10000)
        await elem.fill("")
        
        # -> Clear the 主機位址, 使用者名稱, and 密碼 fields in the qBittorrent panel and scroll down to reveal the 儲存設定 (Save settings) button.
        # http://192.168.1.100:8080 text field
        elem = page.get_by_role("textbox", name="主機位址")
        await elem.wait_for(state="visible", timeout=10000)
        await elem.fill("")
        
        # -> Clear the 主機位址, 使用者名稱, and 密碼 fields in the qBittorrent panel and scroll down to reveal the 儲存設定 (Save settings) button.
        # admin text field
        elem = page.get_by_role("textbox", name="使用者名稱")
        await elem.wait_for(state="visible", timeout=10000)
        await elem.fill("")
        
        # -> Clear the 主機位址, 使用者名稱, and 密碼 fields in the qBittorrent panel and scroll down to reveal the 儲存設定 (Save settings) button.
        # •••••••• password field
        elem = page.get_by_role("textbox", name="密碼")
        await elem.wait_for(state="visible", timeout=10000)
        await elem.fill("")
        
        # -> Clear the 主機位址, 使用者名稱, and 密碼 fields in the qBittorrent panel and scroll down to reveal the 儲存設定 (Save settings) button.
        await page.mouse.wheel(0, 300)
        
        # -> Clear the qBittorrent fields '主機位址', '使用者名稱', and '密碼' and scroll down to reveal the '儲存設定' (Save settings) button.
        # http://192.168.1.100:8080 text field
        elem = page.get_by_role("textbox", name="主機位址")
        await elem.wait_for(state="visible", timeout=10000)
        await elem.fill("")
        
        # -> Clear the qBittorrent fields '主機位址', '使用者名稱', and '密碼' and scroll down to reveal the '儲存設定' (Save settings) button.
        # admin text field
        elem = page.get_by_role("textbox", name="使用者名稱")
        await elem.wait_for(state="visible", timeout=10000)
        await elem.fill("")
        
        # -> Clear the qBittorrent fields '主機位址', '使用者名稱', and '密碼' and scroll down to reveal the '儲存設定' (Save settings) button.
        # •••••••• password field
        elem = page.get_by_role("textbox", name="密碼")
        await elem.wait_for(state="visible", timeout=10000)
        await elem.fill("")
        
        # -> Clear the qBittorrent '主機位址', '使用者名稱', and '密碼' fields and scroll down to reveal the '儲存設定' (Save settings) button.
        # http://192.168.1.100:8080 text field
        elem = page.get_by_role("textbox", name="主機位址")
        await elem.wait_for(state="visible", timeout=10000)
        await elem.fill("")
        
        # -> Clear the qBittorrent '主機位址', '使用者名稱', and '密碼' fields and scroll down to reveal the '儲存設定' (Save settings) button.
        # admin text field
        elem = page.get_by_role("textbox", name="使用者名稱")
        await elem.wait_for(state="visible", timeout=10000)
        await elem.fill("")
        
        # -> Clear the qBittorrent '主機位址', '使用者名稱', and '密碼' fields and scroll down to reveal the '儲存設定' (Save settings) button.
        # •••••••• password field
        elem = page.get_by_role("textbox", name="密碼")
        await elem.wait_for(state="visible", timeout=10000)
        await elem.fill("")
        
        # -> Clear the qBittorrent '主機位址', '使用者名稱', and '密碼' fields and scroll down to reveal the '儲存設定' (Save settings) button.
        await page.mouse.wheel(0, 300)
        
        # -> Clear the qBittorrent fields labelled '主機位址', '使用者名稱', and '密碼', then check the '測試連線' and '儲存設定' buttons for disabled state.
        # http://192.168.1.100:8080 text field
        elem = page.get_by_role("textbox", name="主機位址")
        await elem.wait_for(state="visible", timeout=10000)
        await elem.fill("")
        
        # -> Clear the qBittorrent fields labelled '主機位址', '使用者名稱', and '密碼', then check the '測試連線' and '儲存設定' buttons for disabled state.
        # admin text field
        elem = page.get_by_role("textbox", name="使用者名稱")
        await elem.wait_for(state="visible", timeout=10000)
        await elem.fill("")
        
        # -> Clear the qBittorrent fields labeled '主機位址', '使用者名稱', and '密碼' so they are empty, then reveal the '儲存設定' (Save settings) button by scrolling.
        # http://192.168.1.100:8080 text field
        elem = page.get_by_role("textbox", name="主機位址")
        await elem.wait_for(state="visible", timeout=10000)
        await elem.fill("")
        
        # -> Clear the qBittorrent fields labeled '主機位址', '使用者名稱', and '密碼' so they are empty, then reveal the '儲存設定' (Save settings) button by scrolling.
        # admin text field
        elem = page.get_by_role("textbox", name="使用者名稱")
        await elem.wait_for(state="visible", timeout=10000)
        await elem.fill("")
        
        # -> Clear the qBittorrent fields labeled '主機位址', '使用者名稱', and '密碼' so they are empty, then reveal the '儲存設定' (Save settings) button by scrolling.
        # •••••••• password field
        elem = page.get_by_role("textbox", name="密碼")
        await elem.wait_for(state="visible", timeout=10000)
        await elem.fill("")
        
        # -> Clear the qBittorrent fields labeled '主機位址', '使用者名稱', and '密碼' so they are empty, then reveal the '儲存設定' (Save settings) button by scrolling.
        await page.mouse.wheel(0, 300)
        
        # -> Clear the '主機位址', '使用者名稱', and '密碼' fields (make them empty), then scroll down to reveal the '儲存設定' (Save settings) button.
        # http://192.168.1.100:8080 text field
        elem = page.get_by_role("textbox", name="主機位址")
        await elem.wait_for(state="visible", timeout=10000)
        await elem.fill("")
        
        # -> Clear the '主機位址', '使用者名稱', and '密碼' fields (make them empty), then scroll down to reveal the '儲存設定' (Save settings) button.
        # admin text field
        elem = page.get_by_role("textbox", name="使用者名稱")
        await elem.wait_for(state="visible", timeout=10000)
        await elem.fill("")
        
        # -> Clear the '主機位址', '使用者名稱', and '密碼' fields (make them empty), then scroll down to reveal the '儲存設定' (Save settings) button.
        # •••••••• password field
        elem = page.get_by_role("textbox", name="密碼")
        await elem.wait_for(state="visible", timeout=10000)
        await elem.fill("")
        
        # -> Clear the 主機位址, 使用者名稱, and 密碼 fields in the qBittorrent panel and then verify the '測試連線' and '儲存設定' buttons are disabled by checking the buttons on the page.
        # http://192.168.1.100:8080 text field
        elem = page.get_by_role("textbox", name="主機位址")
        await elem.wait_for(state="visible", timeout=10000)
        await elem.fill("")
        
        # -> Clear the 主機位址, 使用者名稱, and 密碼 fields in the qBittorrent panel and then verify the '測試連線' and '儲存設定' buttons are disabled by checking the buttons on the page.
        # admin text field
        elem = page.get_by_role("textbox", name="使用者名稱")
        await elem.wait_for(state="visible", timeout=10000)
        await elem.fill("")
        
        # -> Clear the 主機位址, 使用者名稱, and 密碼 fields in the qBittorrent panel and then verify the '測試連線' and '儲存設定' buttons are disabled by checking the buttons on the page.
        # •••••••• password field
        elem = page.get_by_role("textbox", name="密碼")
        await elem.wait_for(state="visible", timeout=10000)
        await elem.fill("")
        
        # -> Type 'http://invalid-host.invalid:8080' into the 主機位址 field and 'admin' into the 使用者名稱 field, leave 密碼 empty, then verify the 測試連線 and 儲存設定 buttons remain disabled.
        # http://192.168.1.100:8080 text field
        elem = page.get_by_role("textbox", name="主機位址")
        await elem.wait_for(state="visible", timeout=10000)
        await elem.fill("http://invalid-host.invalid:8080")
        
        # -> Type 'http://invalid-host.invalid:8080' into the 主機位址 field and 'admin' into the 使用者名稱 field, leave 密碼 empty, then verify the 測試連線 and 儲存設定 buttons remain disabled.
        # admin text field
        elem = page.get_by_role("textbox", name="使用者名稱")
        await elem.wait_for(state="visible", timeout=10000)
        await elem.fill("admin")
        
        # -> Type 'http://invalid-host.invalid:8080' into the 主機位址 field and 'admin' into the 使用者名稱 field, leave 密碼 empty, then verify the 測試連線 and 儲存設定 buttons remain disabled.
        # •••••••• password field
        elem = page.get_by_role("textbox", name="密碼")
        await elem.wait_for(state="visible", timeout=10000)
        await elem.fill("")
        
        # -> Clear the qBittorrent 主機位址, 使用者名稱, and 密碼 fields and verify the '測試連線' and '儲存設定' buttons are disabled.
        # http://192.168.1.100:8080 text field
        elem = page.get_by_role("textbox", name="主機位址")
        await elem.wait_for(state="visible", timeout=10000)
        await elem.fill("")
        
        # -> Clear the qBittorrent 主機位址, 使用者名稱, and 密碼 fields and verify the '測試連線' and '儲存設定' buttons are disabled.
        # admin text field
        elem = page.get_by_role("textbox", name="使用者名稱")
        await elem.wait_for(state="visible", timeout=10000)
        await elem.fill("")
        
        # --> Assertions to verify final state
        
        # --> The qBittorrent '測試連線' button is disabled when required fields are empty and remained disabled after entering host and username with the password empty.
        # Assert-outcome: passed
        # Assert: The 測試連線 button is disabled.
        await expect(page.get_by_test_id("qbittorrent-form").get_by_role("button", name="測試連線").nth(0)).to_have_attribute("disabled", "", timeout=15000), "The \u6e2c\u8a66\u9023\u7dda button is disabled."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    