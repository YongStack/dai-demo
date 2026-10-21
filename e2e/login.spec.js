// @ts-check
const { test, expect } = require('@playwright/test');

/**
 * REQ-007: 用户注册登录
 * 验收标准：手机号+验证码注册登录、登录拦截后返回原页面
 */
test.describe('REQ-007 用户注册登录', () => {
  test('登录页应展示手机号和验证码输入框', async ({ page }) => {
    await page.goto('/login.html');
    await expect(page.locator('#phoneInput')).toBeVisible();
    await expect(page.locator('#codeInput')).toBeVisible();
    await expect(page.locator('#sendCodeBtn')).toBeVisible();
    await expect(page.locator('button[type="submit"]')).toContainText('登录');
  });

  test('手机号格式不正确应提示', async ({ page }) => {
    await page.goto('/login.html');
    await page.locator('#phoneInput').fill('123');
    await page.locator('#sendCodeBtn').click();
    await expect(page.locator('.toast')).toContainText('请输入正确的手机号');
  });

  test('获取验证码后按钮应倒计时', async ({ page }) => {
    await page.goto('/login.html');
    await page.locator('#phoneInput').fill('13800138000');
    await page.locator('#sendCodeBtn').click();
    await expect(page.locator('.toast')).toContainText('验证码已发送');
    // 按钮应变为倒计时状态
    await expect(page.locator('#sendCodeBtn')).toContainText(/s 后重发/);
  });

  test('正确输入手机号和验证码应登录成功', async ({ page }) => {
    await page.goto('/login.html');
    await page.locator('#phoneInput').fill('13800138000');
    await page.locator('#sendCodeBtn').click();
    await page.locator('#codeInput').fill('123456');
    await page.locator('button[type="submit"]').click();
    await expect(page.locator('.toast')).toContainText('登录成功');
  });

  test('登录成功后应跳转返回页面', async ({ page }) => {
    await page.goto('/login.html?returnUrl=cart.html');
    await page.locator('#phoneInput').fill('13800138000');
    await page.locator('#sendCodeBtn').click();
    await page.locator('#codeInput').fill('123456');
    await page.locator('button[type="submit"]').click();
    await expect(page).toHaveURL(/cart\.html/, { timeout: 5000 });
  });

  test('未登录用户点击去结算应跳转登录页带returnUrl', async ({ page }) => {
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
    const url = page.url();
    expect(url).toContain('returnUrl');
  });
});
