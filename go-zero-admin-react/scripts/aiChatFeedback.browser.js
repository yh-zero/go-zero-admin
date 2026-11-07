// 用独立 playwright-cli 会话运行；所有业务请求由本地响应桩处理，不使用真实账号或模型。
// oxlint-disable-next-line no-unused-expressions -- playwright-cli 通过函数表达式执行此可重复入口。
async (page) => {
  const now = Math.floor(Date.now() / 1000), writes = [], renameWrites = [], errors = [];
  let feedback = '', feedback_note = '', rejectWrite = false;
  const conversation = { id: 777, title: '反馈表单回归响应桩', user_id: 1, message_count: 2, created_at: now, updated_at: now, pinned: false };
  page.on('pageerror', error => errors.push(error.message));
  await page.addInitScript(() => localStorage.setItem('go-zero-auth', JSON.stringify({ state: {
    token: 'feedback-browser-fixture-access', refreshToken: 'feedback-browser-fixture-refresh', expiresAt: Date.now() / 1000 + 3600,
    refreshExpiresAt: Date.now() / 1000 + 7200, sessionId: 'feedback-browser-fixture-session',
  }, version: 0 })));
  await page.route('**/v1/**', async route => {
    const request = route.request(), path = request.url().replace(/^https?:\/\/[^/]+/, '').split('?')[0];
    const send = (code, result = {}, msg = '响应桩') => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ code, result, msg }) });
    if (path === '/v1/sys/dept/my') return send(200, { list: [] });
    if (path === '/v1/sys/menu/my/perm-codes') return send(200, { codes: ['sys:ai:conversation:list', 'sys:ai:message:send', 'sys:ai:conversation:create'] });
    if (path === '/v1/sys/menu/my') return send(200, { list: [{ id: 777, parent_id: 0, title: 'AI 对话', name: 'ai_chat', type: 2, path: '/ai/chat', component: 'ai_chat', perm_code: '', icon: '', sort: 0, visible: true, status: 1, created_at: 0 }] });
    if (path === '/v1/sys/profile/') return send(200, { user: { id: 1, username: 'feedback-fixture', nickname: '表单回归', email: '', phone: '', avatar_url: '', dept_id: 0, role_ids: [] } });
    if (path === '/v1/sys/ai/conversations') return send(200, { list: [conversation], total: 1 });
    if (path === '/v1/sys/ai/conversations/777' && request.method() === 'PUT') {
      const input = request.postDataJSON(); renameWrites.push(input); conversation.title = input.title;
      return send(200, { conversation });
    }
    if (path === '/v1/sys/ai/conversations/777') return send(200, { conversation, messages: [
      { id: 778, role: 'user', content: '桩问题', tool_calls: '[]', tokens: 0, created_at: now, status: 'succeeded', error: '', sources: [], feedback: '', feedback_note: '' },
      { id: 779, role: 'assistant', content: '桩回答', tool_calls: '[]', tokens: 5, created_at: now, status: 'succeeded', error: '', sources: [], feedback, feedback_note },
    ], has_more: false, next_before_id: 0 });
    if (path === '/v1/sys/ai/conversations/777/messages/779/feedback' && request.method() === 'PUT') {
      const input = request.postDataJSON(); writes.push(input);
      if (!Object.hasOwn(input, 'feedback')) return send(400, {}, '反馈字段缺失');
      if (rejectWrite) return send(503, {}, '反馈响应桩暂不可用');
      feedback = input.feedback; feedback_note = feedback ? input.note || '' : '';
      return send(200);
    }
    throw new Error(`响应桩未定义业务路径：${request.method()} ${path}`);
  });
  await page.goto('http://127.0.0.1:5999/ai/chat');
  await page.getByRole('button', { name: /反馈表单回归响应桩 2 条消息/ }).click();
  const open = async () => { await page.getByRole('button', { name: '评价回答', exact: true }).click(); return page.getByRole('dialog', { name: '评价这条回答' }); };
  const require = (value, message) => { if (!value) throw new Error(message); };
  let dialog = await open();
  await dialog.getByText('有帮助', { exact: true }).waitFor();
  await dialog.getByRole('textbox', { name: '补充说明（可选）' }).fill('保存后的评价说明');
  await dialog.getByRole('button', { name: /确 定/ }).click();
  await dialog.waitFor({ state: 'hidden' });
  require(writes.length === 1 && writes[0].feedback === 'helpful' && feedback_note === '保存后的评价说明', '默认评价必须提交helpful与说明');
  dialog = await open();
  require(await dialog.getByRole('textbox', { name: '补充说明（可选）' }).inputValue() === '保存后的评价说明', '重开必须展示已保存说明');
  await dialog.getByRole('textbox', { name: '补充说明（可选）' }).fill('取消的草稿');
  await dialog.getByRole('button', { name: '取 消', exact: true }).click();
  dialog = await open();
  require(await dialog.getByRole('textbox', { name: '补充说明（可选）' }).inputValue() === '保存后的评价说明', '取消的草稿不能污染下次打开');
  await dialog.getByRole('combobox', { name: '反馈', exact: true }).click();
  await page.getByText('来源不相关', { exact: true }).click();
  await dialog.getByRole('button', { name: /确 定/ }).click();
  await dialog.waitFor({ state: 'hidden' });
  require(feedback === 'irrelevant_source' && writes.length === 2, '修改评价须保存来源不相关');
  dialog = await open();
  await dialog.getByRole('textbox', { name: '补充说明（可选）' }).fill('中'.repeat(334));
  await dialog.getByRole('button', { name: /确 定/ }).click();
  await dialog.getByText('说明最多 1000 UTF-8 字节', { exact: true }).waitFor();
  require(writes.length === 2, '超字节限制不能提交');
  await dialog.getByRole('textbox', { name: '补充说明（可选）' }).fill('失败后保留的说明');
  rejectWrite = true;
  await dialog.getByRole('button', { name: /确 定/ }).click();
  await page.getByText('反馈响应桩暂不可用', { exact: true }).waitFor();
  await dialog.locator('button.ant-btn-loading').waitFor({ state: 'hidden' });
  require(await dialog.getByRole('textbox', { name: '补充说明（可选）' }).inputValue() === '失败后保留的说明', '失败必须保留表单');
  rejectWrite = false;
  await dialog.getByRole('combobox', { name: '反馈', exact: true }).click();
  await page.getByText('清除评价', { exact: true }).click();
  await dialog.getByRole('button', { name: /确 定/ }).click();
  await dialog.waitFor({ state: 'hidden' });
  require(writes.length === 4 && feedback === '' && feedback_note === '', '清除评价必须提交显式空字符串');
  const openRename = async () => {
    await page.getByRole('button', { name: `整理会话 ${conversation.title}`, exact: true }).click();
    await page.getByRole('menuitem', { name: '重命名', exact: true }).click();
    return page.getByRole('dialog', { name: '重命名会话' });
  };
  dialog = await openRename();
  require(await dialog.getByRole('textbox', { name: /名称/ }).inputValue() === conversation.title, '重命名首开应带入原名称');
  await dialog.getByRole('textbox', { name: /名称/ }).fill('取消的重命名草稿');
  await dialog.getByRole('button', { name: '取 消', exact: true }).click();
  dialog = await openRename();
  require(await dialog.getByRole('textbox', { name: /名称/ }).inputValue() === conversation.title, '取消后重新打开不能带入旧草稿');
  await dialog.getByRole('textbox', { name: /名称/ }).fill('已保存的新会话名');
  await dialog.getByRole('button', { name: /确 定/ }).click();
  await dialog.waitFor({ state: 'hidden' });
  require(conversation.title === '已保存的新会话名' && renameWrites.length === 1, '重命名须保存并刷新列表');
  dialog = await openRename();
  require(await dialog.getByRole('textbox', { name: /名称/ }).inputValue() === conversation.title, '保存后重开应使用新名称');
  await dialog.getByRole('button', { name: '取 消', exact: true }).click();
  require(errors.length === 0, `出现浏览器异常：${errors.join('; ')}`);
  await page.evaluate(result => { window.__chatFeedbackRegression = result; }, { passed: 9, writes: writes.length, renameWrites: renameWrites.length, errors });
}
