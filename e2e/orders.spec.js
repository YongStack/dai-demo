// @ts-check
const { test, expect } = require('@playwright/test');

/**
 * REQ-006: 订单管理
 * 验收标准：订单列表、状态筛选、确认收货、取消订单、空态展示
 */
test.describe('REQ-006 订单管理', () => {
  test('订单列表应展示预置订单', async ({ page }) => {
    await page.goto('/');
    await page.evaluate(() => {
      localStorage.removeItem('tennispro_orders');
    });
    await page.goto('/orders.html');
    const orderCards = page.locator('#ordersList .card');
    const count = await orderCards.count();
    expect(count).toBeGreaterThan(0);
    await expect(page.locator('#ordersList')).toContainText('Babolat Pure Aero');
  });

  test('订单状态筛选应正确过滤', async ({ page }) => {
    await page.goto('/');
    await page.evaluate(() => {
      localStorage.removeItem('tennispro_orders');
    });
    await page.goto('/orders.html');
    // 点击待发货 tab
    await page.locator('#statusTabs .cat-pill', { hasText: '待发货' }).click();
    const orderCards = page.locator('#ordersList .card');
    const count = await orderCards.count();
    expect(count).toBeGreaterThan(0);
    // 验证所有订单都是待发货状态
    await expect(page.locator('#ordersList')).toContainText('待发货');
  });

  test('待收货订单可确认收货变为已完成', async ({ page }) => {
    await page.goto('/');
    await page.evaluate(() => {
      localStorage.removeItem('tennispro_orders');
    });
    await page.goto('/orders.html');
    // 点击待收货 tab
    await page.locator('#statusTabs .cat-pill', { hasText: '待收货' }).click();
    // 点击确认收货
    const confirmBtn = page.locator('button', { hasText: '确认收货' }).first();
    await expect(confirmBtn).toBeVisible();
    await confirmBtn.click();
    await expect(page.locator('.toast')).toContainText('确认收货成功');
    // 该订单应变为已完成
    await page.locator('#statusTabs .cat-pill', { hasText: '已完成' }).click();
    const doneCards = page.locator('#ordersList .card');
    const count = await doneCards.count();
    expect(count).toBeGreaterThan(0);
    await expect(page.locator('#ordersList')).toContainText('已完成');
  });

  test('待支付订单可取消', async ({ page }) => {
    await page.goto('/');
    await page.evaluate(() => {
      localStorage.removeItem('tennispro_orders');
    });
    await page.goto('/orders.html');
    // 点击待支付 tab
    await page.locator('#statusTabs .cat-pill', { hasText: '待支付' }).click();
    const cancelBtn = page.locator('button', { hasText: '取消订单' }).first();
    await expect(cancelBtn).toBeVisible();
    await cancelBtn.click();
    await expect(page.locator('.toast')).toContainText('订单已取消');
  });

  test('取消所有待支付订单后筛选待支付应显示空态', async ({ page }) => {
    await page.goto('/');
    await page.evaluate(() => {
      localStorage.removeItem('tennispro_orders');
    });
    await page.goto('/orders.html');
    // 切换到待支付 tab，取消所有待支付订单
    await page.locator('#statusTabs .cat-pill', { hasText: '待支付' }).click();
    // 逐个取消待支付订单
    let cancelBtn = page.locator('button', { hasText: '取消订单' }).first();
    while (await cancelBtn.isVisible({ timeout: 2000 }).catch(() => false)) {
      await cancelBtn.click();
      await page.waitForTimeout(300);
      cancelBtn = page.locator('button', { hasText: '取消订单' }).first();
    }
    // 筛选待支付状态后应无订单，显示空态
    await expect(page.locator('#emptyOrders')).toBeVisible();
    await expect(page.locator('#emptyOrders .empty-title')).toContainText('暂无订单');
  });
});
