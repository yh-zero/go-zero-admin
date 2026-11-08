// 独立Playwright会话；所有业务接口由响应桩接管，真实业务写入与模型调用均为0。
// oxlint-disable-next-line no-unused-expressions -- playwright-cli读取此函数表达式执行。
async (page) => {
  const now = Math.floor(Date.now() / 1000), writes = [], stops = [], reads = [], errors = [];
  const require = (condition, text) => { if (!condition) throw new Error(text); };
  const document = { key: 'rules', kb: 'project_ops', source: 'ai_document/ai-rag/business-docs/rules.md', title: '发布规则', sha256: 'a'.repeat(64) };
  const first = { id: 'case-1', kb: 'project_ops', split: 'development', category: 'risk', query: '发布前有什么要求？', rationale: '核对依据', expected_documents: ['rules'], evidence: [{ document: 'rules', groups: [['先备份']] }], answer: { kind: 'answerable', required_groups: [['先备份']], forbidden_phrases: [], absence_policy: 'forbidden' }, checks: [{ label: '先备份', kind: 'risk', required_group: 0 }] };
  const dataset = { version: 1, documents: [document], cases: [first, { ...first, id: 'case-2', query: '上线前怎样处理？' }] }, raw = JSON.stringify(dataset);
  let sha = '', runs = new Map();
  const source = { index: 1, chunk_id: 11, kb_id: 7, doc_id: 9, doc_title: document.title, snippet: '先备份，再发布。', score: 0.9 };
  const analysis = { version: 1, type: 'kb_analysis', status: 'completed', analysis: '先备份，再发布。[1]', sources: [source], usage: { scope: 'chat', prompt_tokens: 20, completion_tokens: 10, total_tokens: 30, complete: true } };
  const context = { binary_sha: 'b'.repeat(64), model: { provider: 'local-test', name: 'test' }, embedding: { provider: 'local-test', model: 'embedding-test', dimension: 16 }, config: { max_context_tokens: 32000, analysis_answer_token_cap: 2048, recall_top_k: 20, return_top_k: 5, analysis_temperature: 0 } };
  const kb = { id: 7, owner_id: 1, name: '质量验收回归资料', description: '', doc_count: 1, chunk_count: 1, ready_count: 1, pending_count: 0, failed_count: 0, created_at: now, updated_at: now };
  function makeRun(id, options = {}) {
    const ids = options.ids ?? ['case-1'], status = options.status ?? 'succeeded', version = options.version ?? 2;
    const observations = ids.map((caseId, index) => {
      const item = dataset.cases.find(item => item.id === caseId), waiting = status === 'running' && index > 0;
      return { case_id: caseId, request_id: `c1155555-5555-4555-8555-${String(id * 100 + index).padStart(12, '0')}`, task_id: waiting ? 0 : id * 100 + index,
        status: waiting ? 'waiting' : status === 'running' ? 'running' : status === 'failed' ? 'failed' : 'succeeded', stage: waiting ? 'waiting' : status === 'running' ? 'analysis' : 'done', query: item.query,
        list: waiting ? [] : [{ ...source, index: undefined }], search_client_ms: waiting ? 0 : 20, search_server_ms: waiting ? 0 : 10, analysis_elapsed_ms: waiting ? 0 : 50,
        ...(status !== 'failed' && status !== 'running' ? { analysis } : {}), error: status === 'failed' ? '模型调用未完成' : '' };
    });
    return { id, kb_id: 7, request_id: options.requestId ?? `a1111111-1111-4111-8111-${String(id).padStart(12, '0')}`, dataset_key: 'test', dataset_sha256: sha, group_key: 'project_ops', status,
      case_count: ids.length, completed_count: status === 'succeeded' ? ids.length : 0, failed_count: status === 'failed' ? 1 : 0, stop_requested: false, corpus_verified: status === 'succeeded', error: status === 'failed' ? '本题未完成，后续题目未开始' : '', snapshot: raw,
      bindings: JSON.stringify({ documents: [{ ...document, kb_id: 7, doc_id: 9, version, status: 'ready', embedding_model: 'embedding-test', content_sha256: document.sha256 }], inventory: [{ id: 9, kb: 'project_ops', kb_id: 7, title: document.title, version, status: 'ready', embedding_model: 'embedding-test' }] }),
      context: JSON.stringify({ ...context, binary_sha: options.binary ?? context.binary_sha }), observations: JSON.stringify(observations), created_at: now, updated_at: now, started_at: now, completed_at: status === 'running' ? 0 : now };
  }
  const lite = value => ({ ...value, snapshot: '', bindings: '', context: '', observations: '' });
  page.on('pageerror', error => errors.push(error.message));
  await page.addInitScript(() => {
    localStorage.setItem('go-zero-auth', JSON.stringify({ state: { token: 'quality-fixture-access', refreshToken: 'quality-fixture-refresh', expiresAt: Date.now() / 1000 + 3600, refreshExpiresAt: Date.now() / 1000 + 7200, sessionId: 'quality-fixture-session' }, version: 0 }));
    const original = Storage.prototype.setItem;
    Storage.prototype.setItem = function(key, value) { if (this === sessionStorage && key.startsWith('ai-kb-quality:')) throw new Error('quota fixture'); return original.call(this, key, value); };
  });
  await page.route('**/v1/**', async route => {
    const request = route.request(), relative = request.url().replace(/^https?:\/\/[^/]+/, ''), [path, query = ''] = relative.split('?'), values = new Map(query.split('&').map(pair => pair.split('=')));
    const send = (code, result = {}, msg = '') => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ code, result, msg }) });
    if (path === '/v1/sys/auth/captcha') return send(200, { enabled: false });
    if (path === '/v1/sys/dept/my') return send(200, { list: [] });
    if (path === '/v1/sys/menu/my/perm-codes') return send(200, { codes: ['sys:ai:kb:list', 'sys:ai:kb:evaluate', 'sys:ai:task:list', 'sys:ai:task:feedback'] });
    if (path === '/v1/sys/menu/my') return send(200, { list: [{ id: 771, parent_id: 0, title: '知识库', name: 'ai_kb', type: 2, path: '/ai/knowledge', component: 'ai_kb', perm_code: '', icon: '', sort: 1, visible: true, status: 1, created_at: 0 }] });
    if (path === '/v1/sys/profile/') return send(200, { user: { id: 1, username: 'quality-fixture', nickname: '质量回归', email: '', phone: '', avatar_url: '', dept_id: 0, role_ids: [] } });
    if (path === '/v1/sys/ai/kbs/7') return send(200, { kb, capabilities: { retrieval_configured: true, analysis_configured: true, ingestion_mode: 'async' } });
    if (path === '/v1/sys/ai/kbs/7/docs') return send(200, { list: [{ id: 9, kb_id: 7, title: document.title, status: 'ready', version: 2, embedding_model: 'embedding-test', created_at: now, updated_at: now, chunk_count: 1, stage: 'ready' }], total: 1, ready_count: 1, pending_count: 0, failed_count: 0 });
    if (path === '/v1/sys/ai/kbs/7/evaluation-datasets') return send(200, { list: [{ key: 'test', name: '场景验收题集', sha256: sha, dataset: raw }] });
    if (path === '/v1/sys/ai/kbs/7/evaluations') {
      if (request.method() === 'POST') {
        const data = request.postDataJSON(); writes.push(data);
        require(!Object.hasOwn(data, 'answer') && !Object.hasOwn(data, 'checks'), '不能把黄金标注作为模型运行参数');
        if (writes.length === 1) { runs.set(4, makeRun(4, { ids: data.case_ids, status: 'running', requestId: data.request_id })); return send(503, {}, '提交结果待确认'); }
        require(writes.length === 2 && data.request_id !== writes[0].request_id && data.case_ids.length === 1 && data.case_ids[0] === 'case-1', '单题复测必须明确使用新标识并保留旧批次');
        const run = makeRun(6, { ids: data.case_ids, requestId: data.request_id }); runs.set(6, run); return send(200, { evaluation: run });
      }
      reads.push(relative);
      const list = [...runs.values()].filter(run => (!values.get('request_id') || run.request_id === values.get('request_id')) && (!values.get('status') || run.status === values.get('status'))).sort((a, b) => b.id - a.id);
      return send(200, { list: list.map(lite), total: list.length });
    }
    const detail = path.match(/^\/v1\/sys\/ai\/kbs\/7\/evaluations\/(\d+)(\/stop)?$/);
    if (detail) {
      const id = Number(detail[1]), run = runs.get(id); require(run, '未定义批次');
      if (detail[2]) {
        stops.push(id); require(id === 4 && request.method() === 'POST', '只停止明确批次');
        const observations = JSON.parse(run.observations); observations[0] = { ...observations[0], status: 'succeeded', stage: 'done', analysis };
        const stopped = { ...run, status: 'cancelled', stop_requested: true, completed_count: 1, completed_at: now, observations: JSON.stringify(observations) }; runs.set(id, stopped); return send(200, { evaluation: stopped });
      }
      return send(200, { evaluation: run });
    }
    if (/^\/v1\/sys\/ai\/tasks\/\d+\/feedback$/.test(path) && request.method() === 'GET') return send(200, { list: [] });
    throw new Error(`响应桩未定义业务请求：${request.method()} ${path}`);
  });
  await page.goto('http://127.0.0.1:5999/login');
  sha = await page.evaluate(async value => [...new Uint8Array(await crypto.subtle.digest('SHA-256', new TextEncoder().encode(value)))].map(x => x.toString(16).padStart(2, '0')).join(''), raw);
  runs = new Map([[1, makeRun(1)], [2, makeRun(2, { binary: 'c'.repeat(64) })], [3, makeRun(3, { status: 'failed' })], [5, makeRun(5, { version: 3 })]]);
  await page.goto('http://127.0.0.1:5999/ai/knowledge/7/quality');
  await page.getByRole('combobox', { name: '选择资料题组', exact: true }).click();
  await page.locator('.ant-select-dropdown').getByText('项目运维资料', { exact: true }).click();
  await page.getByRole('button', { name: '开始评测', exact: true }).dblclick();
  await page.getByText('上次提交结果待确认', { exact: true }).waitFor();
  await page.getByText('浏览器无法保存本次标识，当前页面仍保留。离开前请先核对本次批次。', { exact: true }).waitFor();
  require(writes.length === 1, '双击只能创建一次，未知结果不得自动重试');
  await page.getByRole('button', { name: '核对原请求', exact: true }).click();
  await page.getByText('批次 #4', { exact: true }).waitFor();
  require(reads.some(value => value.includes(`request_id=${writes[0].request_id}`)), '核对必须使用原UUID');
  const saved = page.url();
  await page.goto('http://127.0.0.1:5999/ai/knowledge/7');
  await page.getByRole('button', { name: '质量验收', exact: true }).waitFor();
  await page.goto(saved);
  await page.getByRole('button', { name: '停止后续题目', exact: true }).click();
  await page.getByText('批次已停止，未开始的题目没有执行。', { exact: true }).waitFor();
  require(stops.length === 1 && writes.length === 1, '离页恢复和停止不能追加评测提交');
  const row = id => page.getByRole('row').filter({ has: page.getByText(`#${id}`, { exact: true }) });
  await row(1).getByRole('button', { name: '设为对照', exact: true }).click();
  await row(2).getByRole('button', { name: '查看批次 #2', exact: true }).click();
  await page.getByText('对照 #1 → 当前 #2', { exact: true }).waitFor();
  await row(5).getByRole('button', { name: '查看批次 #5', exact: true }).click();
  await page.getByText('这两份结果不能计算提升', { exact: true }).waitFor();
  await page.getByText('语料版本、正文指纹或库存不同', { exact: true }).waitFor();
  await row(3).getByRole('button', { name: '查看批次 #3', exact: true }).click();
  await page.getByText('执行未完成', { exact: true }).first().waitFor();
  await row(2).getByRole('button', { name: '查看批次 #2', exact: true }).click();
  await page.getByRole('button', { name: '单题复测', exact: true }).click();
  await page.getByText('批次 #6', { exact: true }).waitFor();
  require(writes.length === 2 && runs.has(2) && runs.has(4), '复测保留原批次');
  await page.locator('.ant-table-row-expand-icon').first().click();
  await page.getByRole('region', { name: '题目详情 case-1', exact: true }).getByText('字符串命中', { exact: true }).waitFor();
  await page.getByRole('button', { name: '评价分析', exact: true }).click();
  await page.getByRole('dialog', { name: '评价第 1 步分析', exact: true }).waitFor();
  await page.getByRole('dialog', { name: '评价第 1 步分析', exact: true }).getByRole('button', { name: '关闭', exact: true }).click();
  await page.setViewportSize({ width: 390, height: 844 });
  await page.getByRole('heading', { name: '质量验收回归资料 · 质量验收', exact: true }).waitFor();
  const mobile = await page.evaluate(() => ({ viewport: innerWidth, page: document.documentElement.scrollWidth }));
  require(mobile.page <= mobile.viewport + 2, '窄屏不能产生整页横向溢出');
  require(errors.length === 0, `页面错误：${errors.join('; ')}`);
  return { startWrites: writes.length, stopWrites: stops.length, unknownOriginalRecovered: true, storageFailureKeptVisible: true, pageAwayRecovered: true, incompatibleComparisonRejected: true, feedbackOpened: true, mobile, pageErrors: errors.length, realBusinessWrites: 0, realModelCalls: 0 };
}
