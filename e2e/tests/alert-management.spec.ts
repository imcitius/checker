import { test, expect } from '@playwright/test'
import { BasePage } from '../pages/BasePage'

test.describe('Alert Management', () => {
  let basePage: BasePage

  test.beforeEach(async ({ page }) => {
    basePage = new BasePage(page)
    await basePage.goto('/alerts')
  })

  test('navigate to alerts view', async ({ page }) => {
    // URL should be /alerts
    expect(page.url()).toContain('/alerts')

    // The Alerts nav button should be visible
    const alertsLink = page.getByRole('link', { name: 'Alerts' }).first()
    await expect(alertsLink).toBeVisible()

    // Main content area should be present
    await expect(page.locator('main')).toBeVisible()
  })

  test('alert list renders with header and table', async ({ page }) => {
    // The "Alert History" heading should be visible
    await expect(page.getByText('Alert History')).toBeVisible()

    // The total count badge should be present
    await expect(page.getByText(/total/)).toBeVisible()

    // A table with status/check name headers should be rendered
    const table = page.locator('table').first()
    await expect(table).toBeVisible()

    // Table headers
    await expect(page.getByText('Status').first()).toBeVisible()
    await expect(page.getByText('Check Name').first()).toBeVisible()
  })

  test('alert table shows data or empty state', async ({ page }) => {
    // Wait for potential data load
    await page.waitForTimeout(500)

    const table = page.locator('table').first()
    await expect(table).toBeVisible()

    // Either alert rows exist or "No alerts found" empty state is shown
    const noAlerts = page.getByText('No alerts found')
    const hasNoAlerts = await noAlerts.isVisible().catch(() => false)

    if (hasNoAlerts) {
      await expect(noAlerts).toBeVisible()
    } else {
      // There should be at least one row in tbody
      const rows = table.locator('tbody tr')
      await expect(rows.first()).toBeVisible()
    }
  })

  test('silences section is accessible', async ({ page }) => {
    // The "Active Silences" collapsible section should be present
    await expect(page.getByRole('button', { name: 'Active Silences' })).toBeVisible()

    // The silences table should be visible (section is open by default)
    const silencesTable = page.locator('table').nth(1)
    const hasSilencesTable = await silencesTable.isVisible().catch(() => false)

    if (hasSilencesTable) {
      // Either silences exist or "No active silences" is shown
      const noSilences = page.getByText('No active silences')
      const hasNoSilences = await noSilences.isVisible().catch(() => false)

      if (hasNoSilences) {
        await expect(noSilences).toBeVisible()
      }
    }
  })

  test('refresh button works', async ({ page }) => {
    const refreshButton = page.getByRole('button', { name: /refresh/i })
    await expect(refreshButton).toBeVisible()
    await refreshButton.click()

    // After clicking, the alert history should still be visible
    await expect(page.getByText('Alert History')).toBeVisible()
  })
})
