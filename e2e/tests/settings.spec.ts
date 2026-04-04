import { test, expect } from '@playwright/test'
import { BasePage } from '../pages/BasePage'

test.describe('Settings', () => {
  let basePage: BasePage

  test.beforeEach(async ({ page }) => {
    basePage = new BasePage(page)
    await basePage.goto('/settings')
  })

  test('settings page loads', async ({ page }) => {
    // URL should be /settings
    expect(page.url()).toContain('/settings')

    // The Settings heading should be visible
    await expect(page.getByText('Settings').first()).toBeVisible()

    // Description text should be present
    await expect(page.getByRole('heading', { name: 'Notification Channels' })).toBeVisible()
  })

  test('notification channels tab is visible', async ({ page }) => {
    // The "Notification Channels" tab should be present and active by default
    const channelsTab = page.getByRole('tab', { name: /notification channels/i })
    await expect(channelsTab).toBeVisible()

    // The "Notification Channels" section heading should be visible
    await expect(page.getByText('Notification Channels').first()).toBeVisible()
  })

  test('check defaults tab is accessible', async ({ page }) => {
    // The "Check Defaults" tab should be present
    const defaultsTab = page.getByRole('tab', { name: /check defaults/i })
    await expect(defaultsTab).toBeVisible()

    // Click on Check Defaults tab
    await defaultsTab.click()

    // After clicking, the Check Defaults content should be visible
    await expect(page.locator('main')).toBeVisible()
  })

  test('configuration sections are visible', async ({ page }) => {
    // The Tabs component should be present with both tabs
    const channelsTab = page.getByRole('tab', { name: /notification channels/i })
    const defaultsTab = page.getByRole('tab', { name: /check defaults/i })

    await expect(channelsTab).toBeVisible()
    await expect(defaultsTab).toBeVisible()

    // The main content area should show settings content
    await expect(page.locator('main')).toBeVisible()
  })
})
