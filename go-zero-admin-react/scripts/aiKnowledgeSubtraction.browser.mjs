// 已正常登录5999本地开发页后执行：playwright-cli run-code --filename scripts/aiKnowledgeSubtraction.browser.mjs
// 全部 AI 请求在当前浏览器内拦截；写请求统一拒绝，不调用模型或写入业务库。
// oxlint-disable-next-line no-unused-expressions -- 由 Playwright CLI 调用函数表达式。
async page => {
  const assert = (condition, message) => { if (!condition) throw new Error(message); };
  const originalUrl = page.url();
  const origin = new URL(originalUrl).origin;
  assert(new URL(originalUrl).port === '5999', '请在已登录的5999本地开发页运行');
  const pattern = '**/v1/sys/ai/**';
  const originalPermissions = await page.evaluate(async () => (await import('/src/stores/auth.ts')).useAuthStore.getState().permCodes);
  const fixturePermissions = [...new Set([...originalPermissions, 'sys:ai:kb:list', 'sys:ai:kb:edit', 'sys:ai:kb:delete', 'sys:ai:kb:evaluate', 'sys:ai:workflow:run'])];
  const setPermissions = codes => page.evaluate(async values => (await import('/src/stores/auth.ts')).useAuthStore.getState().setPermCodes(values), codes);
  const kbId = 91701101;
  const kb = { id: kbId, name: '减法回归资料库', description: '浏览器请求桩', owner_id: 1, doc_count: 2, chunk_count: 1,
    created_at: 1, updated_at: 1, ready_count: 1, pending_count: 0, failed_count: 1 };
  const capabilities = { analysis_configured: true, retrieval_configured: true, ingestion_mode: 'async' };
  const doc = { id: 91701102, kb_id: kbId, title: '减法回归发布指南', status: 'ready', stage: 'ready', version: 1, error: '',
    chunk_count: 1, embedding_model: 'browser-fixture', created_at: 1, updated_at: 1 };
  const failedDoc = { ...doc, id: 91701103, title: '减法回归失败资料', status: 'failed', stage: 'failed', error: '请求桩中的处理失败' };
  const dataset = { version: 1, documents: [{ key: 'rules', kb: 'project_ops', source: 'ai_document/browser-fixture.md', title: doc.title, sha256: 'a'.repeat(64) }],
    cases: ['发布前应当做什么？', '怎样确认发布完成？'].map((query, index) => ({ id: `browser-case-${index + 1}`, kb: 'project_ops', split: index === 0 ? 'development' : 'validation',
      category: 'risk', query, rationale: '核对浏览器交互', expected_documents: ['rules'], evidence: [{ document: 'rules', groups: [['先备份']] }],
      answer: { kind: 'answerable', required_groups: [['先备份']], forbidden_phrases: [] } })) };
  const raw = JSON.stringify(dataset);
  const sha256 = await page.evaluate(async text => Array.from(new Uint8Array(await crypto.subtle.digest('SHA-256', new TextEncoder().encode(text))), value => value.toString(16).padStart(2, '0')).join(''), raw);
  const catalogue = { key: 'browser-subtraction', name: '减法回归题集', sha256, dataset: raw };
  const run = { id: 91701104, kb_id: kbId, request_id: '11111111-1111-4111-8111-111111111111', dataset_key: catalogue.key, dataset_sha256: sha256,
    group_key: 'project_ops', status: 'cancelled', case_count: 2, completed_count: 0, failed_count: 0, stop_requested: true, corpus_verified: false,
    error: '', snapshot: raw, bindings: '{}', context: '{}', observations: '[]', created_at: 1, updated_at: 1, started_at: 0, completed_at: 1 };
  const storage = await page.evaluate(async id => {
    const auth = (await import('/src/stores/auth.ts')).useAuthStore.getState();
    const { draftRunAttemptKey } = await import('/src/pages/ai/draftRunAttempts.ts');
    const draftKey = draftRunAttemptKey([auth.sessionId, auth.selectedDeptId], `knowledge:${id}`);
    const qualityKey = `ai-kb-quality:${JSON.stringify([auth.sessionId, auth.selectedDeptId, id])}`;
    const keys = [draftKey, `${draftKey}:task`, qualityKey];
    return { draftKey, qualityKey, previous: keys.map(key => [key, sessionStorage.getItem(key)]) };
  }, kbId);
  const gets = [], writes = [], unexpected = [];
  const routeHandler = async route => {
    const request = route.request();
    const url = new URL(request.url());
    if (request.method() !== 'GET') {
      writes.push(`${request.method()} ${url.pathname}`);
      await route.fulfill({ json: { code: 409, msg: '浏览器回归禁止业务写入', result: null } });
      return;
    }
    gets.push(url.pathname);
    let result;
    if (url.pathname === '/v1/sys/ai/kbs') result = { list: [kb], total: 1, capabilities };
    else if (url.pathname === `/v1/sys/ai/kbs/${kbId}`) result = { kb, capabilities };
    else if (url.pathname === `/v1/sys/ai/kbs/${kbId}/docs`) result = { list: [doc, failedDoc], total: 2, ready_count: 1, pending_count: 0, failed_count: 1 };
    else if (url.pathname === `/v1/sys/ai/kbs/${kbId}/docs/${doc.id}`) result = { doc, content: '先备份，再发布。' };
    else if (url.pathname === `/v1/sys/ai/kbs/${kbId}/imports`) result = { list: [], total: 0 };
    else if (url.pathname === `/v1/sys/ai/kbs/${kbId}/evaluation-datasets`) result = { list: [catalogue] };
    else if (url.pathname === `/v1/sys/ai/kbs/${kbId}/evaluations`) result = { list: url.searchParams.has('request_id') ? [] : [run], total: url.searchParams.has('request_id') ? 0 : 1 };
    else if (url.pathname === `/v1/sys/ai/kbs/${kbId}/feedback` || url.pathname === '/v1/sys/ai/tasks') result = { list: [], total: 0 };
    else if (url.pathname === '/v1/sys/ai/tools') result = { list: [], capabilities };
    else {
      unexpected.push(url.pathname);
      await route.fulfill({ json: { code: 503, msg: '未声明的浏览器回归请求', result: null } });
      return;
    }
    await route.fulfill({ json: { code: 200, msg: '操作成功', result } });
  };
  const goto = async path => { await page.goto(`${origin}${path}`); await page.waitForLoadState('networkidle'); await setPermissions(fixturePermissions); };
  const cancelConfirmation = async title => {
    await page.locator('.ant-modal-confirm-title').filter({ hasText: title }).waitFor();
    await page.locator('.ant-modal-confirm').getByRole('button', { name: /^取\s*消$/ }).click();
    await page.locator('.ant-modal-confirm').waitFor({ state: 'hidden' });
  };
  try {
    await page.route(pattern, routeHandler);
    await page.evaluate(previous => { for (const [key] of previous) sessionStorage.removeItem(key); }, storage.previous);
    await goto('/ai/knowledge');
    const kbName = page.getByRole('button', { name: kb.name, exact: true });
    await kbName.waitFor();
    assert(await kbName.count() === 1, '知识库应只有一个名称入口');
    assert(await page.getByRole('button', { name: '进入工作区', exact: true }).count() === 0, '重复进入入口应移除');
    await page.getByRole('button', { name: `更多：${kb.name}`, exact: true }).click();
    await page.getByRole('menuitem', { name: /编辑/ }).click();
    const editKb = page.getByRole('dialog', { name: '编辑知识库', exact: true });
    await editKb.getByRole('textbox', { name: /名称$/ }).waitFor();
    assert(await editKb.getByRole('textbox', { name: /名称$/ }).inputValue() === kb.name, '更多应打开当前知识库编辑');
    await editKb.getByRole('button', { name: /^取\s*消$/ }).click();
    await editKb.waitFor({ state: 'hidden' });
    await page.getByRole('button', { name: `更多：${kb.name}`, exact: true }).click();
    await page.getByRole('menuitem', { name: /删除/ }).click();
    await cancelConfirmation('删除知识库及其全部文档？');
    await kbName.click();
    await page.getByRole('button', { name: doc.title, exact: true }).waitFor();
    assert(await page.getByRole('button', { name: /^原\s*文$/ }).count() === 0, '重复原文按钮应移除');
    assert(await page.getByRole('button', { name: `重试${failedDoc.title}`, exact: true }).isEnabled(), '失败资料应提供明确可用的重试入口');
    await page.getByRole('button', { name: doc.title, exact: true }).click();
    const document = page.getByRole('dialog', { name: '文档原文', exact: true });
    await document.getByText('先备份，再发布。', { exact: true }).waitFor();
    await document.locator('.ant-modal-close').click();
    await document.waitFor({ state: 'hidden' });
    await page.getByRole('button', { name: `更多：${doc.title}`, exact: true }).click();
    await page.getByRole('menuitem', { name: /查看并编辑原文/ }).click();
    await document.getByRole('button', { name: '编辑原文', exact: true }).click();
    const editDocument = page.getByRole('dialog', { name: '编辑原文并重新处理', exact: true });
    assert(await editDocument.getByRole('textbox', { name: /正文（纯文本）/ }).inputValue() === '先备份，再发布。', '资料更多应可进入原文编辑');
    await editDocument.getByRole('button', { name: '取消编辑', exact: true }).click();
    await document.locator('.ant-modal-close').click();
    await document.waitFor({ state: 'hidden' });
    await page.getByRole('button', { name: `更多：${doc.title}`, exact: true }).click();
    await page.getByRole('menuitem', { name: /删除/ }).click();
    await cancelConfirmation('删除这篇资料？');
    await page.getByRole('button', { name: /添加资料/ }).click();
    await page.getByRole('menuitem', { name: /导入 TXT \/ Markdown/ }).click();
    const imports = page.getByRole('dialog', { name: '资料导入与后台批次', exact: true });
    await imports.getByLabel('选择知识库资料文件', { exact: true }).waitFor();
    await imports.getByRole('button', { name: '关闭并返回工作区', exact: true }).click();
    await imports.waitFor({ state: 'hidden' });
    await page.getByRole('button', { name: /添加资料/ }).click();
    await page.getByRole('menuitem', { name: /粘贴资料/ }).click();
    await imports.getByRole('textbox', { name: /正文（纯文本）/ }).waitFor();
    await imports.getByRole('button', { name: '关闭并返回工作区', exact: true }).click();
    await imports.waitFor({ state: 'hidden' });
    assert(await page.getByRole('region', { name: '知识库反馈与复核', exact: true }).count() === 0, '资料页默认不挂载反馈面板');
    await page.getByRole('tab', { name: '反馈与复核', exact: true }).click();
    await page.getByRole('region', { name: '知识库反馈与复核', exact: true }).getByText('暂无待复核评价，可在分析结果下提交评价。', { exact: true }).waitFor();
    await page.getByRole('tab', { name: '资料', exact: true }).click();
    await page.getByRole('region', { name: '知识库反馈与复核', exact: true }).waitFor({ state: 'hidden' });
    const analysis = page.getByRole('region', { name: '用资料分析', exact: true });
    assert(await analysis.getByRole('textbox').count() === 0, '首次展开前不应挂载分析表单');
    await analysis.getByRole('button', { name: /用资料分析/ }).click();
    const question = analysis.getByRole('textbox', { name: /你希望了解什么？/ });
    await question.fill('回归草稿：发布之前需要核对什么？');
    await analysis.getByRole('button', { name: /收起分析/ }).click();
    assert(!await question.isVisible(), '收起分析后输入应不可见');
    await analysis.getByRole('button', { name: /用资料分析/ }).click();
    assert(await question.inputValue() === '回归草稿：发布之前需要核对什么？', '收起分析不能丢失问题草稿');
    await page.evaluate(async key => {
      const { writeDraftRunAttempt } = await import('/src/pages/ai/draftRunAttempts.ts');
      writeDraftRunAttempt(sessionStorage, key, { name: '浏览器未知运行回归', definition: '{"steps":[]}', input: '{"question":"原请求问题"}', request_id: '22222222-2222-4222-8222-222222222222' });
    }, storage.draftKey);
    await goto(`/ai/knowledge/${kbId}`);
    await page.getByText('上次运行结果待确认，展开后核对原请求', { exact: true }).waitFor();
    await page.getByRole('region', { name: '用资料分析', exact: true }).getByRole('button', { name: /用资料分析/ }).click();
    await page.getByRole('button', { name: '核对原运行', exact: true }).waitFor();
    await page.getByText('请求标识：22222222-2222-4222-8222-222222222222', { exact: true }).waitFor();
    await page.getByRole('button', { name: '质量验收', exact: true }).click();
    const historyRow = page.getByRole('row').filter({ hasText: `#${run.id}` });
    await historyRow.getByRole('cell', { name: `#${run.id}`, exact: true }).waitFor();
    // 题集单元格还包含题组子元素，整格精确文本并不等于题集名。
    await historyRow.getByRole('cell').filter({ hasText: catalogue.name }).waitFor();
    assert(await page.getByRole('combobox', { name: '选择评测题集', exact: true }).count() === 0, '质量首屏不应挂载新建表单');
    assert(await page.getByText('历史批次', { exact: true }).isVisible(), '质量页默认显示历史批次');
    const catalogueGets = gets.filter(path => path.endsWith('/evaluation-datasets')).length;
    await page.getByRole('button', { name: '新建评测', exact: true }).click();
    await page.getByRole('combobox', { name: '选择评测题集', exact: true }).waitFor();
    assert(!await page.getByLabel('导入冻结题集JSON', { exact: true }).isVisible(), '冻结题集导入应收进高级设置');
    await page.getByText('高级设置', { exact: true }).click();
    await page.getByLabel('导入冻结题集JSON', { exact: true }).waitFor();
    await page.getByText('高级设置', { exact: true }).click();
    await page.getByRole('combobox', { name: '选择资料题组', exact: true }).click();
    await page.locator('.ant-select-dropdown:visible').getByText('项目运维资料', { exact: true }).click();
    const validation = page.getByRole('row').filter({ hasText: '怎样确认发布完成？' }).getByRole('checkbox');
    await validation.check();
    await page.getByText('已选 2 题', { exact: true }).waitFor();
    await page.getByRole('button', { name: '收起新建评测', exact: true }).click();
    assert(await page.getByRole('combobox', { name: '选择评测题集', exact: true }).count() === 0, '收起后应卸载题集表单');
    await page.getByRole('button', { name: '新建评测', exact: true }).click();
    await page.getByText('已选 2 题', { exact: true }).waitFor();
    assert(await validation.isChecked(), '重新展开应保留所选预留题');
    assert(gets.filter(path => path.endsWith('/evaluation-datasets')).length === catalogueGets, '展开收起不应重复加载题集');
    await page.evaluate(({ key, sha, datasetKey }) => sessionStorage.setItem(key, JSON.stringify({ request_id: '33333333-3333-4333-8333-333333333333', dataset_key: datasetKey, dataset_sha256: sha, group_key: 'project_ops', case_ids: ['browser-case-1'] })), { key: storage.qualityKey, sha: sha256, datasetKey: catalogue.key });
    await goto(`/ai/knowledge/${kbId}/quality`);
    await page.getByText('上次提交结果待确认', { exact: true }).waitFor();
    await page.getByRole('button', { name: '核对原请求', exact: true }).waitFor();
    await page.getByRole('button', { name: '使用原标识继续', exact: true }).waitFor();
    assert(await page.getByRole('combobox', { name: '选择评测题集', exact: true }).count() === 0, '原请求恢复入口不能依赖新建表单');
    await goto(`/ai/knowledge/${kbId}`);
    await setPermissions(fixturePermissions.filter(code => !['sys:ai:kb:edit', 'sys:ai:kb:delete', 'sys:ai:kb:evaluate', 'sys:ai:workflow:run'].includes(code)));
    await page.getByRole('button', { name: /添加资料/ }).waitFor({ state: 'hidden' });
    await page.getByRole('button', { name: doc.title, exact: true }).waitFor();
    assert(await page.getByRole('button', { name: /更多：|添加资料|重试减法回归失败资料/ }).count() === 0, '只读身份不能编辑删除添加或重试');
    assert(await page.getByRole('button', { name: '查看导入批次', exact: true }).isVisible(), '只读身份仍可核对批次');
    await page.getByRole('button', { name: doc.title, exact: true }).click();
    await document.getByText('先备份，再发布。', { exact: true }).waitFor();
    assert(await document.getByRole('button', { name: '编辑原文', exact: true }).count() === 0, '只读原文不能出现编辑入口');
    await document.locator('.ant-modal-close').click();
    assert(writes.length === 0, `交互取消与恢复检查不应写入：${writes.join(', ')}`);
    assert(unexpected.length === 0, `出现未声明接口：${unexpected.join(', ')}`);
    return 'PASS：知识库/资料唯一名称入口、更多编辑和删除取消、添加资料两种方式、分析首次挂载与草稿保留、未知原请求入口、历史优先与选题收起保留、无重复题集请求、只读权限；业务写请求0次';
  } catch (error) {
    const queryErrors = await page.locator('main .query-error').allTextContents();
    throw new Error(`${error.stack ?? error}\n页面查询错误：${queryErrors.join(' | ') || '无'}\n最近请求：${gets.slice(-12).join(', ')}\n未声明请求：${unexpected.join(', ') || '无'}\n拦截写请求数：${writes.length}`);
  } finally {
    await page.evaluate(previous => { for (const [key, value] of previous) { if (value === null) sessionStorage.removeItem(key); else sessionStorage.setItem(key, value); } }, storage.previous);
    await setPermissions(originalPermissions);
    await page.unroute(pattern, routeHandler);
    await page.goto(originalUrl);
  }
}
