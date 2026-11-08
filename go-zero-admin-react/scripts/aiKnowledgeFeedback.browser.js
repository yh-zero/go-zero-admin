// 使用独立 playwright-cli 会话运行；所有业务请求由响应桩处理，不调用数据库或模型。
// oxlint-disable-next-line no-unused-expressions -- playwright-cli 将函数表达式作为可重复入口执行。
async (page) => {
  const now = Math.floor(Date.now() / 1000), writes = [], reviewWrites = [], errors = [], reads = [];
  const require = (value, message) => { if (!value) throw new Error(message); };
  const sources = [{ index: 1, kb_id: 7, doc_id: 9, chunk_id: 11, doc_title: '发布操作', snippet: '发布前必须备份数据库。验收通过后才可切换流量。', score: 0.9 }];
  const result = { version: 1, type: 'kb_analysis', status: 'completed', analysis: '先发布 [1]', sources,
    usage: { scope: 'chat', prompt_tokens: 20, completion_tokens: 5, total_tokens: 25, complete: true } };
  const input = JSON.stringify({ kb_id: 7, query: '如何发布？' });
  const output = JSON.stringify(result);
  const task = { id: 8, workflow_id: 0, conversation_id: 0, status: 'succeeded', input: '{}', result: `[1/1] 发布分析: ${output}`, error: '',
    approved_by: 0, created_at: now, completed_at: now, next_step: 1, updated_at: now, request_id: '11111111-1111-4111-8111-111111111111', title: '反馈体验回归',
    workflow_name: '', input_summary: '如何发布？', definition: JSON.stringify({ steps: [{ name: '发布分析', tool: 'kb_analyze', input: JSON.parse(input) }] }),
    steps: [{ index: 0, id: 'step_one', name: '发布分析', tool: 'kb_analyze', input, output, status: 'succeeded', error: '', started_at: now * 1000, completed_at: now * 1000 + 10 }] };
  const kb = { id: 7, name: '业务回归资料', description: '', owner_id: 1, doc_count: 1, chunk_count: 1, created_at: now, updated_at: now, ready_count: 1, pending_count: 0, failed_count: 0 };
  let record = null, reviewConflict = true, detailUnavailable = false;
  page.on('pageerror', error => errors.push(error.message));
  await page.addInitScript(() => localStorage.setItem('go-zero-auth', JSON.stringify({ state: {
    token: 'kb-feedback-browser-fixture-access', refreshToken: 'kb-feedback-browser-fixture-refresh', expiresAt: Date.now() / 1000 + 3600,
    refreshExpiresAt: Date.now() / 1000 + 7200, sessionId: 'kb-feedback-browser-fixture-session',
  }, version: 0 })));
  await page.route('**/v1/**', async route => {
    const request = route.request(), relative = request.url().replace(/^https?:\/\/[^/]+/, ''), [path, query = ''] = relative.split('?');
    const queryValues = new Map(query.split('&').map(pair => pair.split('=')));
    const send = (code, result = {}, msg = '') => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ code, result, msg }) });
    if (path === '/v1/sys/dept/my') return send(200, { list: [] });
    if (path === '/v1/sys/menu/my/perm-codes') return send(200, { codes: ['sys:ai:task:list', 'sys:ai:task:feedback', 'sys:ai:kb:list', 'sys:ai:kb:review'] });
    if (path === '/v1/sys/menu/my') return send(200, { list: [
      { id: 770, parent_id: 0, title: '运行记录', name: 'ai_task', type: 2, path: '/ai/tasks', component: 'ai_task', perm_code: '', icon: '', sort: 0, visible: true, status: 1, created_at: 0 },
      { id: 771, parent_id: 0, title: '知识库', name: 'ai_kb', type: 2, path: '/ai/knowledge', component: 'ai_kb', perm_code: '', icon: '', sort: 1, visible: true, status: 1, created_at: 0 },
    ] });
    if (path === '/v1/sys/profile/') return send(200, { user: { id: 1, username: 'kb-feedback-fixture', nickname: '反馈回归', email: '', phone: '', avatar_url: '', dept_id: 0, role_ids: [] } });
    if (path === '/v1/sys/ai/tasks') return send(200, { list: [task], total: 1 });
    if (path === '/v1/sys/ai/tasks/8') return send(200, { task });
    if (path === '/v1/sys/ai/tasks/8/feedback') {
      if (request.method() === 'GET') return send(200, { list: record ? [record] : [] });
      const data = request.postDataJSON(); writes.push(data);
      require(data.step_index === 0, '评价必须使用0起的实际步骤');
      record = { id: 3, kb_id: 7, task_id: 8, step_index: 0, feedback: data.feedback, note: data.note, status: 'pending', revision: (record?.revision ?? 0) + 1,
        question: '如何发布？', answer: result.analysis, snapshot: JSON.stringify({ version: 1, kb_id: 7, query: '如何发布？', step_input: input, step_output: output, sources,
          documents: [{ doc_id: 9, kb_id: 7, title: '发布操作', version: 2, content_sha256: 'a'.repeat(64), status: 'verified' }] }),
        review: null, created_at: now, updated_at: now, reviewed_at: 0 };
      return send(200, { feedback: record });
    }
    if (path === '/v1/sys/ai/kbs/7') return send(200, { kb, capabilities: { retrieval_configured: true, analysis_configured: true, ingestion_mode: 'async' } });
    if (path === '/v1/sys/ai/kbs/7/feedback') {
      reads.push(query);
      const status = queryValues.get('status') ?? 'pending';
      const list = record && (status === 'all' || status === record.status) ? [{ ...record, answer: '', snapshot: '', review: null }] : [];
      return send(200, { list, total: list.length });
    }
    if (path === '/v1/sys/ai/kbs/7/feedback/3') { reads.push('detail'); return detailUnavailable ? send(500, {}, '复核详情暂不可用') : send(200, { feedback: record }); }
    if (path === '/v1/sys/ai/kbs/7/feedback/3/review') {
      const data = request.postDataJSON(); reviewWrites.push(data);
      if (reviewConflict) { reviewConflict = false; detailUnavailable = true; record = { ...record, revision: 2, note: '另一个页面补充了说明' }; return send(409, {}, '记录已更新，请刷新后核对'); }
      require(data.expected_revision === record.revision, '复核必须显式使用已核对的最新revision');
      record = { ...record, revision: record.revision + 1, status: data.status, review: data.review, reviewed_at: now };
      return send(200, { feedback: record });
    }
    throw new Error(`响应桩未定义业务请求：${request.method()} ${path}`);
  });

  await page.goto('http://127.0.0.1:5999/ai/tasks?task_id=8');
  await page.getByRole('button', { name: '评价分析', exact: true }).click();
  let dialog = page.getByRole('dialog', { name: '评价第 1 步分析', exact: true });
  await dialog.getByRole('combobox', { name: /评价$/ }).click();
  await page.locator('.ant-select-dropdown').getByText('回答错误', { exact: true }).click();
  await dialog.getByRole('textbox', { name: '补充说明（可选）' }).fill('中'.repeat(334));
  await dialog.getByRole('button', { name: '保存评价', exact: true }).click();
  await dialog.getByText('说明最多 1000 UTF-8 字节', { exact: true }).waitFor();
  require(writes.length === 0, '超字节说明不能发起保存');
  await dialog.getByRole('textbox', { name: '补充说明（可选）' }).fill('遗漏了发布前备份');
  await dialog.getByRole('button', { name: '保存评价', exact: true }).dblclick();
  await dialog.waitFor({ state: 'hidden' });
  require(writes.length === 1 && writes[0].expected_revision === 0 && writes[0].feedback === 'incorrect_answer', '双击评价只能创建一次');

  await page.goto('http://127.0.0.1:5999/ai/knowledge/7?tab=feedback');
  await page.getByRole('tab', { name: '反馈与复核', exact: true }).waitFor();
  await page.getByRole('button', { name: '查看并复核', exact: true }).click();
  dialog = page.getByRole('dialog', { name: '反馈复核 #3', exact: true });
  await dialog.getByText('发布前必须备份数据库。验收通过后才可切换流量。', { exact: true }).waitFor();
  await dialog.getByRole('textbox', { name: /预期答案$/ }).fill('发布前备份数据库，验收通过后切换流量。');
  await dialog.getByRole('textbox', { name: /必须包含的事实（每行一项）$/ }).fill('备份数据库\n切换流量');
  await dialog.getByRole('button', { name: /添加依据摘录$/ }).click();
  await dialog.getByRole('textbox', { name: /逐字摘录$/ }).fill('发布前必须备份数据库。');
  await dialog.getByRole('textbox', { name: /复核说明$/ }).fill('按历史发布操作片段修正');
  await dialog.getByRole('button', { name: /保存并采纳$/ }).click();
  await dialog.getByText('每个事实关键词都须同时出现在预期答案和至少一条依据摘录中', { exact: true }).waitFor();
  require(reviewWrites.length === 0, '摘录遗漏第二事实时不能发送采纳请求');
  require(await dialog.getByRole('textbox', { name: /预期答案$/ }).inputValue() === '发布前备份数据库，验收通过后切换流量。', '事实校验失败须保留草稿');
  await dialog.getByRole('textbox', { name: /逐字摘录$/ }).fill(sources[0].snippet);
  await dialog.getByRole('button', { name: /保存并采纳$/ }).click();
  await dialog.getByText('复核详情暂不可用', { exact: true }).waitFor();
  require(await dialog.getByRole('textbox', { name: /预期答案$/ }).count() === 1, '复核GET失败不能卸载已填写的草稿');
  require(await dialog.getByRole('textbox', { name: /预期答案$/ }).inputValue() === '发布前备份数据库，验收通过后切换流量。', '409不能丢失复核草稿');
  require(await dialog.getByRole('button', { name: /保存并采纳$/ }).isDisabled(), '冲突后不能自动覆盖新版本');
  require(await dialog.getByRole('button', { name: '已核对最新版本，保留草稿继续', exact: true }).isDisabled(), '详情500后不能确认旧缓存版本');
  detailUnavailable = false;
  await dialog.getByRole('button', { name: '刷新反馈详情', exact: true }).click();
  await dialog.getByText('最新版本 2 · 回答错误 · 待复核', { exact: true }).waitFor();
  require(await dialog.getByRole('textbox', { name: /预期答案$/ }).inputValue() === '发布前备份数据库，验收通过后切换流量。', '恢复详情后仍须保留原草稿');
  await dialog.getByRole('button', { name: '已核对最新版本，保留草稿继续', exact: true }).click();
  await dialog.getByRole('button', { name: /保存并采纳$/ }).click();
  await dialog.getByRole('button', { name: '下载回归样本', exact: true }).waitFor();
  require(reviewWrites.length === 2 && reviewWrites[1].expected_revision === 2 && record.status === 'accepted', '核对版本后才能采纳');
  const detailReads = reads.filter(value => value === 'detail').length;
  const downloaded = page.waitForEvent('download');
  await dialog.getByRole('button', { name: '下载回归样本', exact: true }).click();
  const download = await downloaded;
  require(download.suggestedFilename() === 'kb-feedback-3-r3.json', '下载文件名须固定记录和版本');
  require(reads.filter(value => value === 'detail').length > detailReads, '下载前必须重新核对记录存续');
  await dialog.getByRole('textbox', { name: /预期答案$/ }).fill('刷新失败后须保留的复核草稿');
  detailUnavailable = true;
  await dialog.getByRole('button', { name: '刷新反馈详情', exact: true }).click();
  await dialog.getByText('复核详情暂不可用', { exact: true }).waitFor();
  require(await dialog.getByRole('textbox', { name: /预期答案$/ }).inputValue() === '刷新失败后须保留的复核草稿', '已有详情刷新失败不能卸载草稿');
  require(await dialog.getByRole('button', { name: '下载回归样本', exact: true }).isDisabled(), 'GET500期间不能从已采纳旧缓存下载');
  require(await dialog.getByRole('button', { name: /保存并采纳$/ }).isDisabled(), 'GET500期间不能提交旧缓存版本');
  detailUnavailable = false;
  await dialog.getByRole('button', { name: '刷新反馈详情', exact: true }).click();
  await dialog.getByText('最新版本 3 · 回答错误 · 已采纳', { exact: true }).waitFor();
  require(await dialog.getByRole('button', { name: /保存并采纳$/ }).isDisabled(), '恢复相同版本后仍须明确核对');
  await dialog.getByRole('button', { name: '已核对最新版本，保留草稿继续', exact: true }).click();
  require(await dialog.getByRole('textbox', { name: /预期答案$/ }).inputValue() === '刷新失败后须保留的复核草稿', '确认相同版本不能覆盖用户草稿');
  require(reviewWrites.length === 2, '刷新/核对不能自动追加复核写入');
  await dialog.getByRole('button', { name: '关闭', exact: true }).click();
  await page.getByRole('combobox', { name: '反馈状态', exact: true }).click();
  await page.locator('.ant-select-dropdown').getByText('已采纳', { exact: true }).click();
  await page.getByRole('button', { name: '查看并复核', exact: true }).waitFor();
  require(reads.some(value => value.includes('status=accepted') && value.includes('page=1')), '切换筛选必须回到第一页');
  require(errors.length === 0, `浏览器错误：${errors.join('; ')}`);
  return { feedbackWrites: writes.length, reviewWrites: reviewWrites.length, recordRevision: record.revision, downloads: 1, readFailureRecoveryCases: 2,
    rejectedMissingFact: true, pageErrors: errors.length, businessRequests: '全部为响应桩，真实写入0/模型0' };
}
