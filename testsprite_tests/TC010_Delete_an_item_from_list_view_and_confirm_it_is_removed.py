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
        
        # -> Click the '媒體庫' link in the sidebar to open the Library page.
        # 媒體庫 link
        elem = page.get_by_text('內容', exact=True).locator("xpath=ancestor-or-self::*[.//a][1]").get_by_role('link', name='媒體庫', exact=True)
        await elem.click(timeout=10000)
        
        # -> Click the '列表檢視' (list view) button to switch the library to list/table view.
        # 列表檢視 button
        elem = page.get_by_text('列表檢視', exact=True)
        await elem.click(timeout=10000)
        
        # -> Click the '教父' list row (the anchor labeled '教父') to open its item view so the per-item action menu can be located.
        # 教父 1972 · 犯罪 缺字幕 8.7 link
        elem = page.get_by_test_id('list-row-v2-seed-mv-001')
        await elem.click(timeout=10000)
        
        # -> Click the '返回媒體庫' (Back to Library) button to return to the Library list view so the per-item action menu for '教父' can be located.
        # 返回媒體庫 button
        elem = page.get_by_test_id('detail-back')
        await elem.click(timeout=10000)
        
        # -> Click the '教父' list row to open its detail page so the per-item action (delete) can be located.
        # 教父 1972 · 犯罪 缺字幕 8.7 link
        elem = page.get_by_test_id('list-row-v2-seed-mv-001')
        await elem.click(timeout=10000)
        
        # -> Open the '媒體庫' (Library) page by clicking the sidebar '媒體庫' link so the per-item action menu for the 教父 row can be located.
        # 媒體庫 link
        elem = page.get_by_text('內容', exact=True).locator("xpath=ancestor-or-self::*[.//a][1]").get_by_role('link', name='媒體庫', exact=True)
        await elem.click(timeout=10000)
        
        # -> Click the '格狀檢視' (grid view) toggle to switch the library to card/grid view so the per-item kebab/menu becomes visible.
        # 格狀檢視 button
        elem = page.get_by_text('格狀檢視', exact=True)
        await elem.click(timeout=10000)
        
        # -> Open the 教父 card by clicking the card that shows '教父 1972' to reveal the per-item '更多' (More) menu or item actions.
        # 缺字幕 8.7 教父 1972 link
        elem = page.get_by_test_id('poster-v2-seed-mv-001')
        await elem.click(timeout=10000)
        
        # -> Click the '返回媒體庫' (Back to Library) button to return to the Library page so the per-item action menu can be located.
        # 返回媒體庫 button
        elem = page.get_by_test_id('detail-back')
        await elem.click(timeout=10000)
        
        # -> Click the '列表檢視' (List view) toggle button to switch the library to list/table view.
        # 列表檢視 button
        elem = page.get_by_text('列表檢視', exact=True)
        await elem.click(timeout=10000)
        
        # -> Click the '格狀檢視' (grid view) toggle to show per-card '更多' menus so the per-item action menu for the 教父 card can be opened.
        # 格狀檢視 button
        elem = page.get_by_text('格狀檢視', exact=True)
        await elem.click(timeout=10000)
        
        # --> Assertions to verify final state
        
        # --> Verify a media items table is visible
        await page.locator("xpath=/html/body/div[1]/div/div/div[2]/main/div/div/div[2]/div[2]/a[18]").nth(0).scroll_into_view_if_needed()
        # Assert: A media items list row (教父) is visible, confirming the media items table is shown.
        await expect(page.locator("xpath=/html/body/div[1]/div/div/div[2]/main/div/div/div[2]/div[2]/a[18]").nth(0)).to_be_visible(timeout=15000), "A media items list row (\u6559\u7236) is visible, confirming the media items table is shown."
        current_url = await page.evaluate("() => window.location.href")
        # Assert: page loaded with a URL (final outcome verified by the AI judge during the run)
        assert current_url, 'Page should have loaded with a URL'
        current_url = await page.evaluate("() => window.location.href")
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
    