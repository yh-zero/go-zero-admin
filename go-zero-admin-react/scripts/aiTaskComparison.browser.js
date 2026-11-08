// 独立浏览器响应桩；仅放行5999静态资源，业务接口显式mock，其他请求全部拒绝。
// oxlint-disable-next-line no-unused-expressions -- playwright-cli读取此函数表达式执行。
async (page) => {
  const check = (value, message) => { if (!value) throw new Error(message); };
  const now = Math.floor(Date.now() / 1000), reads = [], unexpected = [], errors = [];
  let failRight = false, holdRight = false, otherIdentity = false, releaseHeld, markHeld;
  const held = new Promise(resolve => { markHeld = resolve; });
  const analysis = (body, title, doc) => ({ version: 1, type: 'kb_analysis', status: 'completed', analysis: body,
    sources: [{ index: 1, chunk_id: doc * 10, kb_id: 7, doc_id: doc, doc_title: title, snippet: '保存于任务中的实际历史片段', score: 0.8 }], usage: { scope: 'chat', prompt_tokens: 10, completion_tokens: 5, total_tokens: 15, complete: true } });
  const step = (id, index, tool, output) => ({ id, index, name: tool === 'kb_analyze' ? '资料分析' : '统计文字', tool, input: '{"text":"本次输入"}', output, status: 'succeeded', error: '', started_at: 100, completed_at: 130 });
  const leftAnalysis = analysis('左侧可靠结果[1]', '甲侧历史原文', 9), rightAnalysis = analysis('右侧不同结果[1]', '乙侧历史原文', 10);
  const makeTask = (id, title, steps, result) => ({ id, title, workflow_id: 71, workflow_name: '记录中的流程', conversation_id: 0, status: 'succeeded', input: '{"question":"比较问题"}', input_summary: '比较问题', result, error: '', steps, next_step: steps.length,
    request_id: `fixture-${id}`, approved_by: 0, created_at: now - 8, updated_at: now - 5, completed_at: now - 5,
    definition: JSON.stringify({ version: 2, steps: steps.map(item => ({ id: item.id, name: item.name, tool: item.tool, input: {} })) }) });
  const left = makeTask(101, '运行甲', [step('stats', 0, 'text_stats', '{"characters":4}'), step('analysis', 1, 'kb_analyze', JSON.stringify(leftAnalysis))], `[1/2] 统计: {"characters":4}\n[2/2] 分析: ${JSON.stringify(leftAnalysis)}`);
  const right = makeTask(102, '比较乙', [step('analysis', 0, 'kb_analyze', JSON.stringify(rightAnalysis)), step('stats', 1, 'text_stats', '{"characters":6}')], `[1/2] 分析: ${JSON.stringify(rightAnalysis)}\n[2/2] 统计: {"characters":6}`);
  const lite = task => ({ ...task, input: '', result: '', definition: '', steps: [] });
  page.on('pageerror', error => errors.push(error.message));
  await page.setViewportSize({ width: 1280, height: 900 });
  await page.addInitScript(() => localStorage.setItem('go-zero-auth', JSON.stringify({ state: { token: 'comparison-fixture-access', refreshToken: 'comparison-fixture-refresh', sessionId: 'comparison-fixture-session', expiresAt: Date.now() / 1000 + 3600, refreshExpiresAt: Date.now() / 1000 + 7200 }, version: 0 })));
  await page.route('**/*', route => {
    const url = route.request().url(), path = url.replace(/^https?:\/\/[^/]+/, '');
    if (!url.startsWith('http://127.0.0.1:5999/') || path.startsWith('/v1/')) { unexpected.push({ url: url.split('?')[0], method: route.request().method() }); return route.abort('blockedbyclient'); }
    return route.continue();
  });
  await page.route('**/v1/**', async route => {
    const request = route.request(), url = request.url(), relative = url.replace(/^https?:\/\/[^/]+/, ''), [path, query = ''] = relative.split('?');
    const send = (result, code = 200) => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ code, result, msg: code === 500 ? '比较桩读取失败' : '' }) });
    if (request.method() !== 'GET') { unexpected.push({ path, method: request.method() }); return route.abort('blockedbyclient'); }
    if (path === '/v1/sys/auth/captcha') return send({ enabled: false });
    if (path === '/v1/sys/dept/my') return send({ list: [] });
    if (path === '/v1/sys/menu/my/perm-codes') return send({ codes: ['sys:ai:task:list'] });
    if (path === '/v1/sys/menu/my') return send({ list: [{ id: 901, parent_id: 0, title: '运行记录', name: 'ai_task', path: '/ai/tasks', component: 'ai_task', type: 2, visible: true, status: 1, sort: 0 }] });
    if (path === '/v1/sys/profile/') return send({ user: { id: otherIdentity ? 2 : 1, username: 'comparison-fixture', nickname: otherIdentity ? '其他身份' : '对照验收', email: '', phone: '', avatar_url: '', dept_id: 0, role_ids: [] } });
    if (path === '/v1/sys/ai/tasks') {
      reads.push({ kind: 'list', url: relative });
      if (otherIdentity) return send({ list: [], total: 0 });
      if (query.includes('page_size=10')) {
        if (query.includes('query=') || query.includes('page=2')) return send({ list: [lite(right)], total: query.includes('query=') ? 1 : 11 });
        return send({ list: [lite(left), ...Array.from({ length: 9 }, (_, index) => ({ ...lite(right), id: 110 + index, title: `其他运行${index + 1}` }))], total: 11 });
      }
      return send({ list: [lite(left), lite(right)], total: 2 });
    }
    if (/^\/v1\/sys\/ai\/tasks\/\d+\/feedback$/.test(path)) return send({ list: [] });
    if (path === '/v1/sys/ai/tasks/101' || path === '/v1/sys/ai/tasks/102') {
      const id = path.endsWith('/101') ? 101 : 102;
      reads.push({ kind: 'detail', id });
      if (otherIdentity) return send({}, 404);
      if (id === 102 && failRight) return send({}, 500);
      if (id === 102 && holdRight) { markHeld(); await new Promise(resolve => { releaseHeld = resolve; }); }
      try { return await send({ task: id === 101 ? left : right }); } catch { return; }
    }
    unexpected.push({ path, method: request.method() });
    return route.abort('blockedbyclient');
  });
  await page.goto('http://127.0.0.1:5999/ai/tasks?task_id=101');
  const compare = page.getByRole('button', { name: '对比运行', exact: true });
  await compare.waitFor();
  await page.getByRole('heading', { name: '执行结果' }).waitFor();
  const before = { lists: reads.filter(r => r.kind === 'list').length, left: reads.filter(r => r.kind === 'detail' && r.id === 101).length };
  await compare.click();
  const drawer = page.getByRole('dialog', { name: '对比运行 · 左侧 #101', exact: true });
  await drawer.getByRole('button', { name: '选择运行 #101', exact: true }).waitFor();
  check(await drawer.getByRole('button', { name: '选择运行 #101', exact: true }).isDisabled(), '不能选择自己比较');
  await drawer.getByTitle('2', { exact: true }).click();
  await drawer.getByRole('button', { name: '选择运行 #102', exact: true }).waitFor();
  await drawer.getByLabel('搜索另一条运行', { exact: true }).fill('比较');
  await drawer.getByLabel('搜索另一条运行', { exact: true }).press('Enter');
  await drawer.getByRole('button', { name: '选择运行 #102', exact: true }).click();
  await drawer.getByRole('heading', { name: '结果与实际来源', exact: true }).waitFor();
  await drawer.getByText('甲侧历史原文', { exact: true }).waitFor();
  await drawer.getByText('乙侧历史原文', { exact: true }).waitFor();
  check(await drawer.getByText('执行位置变化', { exact: true }).count() === 2, '重排步骤必须继续按稳定ID配对');
  const initialReads = { candidateLists: reads.filter(r => r.kind === 'list').length - before.lists, leftDetails: reads.filter(r => r.kind === 'detail' && r.id === 101).length - before.left, rightDetails: reads.filter(r => r.kind === 'detail' && r.id === 102).length };
  check(initialReads.candidateLists === 3 && initialReads.leftDetails === 1 && initialReads.rightDetails === 1, `打开/翻页/搜索各一次列表和左右各一次详情：${JSON.stringify(initialReads)}`);
  await page.setViewportSize({ width: 390, height: 844 });
  await page.waitForFunction(() => {
    const rect = document.querySelector('[role="dialog"]').getBoundingClientRect();
    return document.documentElement.scrollWidth <= innerWidth + 2 && rect.left >= -1 && rect.right <= innerWidth + 1;
  });
  const mobile = await drawer.evaluate(element => ({ width: innerWidth, scrollWidth: document.documentElement.scrollWidth, left: element.getBoundingClientRect().left, right: element.getBoundingClientRect().right }));
  await page.screenshot({ path: '.playwright-cli/task-comparison-mobile.png', scale: 'css', type: 'png' });
  failRight = true;
  await drawer.getByRole('button', { name: '重新加载对照记录', exact: true }).click();
  await drawer.getByText('比较桩读取失败', { exact: true }).waitFor();
  check(await drawer.getByLabel('右侧历史结果', { exact: true }).count() === 0, '详情失败不能继续展示旧缓存结果');
  failRight = false;
  await drawer.getByRole('button', { name: '重新加载对照记录', exact: true }).click();
  await drawer.getByRole('heading', { name: '结果与实际来源', exact: true }).waitFor();
  await drawer.getByRole('button', { name: '更换另一条运行', exact: true }).click();
  holdRight = true;
  await drawer.getByRole('button', { name: '选择运行 #102', exact: true }).click();
  await held;
  const cancelled = page.waitForEvent('requestfailed', { predicate: request => request.url().endsWith('/v1/sys/ai/tasks/102'), timeout: 5000 });
  try { await drawer.getByRole('button', { name: '关闭对比', exact: true }).click(); await cancelled; }
  finally { releaseHeld?.(); }
  await drawer.waitFor({ state: 'hidden' });
  holdRight = false;
  await compare.click();
  await drawer.waitFor();
  otherIdentity = true;
  await page.evaluate(() => {
    localStorage.setItem('go-zero-auth', JSON.stringify({ state: { token: 'comparison-other-fixture', refreshToken: 'comparison-other-refresh', sessionId: 'comparison-other-session', expiresAt: Date.now() / 1000 + 3600, refreshExpiresAt: Date.now() / 1000 + 7200 }, version: 0 }));
    window.dispatchEvent(new StorageEvent('storage', { key: 'go-zero-auth', storageArea: localStorage }));
  });
  await drawer.waitFor({ state: 'hidden' });
  await page.getByText('暂无符合条件的任务', { exact: true }).waitFor();
  check(errors.length === 0 && unexpected.length === 0, `不得出现页面错误或未知请求：${JSON.stringify({ errors, unexpected })}`);
  return { selectionSearchAndPaging: true, staleCacheHiddenOnFailure: true, closeCancelledDetail: true, identityIsolated: true, initialReads, mobile, realBusinessWrites: 0, realModelCalls: 0, pageErrors: errors.length };
}
