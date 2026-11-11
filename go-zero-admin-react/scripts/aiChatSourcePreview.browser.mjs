// 在已正常登录的本地浏览器中执行：playwright-cli run-code --filename scripts/aiChatSourcePreview.browser.mjs
// 仅拦截本浏览器的会话/原文 GET；不写业务数据、不调用模型。路由和页面在 finally 恢复。
// oxlint-disable-next-line no-unused-expressions -- Playwright CLI 读取并调用此函数表达式。
async page => {
  const assert = (condition, message) => { if (!condition) throw new Error(message); };
  const origin = new URL(page.url()).origin;
  const conversationRoute = '**/v1/sys/ai/conversations**';
  const documentRoute = '**/v1/sys/ai/kbs/987650/docs/*';
  const conversation = { id: 987650, title: '原文侧边预览回归', user_id: 1, message_count: 1, pinned: false, created_at: 1, updated_at: 1 };
  const sources = [1, 2].map(id => ({ kb_id: 987650, doc_id: id, chunk_id: id, doc_title: `回归资料${id}`, snippet: `历史引用${id}<img src=x onerror=evil()>`, score: 0.8 }));
  const message = { id: 987650, role: 'assistant', content: '回答依据 [1] [2]', status: 'succeeded', tool_calls: '[]', tokens: 1, error: '', feedback: '', feedback_note: '', created_at: 1, sources };
  const content = id => `全文${id}起点\n<script>不能执行</script>\n${'长行'.repeat(800)}\n全文${id}终点`;
  const document = id => ({ doc: { id, kb_id: 987650, title: `当前资料${id}`, version: 2, status: 'ready', stage: 'ready', chunk_count: 1, error: '', embedding_model: 'fixture', created_at: 1, updated_at: 2 }, content: content(id) });
  let failure = 0;
  let delayed = false;
  let release;
  let started;
  let finished;
  let requests = 0;
  let writes = 0;
  const reply = (route, result, code = 200) => route.fulfill({ json: { code, msg: code === 200 ? '操作成功' : `原文读取失败 ${code}`, result } });
  const conversations = async route => {
    if (route.request().method() !== 'GET') { writes++; await route.abort(); return; }
    await reply(route, new URL(route.request().url()).pathname.endsWith('/987650')
      ? { conversation, messages: [message], has_more: false, next_before_id: 0 }
      : { list: [conversation], total: 1 });
  };
  const documents = async route => {
    if (route.request().method() !== 'GET') { writes++; await route.abort(); return; }
    requests++;
    const id = Number(new URL(route.request().url()).pathname.split('/').at(-1));
    const isDelayed = delayed && id === 1;
    if (isDelayed) {
      started?.();
      await new Promise(resolve => { release = resolve; });
    }
    try { await reply(route, failure ? null : document(id), failure || 200); }
    finally { if (isDelayed) finished?.(); }
  };
  const drawer = () => page.getByRole('dialog', { name: '当前原文' });
  const open = id => page.getByRole('button', { name: '查看原文', exact: true }).nth(id - 1).click();
  const close = async () => {
    await drawer().getByRole('button', { name: '关闭', exact: true }).click();
    await drawer().waitFor({ state: 'detached' });
  };
  const full = () => page.getByLabel('当前原文全文', { exact: true });
  try {
    await page.route(conversationRoute, conversations);
    await page.route(documentRoute, documents);
    await page.goto(`${origin}/ai/chat`);
    await page.getByRole('button', { name: /^原文侧边预览回归 1 条消息/ }).click();
    await page.getByRole('button', { name: '查看原文', exact: true }).first().waitFor();
    assert(requests === 0, '打开引用前不应预取全文');
    await page.getByRole('textbox', { name: '消息内容' }).fill('验收草稿（不发送）');
    const excerpt = page.getByText('本次引用片段', { exact: true }).first();
    await excerpt.click();
    assert(await page.getByText(sources[0].snippet, { exact: true }).isVisible(), '历史片段应能展开');
    await page.getByRole('button', { name: '查看原文', exact: true }).first().scrollIntoViewIfNeeded();
    const scrollBefore = await page.getByRole('log', { name: '聊天消息' }).evaluate(element => element.scrollTop);
    await page.getByRole('button', { name: '查看原文', exact: true }).first().dblclick();
    await full().waitFor();
    assert(await drawer().count() === 1, '双击应只打开一个抽屉且不能被第二次点击关闭');
    assert(await full().textContent() === content(1), '原文应完整保留换行及末尾');
    assert(await full().locator('script,img').count() === 0, '原文中的HTML应仅显示为文本');
    await close();
    assert(await page.getByRole('textbox', { name: '消息内容' }).inputValue() === '验收草稿（不发送）', '关闭应保留草稿');
    const scrollAfter = await page.getByRole('log', { name: '聊天消息' }).evaluate(element => element.scrollTop);
    assert(Math.abs(scrollAfter - scrollBefore) <= 2, '关闭应保留滚动位置');
    assert(await page.getByText(sources[0].snippet, { exact: true }).isVisible(), '预览后历史片段不能被当前正文替换');
    for (const code of [403, 404, 503]) {
      failure = code;
      await open(1);
      await drawer().getByText('暂时无法加载数据', { exact: true }).waitFor();
      assert(await full().count() === 0, `读取失败${code}不能显示缓存旧全文`);
      failure = 0;
      await drawer().getByRole('button', { name: '重新加载', exact: true }).click();
      await full().waitFor();
      assert(await full().textContent() === content(1), '重试应恢复全文');
      await close();
    }
    // 等统一错误提示自然退场，避免缩到手机视口后提示覆盖关闭按钮并被鼠标悬停续期。
    await page.mouse.move(0, 0);
    await page.locator('.ant-message-notice').waitFor({ state: 'hidden' });
    delayed = true;
    const pending = new Promise(resolve => { started = resolve; });
    const completed = new Promise(resolve => { finished = resolve; });
    await open(1);
    await pending;
    await close();
    await open(2);
    await full().waitFor();
    assert(await full().textContent() === content(2), '切换应显示第二篇原文');
    release();
    await completed;
    delayed = false;
    await drawer().getByRole('heading', { name: '当前资料2', exact: true }).waitFor();
    assert(await full().textContent() === content(2), '第一篇迟到响应不能覆盖第二篇');
    await page.setViewportSize({ width: 390, height: 844 });
    await page.waitForFunction(() => {
      const element = document.querySelector('.ant-drawer [role="dialog"]');
      const rect = element?.getBoundingClientRect();
      return rect && rect.x >= -1 && rect.right <= innerWidth + 1;
    });
    const bounds = await drawer().boundingBox();
    assert(bounds && bounds.x >= -1 && bounds.x + bounds.width <= 391, '窄屏抽屉不能超出视口');
    assert(await full().evaluate(element => element.scrollWidth <= element.clientWidth + 1), '全文长行应折行');
    await close();
    assert(writes === 0, '预览不能产生业务写请求');
    return 'PASS：按需全文、历史片段、HTML转义、双击、草稿、滚动位置、403/404/503与重试、迟到响应、390px窄屏';
  } finally {
    release?.();
    await page.unroute(conversationRoute, conversations);
    await page.unroute(documentRoute, documents);
    await page.setViewportSize({ width: 1440, height: 1000 });
    await page.goto(`${origin}/ai/chat`);
  }
}
