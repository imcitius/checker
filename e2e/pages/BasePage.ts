import { type Page, type Locator } from '@playwright/test'

export class BasePage {
  readonly page: Page
  readonly topBar: Locator
  readonly navDashboard: Locator
  readonly navManage: Locator
  readonly navAlerts: Locator
  readonly navSettings: Locator
  readonly logo: Locator
  readonly statusBar: Locator

  constructor(page: Page) {
    this.page = page
    this.topBar = page.locator('nav').first()
    this.navDashboard = page.getByRole('link', { name: 'Dashboard' })
    this.navManage = page.getByRole('link', { name: 'Manage' })
    this.navAlerts = page.getByRole('link', { name: 'Alerts' })
    this.navSettings = page.getByRole('link', { name: 'Settings' })
    this.logo = page.getByRole('link').filter({ hasText: 'Checker' }).first()
    this.statusBar = page.locator('text=Connected').first()
  }

  async goto(path: string) {
    await this.page.goto(path)
    await this.page.waitForLoadState('networkidle')
  }

  async navigateToDashboard() {
    await this.navDashboard.first().click()
    await this.page.waitForURL('/')
  }

  async navigateToManage() {
    await this.navManage.first().click()
    await this.page.waitForURL('/manage')
  }

  async navigateToAlerts() {
    await this.navAlerts.first().click()
    await this.page.waitForURL('/alerts')
  }

  async navigateToSettings() {
    await this.navSettings.first().click()
    await this.page.waitForURL('/settings')
  }

  async getPageTitle(): Promise<string> {
    return this.page.title()
  }
}
