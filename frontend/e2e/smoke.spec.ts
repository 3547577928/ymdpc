import { test, expect } from '@playwright/test'

// 冒烟测试：覆盖核心用户流程，确保关键路径不被回归破坏
test.describe('smoke', () => {
  test('home page loads', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('text=Quiet Signal')).toBeVisible()
  })

  test('posts page loads with content', async ({ page }) => {
    await page.goto('/posts')
    await expect(page.locator('h1')).toContainText(/社区文章|热门讨论/)
    // 分页或空态都算加载成功
    await expect(page.locator('.feed-content, .empty-state')).toBeVisible()
  })

  test('about page loads', async ({ page }) => {
    await page.goto('/about')
    await expect(page.locator('text=Quiet Signal')).toBeVisible()
  })

  test('forum page loads', async ({ page }) => {
    await page.goto('/forum')
    await expect(page.locator('h1')).toContainText('论坛')
  })

  test('search page accepts query', async ({ page }) => {
    await page.goto('/search')
    const input = page.locator('input[placeholder*="关键词"]')
    await input.fill('界面')
    // 防抖 300ms + 请求
    await page.waitForTimeout(800)
    // 搜索页应显示结果或空态，不应报错
    await expect(page.locator('.search-results, .empty-state')).toBeVisible()
  })

  test('nav links are functional', async ({ page }) => {
    await page.goto('/')
    await page.click('a[aria-label="文章"]')
    await expect(page).toHaveURL(/\/posts/)
    await page.click('a[aria-label="论坛"]')
    await expect(page).toHaveURL(/\/forum/)
    await page.click('a[aria-label="关于"]')
    await expect(page).toHaveURL(/\/about/)
    await page.click('a[aria-label="搜索"]')
    await expect(page).toHaveURL(/\/search/)
  })
})
