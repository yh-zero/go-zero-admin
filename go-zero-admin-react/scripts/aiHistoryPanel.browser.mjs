// 已登录生产预览后执行；只读核对版本历史模块的加载失败与重试，不恢复版本。
// oxlint-disable-next-line no-unused-expressions -- Playwright CLI 调用。
async page => {
  const origin = new URL(page.url()).origin;
  const pattern = '**/assets/WorkflowHistory-*.js';
  const fail = route => route.abort('failed');
  const response = page.waitForResponse(item => new URL(item.url()).pathname === '/v1/sys/ai/workflows');
  await page.goto(`${origin}/ai/workflows`);
  const workflow = (await (await response).json()).result.list[0];
  if (!workflow) throw new Error('需要已有工作流作为只读历史样本');
  try {
    await page.route(pattern, fail);
    await page.getByRole('button', { name: workflow.name, exact: true }).first().click();
    const detail = page.getByRole('dialog', { name: '工作流详情', exact: true });
    await detail.getByRole('button', { name: '版本历史', exact: true }).click();
    await detail.getByText('面板加载失败，当前页面和草稿已保留', { exact: true }).waitFor({ timeout: 5000 });
    await page.unroute(pattern, fail);
    await detail.getByRole('button', { name: '重新加载面板', exact: true }).click();
    const history = page.getByRole('dialog', { name: `版本历史 · ${workflow.name}`, exact: true });
    await history.getByText('历史保存提交前的完整版本；恢复会创建新版本。', { exact: true }).waitFor();
    await history.locator('.ant-drawer-close').click();
    await detail.getByRole('button', { name: '版本历史', exact: true }).waitFor();
    return 'PASS：版本历史加载失败提示与重试位于可操作的详情抽屉内，重试后历史可关闭返回';
  } finally {
    await page.unroute(pattern, fail);
    await page.goto(`${origin}/ai/workflows`);
  }
}
