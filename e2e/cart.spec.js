// @ts-check
const { test, expect } = require('@playwright/test');

/**
 * REQ-003: 购物车
 * 验收标准：加入购物车、数量修改、删除、勾选、空态展示、运费计算
 */
test.describe('REQ-003 购物车', () => {
  test('空购物车应展示空态文案', async ({ page, context }) => {
    // 清空 localStorage 中的购物车
    await page.goto('/');
    await page.evaluate(() => localStorage.removeItem('tennispro_cart'));
    await page.goto('/cart.html');
    await expect(page.locator('#emptyCart')).toBeVisible();
    await expect(page.locator('#emptyCart .empty-title')).toContainText('购物车空空如也');
    await expect(page.locator('#emptyCart .empty-desc')).toContainText('去逛逛吧');
  });

  test('加入购物车后购物车页应展示商品', async ({ page }) => {
    await page.goto('/');
    await page.evaluate(() => localStorage.removeItem('tennispro_cart'));
    // 加入商品
    await page.goto('/product-detail.html?id=1');
    await page.locator('button', { hasText: '加入购物车' }).click();
    await expect(page.locator('.toast')).toContainText('已加入购物车');
    // 查看购物车
    await page.goto('/cart.html');
    const cartItems = page.locator('#cartContent .card');
    const count = await cartItems.count();
    expect(count).toBeGreaterThan(0);
    await expect(page.locator('#cartContent')).toContainText('Babolat Pure Aero 网球拍');
  });

  test('购物车应展示运费计算 - 未满99元收运费', async ({ page }) => {
    await page.goto('/');
    await page.evaluate(() => localStorage.removeItem('tennispro_cart'));
    // 加入价格低于99元的商品 (id=12, Wilson 网球手胶 59元)
    await page.goto('/product-detail.html?id=12');
    await page.locator('button', { hasText: '加入购物车' }).click();
    await page.goto('/cart.html');
    await expect(page.locator('#cartContent')).toContainText('满99元免运费');
    await expect(page.locator('#cartContent')).toContainText('还差');
  });

  test('购物车应展示运费计算 - 满99元免运费', async ({ page }) => {
    await page.goto('/');
    await page.evaluate(() => localStorage.removeItem('tennispro_cart'));
    // 加入价格高于99元的商品 (id=1, Babolat Pure Aero 1899元)
    await page.goto('/product-detail.html?id=1');
    await page.locator('button', { hasText: '加入购物车' }).click();
    await page.goto('/cart.html');
    await expect(page.locator('#cartContent')).toContainText('已满99元');
    await expect(page.locator('#cartContent')).toContainText('免运费');
  });

  test('购物车应可增加商品数量', async ({ page }) => {
    await page.goto('/');
    await page.evaluate(() => localStorage.removeItem('tennispro_cart'));
    await page.goto('/product-detail.html?id=12');
    await page.locator('button', { hasText: '加入购物车' }).click();
    await page.goto('/cart.html');
    const qtySpan = page.locator('#cartContent .card').nth(1).locator('span', { hasText: /^\d+$/ }).first();
    const initialQty = parseInt(await qtySpan.textContent() || '1');
    // 点击+按钮增加数量
    const plusBtn = page.locator('#cartContent .card').nth(1).locator('button', { hasText: '+' });
    await plusBtn.click();
    const newQty = parseInt(await qtySpan.textContent() || '1');
    expect(newQty).toBe(initialQty + 1);
  });

  test('购物车应可删除商品', async ({ page }) => {
    await page.goto('/');
    await page.evaluate(() => localStorage.removeItem('tennispro_cart'));
    await page.goto('/product-detail.html?id=12');
    await page.locator('button', { hasText: '加入购物车' }).click();
    await page.goto('/cart.html');
    const deleteBtn = page.locator('#cartContent button', { hasText: '删除' });
    await deleteBtn.click();
    // 删除后购物车应为空
    await expect(page.locator('#emptyCart')).toBeVisible();
  });

  test('未登录用户点击去结算应跳转登录页', async ({ page }) => {
    await page.goto('/');
    await page.evaluate(() => {
      localStorage.removeItem('tennispro_cart');
      localStorage.removeItem('tennispro_user');
    });
    await page.goto('/product-detail.html?id=1');
    await page.locator('button', { hasText: '加入购物车' }).click();
    await page.goto('/cart.html');
    await page.locator('button', { hasText: '去结算' }).click();
    await expect(page).toHaveURL(/login\.html/);
  });
});
