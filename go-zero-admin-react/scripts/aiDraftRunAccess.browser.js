// 独立 playwright-cli 会话及 6009 测试端口；所有业务请求均由响应桩接收。
// oxlint-disable-next-line no-unused-expressions -- playwright-cli 执行可重复的函数入口。
async page => {
  const failures = [], errors = [], calls = [];
  const check = (condition, message) => { if (!condition) failures.push(message); };
  let taskAccess = false;
  let runFailure = false;
  const capabilities = { retrieval_configured: true, analysis_configured: true, ingestion_mode: 'background' };
  const definition = JSON.stringify({ steps: [{ id: 'analysis', name: '资料分析', tool: 'kb_analyze', input: { kb_id: { $ref: 'input.kb_id' }, query: { $ref: 'input.question' } } }] });
  const kb = { id: 7, name: '权限回归知识库', description: '', ready_count: 1, pending_count: 0, failed_count: 0, doc_count: 1, chunk_count: 1, updated_at: 1 };
  const task = { id: 41, workflow_id: 0, title: '权限回归分析', workflow_name: '权限回归分析', input_summary: '保留的原问题', status: 'succeeded', next_step: 1, input: '{"kb_id":7,"question":"保留的原问题"}', result: '{}', definition, error: '', created_at: 1, updated_at: 2, completed_at: 2, approved_by: 0, steps: [] };
  const tools = [{ name: 'kb_analyze', description: '资料分析', parameters_schema: JSON.stringify({ type: 'object', properties: { kb_id: { type: 'integer', minimum: 1 }, query: { type: 'string', minLength: 1 } }, required: ['kb_id', 'query'] }) }];
  page.on('pageerror', error => errors.push(error.message));
  await page.addInitScript(() => localStorage.setItem('go-zero-auth', JSON.stringify({ state: { token: 'draft-access-fixture', refreshToken: 'draft-access-fixture-refresh', sessionId: 'draft-access-fixture-session', expiresAt: Date.now() / 1000 + 3600, refreshExpiresAt: Date.now() / 1000 + 7200 }, version: 0 })));
  await page.route('**/v1/**', route => {
    const request = route.request(), path = new URL(request.url()).pathname;
    calls.push({ method: request.method(), path, taskAccess, input: request.method() === 'POST' ? request.postDataJSON() : undefined });
    const send = (code, result = {}) => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ code, result, msg: code === 200 ? '' : '响应桩拒绝' }) });
    if (request.method() === 'POST' && path === '/v1/sys/ai/tasks/run') return send(runFailure ? 503 : 200, { task_id: 41, status: 'pending' });
    if (request.method() !== 'GET') return route.abort('blockedbyclient');
    if (path === '/v1/sys/dept/my') return send(200, { list: [] });
    if (path === '/v1/sys/menu/my/perm-codes') return send(200, { codes: ['sys:ai:kb:list', 'sys:ai:workflow:run', 'sys:ai:workflow:list', 'sys:ai:workflow:create', 'sys:ai:tool:list', ...(taskAccess ? ['sys:ai:task:list'] : [])] });
    if (path === '/v1/sys/menu/my') return send(200, { list: [
      { id: 901, parent_id: 0, title: '知识库', name: 'ai_kb', type: 2, path: '/ai/knowledge', component: 'ai_kb', perm_code: '', icon: '', sort: 0, visible: true, status: 1, created_at: 0 },
      { id: 902, parent_id: 0, title: '常用流程', name: 'ai_workflow', type: 2, path: '/ai/workflows', component: 'ai_workflow', perm_code: '', icon: '', sort: 1, visible: true, status: 1, created_at: 0 },
    ] });
    if (path === '/v1/sys/profile/') return send(200, { user: { id: 1, username: 'draft-access-fixture', nickname: '权限回归', email: '', phone: '', avatar_url: '', dept_id: 0, role_ids: [] } });
    if (path === '/v1/sys/ai/tools') return send(200, { list: tools, capabilities });
    if (path === '/v1/sys/ai/kbs/7') return send(200, { kb, capabilities });
    if (path === '/v1/sys/ai/kbs') return send(200, { list: [kb], total: 1, capabilities });
    if (path === '/v1/sys/ai/kbs/7/docs') return send(200, { list: [], total: 0, ready_count: 1, pending_count: 0, failed_count: 0 });
    if (path === '/v1/sys/ai/workflows') return send(200, { list: [], total: 0 });
    if (path === '/v1/sys/ai/tasks/41') return send(taskAccess ? 200 : 403, { task });
    return send(404);
  });
  await page.goto('http://127.0.0.1:6009/ai/knowledge/7');
  await page.getByRole('heading', { name: kb.name, exact: true }).waitFor();
  await page.getByRole('textbox', { name: /你希望了解什么？$/ }).fill('权限最小组合的问题');
  await page.getByRole('button', { name: /开始分析$/ }).click();
  await page.getByText('已创建运行记录 #41，需要任务查看权限才能查看结果。', { exact: true }).waitFor();
  check(await page.getByRole('button', { name: '保存为常用流程', exact: true }).count() === 0, '缺少任务查看权限时不应提供依赖任务快照的保存入口');
  await page.goto('http://127.0.0.1:6009/ai/workflows/new?from_task=41');
  const forbidden = page.getByText('需要查看运行记录权限，才能使用本次快照', { exact: true });
  await forbidden.waitFor({ timeout: 2000 }).catch(() => {});
  check(await forbidden.count() === 1, '直接打开from_task无任务查看权限时应显示明确403，而非持续加载');
  check(calls.filter(call => !call.taskAccess && call.path === '/v1/sys/ai/tasks/41').length === 0, '无任务查看权限不得读取任务快照');
  if (await forbidden.count()) {
    await page.getByRole('button', { name: '返回列表', exact: true }).click();
    await page.getByRole('heading', { name: '常用流程', exact: true }).waitFor();
  }
  taskAccess = true;
  await page.goto('http://127.0.0.1:6009/ai/knowledge/7');
  await page.getByRole('heading', { name: kb.name, exact: true }).waitFor();
  const save = page.getByRole('button', { name: '保存为常用流程', exact: true });
  await save.waitFor();
  await save.click();
  await page.getByRole('heading', { name: '新建常用流程', exact: true }).waitFor();
  const name = page.getByRole('textbox', { name: /工作流名称$/ });
  await name.waitFor();
  check(await name.inputValue() === task.workflow_name, '有权限时应带入本次任务名称及执行快照');
  check(await page.getByRole('textbox', { name: /本次问题$/ }).inputValue() === '保留的原问题', '有权限时应保留原任务的问题');
  // 必须使用SPA导航，让新组件恢复同一页面运行环境中未落盘的原请求。
  await page.getByRole('menuitem', { name: '知识库', exact: true }).click();
  await page.getByRole('button', { name: kb.name, exact: true }).click();
  await page.getByRole('heading', { name: kb.name, exact: true }).waitFor();
  await page.evaluate(() => {
    const setItem = Storage.prototype.setItem;
    Storage.prototype.setItem = function (key, value) {
      if (this === sessionStorage && key.startsWith('ai-draft-run:')) throw new DOMException('fixture quota', 'QuotaExceededError');
      return setItem.call(this, key, value);
    };
  });
  const runCount = calls.filter(call => call.path === '/v1/sys/ai/tasks/run').length;
  runFailure = true;
  await page.getByRole('textbox', { name: /你希望了解什么？$/ }).fill('存储失败仍应保留原问题');
  await page.getByRole('button', { name: /开始分析$/ }).click();
  await page.getByText('上次运行结果待确认', { exact: true }).waitFor();
  check(await page.getByRole('button', { name: /开始分析$/ }).isDisabled(), '未知提交期间应禁止另发新运行');
  await page.getByRole('menuitem', { name: '常用流程', exact: true }).click();
  await page.getByRole('heading', { name: '常用流程', exact: true }).waitFor();
  await page.getByRole('menuitem', { name: '知识库', exact: true }).click();
  await page.getByRole('button', { name: kb.name, exact: true }).click();
  await page.getByText('上次运行结果待确认', { exact: true }).waitFor();
  check(await page.getByRole('textbox', { name: /你希望了解什么？$/ }).inputValue() === '存储失败仍应保留原问题', '配额不足后SPA返回应恢复原问题');
  check(await page.getByRole('button', { name: /开始分析$/ }).isDisabled(), '配额不足后SPA返回应继续锁住新提交');
  check(calls.filter(call => call.path === '/v1/sys/ai/tasks/run').length === runCount + 1, 'SPA返回不得自动提交新请求');
  runFailure = false;
  await page.getByRole('button', { name: '核对原运行', exact: true }).click();
  await page.getByText('上次运行结果待确认', { exact: true }).waitFor({ state: 'hidden' });
  const resumed = calls.filter(call => call.path === '/v1/sys/ai/tasks/run').slice(runCount);
  check(resumed.length === 2 && JSON.stringify(resumed[0].input) === JSON.stringify(resumed[1].input), '核对必须复用原UUID、定义和输入');
  check(errors.length === 0, `页面运行错误：${errors[0] ?? ''}`);
  if (failures.length) throw new Error(failures.join('；'));
  return { minimalPermissions: true, directForbiddenRoute: true, noUnauthorizedTaskReads: true, allowedSnapshotRestored: true, quotaRestoredAcrossSpa: true, originalRequestReused: true, realBusinessWrites: 0, pageErrors: errors.length };
}
