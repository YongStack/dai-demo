// @ts-check
const { test, expect } = require('@playwright/test');

/**
 * REQ-004: 下单与结算
 * 验收标准：地址选择、运费计算（满99免运费/未满99收10元）、订单创建
 * REQ-005: 支付
 * 验收标准：支付方式选择、支付成功跳转、支付幂等性
 */
test.describe('REQ-004/005 下单结算与支付流程', () => {
  test('完整购物流程：加购→登录→结算→支付→成功', async ({ page }) => {
    // 清理状态
    await page.goto('/');
    await page.evaluate(() => {
      localStorage.removeItem('tennispro_cart');
      localStorage.removeItem('tennispro_user');
      localStorage.removeItem('tennispro_orders');
      sessionStorage.removeItem('tennispro_order');
    });

    // 1. 加入购物车
    await page.goto('/product-detail.html?id=1');
    await page.locator('button', { hasText: '加入购物车' }).click();
    await expect(page.locator('.toast')).toContainText('已加入购物车');

    // 2. 去购物车
    await page.goto('/cart.html');
    await expect(page.locator('#cartContent')).toContainText('Babolat Pure Aero 网球拍');

    // 3. 去结算 - 未登录应跳转登录页
    await page.locator('button', { hasText: '去结算' }).click();
    await expect(page).toHaveURL(/login\.html/);

    // 4. 登录
    await page.locator('#phoneInput').fill('13800138000');
    await page.locator('#sendCodeBtn').click();
    await page.locator('#codeInput').fill('123456');
    await page.locator('button[type="submit"]').click();
    await expect(page.locator('.toast')).toContainText('登录成功');

    // 5. 登录后回到购物车并结算
    await page.goto('/cart.html');
    await page.locator('button', { hasText: '去结算' }).click();
    await expect(page).toHaveURL(/checkout\.html/);

    // 6. 结算页应显示地址、商品清单、费用明细
    await expect(page.locator('h1')).toContainText('确认订单');
    await expect(page.locator('#addressList')).toBeVisible();
    await expect(page.locator('#orderItems')).toBeVisible();
    await expect(page.locator('#totalAmount')).toContainText('¥');

    // 7. 提交订单
    await page.locator('button', { hasText: '提交订单' }).click();
    await expect(page.locator('.toast')).toContainText('订单创建成功');

    // 8. 跳转到支付页
    await expect(page).toHaveURL(/payment\.html/, { timeout: 5000 });
    await expect(page.locator('#payAmount')).toContainText('¥');
    await expect(page.locator('#orderNo')).not.toBeEmpty();

    // 9. 确认支付
    await page.locator('button', { hasText: '确认支付' }).click();
    await expect(page.locator('.toast')).toContainText('支付处理中');

    // 10. 支付成功页
    await expect(page).toHaveURL(/payment-success\.html/, { timeout: 10000 });
    await expect(page.locator('h1')).toContainText('支付成功');
    await expect(page.locator('body')).toContainText('待发货');

    // 11. 验证订单出现在我的订单中
    await page.goto('/orders.html');
    const orderCards = page.locator('#ordersList .card');
    const count = await orderCards.count();
    expect(count).toBeGreaterThan(0);
    // 最新订单应包含刚购买的商品
    await expect(page.locator('#ordersList').first()).toContainText('Babolat Pure Aero');
  });

  test('结算页运费计算 - 满99元免运费', async ({ page }) => {
    await page.goto('/');
    await page.evaluate(() => {
      localStorage.removeItem('tennispro_cart');
      localStorage.removeItem('tennispro_orders');
    });
    // 设置用户为已登录
    await page.evaluate(() => {
      localStorage.setItem('tennispro_user', JSON.stringify({phone:'13800138000', name:'测试用户'}));
    });

    // 加入价格高于99元的商品
    await page.goto('/product-detail.html?id=1');
    await page.locator('button', { hasText: '加入购物车' }).click();
    await page.goto('/cart.html');
    await page.locator('button', { hasText: '去结算' }).click();
    await expect(page).toHaveURL(/checkout\.html/);
    await expect(page.locator('#feeDetails')).toContainText('已满99元');
    await expect(page.locator('#feeDetails')).toContainText('免运费');
  });

  test('结算页运费计算 - 未满99元收10元运费', async ({ page }) => {
    await page.goto('/');
    await page.evaluate(() => {
      localStorage.removeItem('tennispro_cart');
      localStorage.removeItem('tennispro_orders');
    });
    await page.evaluate(() => {
      localStorage.setItem('tennispro_user', JSON.stringify({phone:'13800138000', name:'测试用户'}));
    });

    // 加入价格低于99元的商品 (id=12, Wilson 网球手胶 59元)
    await page.goto('/product-detail.html?id=12');
    await page.locator('button', { hasText: '加入购物车' }).click();
    await page.goto('/cart.html');
    await page.locator('button', { hasText: '去结算' }).click();
    await expect(page).toHaveURL(/checkout\.html/);
    await expect(page.locator('#feeDetails')).toContainText('满99元免运费');
    await expect(page.locator('#feeDetails')).toContainText('还差');
    // 验证总金额包含10元运费
    const totalText = await page.locator('#totalAmount').textContent();
    const total = parseFloat(totalText?.replace('¥', '') || '0');
    // 59 + 10 = 69
    expect(total).toBe(69);
  });

  test('结算页可新增收货地址', async ({ page }) => {
    await page.goto('/');
    await page.evaluate(() => {
      localStorage.removeItem('tennispro_cart');
      localStorage.removeItem('tennispro_addresses');
      localStorage.removeItem('tennispro_orders');
    });
    await page.evaluate(() => {
      localStorage.setItem('tennispro_user', JSON.stringify({phone:'13800138000', name:'测试用户'}));
    });

    await page.goto('/product-detail.html?id=1');
    await page.locator('button', { hasText: '加入购物车' }).click();
    await page.goto('/cart.html');
    await page.locator('button', { hasText: '去结算' }).click();
    await expect(page).toHaveURL(/checkout\.html/);

    // 点击新增地址
    await page.locator('button', { hasText: '新增地址' }).click();
    await page.locator('#addrName').fill('王五');
    await page.locator('#addrPhone').fill('13700137000');
    await page.locator('#addrDetail').fill('广州市天河区珠江新城');
    await page.locator('button', { hasText: '保存地址' }).click();
    await expect(page.locator('.toast')).toContainText('地址已添加');
    await expect(page.locator('#addressList')).toContainText('王五');
  });
});
