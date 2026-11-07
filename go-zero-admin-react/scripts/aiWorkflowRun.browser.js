// 使用独立 playwright-cli 会话 run-code --filename 执行；本脚本拦截全部业务请求，仅使用虚构会话。
// oxlint-disable-next-line no-unused-expressions
async (page) => {
  const assert = (value, message) => { if (!value) throw new Error(message); };
  const calls = [], unexpected = [], errors = [];
  const now = Math.floor(Date.now() / 1000);
  const permissions = ['sys:ai:workflow:list', 'sys:ai:workflow:run', 'sys:ai:task:list', 'sys:ai:tool:list'];
  const workflow = { id: 71, name: '运行回归响应桩', enabled: 1, interval_seconds: 0, created_at: now, updated_at: now,
    definition: JSON.stringify({ version: 2, steps: [{ id: 'text', name: '统计本次文本', tool: 'text_stats', input: { text: { $ref: 'input.text' } } }] }) };
  const task = { id: 81, workflow_id: 71, status: 'succeeded', input: '{}', result: '', error: '', next_step: 1, steps: [], request_id: '', created_at: now, updated_at: now, completed_at: now, approved_by: 0 };
  page.on('pageerror', error => errors.push(error.message));
  page.on('console', message => { if (message.type() === 'error' && /Maximum update depth|Cannot update a component|Rendered more hooks/.test(message.text())) errors.push(message.text()); });
  await page.addInitScript(() => {
    localStorage.setItem('go-zero-auth', JSON.stringify({ state: { token: 'workflow-run-fixture-access', refreshToken: 'workflow-run-fixture-refresh',
      expiresAt: Date.now() / 1000 + 3600, refreshExpiresAt: Date.now() / 1000 + 7200, sessionId: 'workflow-run-fixture-20261107' }, version: 0 }));
    const original = Storage.prototype.setItem;
    Storage.prototype.setItem = function (key, value) {
      if (key.startsWith('ai-workflow-run:')) throw new DOMException('QuotaExceeded', 'QuotaExceededError');
      return original.call(this, key, value);
    };
  });
  await page.route('**/v1/**', async route => {
    const request = route.request(), path = new URL(request.url()).pathname, method = request.method();
    let result;
    if (path === '/v1/sys/dept/my') result = { list: [] };
    else if (path === '/v1/sys/menu/my/perm-codes') result = { codes: permissions };
    else if (path === '/v1/sys/menu/my') result = { list: [['ai_workflow', '/ai/workflows', '工作流管理'], ['ai_task', '/ai/tasks', '任务中心']].map(([name, path, title], index) => ({ id: 901 + index, parent_id: 0, title, name, path, component: name, type: 2, visible: true, status: 1, sort: index })) };
    else if (path === '/v1/sys/profile/') result = { user: { id: 1, username: 'workflow-fixture', nickname: '响应桩', email: '', phone: '', avatar_url: '', dept_id: 0, role_ids: [] } };
    else if (path === '/v1/sys/ai/workflows') result = { list: [workflow], total: 1 };
    else if (path === '/v1/sys/ai/workflows/71' && method === 'GET') result = { workflow };
    else if (path === '/v1/sys/ai/workflows/71/run' && method === 'POST') {
      calls.push(request.postDataJSON());
      if (calls.length === 1) { await route.abort('failed'); return; }
      result = { task_id: 81, status: 'running' };
    } else if (path === '/v1/sys/ai/tasks') result = { list: [task], total: 1 };
    else if (path === '/v1/sys/ai/tasks/81') result = { task };
    else { unexpected.push({ path, method }); await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ code: 404, msg: '响应桩未定义路由', result: {} }) }); return; }
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ code: 200, result }) });
  });
  await page.goto('http://127.0.0.1:5999/ai/workflows?run=71');
  const dialog = page.getByRole('dialog', { name: '确认运行工作流' });
  await dialog.waitFor();
  await page.waitForURL('**/ai/workflows');
  await page.evaluate(() => new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve))));
  assert(await dialog.isVisible(), '自动运行确认必须稳定显示');
  assert(errors.length === 0, `自动打开出现React错误：${errors.join(';')}`);
  await page.getByRole('textbox', { name: /本次文本/ }).fill('存储失败也要保留原请求');
  await page.getByRole('button', { name: '开始后台运行', exact: true }).click();
  await page.getByText('上次运行结果待确认', { exact: true }).waitFor();
  assert(calls.length === 1, '首次提交只能执行一个请求');
  await dialog.getByRole('button', { name: /取\s*消/ }).click();
  await dialog.waitFor({ state: 'hidden' });
  await page.getByRole('button', { name: /运行$/ }).click();
  await page.getByRole('button', { name: '核对原任务', exact: true }).waitFor();
  await page.getByRole('button', { name: '核对原任务', exact: true }).evaluate(button => { button.click(); button.click(); });
  await page.waitForURL('**/ai/tasks?task_id=81');
  assert(calls.length === 2, '双击核对必须只增加一次请求');
  assert(calls[0].request_id === calls[1].request_id && calls[0].input === calls[1].input, '存储配额失败、关闭重开后必须复用相同ID和输入');
  assert(errors.length === 0 && unexpected.length === 0, `出现未预期错误/请求：${JSON.stringify({ errors, unexpected })}`);
  return { requests: calls.length, sameRequestId: true, sameInput: true, storageFailureRecovered: true, reactErrors: errors, unexpected };
}
