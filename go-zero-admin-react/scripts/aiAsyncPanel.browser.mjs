// 已登录生产预览后执行，专门验证按需模块的网络失败和加载中收起；不执行分析。
// oxlint-disable-next-line no-unused-expressions -- Playwright CLI 调用。
async page => {
  const origin = new URL(page.url()).origin;
  const pattern = '**/assets/WorkflowDraftRun-*.js';
  const response = page.waitForResponse(item => new URL(item.url()).pathname.endsWith('/ai/kbs'));
  await page.goto(`${origin}/ai/knowledge`);
  const kbId = (await (await response).json()).result.list[0]?.id;
  if (!kbId) throw new Error('需要已有知识库');
  let release;
  const fail = route => route.abort('failed');
  const delay = async route => { await new Promise(resolve => { release = resolve; }); await route.continue(); };
  try {
    await page.route(pattern, fail);
    await page.goto(`${origin}/ai/knowledge/${kbId}`);
    await page.getByRole('button', { name: /用资料分析$/ }).click();
    await page.getByText('面板加载失败，当前页面和草稿已保留', { exact: true }).waitFor({ timeout: 5000 });
    await page.unroute(pattern, fail);
    await page.getByRole('button', { name: '重新加载面板', exact: true }).click();
    const question = page.getByRole('textbox', { name: /你希望了解什么/ });
    await question.fill('只检查草稿，不执行分析');
    await page.getByRole('button', { name: /收起分析$/ }).click();
    await page.getByRole('button', { name: /用资料分析$/ }).click();
    if (await question.inputValue() !== '只检查草稿，不执行分析') throw new Error('重试后草稿丢失');
    await page.route(pattern, delay);
    await page.goto(`${origin}/ai/knowledge/${kbId}`);
    await page.getByRole('button', { name: /用资料分析$/ }).click();
    await page.getByRole('status', { name: '正在加载面板' }).waitFor();
    await page.getByRole('button', { name: /收起分析$/ }).click();
    release();
    await page.getByRole('button', { name: /用资料分析$/ }).click();
    await question.waitFor();
    return 'PASS:模块加载失败局部重试、工作区保持、草稿保留、加载中收起后恢复';
  } finally {
    release?.(); await page.unroute(pattern, fail); await page.unroute(pattern, delay);
    await page.goto(`${origin}/ai/knowledge`);
  }
}
