// @ts-check
const { test, expect } = require('@playwright/test');

/**
 * REQ-001: 商品浏览 - 商品详情页
 * 验收标准：多图展示、规格选择、库存状态、加购/立即购买按钮
 * REQ-003: 购物车 - 加入购物车
 */
test.describe('REQ-001/003 商品详情页', () => {
  test('有库存商品详情页应展示完整信息且加购按钮可点击', async ({ page }) => {
    await page.goto('/product-detail.html?id=1');
    await expect(page.locator('h1')).toContainText('Babolat Pure Aero 网球拍');
    await expect(page.locator('body')).toContainText('¥1899');
    // 规格选项
    await expect(page.locator('#specContainer .cat-pill')).toHaveCount(4);
    // 加购按钮可点击
    const addCartBtn = page.locator('button', { hasText: '加入购物车' });
    await expect(addCartBtn).toBeVisible();
    await expect(addCartBtn).not.toBeDisabled();
    // 立即购买按钮可点击
    const buyNowBtn = page.locator('button', { hasText: '立即购买' });
    await expect(buyNowBtn).toBeVisible();
    await expect(buyNowBtn).not.toBeDisabled();
  });

  test('缺货商品应显示缺货提示且加购按钮不可点击', async ({ page }) => {
    // Product id=3 (Head Speed Pro) has stock=0
    await page.goto('/product-detail.html?id=3');
    await expect(page.locator('body')).toContainText('该商品暂时缺货');
    const addCartBtn = page.locator('button', { hasText: '加入购物车' });
    await expect(addCartBtn).toBeDisabled();
  });

  test('选择规格后加入购物车应显示成功提示', async ({ page, context }) => {
    await page.goto('/product-detail.html?id=1');
    // 选择规格
    const spec300g = page.locator('#specContainer .cat-pill', { hasText: '300g' });
    await spec300g.click();
    await expect(spec300g).toHaveClass(/active/);
    // 加入购物车
    await page.locator('button', { hasText: '加入购物车' }).click();
    // 验证Toast提示
    await expect(page.locator('.toast')).toBeVisible();
    await expect(page.locator('.toast')).toContainText('已加入购物车');
  });

  test('加入购物车后购物车角标应更新', async ({ page }) => {
    await page.goto('/product-detail.html?id=1');
    await page.locator('button', { hasText: '加入购物车' }).click();
    await expect(page.locator('.toast')).toContainText('已加入购物车');
    // 验证购物车角标显示
    await expect(page.locator('.cart-badge')).toBeVisible();
    const badgeText = await page.locator('.cart-badge').textContent();
    expect(parseInt(badgeText || '0')).toBeGreaterThan(0);
  });

  test('商品详情页应展示商品描述', async ({ page }) => {
    await page.goto('/product-detail.html?id=1');
    await expect(page.locator('body')).toContainText('商品描述');
    await expect(page.locator('body')).toContainText('品牌正品');
  });

  test('商品详情页应展示相关推荐', async ({ page }) => {
    await page.goto('/product-detail.html?id=1');
    const relatedProducts = page.locator('#relatedProducts .product-card');
    const count = await relatedProducts.count();
    expect(count).toBeGreaterThan(0);
  });
});
