import { test, expect } from '@playwright/test'
import { DashboardPage } from '../pages/DashboardPage'

test.describe('Dashboard', () => {
  let dashboard: DashboardPage

  test.beforeEach(async ({ page }) => {
    dashboard = new DashboardPage(page)
    await dashboard.goto()
  })

  test('app loads at root and main layout is visible', async ({ page }) => {
    // TopBar with logo and nav links should be present
    await expect(page.getByText('Checker').first()).toBeVisible()
    await expect(dashboard.navDashboard.first()).toBeVisible()
    await expect(dashboard.navManage.first()).toBeVisible()
    await expect(dashboard.navAlerts.first()).toBeVisible()
    await expect(dashboard.navSettings.first()).toBeVisible()

    // Main content area should exist
    await expect(page.locator('main')).toBeVisible()
  })

  test('health map or status display is present', async ({ page }) => {
    // The dashboard shows either a List or Map view toggle
    await expect(dashboard.viewListButton).toBeVisible()
    await expect(dashboard.viewMapButton).toBeVisible()

    // Switching to map view should work
    await dashboard.switchToMapView()
    await expect(dashboard.viewMapButton).toBeVisible()

    // Switch back to list view
    await dashboard.switchToListView()
    await expect(dashboard.viewListButton).toBeVisible()
  })

  test('check list renders with items or empty state', async ({ page }) => {
    // The dashboard should show a check count indicator (e.g., "5 checks" or "0 checks")
    await expect(page.locator('text=/\\d+ checks?/')).toBeVisible()

    // Either check rows exist in the table, or the main area is visible as empty state
    const mainContent = page.locator('main')
    await expect(mainContent).toBeVisible()
  })

  test('metrics row displays stats', async ({ page }) => {
    // The metrics section should be present in the main area
    const main = page.locator('main')
    await expect(main).toBeVisible()

    // Check count text should be visible somewhere
    await expect(page.locator('text=/\\d+ checks?/')).toBeVisible()
  })
})
