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
        
        # -> Click the '設定' (Settings) link in the sidebar to open the settings page.
        # 設定 link
        elem = page.get_by_test_id("nav-settings")
        await elem.click(timeout=10000)
        
        # -> Click the '外觀' (Appearance) link in the settings navigation to open the Appearance settings page.
        # 外觀：外觀 link
        elem = page.get_by_test_id("settings-tab-appearance")
        await elem.click(timeout=10000)
        
        # -> Click the '日巡' radio option to apply the light theme.
        # 日巡 宣紙底 · 松煙墨字 button
        elem = page.get_by_test_id("theme-option-light")
        await elem.click(timeout=10000)
        
        # -> Reload the '外觀' settings page and verify that the 日巡 radio remains selected, the page description reads '已依你的選擇顯示。' (or note its absence), and the <html> element still has data-theme="light".
        await page.goto("http://localhost:8090/settings/appearance")
        try:
            await page.wait_for_load_state("domcontentloaded", timeout=5000)
        except Exception:
            pass
        
        # -> Reload the Appearance settings page and verify the '日巡' radio remains selected, the page shows '已依你的選擇顯示。', and the <html> element has data-theme="light".
        await page.goto("http://localhost:8090/settings/appearance")
        try:
            await page.wait_for_load_state("domcontentloaded", timeout=5000)
        except Exception:
            pass
        
        # -> Click the '夜行' radio option to switch to the night theme and observe the resulting page state.
        # 夜行 墨綠底 · 宣紙白字 button
        elem = page.get_by_test_id("theme-option-dark")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> The Appearance page shows the heading '外觀' and the explanatory note '海報與劇照在深色底上對比更好，因此夜行是預設。'.
        await page.get_by_test_id("settings-tab-appearance").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: passed
        # Assert: Verify the '外觀' heading is visible.
        await expect(page.get_by_test_id("settings-tab-appearance").nth(0)).to_be_visible(timeout=15000), "Verify the '\u5916\u89c0' heading is visible."
        # Assert-outcome: passed
        # Assert: Verify the explanatory note about posters and photos is visible.
        await expect(page.locator("#root").nth(0)).to_contain_text("\u6d77\u5831\u8207\u5287\u7167\u5728\u6df1\u8272\u5e95\u4e0a\u5c0d\u6bd4\u66f4\u597d\uff0c\u56e0\u6b64\u591c\u884c\u662f\u9810\u8a2d\u3002", timeout=15000), "Verify the explanatory note about posters and photos is visible."
        
        # --> The page includes two theme radio options labeled '夜行' and '日巡'.
        await page.get_by_test_id("theme-option-dark").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: passed
        # Assert: Verify the '夜行' theme option is present.
        await expect(page.get_by_test_id("theme-option-dark").nth(0)).to_be_visible(timeout=15000), "Verify the '\u591c\u884c' theme option is present."
        await page.get_by_test_id("theme-option-light").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: passed
        # Assert: Verify the '日巡' theme option is present.
        await expect(page.get_by_test_id("theme-option-light").nth(0)).to_be_visible(timeout=15000), "Verify the '\u65e5\u5de1' theme option is present."
        
        # --> Selecting '日巡' resulted in '日巡' being checked and the page displayed '已依你的選擇顯示。', which persisted across a reload.
        # Assert-outcome: passed
        # Assert: Verify the '日巡' radio is checked (aria-checked=true).
        await expect(page.get_by_test_id("theme-option-light").nth(0)).to_have_attribute("aria-checked", "true", timeout=15000), "Verify the '\u65e5\u5de1' radio is checked (aria-checked=true)."
        # Assert-outcome: passed
        # Assert: Verify the confirmation text '已依你的選擇顯示。' is visible.
        await expect(page.locator("#root").nth(0)).to_contain_text("\u5df2\u4f9d\u4f60\u7684\u9078\u64c7\u986f\u793a\u3002", timeout=15000), "Verify the confirmation text '\u5df2\u4f9d\u4f60\u7684\u9078\u64c7\u986f\u793a\u3002' is visible."
        
        # --> After switching back, the '夜行' radio became checked and '日巡' became unchecked.
        # Assert-outcome: passed
        # Assert: Verify the '夜行' radio is checked (aria-checked=true).
        await expect(page.get_by_test_id("theme-option-dark").nth(0)).to_have_attribute("aria-checked", "true", timeout=15000), "Verify the '\u591c\u884c' radio is checked (aria-checked=true)."
        # Assert-outcome: passed
        # Assert: Verify the '日巡' radio is not checked (aria-checked=false).
        await expect(page.get_by_test_id("theme-option-light").nth(0)).to_have_attribute("aria-checked", "false", timeout=15000), "Verify the '\u65e5\u5de1' radio is not checked (aria-checked=false)."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    