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
        
        # -> Open the qBittorrent settings page and confirm the page title contains 'qBittorrent'.
        await page.goto("http://localhost:8090/settings/qbittorrent")
        try:
            await page.wait_for_load_state("domcontentloaded", timeout=5000)
        except Exception:
            pass
        
        # -> Open the 'qBittorrent' settings page by navigating to http://localhost:8090/settings/qbittorrent.
        await page.goto("http://localhost:8090/settings/qbittorrent")
        try:
            await page.wait_for_load_state("domcontentloaded", timeout=5000)
        except Exception:
            pass
        
        # -> Open the 'qBittorrent' settings page (Settings → qBittorrent) by navigating to its URL and load the qBittorrent settings UI.
        await page.goto("http://localhost:8090/settings/qbittorrent")
        try:
            await page.wait_for_load_state("domcontentloaded", timeout=5000)
        except Exception:
            pass
        
        # -> Find and reveal the 'Learned Patterns' section by searching the page for 'qBittorrent' and 'Learned Patterns' (and Chinese fallback keywords), then scroll to reveal the section.
        await page.mouse.wheel(0, 300)
        
        # -> Search the settings page for the 'Learned Patterns' heading using English and Chinese keywords ('Learned Patterns', '模式', '已學習'), then scroll the page if not found.
        await page.mouse.wheel(0, 300)
        
        # --> Assertions to verify final state
        
        # --> The page did not load the qBittorrent settings page (title 'qBittorrent' was not found).
        # Assert-outcome: failed
        # Assert: Expected the page URL to contain '/settings/qbittorrent' indicating the qBittorrent settings page.
        await expect(page).to_have_url(re.compile("/settings/qbittorrent"), timeout=15000), "Expected the page URL to contain '/settings/qbittorrent' indicating the qBittorrent settings page."
        
        # --> The heading text "Learned Patterns" was not visible on the Settings page.
        # Assert-outcome: failed
        # Assert: Expected the page to contain the text 'Learned Patterns' in the settings content.
        await expect(page.get_by_test_id("settings-tabs-strip").nth(0)).to_contain_text("Learned Patterns", timeout=15000), "Expected the page to contain the text 'Learned Patterns' in the settings content."
        
        # --> The 'Learned Patterns' list was not found on the Settings page.
        # Assert-outcome: failed
        # Assert: Expected the 'Learned Patterns' list element to be visible on the page.
        await expect(page.get_by_test_id("settings-tabs-strip").nth(0)).to_contain_text("Learned Patterns list", timeout=15000), "Expected the 'Learned Patterns' list element to be visible on the page."
        
        # --> The 'Pattern statistics' area was not found on the Settings page.
        # Assert-outcome: failed
        # Assert: Expected the 'Pattern statistics' area to be visible on the page.
        await expect(page.get_by_test_id("settings-tabs-strip").nth(0)).to_contain_text("Pattern statistics", timeout=15000), "Expected the 'Pattern statistics' area to be visible on the page."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    