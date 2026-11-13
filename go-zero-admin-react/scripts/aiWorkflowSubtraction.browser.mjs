// 已正常登录本地开发栈后执行：playwright-cli run-code --filename scripts/aiWorkflowSubtraction.browser.mjs
// 所有 AI 请求使用响应桩，额外拦截全部 /v1/ 写请求；仅检查界面，不执行模型或真实业务写入。
// oxlint-disable-next-line no-unused-expressions -- 由 Playwright CLI 调用函数表达式。
async page => {
  const assert = (condition, message) => { if (!condition) throw new Error(message); };
  const origin = new URL(page.url()).origin;
  const aiPattern = '**/v1/sys/ai/**';
  const safetyPattern = '**/v1/**';
  const originalPermissions = await page.evaluate(async () => [...(await import('/src/stores/auth.ts')).useAuthStore.getState().permCodes]);
  const requiredPermissions = ['sys:ai:workflow:list', 'sys:ai:workflow:create', 'sys:ai:workflow:edit', 'sys:ai:workflow:run', 'sys:ai:workflow:delete', 'sys:ai:tool:list', 'sys:ai:task:list'];
  const fullPermissions = [...new Set([...originalPermissions, ...requiredPermissions])];
  const setPermissions = codes => page.evaluate(async values => (await import('/src/stores/auth.ts')).useAuthStore.getState().setPermCodes(values), codes);
  const workflowId = 810041;
  const completeId = 810051;
  const runningId = 810052;
  const name = '浏览器桩流程减法';
  const definition = JSON.stringify({ version: 2, steps: [{ id: 'fixture_text', name: '文本统计', tool: 'text_stats', input: { text: '来源折叠保留文本' }, approval: false }] });
  const workflow = { id: workflowId, name, definition, interval_seconds: 0, enabled: 1, next_run_at: 0, created_by: 1, created_at: 1, updated_at: 2, step_count: 1, revision: 1 };
  const task = (id, status) => ({ id, workflow_id: workflowId, conversation_id: 0, status, input: '{"text":"运行输入"}', result: status === 'succeeded' ? '浏览器桩执行结果' : '', error: '', approved_by: 0,
    created_at: 1, completed_at: status === 'succeeded' ? 2 : 0, next_step: status === 'succeeded' ? 3 : 1, steps: [], updated_at: 2,
    request_id: '11111111-1111-4111-8111-111111111111', title: status === 'succeeded' ? '已完成桩记录' : '执行中桩记录', workflow_name: name, input_summary: '桩问题', definition });
  const tasks = [task(completeId, 'succeeded'), task(runningId, 'running')];
  const tool = { name: 'text_stats', description: '文本统计', parameters_schema: '{"type":"object","properties":{"text":{"type":"string"}},"required":["text"]}' };
  const capabilities = { retrieval_configured: false, analysis_configured: false, ingestion_mode: 'disabled' };
  const gets = [];
  const writes = [];
  const unexpected = [];
  let savedStorage = [];
  const handleDialog = dialog => dialog.type() === 'beforeunload' ? dialog.accept() : dialog.dismiss();
  const fulfill = (route, result) => route.fulfill({ json: { code: 200, msg: '操作成功', result } });
  const safetyHandler = async route => {
    if (['GET', 'HEAD', 'OPTIONS'].includes(route.request().method())) { await route.fallback(); return; }
    writes.push({ method: route.request().method(), path: new URL(route.request().url()).pathname });
    await route.fulfill({ json: { code: 409, msg: '浏览器回归已阻止写入', result: null } });
  };
  const aiHandler = async route => {
    const request = route.request(); const url = new URL(request.url());
    if (request.method() !== 'GET') {
      writes.push({ method: request.method(), path: url.pathname });
      await route.fulfill({ json: { code: 409, msg: '浏览器回归已阻止写入', result: null } }); return;
    }
    gets.push(url);
    const path = url.pathname.replace('/v1/sys/ai', '');
    if (path === '/tools') { await fulfill(route, { list: [tool], capabilities }); return; }
    if (path === '/workflows') { await fulfill(route, { list: [workflow], total: 1 }); return; }
    if (path === `/workflows/${workflowId}`) { await fulfill(route, { workflow }); return; }
    if (path === '/tasks') {
      const status = url.searchParams.get('status'); const list = status ? tasks.filter(item => item.status === status) : tasks;
      await fulfill(route, { list, total: list.length }); return;
    }
    if (path === `/tasks/${completeId}` || path === `/tasks/${runningId}`) { await fulfill(route, { task: tasks.find(item => path.endsWith(`/${item.id}`)) }); return; }
    if (path === '/kbs') { await fulfill(route, { list: [], total: 0, capabilities }); return; }
    unexpected.push(path);
    await route.fulfill({ json: { code: 404, msg: '浏览器桩未登记此读取', result: null } });
  };
  const listRow = () => page.getByRole('row').filter({ has: page.getByRole('button', { name, exact: true }) });
  const moreRow = () => listRow().getByRole('button', { name: `更多操作：${name}`, exact: true });
  const waitEditorUnmount = async () => {
    await page.getByRole('button', { name: '返回常用流程列表', exact: true }).waitFor({ state: 'detached' });
    await page.evaluate(() => new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve))));
  };
  const leaveEditor = async () => {
    const path = new URL(page.url()).pathname;
    if (!/^\/ai\/workflows\/(?:new|[1-9]\d*\/edit)$/.test(path)) return;
    const back = page.getByRole('button', { name: '返回常用流程列表', exact: true });
    if (!await back.isVisible()) return;
    await back.click();
    await page.waitForFunction(() => location.pathname === '/ai/workflows'
      || [...document.querySelectorAll('[role="dialog"]')].some(dialog => dialog.textContent.includes('离开常用流程编辑？')));
    const confirmation = page.getByRole('dialog', { name: '离开常用流程编辑？', exact: true });
    if (await confirmation.isVisible()) await confirmation.getByRole('button', { name: /保留草稿并离开$/ }).click();
    await page.waitForURL(`${origin}/ai/workflows`);
    await waitEditorUnmount();
  };
  let firstNavigation = true;
  const go = async path => {
    await leaveEditor();
    if (firstNavigation) {
      firstNavigation = false;
      await page.goto(`${origin}${path}`);
    } else if (new URL(page.url()).pathname !== path) {
      const menuName = path === '/ai/tasks' ? /(?:运行记录|任务中心)$/ : /(?:常用流程|工作流管理)$/;
      await page.getByRole('menuitem', { name: menuName }).click();
    }
    await page.waitForURL(`${origin}${path}`);
    await waitEditorUnmount();
    await setPermissions(fullPermissions);
  };
  const taskReads = () => gets.filter(url => url.pathname.endsWith('/ai/tasks') && url.searchParams.get('page_size') === '20');
  const detailReads = () => gets.filter(url => url.pathname.endsWith(`/tasks/${completeId}`) && !url.searchParams.has('meta_only'));
  try {
    page.on('dialog', handleDialog);
    await page.route(safetyPattern, safetyHandler);
    await page.route(aiPattern, aiHandler);
    // 仅暂存本轮会触碰的两个编辑草稿及对应试运行槽，结束时逐键还原。
    savedStorage = await page.evaluate(async id => {
      const { useAuthStore } = await import('/src/stores/auth.ts');
      const { workflowDraftKey } = await import('/src/pages/ai/workflowDraft.ts');
      const { draftRunAttemptKey } = await import('/src/pages/ai/draftRunAttempts.ts');
      const auth = useAuthStore.getState();
      const keys = [workflowDraftKey(auth.sessionId, auth.selectedDeptId, 'new'), workflowDraftKey(auth.sessionId, auth.selectedDeptId, id)];
      for (const attempt of ['editor:new', `editor:${id}`]) { const key = draftRunAttemptKey([auth.sessionId, auth.selectedDeptId], attempt); keys.push(key, `${key}:task`); }
      return keys.map(key => { const previous = sessionStorage.getItem(key); sessionStorage.removeItem(key); return [key, previous]; });
    }, workflowId);
    await go('/ai/workflows');
    await listRow().waitFor();
    assert(await listRow().getByRole('button', { name: /运\s*行$/ }).isVisible(), '流程行应常显运行');
    assert(await listRow().getByRole('button', { name: /编\s*辑$/ }).isVisible(), '流程行应常显编辑');
    assert(!await listRow().getByRole('button', { name: /复\s*制$/ }).count() && !await listRow().getByRole('button', { name: /删\s*除$/ }).count(), '复制和删除不应常显');
    assert(!(await listRow().textContent()).includes('手动运行') && !(await listRow().textContent()).includes(`#${workflowId}`), '流程行不重复显示ID与手动运行');
    assert(!await page.getByRole('link', { name: '能力目录', exact: true }).count(), '列表不再提供编辑器已有的能力目录跳转');
    await listRow().getByRole('button', { name: /运\s*行$/ }).click();
    const runModal = page.getByRole('dialog', { name: '确认运行常用流程', exact: true });
    await runModal.waitFor();
    await runModal.getByRole('button', { name: /^取\s*消$/ }).click();
    await runModal.waitFor({ state: 'hidden' });
    await moreRow().click();
    await page.getByRole('menuitem', { name: /删除/ }).click();
    const deleteModal = page.getByRole('dialog', { name: '删除常用流程', exact: true });
    await deleteModal.waitFor();
    assert((await deleteModal.textContent()).includes(name), '删除确认应保留明确目标');
    await deleteModal.getByRole('button', { name: /^取\s*消$/ }).click();
    await deleteModal.waitFor({ state: 'hidden' });
    await moreRow().click();
    await page.getByRole('menuitem', { name: /复制/ }).click();
    await page.waitForURL(`**/ai/workflows/new?copy=${workflowId}`);
    await page.getByLabel(/常用流程名称$/).waitFor();
    assert(await page.getByLabel(/常用流程名称$/).inputValue() === `${name} 副本`, '更多中的复制应带入原定义');
    await go('/ai/workflows');
    await listRow().getByRole('button', { name: /编\s*辑$/ }).click();
    await page.waitForURL(`**/ai/workflows/${workflowId}/edit`);
    await page.getByLabel(/常用流程名称$/).waitFor();
    const parameter = page.locator('[data-parameter="text"]');
    const source = parameter.locator('details').first();
    assert(await source.getAttribute('open') === null, '参数来源应默认收起');
    assert((await source.locator('summary').textContent()).includes('填写固定内容'), '参数来源应展示当前摘要');
    await parameter.getByRole('textbox', { name: /处理文本$/ }).fill('来源展开收起后仍保留');
    await source.locator('summary').click();
    await source.getByRole('combobox', { name: '处理文本来源', exact: true }).waitFor();
    await source.locator('summary').click();
    assert(await parameter.getByRole('textbox', { name: /处理文本$/ }).inputValue() === '来源展开收起后仍保留', '来源折叠不能清空固定参数');
    await source.locator('summary').click();
    await source.getByRole('combobox', { name: '处理文本来源', exact: true }).click();
    await page.locator('.ant-select-item-option[title="使用本次运行的文本"]').click();
    await source.locator('summary').click();
    assert((await source.locator('summary').textContent()).includes('使用本次运行的文本'), '来源摘要应反映当前引用');
    const advanced = page.locator('details').filter({ has: page.locator('summary').filter({ hasText: /^高级参数$/ }) });
    await advanced.locator('summary').first().click();
    assert(JSON.parse(await page.getByRole('textbox', { name: '固定参数 JSON', exact: true }).inputValue()).text.$ref === 'input.text', '收起后引用语义必须保留');
    const trial = page.getByRole('region', { name: '试运行', exact: true });
    assert(await trial.getByRole('textbox', { name: /本次文本$/ }).count() === 0, '试运行首次展开前不挂载运行表单');
    await trial.getByRole('button', { name: /试运行$/, exact: false }).click();
    await trial.getByRole('textbox', { name: /本次文本$/ }).fill('试运行折叠保留输入');
    await page.getByLabel(/常用流程名称$/).fill('编辑器保留草稿');
    await trial.getByRole('button', { name: /收起试运行$/ }).click();
    assert(!await trial.getByRole('textbox', { name: /本次文本$/ }).isVisible(), '收起试运行应隐藏表单');
    assert(await page.getByLabel(/常用流程名称$/).inputValue() === '编辑器保留草稿', '收起试运行不能改变编辑草稿');
    await trial.getByRole('button', { name: /试运行$/ }).click();
    assert(await trial.getByRole('textbox', { name: /本次文本$/ }).inputValue() === '试运行折叠保留输入', '再次展开应保留试运行输入');
    await setPermissions(fullPermissions.filter(code => !['sys:ai:workflow:run', 'sys:ai:workflow:create', 'sys:ai:workflow:delete'].includes(code)));
    assert(await trial.getByRole('button', { name: /试运行并查看结果$/ }).isDisabled(), '无运行权限不能提交试运行');
    await go('/ai/tasks');
    const tabs = page.getByRole('tab');
    await tabs.filter({ hasText: '全部' }).waitFor();
    assert(await tabs.count() === 3, '运行记录只常显全部、待确认、执行失败三个标签');
    const records = page.getByLabel('我的运行记录');
    await records.getByRole('button').filter({ hasText: '已完成桩记录' }).waitFor();
    assert(!(await records.getByRole('button').filter({ hasText: '已完成桩记录' }).textContent()).includes('已完成 3 步'), '终态列表不重复步骤数');
    assert((await records.getByRole('button').filter({ hasText: '执行中桩记录' }).textContent()).includes('第 2 步'), '执行中仍保留进度');
    for (const [status, label] of [['pending', '等待执行'], ['running', '执行中'], ['succeeded', '已完成'], ['rejected', '已拒绝'], ['cancelled', '已取消']]) {
      await page.getByRole('button', { name: /更多状态/ }).click();
      const count = taskReads().length;
      const response = page.waitForResponse(item => { const url = new URL(item.url()); return url.pathname.endsWith('/ai/tasks') && url.searchParams.get('status') === status && url.searchParams.get('page_size') === '20'; });
      await page.getByRole('menuitem', { name: label, exact: true }).click(); await response;
      const added = taskReads().slice(count);
      assert(added.length === 1 && added[0].searchParams.getAll('status').length === 1 && added[0].searchParams.get('status') === status, '每个状态只提交一次单status服务端分页查询');
    }
    await page.getByRole('tab', { name: '全部', exact: true }).click();
    await records.getByRole('button').filter({ hasText: '已完成桩记录' }).click();
    const result = page.getByRole('region', { name: '运行结果', exact: true });
    await result.getByText('浏览器桩执行结果', { exact: true }).waitFor();
    assert(!await result.getByRole('button', { name: '对比运行', exact: true }).count(), '对比入口应收进详情更多');
    const beforeMore = detailReads().length;
    await result.getByRole('button', { name: /更多操作$/ }).click();
    const compare = page.getByRole('menuitem', { name: '对比运行', exact: true });
    await compare.waitFor();
    assert(await compare.getAttribute('aria-disabled') !== 'true', '已确认终态允许对比');
    assert(detailReads().length === beforeMore, '展示更多入口不能额外读取详情');
    await compare.click();
    await page.getByRole('dialog', { name: `对比运行 · 左侧 #${completeId}`, exact: true }).waitFor();
    await page.getByRole('button', { name: '关闭对比', exact: true }).click();
    await go('/ai/workflows');
    await setPermissions(fullPermissions.filter(code => !['sys:ai:workflow:create', 'sys:ai:workflow:edit', 'sys:ai:workflow:run', 'sys:ai:workflow:delete', 'sys:ai:tool:list'].includes(code)));
    await listRow().waitFor();
    assert(await listRow().getByRole('button').count() === 1, '只读流程仅保留名称详情入口');
    assert(!await page.getByRole('link', { name: '能力目录', exact: true }).count(), '无工具读取权限时隐藏能力目录');
    await listRow().getByRole('button', { name, exact: true }).click();
    await page.getByRole('dialog', { name: '常用流程详情', exact: true }).waitFor();
    assert(writes.length === 0, '界面操作不应产生任何写请求');
    assert(unexpected.length === 0, `出现未登记的AI读取：${unexpected.join(', ')}`);
    return 'PASS：流程常用操作/更多/删除取消、3标签与5状态单请求、终态进度、缓存对比入口、参数摘要与值引用保留、试运行按需挂载与草稿保留、只读权限；零业务写入';
  } catch (error) {
    throw new Error(`${error.stack ?? error}\n最近AI读取：${gets.slice(-10).map(url => `${url.pathname}${url.search}`).join('\n')}`);
  } finally {
    // 先通过应用返回入口和侧栏做 SPA 导航，等待编辑器及其离开监听卸载，再逐键恢复。
    await go('/ai/workflows');
    await page.evaluate(entries => { for (const [key, previous] of entries) { if (previous === null) sessionStorage.removeItem(key); else sessionStorage.setItem(key, previous); } }, savedStorage);
    await setPermissions(originalPermissions);
    await page.unroute(aiPattern, aiHandler);
    await page.unroute(safetyPattern, safetyHandler);
    page.off('dialog', handleDialog);
    await page.reload();
  }
}
