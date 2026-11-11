// 已正常登录本地开发栈后执行：playwright-cli run-code --filename scripts/aiUsagePage.browser.mjs
// 仅使用浏览器请求桩和内存权限快照验证；策略 PUT 被拦截，不写真实额度。
// oxlint-disable-next-line no-unused-expressions -- 由 Playwright CLI 调用函数表达式。
async page => {
  const assert = (condition, message) => { if (!condition) throw new Error(message); };
  const origin = new URL(page.url()).origin;
  const pattern = '**/v1/sys/ai/usage/**';
  const originalPermissions = await page.evaluate(async () => (await import('/src/stores/auth.ts')).useAuthStore.getState().permCodes);
  const setPermissions = codes => page.evaluate(async values => (await import('/src/stores/auth.ts')).useAuthStore.getState().setPermCodes(values), codes);
  const base = { confirmed_requests: 12, reserved_requests: 0, not_sent_requests: 0, uncertain_requests: 0, succeeded_requests: 12, failed_requests: 0,
    cancelled_requests: 0, complete_requests: 12, partial_requests: 0, unknown_requests: 0, invalid_requests: 0,
    known_prompt_tokens: 8000, known_completion_tokens: 253, known_embedding_tokens: 466, known_total_tokens: 8719 };
  const policy = { owner_id: 9, quota_day: '2026-11-11', daily_request_limit: 1000, used_requests: 2, remaining_requests: 998, unlimited: false,
    inherited: false, daily_token_warning: 0, known_tokens: 20, warning_reached: false, incomplete_requests: 0, version: 4,
    global_concurrent_limit: 8, active_calls: 0, active_known: true };
  const call = { id: 1, owner_id: 9, source: 'chat', operation: 'chat', provider: 'fixture-provider', model: 'fixture-model', status: 'succeeded', usage_state: 'complete', total_tokens: 8253,
    quota_charged: true, request_id: 'fixture-request', provider_request_id: 'fixture-provider-request', attempt_id: 'fixture-attempt', error_code: '', started_at: 1 };
  let failure = false;
  let anomalies = false;
  const gets = [];
  const writes = [];
  const routeHandler = async route => {
    const request = route.request(); const url = new URL(request.url());
    if (request.method() === 'PUT') {
      writes.push({ path: url.pathname, body: request.postDataJSON() });
      await route.fulfill({ json: { code: 409, msg: '策略版本已变化', result: null } });
      return;
    }
    assert(request.method() === 'GET', '出现未预期的写请求');
    gets.push(url);
    let result;
    if (url.pathname.endsWith('/summary')) {
      if (failure) { await route.fulfill({ json: { code: 503, msg: '统计暂不可用', result: null } }); return; }
      result = { summary: { ...base, ...(anomalies ? { uncertain_requests: 2, partial_requests: 1, unknown_requests: 1 } : {}),
        period: { from_day: url.searchParams.get('from_day'), to_day: url.searchParams.get('to_day'), timezone: 'Asia/Shanghai' }, recording_started_at: 1, groups: [] } };
    } else if (url.pathname.endsWith('/calls')) result = { list: failure ? [] : [call], total: failure ? 0 : 1 };
    else if (url.pathname.endsWith('/policies')) result = { list: [{ ...policy, owner_id: 0, inherited: false }, policy], total: 2 };
    else if (url.pathname.endsWith('/policy')) result = { policy };
    else throw new Error('未预期的用量接口');
    await route.fulfill({ json: { code: 200, msg: '操作成功', result } });
  };
  const refresh = () => page.getByRole('button', { name: 'reload 刷新', exact: true }).click();
  const waitSummary = predicate => page.waitForResponse(response => {
    const url = new URL(response.url());
    return url.pathname.endsWith('/usage/summary') && predicate(url.searchParams);
  });
  try {
    await page.route(pattern, routeHandler);
    await page.goto(`${origin}/ai/usage`);
    await page.getByRole('region', { name: '今日额度' }).getByText('998', { exact: true }).waitFor();
    await page.locator('.ant-statistic').first().waitFor();
    assert(await page.locator('.ant-statistic').count() === 3, '默认期间统计应只有三项');
    assert((await page.getByRole('region', { name: '今日额度' }).textContent()).includes('2 / 1,000'), '今日占用不能由期间12次推算');
    assert(!await page.getByRole('textbox', { name: '供应商', exact: true }).isVisible(), '高级筛选应默认收起');
    assert(gets.every(url => !url.pathname.includes('/admin/')), '我的用量不应加载全局/额度策略');
    await page.getByRole('button', { name: /更多筛选/ }).click();
    await page.getByRole('textbox', { name: '供应商', exact: true }).fill('fixture-provider');
    await page.getByRole('combobox', { name: '请求状态', exact: true }).click();
    await page.getByText('待核对', { exact: true }).last().click();
    await page.getByRole('button', { name: /收起筛选/ }).click();
    const filtered = waitSummary(params => params.get('provider') === 'fixture-provider' && params.get('status') === 'unknown');
    await page.getByRole('button', { name: '查询用量', exact: true }).click(); await filtered;
    await page.getByRole('button', { name: /更多筛选/ }).click();
    assert(await page.getByRole('textbox', { name: '供应商', exact: true }).inputValue() === 'fixture-provider', '折叠不能丢失高级条件');
    const reset = waitSummary(params => !params.has('provider') && !params.has('status'));
    await page.getByRole('button', { name: '重置筛选', exact: true }).click();
    await refresh(); await reset;
    const today = waitSummary(params => params.get('from_day') === params.get('to_day'));
    await page.getByRole('button', { name: '今天', exact: true }).click(); await today;
    const thirty = waitSummary(params => (Date.parse(params.get('to_day')) - Date.parse(params.get('from_day'))) / 86400000 === 29);
    await page.getByRole('button', { name: '最近30天', exact: true }).click(); await thirty;
    const requestCount = gets.length;
    await page.getByRole('textbox', { name: '开始日期', exact: true }).fill('2020-01-01');
    await page.getByRole('button', { name: '查询用量', exact: true }).click();
    await page.getByText('单次最多查询 31 日（包含开始和结束日期）', { exact: true }).waitFor();
    assert(gets.length === requestCount, '超31天不能发出查询');
    await page.getByRole('button', { name: '重置筛选', exact: true }).click();
    await page.getByRole('button', { name: '展开行', exact: true }).first().click();
    await page.getByText('请求标识：fixture-request', { exact: true }).waitFor();
    await page.getByText('额度：已占用', { exact: true }).waitFor();
    anomalies = true;
    await refresh();
    await page.getByText('2 次请求结果待核对', { exact: true }).waitFor();
    await page.getByText('2 次请求用量不完整', { exact: true }).waitFor();
    failure = true;
    await refresh();
    await page.getByText('暂时无法加载数据', { exact: true }).waitFor();
    assert(await page.locator('.ant-statistic').count() === 0, '读取失败不能继续显示缓存旧统计');
    await page.getByText('此范围尚无账本记录', { exact: true }).waitFor();
    failure = false; anomalies = false;
    await refresh();
    await page.locator('.ant-statistic').first().waitFor();
    await page.getByRole('tab', { name: '全局用量', exact: true }).click();
    await page.getByRole('button', { name: /更多筛选|收起筛选/ }).click();
    if (!await page.getByRole('spinbutton', { name: '后台用户 ID', exact: true }).isVisible()) await page.getByRole('button', { name: /更多筛选/ }).click();
    await page.getByRole('spinbutton', { name: '后台用户 ID', exact: true }).fill('9');
    const globalQuery = page.waitForResponse(response => response.url().includes('/usage/admin/summary') && new URL(response.url()).searchParams.get('user_id') === '9');
    await page.getByRole('button', { name: '查询用量', exact: true }).click(); await globalQuery;
    const personal = waitSummary(params => !params.has('user_id'));
    await page.getByRole('tab', { name: '我的用量', exact: true }).click();
    await refresh(); await personal;
    await page.getByRole('tab', { name: '额度管理', exact: true }).click();
    const owner = page.getByRole('row').filter({ hasText: '后台用户 9' });
    await owner.getByRole('button', { name: /^设\s*置$/ }).click();
    const drawer = page.getByRole('dialog', { name: '额度策略' });
    await drawer.getByRole('spinbutton', { name: '每日供应商请求次数（0 不限）' }).fill('1200');
    await drawer.getByRole('button', { name: '保存策略', exact: true }).click();
    await drawer.getByText('保存未确认或版本已变化。草稿已保留，请先读取当前策略核对，勿直接重复提交。', { exact: true }).waitFor();
    assert(writes.length === 1 && writes[0].body.expected_version === 4 && writes[0].body.daily_request_limit === 1200, '策略必须保留原版本和字段');
    assert(await drawer.getByRole('spinbutton', { name: '每日供应商请求次数（0 不限）' }).inputValue() === '1200', '冲突不能丢失草稿');
    await page.keyboard.press('Escape');
    await drawer.waitFor({ state: 'hidden' });
    await setPermissions(originalPermissions.filter(code => code !== 'sys:ai:usage:manage'));
    assert(await page.getByRole('button', { name: /^设\s*置$/ }).count() === 0, '只读管理员不能修改额度');
    assert(await page.getByRole('button', { name: '设置用户额度', exact: true }).count() === 0, '只读管理员不能打开修改入口');
    await setPermissions(originalPermissions.filter(code => !['sys:ai:usage:admin', 'sys:ai:usage:manage'].includes(code)));
    await page.getByRole('tab', { name: '我的用量', exact: true }).waitFor();
    assert(await page.getByRole('tab', { name: '全局用量', exact: true }).count() === 0 && await page.getByRole('tab', { name: '额度管理', exact: true }).count() === 0, '普通用户只能查看本人');
    await page.setViewportSize({ width: 390, height: 844 });
    await page.waitForFunction(() => document.documentElement.scrollWidth <= innerWidth + 1);
    assert(await page.getByRole('region', { name: '今日额度' }).isVisible(), '窄屏仍能查看今日额度');
    await page.getByRole('button', { name: /更多筛选|收起筛选/ }).click();
    await setPermissions(originalPermissions.filter(code => !code.startsWith('sys:ai:usage:')));
    await page.getByText('无权查看用量与额度', { exact: true }).waitFor();
    assert(writes.length === 1, '权限核对期间不能产生额外写请求');
    return 'PASS：日期/31日边界、折叠保留、重置、今日与期间口径、两类告警、错误与空状态、技术明细、全局/本人隔离、策略版本409草稿、读写权限和390px窄屏';
  } catch (error) {
    throw new Error(`${error.stack ?? error}\n最近用量请求：${gets.slice(-8).map(url => `${url.pathname}${url.search}`).join('\n')}`);
  } finally {
    await setPermissions(originalPermissions);
    await page.unroute(pattern, routeHandler);
    await page.setViewportSize({ width: 1440, height: 1000 });
    await page.goto(`${origin}/ai/usage`);
  }
}
