// 独立浏览器会话；全部业务API均为响应桩，供应商与真实业务写入为0。
// oxlint-disable-next-line no-unused-expressions -- playwright-cli读取函数表达式执行。
async (page) => {
  await page.unroute('**/v1/**');
  const origin = page.url().replace(/^(https?:\/\/[^/]+).*$/, '$1'), now = Math.floor(Date.now() / 1000), writes = [], reads = [], errors = [];
  let admin = true, version = 1, dailyLimit = 2;
  const require = (condition, message) => { if (!condition) throw new Error(message); };
  const counts = { confirmed_requests: 2, reserved_requests: 1, not_sent_requests: 1, uncertain_requests: 1, succeeded_requests: 1, failed_requests: 1, cancelled_requests: 0, complete_requests: 0, partial_requests: 1, unknown_requests: 1, invalid_requests: 0, known_prompt_tokens: 10, known_completion_tokens: 5, known_embedding_tokens: 7, known_total_tokens: 22 };
  const policy = owner => ({ owner_id: owner, quota_day: '2026-11-08', daily_request_limit: owner === 0 ? 1000 : dailyLimit, used_requests: 2, remaining_requests: owner === 0 ? 0 : Math.max(0, dailyLimit - 2), unlimited: false, inherited: false, daily_token_warning: 10, known_tokens: 15, warning_reached: true, incomplete_requests: 1, version, global_concurrent_limit: 8, active_calls: owner === 0 ? 1 : 0, active_known: owner === 0 });
  const summary = { ...counts, period: { from_day: '2026-11-02', to_day: '2026-11-08', timezone: 'Asia/Shanghai' }, recording_started_at: now, groups: [{ ...counts, source: 'analysis', operation: 'chat', provider: 'local-test', model: 'test' }] };
  const row = { id: 100, owner_id: 1, attempt_id: 'usage-attempt-100', source: 'analysis', operation: 'chat', request_id: 'usage-request-100', source_id: 31, step_index: 0, batch_index: 0, provider: 'local-test', model: 'test', quota_day: '2026-11-08', quota_charged: true, dispatch_confirmed: true, status: 'failed', usage_state: 'unknown', http_status: 503, provider_request_id: '', error_code: 'provider_http_error', started_at: now, dispatched_at: now, completed_at: now };
  page.on('pageerror', error => errors.push(error.message));
  await page.addInitScript(() => {
    if (location.protocol === 'http:') localStorage.setItem('go-zero-auth', JSON.stringify({ state: { token: 'usage-fixture-access', refreshToken: 'usage-fixture-refresh', expiresAt: Date.now() / 1000 + 3600, refreshExpiresAt: Date.now() / 1000 + 7200, sessionId: 'usage-fixture-session' }, version: 0 }));
  });
  await page.route('**/v1/**', async route => {
    const request = route.request(), [path, query = ''] = request.url().replace(/^https?:\/\/[^/]+/, '').split('?'), values = new Map(query.split('&').map(pair => pair.split('=')));
    const send = (code, result = {}, msg = '') => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ code, result, msg }) });
    if (path === '/v1/sys/auth/captcha') return send(200, { enabled: false });
    if (path === '/v1/sys/dept/my') return send(200, { list: [] });
    if (path === '/v1/sys/menu/my/perm-codes') return send(200, { codes: ['sys:ai:usage:read', ...(admin ? ['sys:ai:usage:admin', 'sys:ai:usage:manage'] : [])] });
    if (path === '/v1/sys/menu/my') return send(200, { list: [{ id: 9001450, parent_id: 0, title: '用量与额度', name: 'ai_usage', type: 2, path: '/ai/usage', component: 'ai_usage', perm_code: '', icon: 'audit', sort: 1, visible: true, status: 1, created_at: now }] });
    if (path === '/v1/sys/profile/') return send(200, { user: { id: 1, username: 'usage-fixture', nickname: '用量回归', email: '', phone: '', avatar_url: '', dept_id: 0, role_ids: [], is_super_admin: admin } });
    if (!path.startsWith('/v1/sys/ai/usage/')) throw new Error(`默认拒绝未定义业务请求：${request.method()} ${path}`);
    reads.push(path);
    if (path.includes('/admin/') && !admin) throw new Error(`本人权限页面不能发全局请求：${path}`);
    if (path.endsWith('/summary')) return send(200, { summary });
    if (path.endsWith('/calls')) return send(200, { list: [row, { ...row, id: 101, attempt_id: 'usage-attempt-101', usage_state: 'partial', prompt_tokens: 10, completion_tokens: 5, total_tokens: 15 }], total: 2 });
    if (path.endsWith('/policy')) return send(200, { policy: policy(1) });
    if (path.endsWith('/admin/policies')) {
      const user = Number(values.get('user_id') ?? 0);
      return send(200, { list: user ? [policy(user)] : [policy(0), policy(1)], total: user ? 1 : 2 });
    }
    if (path === '/v1/sys/ai/usage/admin/policies/1' && request.method() === 'PUT') {
      writes.push(request.postDataJSON());
      if (writes.length === 1) { version = 2; await page.waitForTimeout(100); return send(409, {}, '策略已变化'); }
      require(writes.length === 2 && writes[1].expected_version === 2, '核对后保存必须使用新读取版本');
      version = 3; dailyLimit = 3; return send(200, { policy: policy(1) });
    }
    throw new Error(`默认拒绝未定义用量请求：${request.method()} ${path}`);
  });
  await page.goto(`${origin}/ai/usage`);
  await page.getByRole('cell', { name: '未取得 未取得用量', exact: true }).waitFor();
  await page.getByRole('cell', { name: '15（不完整） 部分用量', exact: true }).waitFor();
  await page.getByText('按来源与模型查看汇总', { exact: true }).click();
  require(await page.locator('details').evaluate(node => node.open), '汇总折叠应可展开');
  await page.getByText('按来源与模型查看汇总', { exact: true }).click();
  await page.locator('.ant-table-row-expand-icon').first().click();
  await page.getByText('尝试标识：usage-attempt-100', { exact: false }).waitFor();
  const globalRead = page.waitForResponse('**/v1/sys/ai/usage/admin/summary**');
  await page.getByRole('tab', { name: '全局用量', exact: true }).click();
  await globalRead;
  require(reads.some(path => path.endsWith('/admin/summary')), '切换全局必须读取管理员端点');
  const policyRow = page.getByRole('row').filter({ hasText: '后台用户 1' });
  await policyRow.getByRole('button', { name: '设置', exact: true }).click();
  await page.getByRole('button', { name: '保存策略', exact: true }).dblclick();
  await page.getByText('保存未确认或版本已变化。草稿已保留，请先读取当前策略核对，勿直接重复提交。', { exact: true }).waitFor();
  require(writes.length === 1 && writes[0].expected_version === 1, '双击和版本冲突不能重复保存');
  await page.getByRole('button', { name: '读取当前策略', exact: true }).click();
  await page.getByText('当前版本 2', { exact: false }).waitFor();
  await page.getByLabel('每日供应商请求次数（0 不限）', { exact: true }).fill('3');
  await page.getByRole('button', { name: '保存策略', exact: true }).click();
  await page.getByText('策略已保存，正在执行的请求沿用已领取额度，后续准入使用新策略。', { exact: true }).waitFor();
  require(writes.length === 2, '只有明确核对后的再次保存能提交');
  await page.waitForLoadState('networkidle');
  // 先卸载已获管理员权限的旧文档，独立验证本人权限首次加载。
  await page.goto('about:blank');
  admin = false; reads.length = 0;
  await page.goto(`${origin}/ai/usage`);
  await page.getByRole('cell', { name: '未取得 未取得用量', exact: true }).waitFor();
  require(await page.getByRole('tab', { name: '全局用量', exact: true }).count() === 0, '本人权限不展示全局入口');
  require(await page.getByRole('button', { name: '设置用户额度', exact: true }).count() === 0 && !reads.some(path => path.includes('/admin/')), '本人页面不发全用户读取或设置请求');
  await page.setViewportSize({ width: 390, height: 844 });
  require(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth + 1), '窄屏不能使页面整体横向溢出');
  require(errors.length === 0, `页面错误：${errors.join(';')}`);
  return { success: true, provider_calls: 0, real_business_writes: 0, mocked_policy_writes: writes.length, checks: ['unknown与partial', '分组/行折叠', '本人/全局隔离', '双击防重', '409草稿保留', '显式读版本再保存', '390px布局'] };
}
