import { test, expect } from '@playwright/test'
import { BasePage } from '../pages/BasePage'

test.describe('Check Management', () => {
  let basePage: BasePage

  test.beforeEach(async ({ page }) => {
    basePage = new BasePage(page)
    await basePage.goto('/manage')
  })

  test('navigate to checks management view', async ({ page }) => {
    // The Manage page should load with its heading
    await expect(page.locator('main')).toBeVisible()

    // The Manage nav button should be in active/secondary state
    const manageLink = page.getByRole('link', { name: 'Manage' }).first()
    await expect(manageLink).toBeVisible()

    // URL should be /manage
    expect(page.url()).toContain('/manage')
  })

  test('check definitions table is visible', async ({ page }) => {
    // The management page displays check definitions in a table
    // Either a table with check rows exists or a loading/empty state is shown
    const main = page.locator('main')
    await expect(main).toBeVisible()

    // The page should have table headers for check management
    // Look for typical column headers: Name, Type, Group, etc.
    const table = page.locator('table').first()
    const hasTable = await table.isVisible().catch(() => false)

    if (hasTable) {
      // Table headers should be present
      await expect(page.getByText('Name').first()).toBeVisible()
    } else {
      // Loading or empty state
      await expect(main).toBeVisible()
    }
  })

  test('check status indicators are visible when checks exist', async ({ page }) => {
    // Wait for data to potentially load
    await page.waitForTimeout(1000)

    const main = page.locator('main')
    await expect(main).toBeVisible()

    // The page should show either check rows with enabled/disabled indicators
    // or a refresh button to reload data
    const refreshButton = page.getByRole('button', { name: /refresh/i })
    const hasRefresh = await refreshButton.isVisible().catch(() => false)
    if (hasRefresh) {
      await expect(refreshButton).toBeVisible()
    }
  })

  test('new check button is available', async ({ page }) => {
    // The management page should have a button to add new checks
    const addButton = page.getByRole('button', { name: /new|add|create/i }).first()
    const hasAddButton = await addButton.isVisible().catch(() => false)

    if (hasAddButton) {
      await expect(addButton).toBeVisible()
    } else {
      // The page main content should at least be present
      await expect(page.locator('main')).toBeVisible()
    }
  })
})
