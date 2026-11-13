// 独立playwright-cli会话；全部业务请求由显式响应桩接管，供应商及未知业务请求拒绝。
// oxlint-disable-next-line no-unused-expressions -- playwright-cli读取此函数表达式执行。
async (page) => {
  const origin = new URL(page.url()).origin;
  const check = (value, message) => { if (!value) throw new Error(message); };
  const now = Math.floor(Date.now() / 1000), creates = [], updates = [], restores = [], unexpected = [], errors = [];
  let permissions = ['sys:ai:workflow:list', 'sys:ai:workflow:create', 'sys:ai:workflow:edit', 'sys:ai:tool:list'];
  const handleDialog = dialog => dialog.type() === 'beforeunload' ? dialog.accept() : dialog.dismiss();
  page.on('dialog', handleDialog);
  let workflow = { id: 71, name: '原始流程', enabled: 1, interval_seconds: 0, next_run_at: 0, created_by: 1, created_at: now, updated_at: now, revision: 1, step_count: 1,
    definition: JSON.stringify({ steps: [{ name: '读取时间', tool: 'get_current_time', input: {}, approval: false }] }) };
  const history = new Map();
  function archive() { history.set(workflow.revision, { id: 1000 + workflow.revision, workflow_id: workflow.id, revision: workflow.revision, name: workflow.name, definition: workflow.definition, interval_seconds: 0, enabled: workflow.enabled, step_count: 1, created_at: now }); }
  page.on('pageerror', error => errors.push(error.message));
  await page.addInitScript(() => localStorage.setItem('go-zero-auth', JSON.stringify({ state: {
    token: 'workflow-history-fixture-access', refreshToken: 'workflow-history-fixture-refresh',
    expiresAt: Date.now() / 1000 + 3600, refreshExpiresAt: Date.now() / 1000 + 7200, sessionId: 'workflow-history-fixture-session',
  }, version: 0 })));
  await page.route('**/*', route => {
    const url = route.request().url(), path = url.replace(/^https?:\/\/[^/]+/, '');
    if (!url.startsWith(`${origin}/`) || path.startsWith('/v1/')) { unexpected.push({ url: url.split('?')[0], method: route.request().method() }); return route.abort('blockedbyclient'); }
    return route.continue();
  });
  await page.route('**/v1/**', async route => {
    const request = route.request(), path = request.url().replace(/^https?:\/\/[^/]+/, '').split('?')[0], method = request.method();
    const send = (result, code = 200) => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ code, result, msg: code === 409 ? '工作流版本已变化' : code === 503 ? '恢复结果待核对' : '' }) });
    if (path === '/v1/sys/dept/my') return send({ list: [] });
    if (path === '/v1/sys/auth/captcha') return send({ enabled: false });
    if (path === '/v1/sys/menu/my/perm-codes') return send({ codes: permissions });
    if (path === '/v1/sys/menu/my') return send({ list: [{ id: 901, parent_id: 0, title: '常用流程', name: 'ai_workflow', path: '/ai/workflows', component: 'ai_workflow', type: 2, visible: true, status: 1, sort: 0 }] });
    if (path === '/v1/sys/profile/') return send({ user: { id: 1, username: 'workflow-fixture', nickname: '版本验收', email: '', phone: '', avatar_url: '', dept_id: 0, role_ids: [] } });
    if (path === '/v1/sys/ai/tools') return send({ list: [
      { name: 'get_current_time', description: '读取当前时间', parameters_schema: '{"type":"object","properties":{}}' },
      { name: 'text_stats', description: '统计文本', parameters_schema: '{"type":"object","properties":{"text":{"type":"string"}},"required":["text"]}' },
    ], capabilities: { retrieval_configured: false, analysis_configured: false, ingestion_mode: 'background' } });
    if (path === '/v1/sys/ai/workflows' && method === 'GET') return send({ list: [{ ...workflow, definition: '' }], total: 1 });
    if (path === '/v1/sys/ai/workflows' && method === 'POST') {
      const data = request.postDataJSON(); creates.push(data);
      check(creates.length === 1 && data.name === 'Form提交持久化回归', '新建只能由本次明确保存触发一次');
      check(JSON.parse(data.definition).steps[0].input.text === '提交中刷新必须保留这段文本', '实际 Form 提交必须保留工具输入');
      return send({}, 503);
    }
    if (path === '/v1/sys/ai/workflows/71' && method === 'GET') return send({ workflow });
    if (path === '/v1/sys/ai/workflows/71' && method === 'PUT') {
      const data = request.postDataJSON(); updates.push(data);
      if (updates.length === 1) { archive(); workflow = { ...workflow, name: '其他编辑者的新名称', revision: 2 }; return send({}, 409); }
      check(data.expected_revision === workflow.revision, '保存必须使用人工核对后的当前版本');
      archive(); workflow = { ...workflow, ...data, revision: workflow.revision + 1 };
      return send({ workflow });
    }
    if (path === '/v1/sys/ai/workflows/71/revisions') return send({ list: [...history.values()].sort((a, b) => b.revision - a.revision).map(item => ({ ...item, definition: '' })), total: history.size });
    const match = path.match(/^\/v1\/sys\/ai\/workflows\/71\/revisions\/(\d+)(\/restore)?$/);
    if (match) {
      const old = history.get(Number(match[1]));
      check(!!old, '只能读取已有归档版本');
      if (method === 'GET' && !match[2]) return send({ revision: old });
      if (method === 'POST' && match[2]) {
        const data = request.postDataJSON(); restores.push({ target: old.revision, ...data });
        check(data.expected_revision === workflow.revision, '恢复须使用确认时捕获的版本');
        archive(); workflow = { ...workflow, name: old.name, definition: old.definition, enabled: old.enabled, revision: workflow.revision + 1 };
        return send(restores.length === 1 ? {} : { workflow }, restores.length === 1 ? 503 : 200);
      }
    }
    unexpected.push({ path, method });
    return route.abort('blockedbyclient');
  });
  // 真实 antd Form：新建未挂载 enabled 字段，切换工具仍应保存完整草稿。
  await page.goto(`${origin}/ai/workflows/new`);
  await page.getByRole('textbox', { name: '工作流名称', exact: true }).fill('Form提交持久化回归');
  await page.getByRole('textbox', { name: '处理文本', exact: true }).fill('提交中刷新必须保留这段文本');
  await page.getByLabel('步骤类型', { exact: true }).click();
  await page.locator('.ant-select-item-option[title="当前时间"]').click();
  await page.getByLabel('步骤类型', { exact: true }).click();
  await page.locator('.ant-select-item-option[title="文本统计"]').click();
  check(await page.getByRole('textbox', { name: '处理文本', exact: true }).inputValue() === '提交中刷新必须保留这段文本', '切回工具必须还原文本缓存');
  await page.getByRole('button', { name: /保存为常用流程$/ }).click();
  await page.getByText('上次保存结果未知，草稿已保留', { exact: true }).waitFor();
  const draftKey = `ai-workflow-draft:${JSON.stringify(['workflow-history-fixture-session', null, 'new'])}`;
  const submitted = await page.evaluate(key => JSON.parse(sessionStorage.getItem(key)), draftKey);
  check(submitted.draft.enabled === true, '未挂载的 enabled 字段必须保留，刷新时才能恢复合法草稿');
  check(submitted.submissionState === 'unknown', '未知提交必须落盘为 unknown');
  check(JSON.parse(submitted.draft.steps[0].toolInputs.text_stats).text === '提交中刷新必须保留这段文本', '实际 Form 提交必须保留工具缓存');
  await page.reload();
  await page.getByText('上次保存结果未知，草稿已保留', { exact: true }).waitFor();
  check(await page.getByRole('textbox', { name: '工作流名称', exact: true }).inputValue() === 'Form提交持久化回归', '未知提交刷新后名称必须恢复');
  check(await page.getByRole('textbox', { name: '处理文本', exact: true }).inputValue() === '提交中刷新必须保留这段文本', '未知提交刷新后文本必须恢复');
  check(await page.getByRole('button', { name: /保存为常用流程$/ }).isDisabled() && creates.length === 1, '未知提交刷新后必须锁住保存且不自动重发POST');
  await page.getByRole('button', { name: '丢弃草稿', exact: true }).click();
  await page.getByRole('button', { name: '丢弃并返回列表', exact: true }).click();
  await page.waitForURL('**/ai/workflows');
  await page.goto(`${origin}/ai/workflows/71/edit`);
  const name = page.getByRole('textbox', { name: '工作流名称', exact: true });
  await name.fill('我的未保存草稿');
  await page.getByRole('button', { name: /保存为常用流程$/ }).click();
  await page.getByText('已保存的工作流在编辑期间发生变化', { exact: true }).waitFor();
  check(await name.inputValue() === '我的未保存草稿', '409后必须保留表单草稿');
  check(await page.getByRole('button', { name: /保存为常用流程$/ }).isDisabled(), '409后不能绕过版本核对继续保存');
  check(updates.length === 1 && updates[0].expected_revision === 1, '首次保存只能提交原基线一次');
  await page.getByRole('button', { name: '版本历史', exact: true }).click();
  const drawer = page.getByRole('dialog', { name: /版本历史/ });
  await drawer.getByRole('button', { name: '查看版本 1', exact: true }).click();
  await drawer.getByRole('button', { name: '恢复版本 1', exact: true }).click();
  await page.getByRole('button', { name: '确认恢复为新版本', exact: true }).click();
  await drawer.getByText('上次恢复结果待核对', { exact: true }).waitFor();
  await drawer.getByRole('button', { name: '关闭', exact: true }).click();
  await drawer.waitFor({ state: 'hidden' });
  await page.getByRole('button', { name: '版本历史', exact: true }).click();
  await drawer.getByText('上次恢复结果待核对', { exact: true }).waitFor();
  check(restores.length === 1, '关闭重开不能自动重复恢复');
  await drawer.getByRole('button', { name: '刷新版本与历史', exact: true }).click();
  await drawer.getByText('当前版本：3', { exact: true }).waitFor();
  await drawer.getByRole('button', { name: '已核对，允许后续操作', exact: true }).click();
  await drawer.getByRole('button', { name: '查看版本 1', exact: true }).click();
  await drawer.getByRole('button', { name: '恢复版本 1', exact: true }).click();
  await page.getByRole('button', { name: '确认恢复为新版本', exact: true }).click();
  await drawer.getByText(/已恢复为版本 4/).waitFor();
  await drawer.getByRole('button', { name: '关闭', exact: true }).click();
  await drawer.waitFor({ state: 'hidden' });
  check(await name.inputValue() === '我的未保存草稿', '恢复后也必须保留未保存草稿');
  await page.getByRole('button', { name: '核对后继续', exact: true }).click();
  await page.getByRole('button', { name: '已核对，保留草稿', exact: true }).click();
  await page.getByRole('button', { name: /保存为常用流程$/ }).click();
  await page.waitForURL('**/ai/workflows');
  check(updates.length === 2 && updates[1].expected_revision === 4, '人工核对后的保存使用新版本4');
  check(restores.length === 2 && restores[0].expected_revision === 2 && restores[1].expected_revision === 3, '恢复未知后只有人工新操作才提交另一条件版本');
  permissions = ['sys:ai:workflow:list'];
  await page.reload();
  await page.getByRole('button', { name: '我的未保存草稿', exact: true }).click();
  await page.getByRole('button', { name: '版本历史', exact: true }).click();
  await drawer.getByRole('button', { name: '查看版本 1', exact: true }).click();
  check(await drawer.getByRole('button', { name: '恢复版本 1', exact: true }).isDisabled(), '只读权限能看历史但不能恢复');
  await page.setViewportSize({ width: 390, height: 844 });
  await page.waitForFunction(() => document.documentElement.scrollWidth <= innerWidth + 2);
  const mobile = await page.evaluate(() => ({ width: innerWidth, scrollWidth: document.documentElement.scrollWidth }));
  check(mobile.scrollWidth <= mobile.width + 2, '手机宽度不能产生页面横向溢出');
  check(errors.length === 0 && unexpected.length === 0, `不得出现页面错误或未mock请求：${JSON.stringify({ errors, unexpected })}`);
  page.off('dialog', handleDialog);
  const result = { actualFormEnabledPreserved: true, toolCachePreserved: true, unknownCreateRestoredWithoutReplay: true, createRequests: creates.length, staleSavePreservedDraft: true, unknownRestoreKeptOriginal: true, explicitRestoreCreatedNewVersion: true, readOnlyHistory: true, mobile, writes: creates.length + updates.length + restores.length, realBusinessWrites: 0, realModelCalls: 0, pageErrors: errors.length };
  console.log(JSON.stringify(result));
  return result;
}
