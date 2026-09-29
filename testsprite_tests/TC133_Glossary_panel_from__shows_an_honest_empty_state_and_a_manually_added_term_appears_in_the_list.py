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
        
        # -> Open the movie detail page for 駭客任務 by navigating to /media/movie/seed-mv-003.
        await page.goto("http://localhost:8090/media/movie/seed-mv-003")
        try:
            await page.wait_for_load_state("domcontentloaded", timeout=5000)
        except Exception:
            pass
        
        # -> Click the '管理字幕' button to open the subtitle management dialog for 駭客任務.
        # 管理字幕 button
        elem = page.get_by_test_id("action-manage-subtitle")
        await elem.click(timeout=10000)
        
        # -> Click the '名詞對照表' row in the 管理字幕 dialog to open the glossary panel.
        # 名詞對照表 （ 0 條） button
        elem = page.get_by_test_id("open-glossary")
        await elem.click(timeout=10000)
        
        # -> Click the '新增詞彙' button to open the add-term form.
        # 新增詞彙 button
        elem = page.get_by_test_id("glossary-empty-add")
        await elem.click(timeout=10000)
        
        # -> Fill the 原文詞彙 field with 'Neo' and 中文譯名 with '尼歐', then click the '新增' button to add the glossary term.
        # 原文詞彙 text field
        elem = page.get_by_test_id("glossary-add-src")
        await elem.wait_for(state="visible", timeout=10000)
        await elem.fill("Neo")
        
        # -> Fill the 原文詞彙 field with 'Neo' and 中文譯名 with '尼歐', then click the '新增' button to add the glossary term.
        # 中文譯名 text field
        elem = page.get_by_test_id("glossary-add-zh")
        await elem.wait_for(state="visible", timeout=10000)
        await elem.fill("\u5c3c\u6b50")
        
        # -> Fill the 原文詞彙 field with 'Neo' and 中文譯名 with '尼歐', then click the '新增' button to add the glossary term.
        # 新增 button
        elem = page.get_by_test_id("glossary-add-submit")
        await elem.click(timeout=10000)
        
        # -> Click the '刪除' button for the glossary entry 'Neo' to open the 刪除詞彙 confirmation dialog.
        # 刪除 Neo button
        elem = page.get_by_test_id("glossary-delete-06c680df-f1dc-425c-95a3-4e68e285494a")
        await elem.click(timeout=10000)
        
        # -> Click the '刪除' button in the 刪除詞彙 confirmation dialog to confirm deletion of the Neo term.
        # 刪除 button
        elem = page.get_by_test_id("glossary-delete-confirm-06c680df-f1dc-425c-95a3-4e68e285494a")
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> The 名詞對照表 panel for 駭客任務 is open and shows the empty-state text and the 新增詞彙 button.
        # Assert-outcome: passed
        # Assert: Panel shows the empty-state text '尚無詞彙'.
        await expect(page.get_by_test_id("glossary-panel-v2").nth(0)).to_contain_text("\u5c1a\u7121\u8a5e\u5f59", timeout=15000), "Panel shows the empty-state text '\u5c1a\u7121\u8a5e\u5f59'."
        await page.get_by_test_id("glossary-empty-add").nth(0).scroll_into_view_if_needed()
        # Assert-outcome: passed
        # Assert: The '新增詞彙' button is visible.
        await expect(page.get_by_test_id("glossary-empty-add").nth(0)).to_be_visible(timeout=15000), "The '\u65b0\u589e\u8a5e\u5f59' button is visible."
        
        # --> After adding, the glossary contained the term 'Neo' with the translation '尼歐'.
        # Assert-outcome: passed
        # Assert: Glossary lists the source term 'Neo'.
        await expect(page.get_by_test_id("glossary-panel-v2").nth(0)).to_contain_text("Neo", timeout=15000), "Glossary lists the source term 'Neo'."
        # Assert-outcome: passed
        # Assert: Glossary lists the translation '尼歐'.
        await expect(page.get_by_test_id("glossary-panel-v2").nth(0)).to_contain_text("\u5c3c\u6b50", timeout=15000), "Glossary lists the translation '\u5c3c\u6b50'."
        
        # --> After deletion, the 名詞對照表 panel returned to the empty state '尚無詞彙'.
        # Assert-outcome: passed
        # Assert: Empty-state '尚無詞彙' is visible after deletion.
        await expect(page.get_by_test_id("glossary-panel-v2").nth(0)).to_contain_text("\u5c1a\u7121\u8a5e\u5f59", timeout=15000), "Empty-state '\u5c1a\u7121\u8a5e\u5f59' is visible after deletion."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    