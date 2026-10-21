// @ts-check
const { test, expect } = require('@playwright/test');

/**
 * REQ-001: 商品浏览与展示
 * 验收标准：首页展示推荐商品轮播和六大品类分类入口、热销商品
 */
test.describe('REQ-001 商品浏览与展示 - 首页', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
  });

  test('首页应展示轮播图', async ({ page }) => {
    const carousel = page.locator('#heroCarousel');
    await expect(carousel).toBeVisible();
    const slides = page.locator('.carousel-slide');
    await expect(slides).toHaveCount(3);
  });

  test('首页应展示六大品类分类入口', async ({ page }) => {
    const categoryGrid = page.locator('#categoryGrid');
    await expect(categoryGrid).toBeVisible();
    const categoryCards = page.locator('#categoryGrid .card');
    await expect(categoryCards).toHaveCount(6);
    // 验证包含网球拍分类
    const racketCategory = page.locator('#categoryGrid .card', { hasText: '网球拍' });
    await expect(racketCategory).toBeVisible();
  });

  test('首页应展示热销商品', async ({ page }) => {
    const hotProducts = page.locator('#hotProducts .product-card');
    const count = await hotProducts.count();
    expect(count).toBeGreaterThan(0);
  });

  test('首页应展示Slogan文案"你的网球装备，一站搞定"', async ({ page }) => {
    await expect(page.locator('body')).toContainText('你的网球装备');
    await expect(page.locator('body')).toContainText('一站搞定');
  });

  test('点击分类入口应跳转到对应分类商品列表', async ({ page }) => {
    const racketLink = page.locator('#categoryGrid .card', { hasText: '网球拍' });
    await racketLink.click();
    await expect(page).toHaveURL(/products\.html\?cat=rackets/);
    await expect(page.locator('#pageTitle')).toContainText('网球拍');
  });

  test('导航栏应包含搜索框和购物车图标', async ({ page }) => {
    await expect(page.locator('#navSearch')).toBeVisible();
    await expect(page.locator('a[href="cart.html"]')).toBeVisible();
  });
});
