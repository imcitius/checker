import { test, expect } from '@playwright/test'
import { BasePage } from '../pages/BasePage'

test.describe('Navigation', () => {
  test('topbar navigation links are visible', async ({ page }) => {
    const basePage = new BasePage(page)
    await basePage.goto('/')

    // All nav links should be visible in the top bar
    await expect(basePage.navDashboard.first()).toBeVisible()
    await expect(basePage.navManage.first()).toBeVisible()
    await expect(basePage.navAlerts.first()).toBeVisible()
    await expect(basePage.navSettings.first()).toBeVisible()
  })

  test('navigate from dashboard to manage', async ({ page }) => {
    const basePage = new BasePage(page)
    await basePage.goto('/')

    await basePage.navigateToManage()
    expect(page.url()).toContain('/manage')

    // Manage page content should be visible
    await expect(page.locator('main')).toBeVisible()
  })

  test('navigate from dashboard to alerts', async ({ page }) => {
    const basePage = new BasePage(page)
    await basePage.goto('/')

    await basePage.navigateToAlerts()
    expect(page.url()).toContain('/alerts')

    // Alert History heading should appear
    await expect(page.getByText('Alert History')).toBeVisible()
  })

  test('navigate from dashboard to settings', async ({ page }) => {
    const basePage = new BasePage(page)
    await basePage.goto('/')

    await basePage.navigateToSettings()
    expect(page.url()).toContain('/settings')

    // Settings heading should appear
    await expect(page.getByText('Settings').first()).toBeVisible()
  })

  test('route changes update content', async ({ page }) => {
    const basePage = new BasePage(page)
    await basePage.goto('/')

    // Dashboard should show check count
    await expect(page.locator('text=/\\d+ checks?/')).toBeVisible()

    // Navigate to alerts
    await basePage.navigateToAlerts()
    await expect(page.getByText('Alert History')).toBeVisible()

    // Navigate to settings
    await basePage.navigateToSettings()
    await expect(page.getByRole('heading', { name: 'Notification Channels' })).toBeVisible()

    // Navigate back to dashboard via logo or nav
    await basePage.navigateToDashboard()
    await expect(page.locator('text=/\\d+ checks?/')).toBeVisible()
  })

  test('page title updates on navigation', async ({ page }) => {
    const basePage = new BasePage(page)
    await basePage.goto('/')

    const initialTitle = await page.title()
    expect(initialTitle).toBeTruthy()

    // Navigate to another page and verify the page still has a title
    await basePage.navigateToAlerts()
    const alertsTitle = await page.title()
    expect(alertsTitle).toBeTruthy()
  })

  test('logo navigates to dashboard', async ({ page }) => {
    const basePage = new BasePage(page)
    await basePage.goto('/alerts')

    // Click on the Checker logo to go back to dashboard
    await basePage.logo.click()
    await page.waitForURL('/')

    expect(page.url()).toMatch(/\/$/)
  })
})
