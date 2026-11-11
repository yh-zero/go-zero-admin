// 登录独立生产预览后执行：playwright-cli run-code --filename scripts/aiUiPerformance.browser.mjs --raw
// 仅导航和编辑搜索草稿，不提交查询、执行模型或修改业务数据；记录相同页面的冷缓存样本。
// oxlint-disable-next-line no-unused-expressions -- 由 Playwright CLI 调用。
async page => {
  const origin = new URL(page.url()).origin;
  const firstKb = page.waitForResponse(response => new URL(response.url()).pathname.endsWith('/ai/kbs'));
  await page.goto(`${origin}/ai/knowledge`);
  const kbId = (await (await firstKb).json()).result.list[0]?.id;
  if (!kbId) throw new Error('需要一个现有知识库作为一致的只读样本');
  const cdp = await page.context().newCDPSession(page);
  await cdp.send('Network.enable');
  await cdp.send('Network.setCacheDisabled', { cacheDisabled: true });
  const routes = ['/ai/chat', '/ai/knowledge', `/ai/knowledge/${kbId}`, '/ai/workflows', '/ai/workflows/new', '/ai/tasks', '/ai/tools', `/ai/knowledge/${kbId}/quality`, '/ai/usage'];
  const searches = { '/ai/chat': '搜索会话', '/ai/knowledge': '搜索知识库名称', '/ai/workflows': '搜索流程名称', '/ai/tasks': '搜索运行名称或问题' };
  const samples = [];
  try {
    for (const path of routes) {
      for (let sample = 0; sample < 3; sample++) {
        const requests = [];
        const listen = request => { const url = new URL(request.url()); if (url.pathname.startsWith('/v1/')) requests.push({ method: request.method(), path: url.pathname }); };
        page.on('request', listen);
        try {
          await page.goto(`${origin}${path}`);
          await page.getByRole('main').waitFor();
          await page.waitForLoadState('networkidle');
          const metrics = await page.evaluate(() => {
            const entries = performance.getEntriesByType('resource');
            const scripts = entries.filter(item => new URL(item.name).pathname.endsWith('.js'));
            const api = entries.filter(item => new URL(item.name).pathname.startsWith('/v1/'));
            return { scriptBytes: scripts.reduce((sum, item) => sum + item.decodedBodySize, 0), scriptRequests: scripts.length,
              readyMs: Math.max(0, ...api.map(item => item.responseEnd)), domNodes: document.querySelector('main')?.querySelectorAll('*').length ?? 0 };
          });
          let inputMs = null;
          if (searches[path]) {
            const field = page.getByRole('textbox', { name: searches[path], exact: true }).or(page.getByRole('searchbox', { name: searches[path], exact: true })).first();
            if (await field.count()) {
              const timings = [];
              for (let i = 0; i < 5; i++) {
                const start = Date.now(); await field.fill(`性能检查${i}`); await field.fill(''); timings.push(Date.now() - start);
              }
              inputMs = timings.sort((a, b) => a - b)[2];
            }
          }
          if (requests.some(item => item.method !== 'GET')) throw new Error('只读测量出现非GET请求');
          samples.push({ path, sample, ...metrics, inputMs, apiRequests: requests.length, aiRequests: requests.filter(item => item.path.startsWith('/v1/sys/ai/')).map(item => item.path) });
        } finally { page.off('request', listen); }
      }
    }
    return { origin, kbId, cache: 'disabled', samples };
  } finally { await cdp.send('Network.setCacheDisabled', { cacheDisabled: false }); await cdp.detach(); }
}
