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
        
        # -> Open the specified movie page for 沙丘:第二部 and inspect its visible controls.
        await page.goto("http://localhost:8090/media/movie/seed-mv-012")
        try:
            await page.wait_for_load_state("domcontentloaded", timeout=5000)
        except Exception:
            pass
        
        # -> Open the movie’s 修改資訊 form and review its visible fields.
        # 修改資訊 button
        elem = page.get_by_test_id("action-edit-metadata")
        await elem.click(timeout=10000)
        
        # -> Replace the 片名 value with the test title and choose 重新選片.
        # title text field
        elem = page.get_by_role("textbox", name="片名 *")
        await elem.wait_for(state="visible", timeout=10000)
        await elem.fill("\u4e0d\u8a72\u88ab\u5b58\u4e0b\u4f86\u7684\u7247\u540d")
        
        # -> Replace the 片名 value with the test title and choose 重新選片.
        # 重新選片 button
        elem = page.get_by_role("button", name="重新選片")
        await elem.click(timeout=10000)
        
        # -> Reopen 修改資訊 and verify the 片名 field contains 沙丘:第二部.
        # 修改資訊 button
        elem = page.get_by_test_id("action-edit-metadata")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> The replacement title was not saved: the movie page and reopened title field still show 沙丘:第二部.
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: The movie page shows 沙丘:第二部.
        await expect(page.locator("#root").nth(0)).to_contain_text("\u6c99\u4e18:\u7b2c\u4e8c\u90e8", timeout=15000), "The movie page shows \u6c99\u4e18:\u7b2c\u4e8c\u90e8."
        # Assert-outcome: not verified — the export-time check could not be evaluated
        # Assert: The title field contains 沙丘:第二部.
        await expect(page.locator("xpath=/html/body/div[4]/div[3]/form/fieldset/div[1]/input").nth(0)).to_have_value("\u6c99\u4e18:\u7b2c\u4e8c\u90e8", timeout=15000), "The title field contains \u6c99\u4e18:\u7b2c\u4e8c\u90e8."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    