// 用独立 playwright-cli 会话运行；全部业务请求由只读响应桩处理，不使用真实会话或模型。
// oxlint-disable-next-line no-unused-expressions -- playwright-cli 通过函数表达式执行此可重复入口。
async (page) => {
  const assert = (condition, message) => { if (!condition) throw new Error(message); };
  const calls = [], errors = [];
  let mode = 'ready';
  const list = ['get_current_time', 'text_stats', 'json_formatter', 'kb_search', 'kb_analyze'].map(name => ({ name, description: '', parameters_schema: '{bad <svg onload=evil()>' }));
  page.on('pageerror', error => errors.push(error.message));
  await page.addInitScript(() => localStorage.setItem('go-zero-auth', JSON.stringify({ state: {
    token: 'tools-browser-fixture-access', refreshToken: 'tools-browser-fixture-refresh', expiresAt: Date.now() / 1000 + 3600,
    refreshExpiresAt: Date.now() / 1000 + 7200, sessionId: 'tools-browser-fixture-session',
  }, version: 0 })));
  await page.route('**/v1/**', async route => {
    const request = route.request(), path = request.url().replace(/^https?:\/\/[^/]+/, '').split('?')[0];
    calls.push({ method: request.method(), path });
    const send = (code, result = {}, msg = '响应桩') => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ code, result, msg }) });
    if (request.method() !== 'GET') return route.abort('blockedbyclient');
    if (path === '/v1/sys/dept/my') return send(200, { list: [] });
    if (path === '/v1/sys/menu/my/perm-codes') return send(200, { codes: ['sys:ai:tool:list', 'sys:ai:workflow:list', 'sys:ai:workflow:create'] });
    if (path === '/v1/sys/menu/my') return send(200, { list: [{ id: 901, parent_id: 0, title: '工具与能力', name: 'ai_tool', type: 2, path: '/ai/tools', component: 'ai_tool', perm_code: '', icon: '', sort: 0, visible: true, status: 1, created_at: 0 }] });
    if (path === '/v1/sys/profile/') return send(200, { user: { id: 1, username: 'tools-fixture', nickname: '工具回归', email: '', phone: '', avatar_url: '', dept_id: 0, role_ids: [] } });
    if (path === '/v1/sys/ai/tools') {
      if (mode === 'failure') return send(503, {}, '响应桩工具目录暂不可用');
      if (mode === 'empty') return send(200, { list: [], capabilities: { retrieval_configured: true, analysis_configured: true, ingestion_mode: 'background' } });
      if (mode === 'unknown') return send(200, { list });
      return send(200, { list, capabilities: { retrieval_configured: true, analysis_configured: mode !== 'missing', ingestion_mode: 'background' } });
    }
    return send(404, {}, '响应桩未定义的只读路径');
  });
  await page.setViewportSize({ width: 1440, height: 1000 });
  await page.goto('http://127.0.0.1:5999/ai/tools');
  await page.getByRole('heading', { name: '资料分析', exact: true }).waitFor();
  const search = page.getByRole('textbox', { name: '搜索工具名称或说明', exact: true });
  await search.fill('分析资料');
  assert(await page.locator('article').count() === 1, '中文用途搜索应只保留资料分析');
  await page.getByRole('button', { name: '看输入输出', exact: true }).click();
  const drawer = page.getByRole('dialog');
  await drawer.getByRole('heading', { name: '示例输入', exact: true }).waitFor();
  const schema = drawer.locator('details');
  assert(await schema.evaluate(element => !element.open), '参数Schema必须默认折叠');
  await schema.locator('summary').click();
  assert(await schema.evaluate(element => element.open), '参数Schema应可展开');
  assert((await schema.innerText()).includes('{bad <svg'), '坏Schema应保留原始文本');
  assert(await drawer.locator('svg[onload]').count() === 0, '坏Schema不得执行HTML');
  await schema.locator('summary').click();
  assert(await schema.evaluate(element => !element.open), '参数Schema应可重新折叠');
  await drawer.getByRole('button', { name: '关闭', exact: true }).click();
  await drawer.waitFor({ state: 'hidden' });
  await search.fill('整理 JSON');
  assert(await page.locator('article').count() === 1, '多关键词用途搜索应命中JSON工具');
  await search.fill('不存在的工具');
  await page.getByText('没有找到对应工具', { exact: true }).waitFor();
  await search.fill('');
  await page.screenshot({ path: 'tmp/ai-product-20261107/tools-ux-desktop.png', fullPage: true });
  mode = 'missing';
  await page.getByRole('button', { name: /刷新$/ }).click();
  await page.getByText('分析模型未配置', { exact: true }).waitFor();
  mode = 'unknown';
  await page.getByRole('button', { name: /刷新$/ }).click();
  await page.getByText('配置状态未知', { exact: true }).first().waitFor();
  mode = 'failure';
  await page.getByRole('button', { name: /刷新$/ }).click();
  await page.getByText('响应桩工具目录暂不可用', { exact: true }).first().waitFor();
  assert(await page.locator('article').count() === 0, '请求失败不得继续展示旧能力卡');
  mode = 'empty';
  await page.getByRole('button', { name: /刷新$/ }).click();
  await page.getByText('暂无注册工具', { exact: true }).waitFor();
  mode = 'ready';
  await page.getByRole('button', { name: /刷新$/ }).click();
  await page.getByRole('heading', { name: '资料分析', exact: true }).waitFor();
  await page.setViewportSize({ width: 390, height: 844 });
  await page.waitForFunction(() => document.documentElement.scrollWidth <= window.innerWidth);
  assert(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth), '窄屏目录不可横向溢出');
  await page.screenshot({ path: 'tmp/ai-product-20261107/tools-ux-mobile.png', fullPage: true });
  assert(calls.every(call => call.method === 'GET'), '只读目录验收不得发出业务写请求');
  assert(errors.length === 0, `浏览器运行错误数量：${errors.length}`);
  return { regression: 'tool-catalog-capabilities', schemaToggle: true, purposeSearch: true, configurationStates: true, failedAndEmptyStates: true, mobileNoOverflow: true, businessWrites: 0, pageErrors: 0 };
}
