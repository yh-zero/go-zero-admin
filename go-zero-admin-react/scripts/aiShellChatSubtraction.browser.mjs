// 已登录5999开发栈后执行；仅本浏览器GET响应桩，不发送消息或修改权限数据。
// oxlint-disable-next-line no-unused-expressions -- Playwright CLI 调用。
async page => {
  const origin = new URL(page.url()).origin;
  const assert = (value, message) => { if (!value) throw new Error(message); };
  const codes = await page.evaluate(async () => [...(await import('/src/stores/auth.ts')).useAuthStore.getState().permCodes]);
  const permPattern = '**/v1/sys/menu/my/perm-codes';
  const chatPattern = '**/v1/sys/ai/conversations**';
  const kbPattern = '**/v1/sys/ai/kbs**';
  const safetyPattern = '**/v1/**';
  let toolsOnly = false;
  const handleDialog = dialog => dialog.type() === 'beforeunload' ? dialog.accept() : dialog.dismiss();
  const safety = route => ['GET', 'HEAD', 'OPTIONS'].includes(route.request().method()) ? route.fallback() : route.abort('blockedbyclient');
  const permissions = route => route.fulfill({ json: { code: 200, msg: '成功', result: { codes: toolsOnly ? [...codes.filter(code => !code.startsWith('sys:ai:')), 'sys:ai:tool:list'] : codes } } });
  const conversation = { id: 820031, title: '减法对话验收', user_id: 1, message_count: 1, created_at: 1, updated_at: 1, pinned: false };
  const message = { id: 820032, role: 'assistant', status: 'succeeded', content: '精简回答', error: '', created_at: 1, tokens: 13, feedback: '', feedback_note: '', sources: [],
    tool_calls: JSON.stringify([{ id: 'fixture', function: { name: '<script>工具</script>', arguments: '<img src=x onerror=evil()>' } }]) };
  const chat = route => {
    if (route.request().method() !== 'GET') throw new Error('验收不允许写请求');
    const detail = new URL(route.request().url()).pathname.endsWith('/820031');
    return route.fulfill({ json: { code: 200, msg: '成功', result: detail ? { conversation, messages: [message], has_more: false, next_before_id: 0 } : { list: [conversation], total: 1 } } });
  };
  const knowledge = route => {
    if (route.request().method() !== 'GET') throw new Error('验收不允许知识库写请求');
    const kb = { id: 7, name: '范围回归知识库', description: '', doc_count: 1, chunk_count: 1, ready_count: 1, pending_count: 0, failed_count: 0, created_at: 1, updated_at: 1 };
    const capabilities = { retrieval_configured: true, analysis_configured: true, ingestion_mode: 'background' };
    return route.fulfill({ json: { code: 200, msg: '成功', result: new URL(route.request().url()).pathname.endsWith('/7') ? { kb, capabilities } : { list: [kb], total: 1, capabilities } } });
  };
  const checkKnowledgePicker = async () => {
    assert(await page.getByRole('switch', { name: '仅依据所选资料回答', exact: true }).count() === 0, '自动检索不应显示不可操作的约束开关');
    await page.getByRole('combobox', { name: '对话知识库', exact: true }).click();
    await page.getByText('范围回归知识库（1 篇）', { exact: true }).last().click();
    const constraint = page.getByRole('switch', { name: '仅依据所选资料回答', exact: true });
    await constraint.waitFor();
    await constraint.click();
    assert(await constraint.getAttribute('aria-checked') === 'true', '指定知识库仍可启用资料约束');
    await page.getByRole('combobox', { name: '对话知识库', exact: true }).click();
    await page.getByText('自动检索我的知识库', { exact: true }).last().click();
    assert(await constraint.count() === 0, '切回自动范围应隐藏约束开关');
  };
  try {
    page.on('dialog', handleDialog);
    await page.route(safetyPattern, safety);
    await page.route(permPattern, permissions);
    await page.route(kbPattern, knowledge);
    await page.goto(`${origin}/ai/workflows`); await page.waitForLoadState('networkidle');
    assert(await page.getByRole('menuitem', { name: /工具与能力$/ }).count() === 0, '有流程入口的用户应下沉工具侧栏入口');
    assert(await page.getByRole('link', { name: '能力目录', exact: true }).count() === 0, '常用流程列表不应重复提供能力目录跳转');
    await page.getByRole('button', { name: /新建常用流程$/ }).click();
    await page.waitForURL('**/ai/workflows/new');
    assert(await page.getByRole('button', { name: /JSON 格式整理$/ }).count() === 0, 'JSON模板不应占用新建首屏');
    await page.getByRole('button', { name: '添加步骤', exact: true }).click();
    const capability = page.getByRole('dialog', { name: '按用途选择能力', exact: true });
    await capability.getByRole('heading', { name: 'JSON 格式整理', exact: true }).waitFor();
    await page.keyboard.press('Escape');
    await capability.waitFor({ state: 'hidden' });
    const compatibility = await page.context().newPage();
    try {
      await compatibility.route(safetyPattern, safety);
      await compatibility.goto(`${origin}/ai/workflows/new?tool=json_formatter`);
      const name = compatibility.getByRole('textbox', { name: '常用流程名称', exact: true });
      await name.waitFor();
      assert(await name.inputValue() === 'JSON 格式整理', '旧工具参数入口应继续初始化JSON流程');
    } finally { await compatibility.close(); }
    toolsOnly = true;
    await page.goto(`${origin}/ai/tools`); await page.waitForLoadState('networkidle');
    await page.getByRole('menuitem', { name: /工具与能力$/ }).waitFor();
    assert(!await page.getByRole('link', { name: /用于.*流程/ }).count(), '只有工具权限时不能出现创建流程入口');
    toolsOnly = false;
    await page.route(chatPattern, chat);
    await page.goto(`${origin}/ai/chat`);
    await page.getByRole('combobox', { name: '对话知识库', exact: true }).waitFor();
    await checkKnowledgePicker();
    await page.getByRole('button', { name: /^减法对话验收 1 条消息/ }).click();
    const article = page.getByRole('article').filter({ hasText: '精简回答' });
    await article.getByText('精简回答', { exact: true }).waitFor();
    await checkKnowledgePicker();
    assert(await article.getByText('已完成', { exact: true }).count() === 0, '正常消息不重复完成标签');
    assert(await article.getByText(/已知 token：13/).count() === 0, '用量详情首屏不挂载');
    const copy = await article.getByRole('button', { name: /复制回答$/ }).boundingBox();
    const feedback = await article.getByRole('button', { name: '评价回答', exact: true }).boundingBox();
    assert(copy && feedback && Math.abs(copy.y - feedback.y) <= 3, '复制与评价应在同一操作行');
    await article.getByText('用量与调用详情', { exact: true }).click();
    await article.getByText(/已知 token：13/).waitFor();
    await article.getByText('<script>工具</script>', { exact: true }).waitFor();
    assert(await article.locator('script,img').count() === 0, '工具参数必须文本转义');
    await page.unroute(chatPattern, chat);
    await page.setViewportSize({ width: 390, height: 844 });
    const paths = ['/ai/chat', '/ai/workflows', '/ai/tasks', '/ai/knowledge', '/ai/tools'];
    for (const path of paths) {
      await page.goto(`${origin}${path}`); await page.waitForLoadState('networkidle');
      assert(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1), `${path} 窄屏页面横向溢出`);
    }
    return 'PASS:常用流程入口收口/JSON能力保留/仅工具权限回退、新旧会话知识库约束显隐、聊天单行操作/状态精简/详情按需/转义、5个页面390px无横向溢出';
  } finally {
    await page.unroute(permPattern, permissions); await page.unroute(chatPattern, chat); await page.unroute(kbPattern, knowledge); await page.unroute(safetyPattern, safety);
    await page.setViewportSize({ width: 1440, height: 1000 }); await page.goto(`${origin}/ai/chat`);
    page.off('dialog', handleDialog);
  }
}
