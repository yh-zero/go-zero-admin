// 全拦截独立浏览器：仅当前同源静态资源放行，业务读及显式评测提交均由响应桩处理。
// oxlint-disable-next-line no-unused-expressions -- playwright-cli执行此函数表达式。
async (page) => {
  const origin = new URL(page.url()).origin;
  const check = (condition, message) => { if (!condition) throw new Error(message); };
  const ensureAdvancedOpen = async () => {
    const summary = page.getByText('高级设置', { exact: true });
    await summary.waitFor();
    if (!await summary.evaluate(node => node.parentElement.open)) await summary.click();
  };
  const now = Math.floor(Date.now() / 1000), reads = [], writes = [], unexpected = [], errors = [];
  const content = '发布前必须备份数据库。验收通过后才可切换流量。';
  const doc = { id: 9, kb_id: 7, title: '发布资料', status: 'ready', version: 2, embedding_model: 'fixture', chunk_count: 1, stage: 'ready', error: '', created_at: now, updated_at: now };
  const source = { index: 1, kb_id: 7, doc_id: 9, chunk_id: 11, doc_title: doc.title, snippet: content, score: 0.8 };
  let digest = '', stale = true, negative = false, denyRead = false, run = null, lookupReads = 0;
  const feedback = () => ({ id: 3, kb_id: 7, task_id: 8, step_index: 0, feedback: 'incorrect_answer', note: '', status: 'accepted', revision: 4, question: '如何发布？', answer: '历史回答', created_at: now, updated_at: now, reviewed_at: now,
    snapshot: JSON.stringify({ version: 1, kb_id: 7, query: '如何发布？', step_input: '{}', step_output: '{}', sources: negative ? [] : [source], documents: negative ? [] : [{ doc_id: 9, kb_id: 7, title: doc.title, version: 2, status: 'verified', content_sha256: digest }] }),
    review: negative ? { answerable: false, expected_answer: '当前资料不足，无法确定。', required_facts: [], evidence: [], reason: '人工核对当前资料' }
      : { answerable: true, expected_answer: '先备份数据库，再切换流量。', required_facts: ['备份数据库', '切换流量'], evidence: [{ source_index: 1, quote: content }], reason: '人工逐字核对' } });
  const kb = { id: 7, owner_id: 1, name: '冻结题集回归库', description: '', doc_count: 1, chunk_count: 1, ready_count: 1, pending_count: 0, failed_count: 0, created_at: now, updated_at: now };
  const context = { binary_sha: 'b'.repeat(64), model: { provider: 'fixture', name: 'fixture' }, embedding: { provider: 'fixture', model: 'fixture', dimension: 16 }, config: { max_context_tokens: 32000, analysis_answer_token_cap: 2048, recall_top_k: 20, return_top_k: 5, analysis_temperature: 0 } };
  const lite = value => ({ ...value, snapshot: '', bindings: '', context: '', observations: '' });
  const makeRun = request => {
    const dataset = JSON.parse(request.dataset_snapshot), documents = dataset.documents.map(({ content: _content, source_doc_id: _id, source_doc_version: _version, ...item }) => ({ ...item, kb_id: 7, doc_id: doc.id, version: doc.version, status: 'ready', embedding_model: 'fixture', content_sha256: item.sha256 }));
    return { id: 4, kb_id: 7, request_id: request.request_id, dataset_key: request.dataset_key, dataset_sha256: request.dataset_sha256, group_key: request.group_key,
      status: 'cancelled', case_count: request.case_ids.length, completed_count: 0, failed_count: 0, stop_requested: true, corpus_verified: false, error: '', snapshot: request.dataset_snapshot,
      bindings: JSON.stringify({ documents, inventory: [{ id: doc.id, kb: request.group_key, kb_id: 7, title: doc.title, version: doc.version, status: 'ready', embedding_model: 'fixture' }] }), context: JSON.stringify(context),
      observations: JSON.stringify(request.case_ids.map(id => ({ case_id: id, request_id: 'c1155555-5555-4555-8555-000000000401', task_id: 0, status: 'waiting', stage: 'waiting', query: dataset.cases.find(item => item.id === id).query, list: [], search_client_ms: 0, search_server_ms: 0, analysis_elapsed_ms: 0, error: '' }))),
      created_at: now, updated_at: now, started_at: 0, completed_at: now };
  };
  page.on('pageerror', error => errors.push(error.message));
  await page.setViewportSize({ width: 1280, height: 900 });
  await page.addInitScript(() => localStorage.setItem('go-zero-auth', JSON.stringify({ state: { token: 'dataset-fixture-access', refreshToken: 'dataset-fixture-refresh', sessionId: 'dataset-fixture-session', expiresAt: Date.now() / 1000 + 3600, refreshExpiresAt: Date.now() / 1000 + 7200 }, version: 0 })));
  await page.route('**/*', route => {
    const url = route.request().url(), path = url.replace(/^https?:\/\/[^/]+/, '');
    if (!url.startsWith(`${origin}/`) || path.startsWith('/v1/')) { unexpected.push(url.split('?')[0]); return route.abort('blockedbyclient'); }
    return route.continue();
  });
  await page.route('**/v1/**', async route => {
    const request = route.request(), relative = request.url().replace(/^https?:\/\/[^/]+/, ''), [path, query = ''] = relative.split('?');
    const send = (result, code = 200, msg = '') => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ code, result, msg }) });
    if (request.method() !== 'GET' && !(path === '/v1/sys/ai/kbs/7/evaluations' && request.method() === 'POST')) { unexpected.push(`${request.method()} ${path}`); return route.abort('blockedbyclient'); }
    reads.push(relative);
    if (path === '/v1/sys/auth/captcha') return send({ enabled: false });
    if (path === '/v1/sys/dept/my') return send({ list: [] });
    if (path === '/v1/sys/menu/my/perm-codes') return send({ codes: denyRead ? [] : ['sys:ai:kb:list', 'sys:ai:kb:evaluate'] });
    if (path === '/v1/sys/menu/my') return send({ list: [{ id: 771, parent_id: 0, title: '知识库', name: 'ai_kb', path: '/ai/knowledge', component: 'ai_kb', type: 2, visible: true, status: 1, sort: 0 }] });
    if (path === '/v1/sys/profile/') return send({ user: { id: denyRead ? 2 : 1, username: 'dataset-fixture', nickname: '题集回归', email: '', phone: '', avatar_url: '', dept_id: 0, role_ids: [] } });
    if (path === '/v1/sys/ai/kbs/7') return send({ kb, capabilities: { retrieval_configured: true, analysis_configured: true, ingestion_mode: 'async' } });
    if (path === '/v1/sys/ai/kbs/7/docs') return send({ list: [doc], total: 1, ready_count: 1, pending_count: 0, failed_count: 0 });
    if (path === '/v1/sys/ai/kbs/7/docs/9') return send({ doc: stale ? { ...doc, version: 3 } : doc, content });
    if (path === '/v1/sys/ai/kbs/7/imports') return send({ list: [], total: 0 });
    if (path === '/v1/sys/ai/kbs/7/feedback') return send({ list: [{ ...feedback(), snapshot: '', review: null }], total: 1 });
    if (path === '/v1/sys/ai/kbs/7/feedback/3') return send({ feedback: feedback() });
    if (path === '/v1/sys/ai/kbs/7/evaluation-datasets') return send({ list: [] });
    if (path === '/v1/sys/ai/kbs/7/evaluations') {
      if (request.method() === 'POST') {
        const data = request.postDataJSON(); writes.push(data);
        check(typeof data.dataset_snapshot === 'string' && data.dataset_key === `custom-${data.dataset_sha256.slice(0, 32)}`, '必须提交完整原始快照与指纹');
        if (writes.length > 1) check(JSON.stringify(data) === JSON.stringify(writes[0]), '未知结果只能显式继续原UUID与原raw');
        run = makeRun(data); return send({}, 503, '桩提交结果待确认');
      }
      if (query.includes('request_id=')) { lookupReads++; return send({ list: lookupReads === 1 ? [] : [lite(run)], total: lookupReads === 1 ? 0 : 1 }); }
      return send({ list: run ? [lite(run)] : [], total: run ? 1 : 0 });
    }
    if (path === '/v1/sys/ai/kbs/7/evaluations/4') return send({ evaluation: run });
    unexpected.push(`${request.method()} ${path}`); return route.abort('blockedbyclient');
  });
  await page.goto(`${origin}/login`);
  digest = await page.evaluate(async text => [...new Uint8Array(await crypto.subtle.digest('SHA-256', new TextEncoder().encode(text)))].map(value => value.toString(16).padStart(2, '0')).join(''), content);
  await page.goto(`${origin}/ai/knowledge/7?tab=feedback`);
  await page.getByRole('button', { name: '查看评价', exact: true }).click();
  let drawer = page.getByRole('dialog', { name: '反馈复核 #3', exact: true });
  await drawer.getByRole('button', { name: '用此样本新建评测', exact: true }).click();
  await drawer.getByText('来源ID、版本、标题或正文指纹已变化，请重新复核反馈', { exact: true }).waitFor();
  check(writes.length === 0, '来源变化拒绝后不能提交');
  stale = false; await drawer.getByRole('button', { name: /重试|重新加载/ }).click();
  await drawer.getByRole('button', { name: '确认题集，前往新建评测', exact: true }).waitFor();
  await drawer.getByText('核对后可前往新建评测，只有点击“开始评测”才执行模型。', { exact: true }).waitFor();
  check(await drawer.getByRole('button', { name: '下载冻结题集', exact: true }).count() === 0, '冻结题集下载须收进更多导出');
  check(writes.length === 0, '生成预览不得提交评测');
  await page.evaluate(() => {
    const create = URL.createObjectURL;
    URL.createObjectURL = blob => { window.feedbackDatasetExportRaw = blob.text(); return create.call(URL, blob); };
  });
  const downloaded = page.waitForEvent('download');
  await drawer.getByRole('region', { name: '评测题集预览', exact: true }).getByRole('button', { name: '更多导出', exact: true }).click();
  await page.getByRole('menuitem', { name: '下载冻结题集', exact: true }).click();
  check((await downloaded).suggestedFilename() === 'kb-7-feedback-3-r4.json', '下载须标识反馈版本');
  const frozenExportRaw = await page.evaluate(async () => await window.feedbackDatasetExportRaw);
  check(writes.length === 0, '菜单下载不能提交评测');
  await drawer.getByRole('button', { name: '确认题集，前往新建评测', exact: true }).click();
  await page.getByRole('heading', { name: '冻结题集回归库 · 质量验收', exact: true }).waitFor();
  await ensureAdvancedOpen();
  await page.getByRole('button', { name: '移除导入题集', exact: true }).waitFor();
  await page.getByLabel('冻结题集预览', { exact: true }).waitFor();
  check(writes.length === 0, '跨页导入仍不能自动运行');
  await page.setViewportSize({ width: 390, height: 844 });
  await page.waitForFunction(() => document.documentElement.scrollWidth <= innerWidth + 2 && [...document.querySelectorAll('button')].some(button => button.textContent.includes('开始评测') && !button.disabled));
  const mobile = await page.evaluate(() => ({ width: innerWidth, scrollWidth: document.documentElement.scrollWidth }));
  await page.screenshot({ path: '.playwright-cli/feedback-dataset-mobile.png', type: 'png', scale: 'css' });
  await page.getByRole('button', { name: '开始评测', exact: true }).dblclick();
  await page.getByText('上次提交结果待确认', { exact: true }).waitFor();
  check(writes.length === 1, '双击不得重复提交');
  const raw = writes[0].dataset_snapshot;
  check(raw === frozenExportRaw, '新建评测必须沿用菜单下载的冻结题集原字节');
  await page.reload(); await page.getByText('上次提交结果待确认', { exact: true }).waitFor();
  check(writes.length === 1, '重载不得自动重提');
  await page.getByRole('button', { name: '核对原请求', exact: true }).click();
  await page.getByText('尚未查到匹配批次，请稍后再核对；也可使用原标识继续原提交。', { exact: true }).waitFor();
  await page.getByRole('button', { name: '使用原标识继续', exact: true }).click();
  check(writes.length === 2 && writes[1].dataset_snapshot === raw, '显式继续必须保留原字节');
  await page.getByRole('button', { name: '核对原请求', exact: true }).click();
  await page.getByText('批次 #4', { exact: true }).waitFor();
  check(reads.some(url => url === '/v1/sys/ai/kbs/7/evaluations/4'), '核对必须补读详情验证原快照');
  await page.getByRole('button', { name: '使用此批次冻结题集', exact: true }).click();
  check(writes.length === 2, '复用快照仅设置下一次题集');
  await ensureAdvancedOpen();
  await page.getByRole('button', { name: '移除导入题集', exact: true }).click();
  await ensureAdvancedOpen();
  await page.getByLabel('导入冻结题集JSON', { exact: true }).evaluate((input, value) => {
    const transfer = new DataTransfer(); transfer.items.add(new File([value], 'merged.json', { type: 'application/json' })); input.files = transfer.files; input.dispatchEvent(new Event('change', { bubbles: true }));
  }, raw);
  await page.getByLabel('冻结题集预览', { exact: true }).waitFor();
  check(writes.length === 2, '上传合并题集不能提交');
  negative = true;
  await page.goto(`${origin}/ai/knowledge/7?tab=feedback`);
  await page.getByRole('button', { name: '查看评价', exact: true }).click();
  drawer = page.getByRole('dialog', { name: '反馈复核 #3', exact: true });
  const beforeDocs = reads.filter(url => url.includes('/docs/9')).length;
  await drawer.getByRole('button', { name: '用此样本新建评测', exact: true }).click();
  await drawer.getByText('原反馈不能证明整个知识库无答案', { exact: true }).waitFor();
  check(reads.filter(url => url.includes('/docs/9')).length === beforeDocs, '未确认负例不得拉取全文并生成');
  await drawer.getByRole('checkbox', { name: '我已核对当前库全部就绪资料，确认本题资料不足', exact: true }).check();
  await drawer.getByRole('button', { name: '确认题集，前往新建评测', exact: true }).waitFor();
  check(await drawer.getByRole('region', { name: '评测题集预览', exact: true }).getByRole('button', { name: '更多导出', exact: true }).isEnabled(), '明确核对全部当前资料后可导出冻结题集');
  denyRead = true;
  await page.evaluate(() => {
    localStorage.setItem('go-zero-auth', JSON.stringify({ state: { token: 'dataset-other-access', refreshToken: 'dataset-other-refresh', sessionId: 'dataset-other-session', expiresAt: Date.now() / 1000 + 3600, refreshExpiresAt: Date.now() / 1000 + 7200 }, version: 0 }));
    window.dispatchEvent(new StorageEvent('storage', { key: 'go-zero-auth', storageArea: localStorage }));
  });
  await drawer.waitFor({ state: 'hidden' });
  check(await page.getByLabel('冻结题集预览', { exact: true }).count() === 0, '换身份不能保留原文预览');
  check(errors.length === 0 && unexpected.length === 0, JSON.stringify({ errors, unexpected }));
  return { changedSourceRejected: true, generationAndImportWithoutRun: true, unknownPreservesRawAndUuid: true, batchSnapshotReusable: true, uploadWithoutRun: true, negativeRequiresConfirmation: true, identityIsolated: true, mobile, mockEvaluationSubmissions: writes.length, realBusinessWrites: 0, realModelCalls: 0, pageErrors: errors.length };
}
