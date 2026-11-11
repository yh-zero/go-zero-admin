// 已登录5999开发栈后执行；仅本浏览器GET响应桩，不发送消息或修改权限数据。
// oxlint-disable-next-line no-unused-expressions -- Playwright CLI 调用。
async page => {
  const origin = new URL(page.url()).origin;
  const assert = (value, message) => { if (!value) throw new Error(message); };
  const codes = await page.evaluate(async () => [...(await import('/src/stores/auth.ts')).useAuthStore.getState().permCodes]);
  const permPattern = '**/v1/sys/menu/my/perm-codes';
  const chatPattern = '**/v1/sys/ai/conversations**';
  let toolsOnly = false;
  const permissions = route => route.fulfill({ json: { code: 200, msg: '成功', result: { codes: toolsOnly ? [...codes.filter(code => !code.startsWith('sys:ai:')), 'sys:ai:tool:list'] : codes } } });
  const conversation = { id: 820031, title: '减法对话验收', user_id: 1, message_count: 1, created_at: 1, updated_at: 1, pinned: false };
  const message = { id: 820032, role: 'assistant', status: 'succeeded', content: '精简回答', error: '', created_at: 1, tokens: 13, feedback: '', feedback_note: '', sources: [],
    tool_calls: JSON.stringify([{ id: 'fixture', function: { name: '<script>工具</script>', arguments: '<img src=x onerror=evil()>' } }]) };
  const chat = route => {
    if (route.request().method() !== 'GET') throw new Error('验收不允许写请求');
    const detail = new URL(route.request().url()).pathname.endsWith('/820031');
    return route.fulfill({ json: { code: 200, msg: '成功', result: detail ? { conversation, messages: [message], has_more: false, next_before_id: 0 } : { list: [conversation], total: 1 } } });
  };
  try {
    await page.route(permPattern, permissions);
    await page.goto(`${origin}/ai/workflows`); await page.waitForLoadState('networkidle');
    assert(await page.getByRole('menuitem', { name: /工具与能力$/ }).count() === 0, '有流程入口的用户应下沉工具侧栏入口');
    await page.getByRole('link', { name: '能力目录', exact: true }).click();
    await page.waitForURL('**/ai/tools');
    toolsOnly = true;
    await page.goto(`${origin}/ai/tools`); await page.waitForLoadState('networkidle');
    await page.getByRole('menuitem', { name: /工具与能力$/ }).waitFor();
    assert(!await page.getByRole('link', { name: /用于工作流/ }).count(), '只有工具权限时不能出现创建流程入口');
    toolsOnly = false;
    await page.route(chatPattern, chat);
    await page.goto(`${origin}/ai/chat`);
    await page.getByRole('button', { name: /^减法对话验收 1 条消息/ }).click();
    const article = page.getByRole('article').filter({ hasText: '精简回答' });
    await article.getByText('精简回答', { exact: true }).waitFor();
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
    return 'PASS:工具目录下沉与仅工具权限回退、聊天单行操作/状态精简/详情按需/转义、5个页面390px无横向溢出';
  } finally {
    await page.unroute(permPattern, permissions); await page.unroute(chatPattern, chat);
    await page.setViewportSize({ width: 1440, height: 1000 }); await page.goto(`${origin}/ai/chat`);
  }
}
