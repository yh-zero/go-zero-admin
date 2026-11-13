import test from 'node:test';
import assert from 'node:assert/strict';
import { createServer } from 'vite';
import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import { App } from 'antd';

test('任务审批、状态与工具安全展示', async t => {
  const originalWindow = globalThis.window;
  const memoryStorage = () => {
    const values = new Map();
    return { getItem: key => values.get(key) ?? null, setItem: (key, value) => values.set(key, value), removeItem: key => values.delete(key) };
  };
  globalThis.window = { localStorage: memoryStorage(), sessionStorage: memoryStorage(), addEventListener() {} };
  const server = await createServer({ configFile: false, envFile: false, logLevel: 'error', server: { middlewareMode: true, watch: null, ws: false } });
  try {
    let helpers;
    await t.test('任务状态辅助函数已提供', async () => {
      helpers = await server.ssrLoadModule('/src/pages/ai/taskState.ts');
      assert.equal(typeof helpers.captureTaskDecision, 'function');
    });
    if (!helpers) return;
    await t.test('审批移除末页唯一任务后回合法页，旧筛选与加载失败不得回写参数', () => {
      const requested = { page: 2, page_size: 20, status: 'pending_approval', workflow_id: 5 };
      const settled = { total: 20, isSuccess: true, isFetching: false };
      assert.deepEqual(helpers.correctTaskPageParams(requested, requested, settled), { ...requested, page: 1 });
      assert.equal(helpers.correctTaskPageParams(requested, requested, { ...settled, total: 21 }), requested);
      const newFilter = { ...requested, status: 'failed', page: 3 };
      assert.equal(helpers.correctTaskPageParams(newFilter, requested, settled), newFilter);
      assert.equal(helpers.correctTaskPageParams(requested, requested, { ...settled, isFetching: true }), requested);
      assert.equal(helpers.correctTaskPageParams(requested, requested, { ...settled, isSuccess: false }), requested);
      assert.equal(helpers.correctTaskPageParams(requested, requested, { ...settled, total: undefined }), requested);
    });
    await t.test('当前步骤从1展示，终态显示已完成数且不推断总步数', () => {
      assert.equal(helpers.taskStepLabel({ next_step: 0, status: 'pending_approval' }), '第 1 步');
      assert.equal(helpers.taskStepLabel({ next_step: 2, status: 'running' }), '第 3 步');
      assert.equal(helpers.taskStepLabel({ next_step: 3, status: 'succeeded' }), '已完成 3 步');
      assert.equal(helpers.taskStepLabel({ next_step: 0, status: 'rejected' }), '已完成 0 步');
    });
    const task = { id: 7, status: 'pending_approval', next_step: 0 };
    await t.test('审批捕获详情断点且0有效，状态变化不能替换旧断点', () => {
      const decision = helpers.captureTaskDecision(task, 'approve');
      assert.deepEqual(decision, { id: 7, expected_step: 0, action: 'approve' });
      task.next_step = 2;
      assert.equal(decision.expected_step, 0);
      assert.equal(helpers.isTaskDecisionCurrent(decision, task), false);
      assert.equal(helpers.captureTaskDecision({ ...task, status: 'running' }, 'approve'), null);
      assert.equal(helpers.captureTaskDecision({ ...task, next_step: -1 }, 'approve'), null);
    });
    await t.test('拒绝原因trim非空，按Unicode字符计算512上限', () => {
      assert.match(helpers.validateRejectReason(' \n '), /原因/);
      assert.equal(helpers.validateRejectReason('😀'.repeat(512)), undefined);
      assert.match(helpers.validateRejectReason('😀'.repeat(513)), /512/);
      assert.equal(helpers.validateRejectReason('  可以拒绝  '), undefined);
    });
    await t.test('仅运行中限定轮询，终态、关闭、写入及失败停止', () => {
      assert.equal(helpers.taskPollInterval('pending', true, 0, 0), 3000);
      assert.equal(helpers.taskPollInterval('running', true, 6000, 0), 3000);
      for (const status of ['pending_approval', 'succeeded', 'failed', 'rejected']) assert.equal(helpers.taskPollInterval(status, true, 0, 0), false);
      assert.equal(helpers.taskPollInterval('running', false, 0, 0), false);
      assert.equal(helpers.taskPollInterval('running', true, 120000, 0), false);
      assert.equal(helpers.taskPollInterval('running', true, 0, 1), false);
    });
    await t.test('任务链接只接受正安全整数', () => {
      assert.equal(helpers.parseTaskId('7'), 7);
      for (const value of [null, '', '0', '-1', '1.5', '1e3', '9007199254740992', '<script>']) assert.equal(helpers.parseTaskId(value), null);
    });
    await t.test('JSON解析失败保留原文且渲染转义', () => {
      assert.equal(helpers.formatTaskValue('{bad <img src=x onerror=alert(1)>'), '{bad <img src=x onerror=alert(1)>');
      assert.equal(helpers.formatTaskValue('{"x":1}'), '{\n  "x": 1\n}');
      const markup = renderToStaticMarkup(createElement('pre', null, helpers.formatTaskValue('<script>evil()</script>')));
      assert.ok(markup.includes('&lt;script&gt;'));
      assert.ok(!markup.includes('<script>'));
    });
    await t.test('任务详情实际渲染转义输入、结果和错误，不推断工作流总进度', async () => {
      const { TaskDetailContent } = await server.ssrLoadModule('/src/pages/ai/TaskDetail.tsx');
      const markup = renderToStaticMarkup(createElement(TaskDetailContent, { task: {
        id: 7, workflow_id: 8, conversation_id: 0, status: 'pending_approval', next_step: 0,
        input: '<img src=x onerror=alert(1)>', result: '{bad <script>result()</script>', error: '<svg onload=alert(1)>',
        approved_by: 0, created_at: 1, completed_at: 0,
      } }));
      for (const text of ['&lt;img', '&lt;script&gt;', '&lt;svg', '当前步骤', '第 1 步', '本人任务']) assert.ok(markup.includes(text), text);
      for (const text of ['<img src=x', '<script>', '总进度', '时间线', 'next_step', '服务端断点', '从 0 开始']) assert.ok(!markup.includes(text), text);
      const raw = ' { "a": 1 } ';
      const rawMarkup = renderToStaticMarkup(createElement(TaskDetailContent, { task: { id: 7, workflow_id: 8, status: 'succeeded', next_step: 1,
        input: raw, result: raw, error: '', approved_by: 0, created_at: 1, completed_at: 2 } }));
      const escapedRaw = renderToStaticMarkup(createElement('pre', null, raw)).slice(5, -6);
      assert.ok(rawMarkup.includes('查看原文'));
      assert.ok(rawMarkup.includes(escapedRaw), '格式化后仍可核对原始输入和结果文本');
    });
    await t.test('工具实际渲染保留坏Schema原文并转义，使用链接由权限控制', async () => {
      const { ToolDetailContent, ToolCard } = await server.ssrLoadModule('/src/pages/ai/AITools.tsx');
      const { MemoryRouter } = await import('react-router-dom');
      const tool = { name: '<script>tool</script>', description: '<img src=x>', parameters_schema: '{bad <svg onload=alert(1)>' };
      const markup = renderToStaticMarkup(createElement(ToolDetailContent, { tool }));
      assert.ok(markup.includes('&lt;script&gt;tool'));
      assert.ok(markup.includes('{bad &lt;svg'));
      assert.ok(!markup.includes('<img src=x'));
      const denied = renderToStaticMarkup(createElement(ToolCard, { tool, canUse: false, onView() {} }));
      assert.ok(!denied.includes('/ai/workflows'));
      const allowed = renderToStaticMarkup(createElement(MemoryRouter, null, createElement(ToolCard, { tool, canUse: true, onView() {} })));
      assert.ok(allowed.includes('/ai/workflows/new?tool=%3Cscript%3Etool%3C%2Fscript%3E'));
    });
    await t.test('运行结果先于输入和执行过程，名称及问题摘要直接可读', async () => {
      const { TaskDetailContent } = await server.ssrLoadModule('/src/pages/ai/TaskDetail.tsx');
      const html = renderToStaticMarkup(createElement(TaskDetailContent, { task: { id: 1, workflow_id: 0, title: '资料风险分析', workflow_name: '一次性分析', input_summary: '本次需要核对什么', status: 'succeeded', next_step: 1, input: '{}', result: '本次分析结果', error: '', created_at: 1, updated_at: 2, completed_at: 2, approved_by: 0, steps: [{ id: 'one', index: 0, name: '资料分析', tool: 'kb_analyze', status: 'succeeded', input: '{}', output: '结果', error: '', started_at: 1, completed_at: 2 }] } }));
      assert.ok(html.includes('资料风险分析'));
      assert.ok(html.includes('本次需要核对什么'));
      assert.ok(html.indexOf('执行结果') < html.indexOf('本次输入'));
      assert.ok(html.indexOf('执行结果') < html.indexOf('步骤执行过程'));
    });
    await t.test('任务错误仅在结果区域安全展示一次，技术字段继续保留', async () => {
      const { TaskDetailContent } = await server.ssrLoadModule('/src/pages/ai/TaskDetail.tsx');
      const html = renderToStaticMarkup(createElement(TaskDetailContent, { task: { id: 1, workflow_id: 3, status: 'failed', next_step: 0, input: '{}', result: '', error: '<svg>唯一错误</svg>', approved_by: 0, created_at: 1, completed_at: 2 } }));
      assert.equal((html.match(/唯一错误/g) ?? []).length, 1);
      assert.ok(html.includes('&lt;svg&gt;'));
      assert.ok(html.includes('审批人 ID'));
      assert.ok(html.includes('本次输入'));
    });
    await t.test('记录首屏仅三个常用状态，终态条目不显示步骤进度', async () => {
      const { useAuthStore } = await server.ssrLoadModule('/src/stores/auth.ts');
      const scope = ['tasks-presentation', 1, 0];
      const auth = { token: 'local-fixture', sessionId: scope[0], sessionVersion: 1, departmentVersion: 0, permissionsReady: true, permCodes: ['sys:ai:task:list'] };
      Object.assign(useAuthStore.getInitialState(), auth);
      useAuthStore.setState(auth);
      const queries = await server.ssrLoadModule('/src/pages/ai/workspaceQueries.ts');
      const { AITaskSession } = await server.ssrLoadModule('/src/pages/ai/AITasks.tsx');
      const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
      try {
        client.setQueryData(queries.taskListOptions(scope, { page: 1, page_size: 20 }).queryKey, { total: 2, list: [
          { id: 1, title: '完成的分析', status: 'succeeded', next_step: 3, created_at: 1 },
          { id: 2, title: '进行的分析', status: 'running', next_step: 1, created_at: 1 },
        ] });
        const html = renderToStaticMarkup(createElement(QueryClientProvider, { client }, createElement(MemoryRouter, { initialEntries: ['/ai/tasks'] }, createElement(App, null, createElement(AITaskSession, { scope })))));
        assert.equal((html.match(/role="tab"/g) ?? []).length, 3);
        assert.ok(html.includes('更多状态'));
        assert.ok(!html.includes('已完成 3 步'));
        assert.ok(html.includes('第 2 步'));
      } finally { client.clear(); }
    });
  } finally { await server.close(); globalThis.window = originalWindow; }
});
