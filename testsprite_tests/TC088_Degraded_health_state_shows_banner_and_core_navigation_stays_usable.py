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
        
        # -> Search the dashboard for the word 'degraded', then click the '媒體庫' (Library) link to open the Library page and check for media poster cards.
        # 媒體庫 link
        elem = page.get_by_text('內容', exact=True).locator("xpath=ancestor-or-self::*[.//a][1]").get_by_role('link', name='媒體庫', exact=True)
        await elem.click(timeout=10000)
        
        # -> Navigate to the homepage (root) and inspect the page for a visible degraded state indicator and the text 'degraded'.
        await page.goto("http://localhost:8090")
        try:
            await page.wait_for_load_state("domcontentloaded", timeout=5000)
        except Exception:
            pass
        
        # -> Search the page for the text 'degraded' (substring 'degrad'), then click the '媒體庫' (Library) link to open the Library page.
        # 媒體庫 link
        elem = page.get_by_text('內容', exact=True).locator("xpath=ancestor-or-self::*[.//a][1]").get_by_role('link', name='媒體庫', exact=True)
        await elem.click(timeout=10000)
        
        # -> Open the homepage (首頁) and check the page for a degraded indicator or the text 'degraded'.
        await page.goto("http://localhost:8090")
        try:
            await page.wait_for_load_state("domcontentloaded", timeout=5000)
        except Exception:
            pass
        
        # -> Search the homepage for the text 'degrad' and the Chinese term '降級', then click the '媒體庫' (Library) link to open the Library page.
        # 媒體庫 link
        elem = page.get_by_text('內容', exact=True).locator("xpath=ancestor-or-self::*[.//a][1]").get_by_role('link', name='媒體庫', exact=True)
        await elem.click(timeout=10000)
        
        # -> Open the homepage (首頁) and inspect the service status area and page text for a degraded indicator or the words 'degraded' / '降級'.
        await page.goto("http://localhost:8090/")
        try:
            await page.wait_for_load_state("domcontentloaded", timeout=5000)
        except Exception:
            pass
        
        # --> Assertions to verify final state
        
        # --> Verify element "degraded state indicator" is visible
        # Assert: Expected TMDb API status icon to display a degraded state.
        await expect(page.locator("xpath=/html/body/div/div/div/div[1]/aside/div[2]/div[2]/span[2]/span[1]").nth(0)).to_have_attribute("aria-label", "TMDb API\uff1a\u964d\u7d1a", timeout=15000), "Expected TMDb API status icon to display a degraded state."
        # Assert: Expected Douban Scraper status icon to display a degraded state.
        await expect(page.locator("xpath=/html/body/div/div/div/div[1]/aside/div[2]/div[2]/span[2]/span[2]").nth(0)).to_have_attribute("aria-label", "Douban Scraper\uff1a\u964d\u7d1a", timeout=15000), "Expected Douban Scraper status icon to display a degraded state."
        # Assert: Expected Wikipedia API status icon to display a degraded state.
        await expect(page.locator("xpath=/html/body/div/div/div/div[1]/aside/div[2]/div[2]/span[2]/span[3]").nth(0)).to_have_attribute("aria-label", "Wikipedia API\uff1a\u964d\u7d1a", timeout=15000), "Expected Wikipedia API status icon to display a degraded state."
        # Assert: Expected AI Parser status icon to display a degraded state.
        await expect(page.locator("xpath=/html/body/div/div/div/div[1]/aside/div[2]/div[2]/span[2]/span[4]").nth(0)).to_have_attribute("aria-label", "AI Parser\uff1a\u964d\u7d1a", timeout=15000), "Expected AI Parser status icon to display a degraded state."
        # Assert: Expected qBittorrent status icon to display a degraded state.
        await expect(page.locator("xpath=/html/body/div/div/div/div[1]/aside/div[2]/div[2]/span[2]/span[5]").nth(0)).to_have_attribute("aria-label", "qBittorrent\uff1a\u964d\u7d1a", timeout=15000), "Expected qBittorrent status icon to display a degraded state."
        
        # --> Verify text "degraded" is visible
        # Assert: Expected text "degraded" to be visible on the page.
        await expect(page.locator("xpath=/html/body/div").nth(0)).to_contain_text("degraded", timeout=15000), "Expected text \"degraded\" to be visible on the page."
        
        # --> Verify a grid of media poster cards is still visible (core feature unaffected by degradation)
        # Assert: Expected the browser to be on /library so the media poster grid could be verified.
        await expect(page).to_have_url(re.compile("/library"), timeout=15000), "Expected the browser to be on /library so the media poster grid could be verified."
        await asyncio.sleep(5)

    finally:
        if context:
            await context.close()
        if browser:
            await browser.close()
        if pw:
            await pw.stop()

asyncio.run(run_test())
    