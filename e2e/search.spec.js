// @ts-check
const { test, expect } = require('@playwright/test');

/**
 * REQ-002: 商品搜索
 * 验收标准：关键词搜索返回匹配商品、无结果展示空态页和热门推荐
 */
test.describe('REQ-002 商品搜索', () => {
  test('搜索有结果时应展示匹配商品', async ({ page }) => {
    await page.goto('/search.html?q=Babolat');
    await expect(page.locator('#searchTitle')).toContainText('Babolat');
    const results = page.locator('#searchResults .product-card');
    const count = await results.count();
    expect(count).toBeGreaterThan(0);
    await expect(page.locator('#searchSubtitle')).toContainText('找到');
  });

  test('搜索无结果时应展示空态页和热门推荐', async ({ page }) => {
    await page.goto('/search.html?q=zzzznonexist');
    await expect(page.locator('#emptyState')).toBeVisible();
    await expect(page.locator('#emptyState .empty-title')).toContainText('没有找到相关商品');
    await expect(page.locator('#emptyState .empty-desc')).toContainText('看看这些热门推荐');
    // 热门推荐区域应可见
    await expect(page.locator('#hotSection')).toBeVisible();
    const hotProducts = page.locator('#hotProducts .product-card');
    const count = await hotProducts.count();
    expect(count).toBeGreaterThan(0);
  });

  test('导航栏搜索框应能触发搜索', async ({ page }) => {
    await page.goto('/');
    await page.locator('#navSearch').fill('Wilson');
    await page.locator('#navSearch').press('Enter');
    await expect(page).toHaveURL(/search\.html\?q=Wilson/);
  });

  test('搜索结果应支持排序', async ({ page }) => {
    await page.goto('/search.html?q=Babolat');
    await page.locator('#sortSelect').selectOption('price-asc');
    const priceElements = page.locator('.product-card-price');
    const prices = await priceElements.allTextContents();
    const numericPrices = prices.map(p => parseFloat(p.replace('¥', '')));
    for (let i = 1; i < numericPrices.length; i++) {
      expect(numericPrices[i]).toBeGreaterThanOrEqual(numericPrices[i - 1]);
    }
  });
});
