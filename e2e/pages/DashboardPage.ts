import { type Page, type Locator } from '@playwright/test'
import { BasePage } from './BasePage'

export class DashboardPage extends BasePage {
  readonly metricsRow: Locator
  readonly checkList: Locator
  readonly healthMap: Locator
  readonly eventLog: Locator
  readonly viewListButton: Locator
  readonly viewMapButton: Locator
  readonly searchInput: Locator
  readonly filterCount: Locator

  constructor(page: Page) {
    super(page)
    this.metricsRow = page.locator('main').first()
    this.checkList = page.locator('table').first()
    this.healthMap = page.locator('main').first()
    this.eventLog = page.locator('main').first()
    this.viewListButton = page.getByRole('button', { name: 'List' })
    this.viewMapButton = page.getByRole('button', { name: 'Map' })
    this.searchInput = page.getByPlaceholder(/search/i).first()
    this.filterCount = page.locator('text=/\\d+ checks?/')
  }

  async goto() {
    await super.goto('/')
  }

  async switchToMapView() {
    await this.viewMapButton.click()
  }

  async switchToListView() {
    await this.viewListButton.click()
  }
}
