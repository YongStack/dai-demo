// @ts-check
const { test, expect } = require('@playwright/test');

/**
 * REQ-001: 商品浏览与展示 - 商品列表页
 * 验收标准：分类筛选、排序、分页浏览、商品卡片信息
 */
test.describe('REQ-001 商品浏览 - 商品列表页', () => {
  test('商品列表页应展示全部商品', async ({ page }) => {
    await page.goto('/products.html');
    await expect(page.locator('#pageTitle')).toContainText('全部商品');
    const productCards = page.locator('#productsGrid .product-card');
    const count = await productCards.count();
    expect(count).toBeGreaterThan(0);
    await expect(page.locator('#pageSubtitle')).toContainText('件商品');
  });

  test('分类筛选应正确过滤商品', async ({ page }) => {
    await page.goto('/products.html');
    // 点击网球鞋分类
    const shoesPill = page.locator('#catFilters .cat-pill', { hasText: '网球鞋' });
    await shoesPill.click();
    await expect(page.locator('#pageTitle')).toContainText('网球鞋');
    const productCards = page.locator('#productsGrid .product-card');
    const count = await productCards.count();
    expect(count).toBeGreaterThan(0);
    await expect(page.locator('#pageSubtitle')).toContainText('件商品');
  });

  test('价格升序排序应正确排列商品', async ({ page }) => {
    await page.goto('/products.html');
    await page.locator('#sortSelect').selectOption('price-asc');
    const priceElements = page.locator('.product-card-price');
    const prices = await priceElements.allTextContents();
    const numericPrices = prices.map(p => parseFloat(p.replace('¥', '')));
    for (let i = 1; i < numericPrices.length; i++) {
      expect(numericPrices[i]).toBeGreaterThanOrEqual(numericPrices[i - 1]);
    }
  });

  test('价格降序排序应正确排列商品', async ({ page }) => {
    await page.goto('/products.html');
    await page.locator('#sortSelect').selectOption('price-desc');
    const priceElements = page.locator('.product-card-price');
    const prices = await priceElements.allTextContents();
    const numericPrices = prices.map(p => parseFloat(p.replace('¥', '')));
    for (let i = 1; i < numericPrices.length; i++) {
      expect(numericPrices[i]).toBeLessThanOrEqual(numericPrices[i - 1]);
    }
  });

  test('按销量排序应正确排列商品', async ({ page }) => {
    await page.goto('/products.html');
    await page.locator('#sortSelect').selectOption('sales');
    const salesElements = page.locator('.product-card-sales');
    const salesTexts = await salesElements.allTextContents();
    const sales = salesTexts.map(s => parseInt(s.match(/\d+/)?.[0] || '0'));
    for (let i = 1; i < sales.length; i++) {
      expect(sales[i]).toBeLessThanOrEqual(sales[i - 1]);
    }
  });

  test('商品卡片应展示名称和价格', async ({ page }) => {
    await page.goto('/products.html');
    const firstCard = page.locator('#productsGrid .product-card').first();
    await expect(firstCard.locator('.product-card-name')).toBeVisible();
    await expect(firstCard.locator('.product-card-price')).toContainText('¥');
  });

  test('点击商品卡片应跳转到商品详情页', async ({ page }) => {
    await page.goto('/products.html');
    const firstCard = page.locator('#productsGrid .product-card').first();
    await firstCard.click();
    await expect(page).toHaveURL(/product-detail\.html\?id=\d+/);
  });
});
