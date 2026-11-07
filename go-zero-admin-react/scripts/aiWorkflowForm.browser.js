// 通过 playwright-cli run-code --filename 执行；先加载本轮独立浏览器响应桩。
// 仅测试伪会话与已拦截请求，不可在真实业务会话直接执行。
async (page) => {
  const assert = (condition, message) => { if (!condition) throw new Error(message); };
  const fixture = await page.evaluate(() => typeof window.workflowFixtureState === 'function');
  assert(fixture, '必须先加载工作流浏览器响应桩，禁止向真实业务提交');
  const sessionId = await page.evaluate(() => JSON.parse(localStorage.getItem('go-zero-auth')).state.sessionId);
  assert(sessionId === 'workflow-editor-fixture-20261107', '仅允许本轮伪会话');
  const key = `ai-workflow-draft:${JSON.stringify([sessionId, null, 'new'])}`;
  const handleDialog = dialog => dialog.type() === 'beforeunload' ? dialog.accept() : dialog.dismiss();
  page.on('dialog', handleDialog);
  try {
    const discard = page.getByRole('button', { name: '丢弃草稿', exact: true });
    if (await discard.isVisible()) {
      await discard.click();
      await page.getByRole('button', { name: '丢弃并返回列表', exact: true }).click();
      await page.waitForURL('**/ai/workflows');
    }
    await page.evaluate(key => sessionStorage.removeItem(key), key);
    await page.goto('http://127.0.0.1:5999/ai/workflows/new');
    await page.getByRole('button', { name: /文本处理/ }).click();
    await page.getByLabel('工作流名称', { exact: true }).fill('Form提交持久化回归');
    await page.getByRole('textbox', { name: '处理文本', exact: true }).fill('提交中刷新必须保留这段文本');
    await page.getByLabel('步骤类型', { exact: true }).click();
    await page.locator('.ant-select-item-option[title="当前时间"]').click();
    await page.getByLabel('步骤类型', { exact: true }).click();
    await page.locator('.ant-select-item-option[title="文本统计"]').click();
    assert(await page.getByRole('textbox', { name: '处理文本', exact: true }).inputValue() === '提交中刷新必须保留这段文本', '切回步骤类型丢失文本');
    const before = await page.evaluate(() => window.workflowFixtureState({ failCreate: true }));
    const beforePosts = before.calls.filter(call => call.path === '/v1/sys/ai/workflows' && call.method === 'POST').length;
    await page.getByRole('button', { name: /保存工作流$/ }).click();
    await page.getByText('上次保存结果未知，草稿已保留', { exact: true }).waitFor();
    const submitted = await page.evaluate(key => JSON.parse(sessionStorage.getItem(key)), key);
    assert(submitted.draft.enabled === true, '实际 Form 提交丢失未展示的 enabled，新建草稿将无法刷新恢复');
    assert(submitted.submissionState === 'unknown', '网络结果未知后未持久化 unknown');
    assert(JSON.parse(submitted.draft.steps[0].toolInputs.text_stats).text === '提交中刷新必须保留这段文本', '实际 Form 提交丢失工具参数缓存');
    await page.reload();
    await page.getByText('上次保存结果未知，草稿已保留', { exact: true }).waitFor();
    assert(await page.getByRole('textbox', { name: '处理文本', exact: true }).inputValue() === '提交中刷新必须保留这段文本', '未知提交刷新后未恢复实际文本');
    const after = await page.evaluate(() => window.workflowFixtureState({ failCreate: false }));
    const afterPosts = after.calls.filter(call => call.path === '/v1/sys/ai/workflows' && call.method === 'POST').length;
    assert(afterPosts === beforePosts + 1, `未知提交或刷新发生重复创建：${afterPosts - beforePosts}`);
    const result = { regression: 'actual-form-submit-draft', enabledPreserved: true, toolCachePreserved: true, unknownRestored: true, createdRequests: afterPosts - beforePosts };
    await page.evaluate(result => { window.workflowFormRegressionResult = result; }, result);
    console.log(JSON.stringify(result));
  } finally { page.off('dialog', handleDialog); }
}
