// 独立 playwright-cli 会话；业务读写全部由响应桩处理。
// 初始页面 URL 的 receipt_scenario 可选 quota（默认）、partial 或 edit。
// oxlint-disable-next-line no-unused-expressions -- playwright-cli 执行函数表达式。
async page => {
  const origin = new URL(page.url()).origin;
  const scenario = new URL(page.url()).searchParams.get('receipt_scenario') ?? 'quota';
  const assert = (value, message) => { if (!value) throw new Error(message); };
  assert(['quota', 'partial', 'edit'].includes(scenario), '场景仅允许 quota、partial 或 edit');
  const sessionId = 'import-receipt-browser-session';
  const batchId = '11111111-1111-4111-8111-111111111111';
  const requestA = '22222222-2222-4222-8222-222222222222';
  const requestB = '33333333-3333-4333-8333-333333333333';
  const errors = [], writes = [], importReads = [];
  page.on('pageerror', error => errors.push(error.message));
  page.on('console', message => { if (/Maximum update depth|Too many re-renders/.test(message.text())) errors.push(message.text()); });
  await page.addInitScript(({ sessionId }) => {
    localStorage.setItem('go-zero-auth', JSON.stringify({ state: { token: 'receipt-browser-fixture-access', refreshToken: 'receipt-browser-fixture-refresh', sessionId, expiresAt: Date.now() / 1000 + 3600, refreshExpiresAt: Date.now() / 1000 + 7200 }, version: 0 }));
    if (window.__receiptStoragePatched) return;
    window.__receiptStoragePatched = true;
    const original = Storage.prototype.setItem;
    const originalGet = Storage.prototype.getItem;
    Storage.prototype.getItem = function (key) {
      if (this === sessionStorage && key.startsWith('ai-knowledge-import:') && key.endsWith(':pending')) window.__receiptPendingReads = (window.__receiptPendingReads ?? 0) + 1;
      return originalGet.call(this, key);
    };
    Storage.prototype.setItem = function (key, value) {
      if (this === sessionStorage && key.startsWith('ai-knowledge-import:') && (window.__receiptQuota === 'all' || window.__receiptQuota === 'pointer' && !key.endsWith(':pending'))) throw new DOMException('fixture quota', 'QuotaExceededError');
      return original.call(this, key, value);
    };
  }, { sessionId });
  const kb = { id: 7, name: '回执组件回归', description: '', ready_count: 1, pending_count: 0, failed_count: 0, doc_count: 1, chunk_count: 1, updated_at: 1 };
  let document = { id: 9, kb_id: 7, title: '待编辑资料', status: 'ready', stage: 'ready', chunk_count: 1, embedding_model: 'local-fixture', version: 1, error: '', created_at: 1, updated_at: 1 };
  let savedContent = '原始正文';
  await page.route('**/v1/**', route => {
    const request = route.request(), path = request.url().replace(/^https?:\/\/[^/]+/, '').split('?')[0];
    const send = (code, result = {}) => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ code, result, msg: code === 200 ? '' : '响应桩结果未知' }) });
    if (request.method() === 'POST' && path === '/v1/sys/ai/kbs/7/docs') { writes.push(request.postDataJSON()); return send(503); }
    if (scenario === 'edit' && path === '/v1/sys/ai/kbs/7/docs/9') {
      if (request.method() === 'GET') return send(200, { doc: document, content: savedContent });
      if (request.method() === 'PUT') {
        const patch = request.postDataJSON();
        assert(patch.expected_version === document.version && !!patch.request_id, '保存提交原文版本和请求标识');
        writes.push(patch);
        const status = patch.content === savedContent ? 'ready' : 'pending';
        document = { ...document, title: patch.title, status, stage: status, version: document.version + 1, updated_at: document.updated_at + 1 };
        savedContent = patch.content;
        return send(200, { doc: document, job_id: document.version, batch_id: patch.request_id });
      }
    }
    if (request.method() !== 'GET') return route.abort('blockedbyclient');
    if (path.startsWith('/v1/sys/ai/kbs/7/imports')) importReads.push(path);
    if (path === '/v1/sys/dept/my') return send(200, { list: [] });
    if (path === '/v1/sys/menu/my/perm-codes') return send(200, { codes: ['sys:ai:kb:list', 'sys:ai:kb:edit'] });
    if (path === '/v1/sys/menu/my') return send(200, { list: [{ id: 901, parent_id: 0, title: '知识库', name: 'ai_kb', type: 2, path: '/ai/knowledge', component: 'ai_kb', perm_code: '', icon: '', sort: 0, visible: true, status: 1, created_at: 0 }] });
    if (path === '/v1/sys/profile/') return send(200, { user: { id: 1, username: 'receipt-fixture', nickname: '回执回归', email: '', phone: '', avatar_url: '', dept_id: 0, role_ids: [] } });
    if (path === '/v1/sys/ai/kbs/7') return send(200, { kb, capabilities: { retrieval_configured: true, analysis_configured: true, ingestion_mode: 'background' } });
    if (path === '/v1/sys/ai/kbs/7/docs') return send(200, { list: scenario === 'edit' ? [document] : [], total: scenario === 'edit' ? 1 : 0, ready_count: document.status === 'ready' ? 1 : 0, pending_count: document.status === 'pending' ? 1 : 0, failed_count: 0 });
    if (path === '/v1/sys/ai/kbs/7/imports') return send(200, { list: scenario === 'partial' ? [{ id: batchId, kb_id: 7, total: 1, queued: 0, running: 0, succeeded: 1, failed: 0, cancelled: 0, reused: 0, created_at: 1, updated_at: 1, stopped: false }] : [], total: scenario === 'partial' ? 1 : 0 });
    if (path.startsWith('/v1/sys/ai/kbs/7/imports/')) {
      if (scenario !== 'partial') return send(404);
      return send(200, { batch: { id: batchId, kb_id: 7, total: 1, queued: 0, running: 0, succeeded: 1, failed: 0, cancelled: 0, reused: 0, stopped: false }, list: [{ id: 1, request_id: requestA, doc_id: 9, status: 'succeeded', operation: 'create', error: '', doc: { id: 9, title: '已接收资料', status: 'ready' } }] });
    }
    return send(404);
  });
  await page.goto(`${origin}/ai/knowledge/7`);
  await page.getByRole('heading', { name: '回执组件回归', exact: true }).waitFor();
  if (scenario === 'edit') {
    await page.getByRole('button', { name: document.title, exact: true }).click();
    const dialog = page.getByRole('dialog');
    await dialog.getByRole('button', { name: '编辑原文', exact: true }).click();
    assert(await dialog.getByRole('button', { name: '保存修改' }).isDisabled(), '未编辑时不提交');
    await dialog.getByRole('textbox', { name: /标题$/ }).fill('改名后的资料');
    await dialog.getByRole('button', { name: '保存标题' }).click();
    await dialog.getByText('“改名后的资料”已就绪，可以检索', { exact: true }).waitFor();
    await dialog.getByRole('button', { name: '编辑原文' }).click();
    await dialog.getByRole('textbox', { name: /正文（纯文本）$/ }).fill('新版正文');
    await dialog.getByRole('button', { name: '保存并重新处理' }).click();
    await dialog.getByText(/资料已接收，后台继续处理/).waitFor();
    assert(writes.length === 2 && writes[0].content === '原始正文' && writes[1].content === '新版正文', '改名提交原始正文，正文变更使用新版本');
    assert(errors.length === 0, '修改后无页面错误');
    return { scenario, unchangedWrites: 0, renameReady: true, editedBodyPending: true, realBusinessWrites: 0 };
  }
  if (scenario === 'partial') {
    await page.evaluate(({ sessionId, batchId, requestA, requestB }) => sessionStorage.setItem(`ai-knowledge-import:${JSON.stringify([sessionId, null, 7])}:pending`, JSON.stringify({ batchId, requestIds: [requestA, requestB], unsubmittedIds: [] })), { sessionId, batchId, requestA, requestB });
    await page.getByRole('button', { name: '查看导入批次', exact: true }).click();
    await page.getByText('部分接收结果尚未确认', { exact: true }).waitFor();
    await page.waitForTimeout(300); // 检查无变化effect在多个浏览器帧后仍稳定，而非等待接口。
    const pendingReads = await page.evaluate(() => window.__receiptPendingReads ?? 0);
    assert(pendingReads < 16, `无变化effect应稳定，实际回执读取次数：${pendingReads}`);
    assert(errors.length === 0, `不应出现effect循环：${errors[0] ?? ''}`);
    const pending = await page.evaluate(sessionId => JSON.parse(sessionStorage.getItem(`ai-knowledge-import:${JSON.stringify([sessionId, null, 7])}:pending`)), sessionId);
    assert(pending.requestIds.length === 1 && pending.requestIds[0] === requestB, 'GET只应移除已接收的A，保留B');
    await page.getByRole('button', { name: '关闭并返回工作区', exact: true }).click({ timeout: 2000 });
    return { scenario, stableEffect: true, pendingReads, remainingUnknown: 1, writes: writes.length, pageErrors: errors.length };
  }
  for (const quota of ['pointer', 'all']) {
    await page.evaluate(({ quota, sessionId, batchId }) => {
      sessionStorage.clear();
      sessionStorage.setItem(`ai-knowledge-import:${JSON.stringify([sessionId, null, 7])}`, batchId);
      window.__receiptQuota = quota;
    }, { quota, sessionId, batchId });
    const readsBeforePaste = importReads.length;
    await page.getByRole('button', { name: /添加资料$/ }).click();
    await page.getByRole('menuitem', { name: /粘贴资料$/ }).click();
    const dialog = page.getByRole('dialog');
    await dialog.getByRole('textbox', { name: /标题$/ }).fill(`配额回归${quota}`);
    await dialog.getByRole('textbox', { name: /正文（纯文本）$/ }).fill('回执组件的受控资料。');
    assert(importReads.length === readsBeforePaste, '新建导入时不请求历史批次或旧批次详情');
    await dialog.getByRole('button', { name: '接收并后台处理', exact: true }).click();
    await dialog.getByText('部分接收结果尚未确认', { exact: true }).waitFor({ timeout: 2500 });
    assert(await dialog.getByRole('button', { name: '接收并后台处理', exact: true }).isDisabled(), '未知POST后必须禁止新ID重发');
    await dialog.getByRole('button', { name: '关闭并返回工作区', exact: true }).click();
    await page.getByRole('button', { name: '查看导入批次', exact: true }).click();
    await page.getByRole('dialog').getByText('部分接收结果尚未确认', { exact: true }).waitFor({ timeout: 2500 });
    await page.getByRole('dialog').getByRole('button', { name: '关闭并返回工作区', exact: true }).click();
    await page.reload();
    await page.getByRole('heading', { name: '回执组件回归', exact: true }).waitFor();
  }
  assert(writes.length === 2 && writes.every(write => write.request_id && write.batch_id), '每个用户意图只发送一次，保留原请求和批次ID');
  assert(errors.length === 0, `浏览器错误：${errors[0] ?? ''}`);
  return { scenario, pointerQuota: true, allQuota: true, unknownLockedAcrossModal: true, businessWrites: writes.length, realBusinessWrites: 0 };
}
