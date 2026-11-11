import test from 'node:test';
import assert from 'node:assert/strict';
import { createServer } from 'vite';
import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import { App } from 'antd';

test('流程与运行记录减法保留必要状态和权限', async t => {
  const originalWindow = globalThis.window;
  const values = new Map();
  const storage = { getItem: key => values.get(key) ?? null, setItem: (key, value) => values.set(key, value), removeItem: key => values.delete(key) };
  globalThis.window = { localStorage: storage, sessionStorage: storage, addEventListener() {} };
  const server = await createServer({ configFile: false, envFile: false, logLevel: 'error', server: { middlewareMode: true, watch: null, ws: false } });
  try {
    const { useAuthStore } = await server.ssrLoadModule('/src/stores/auth.ts');
    const scope = ['simplification-test', 1, 0];
    const auth = { token: 'local-fixture', sessionId: scope[0], sessionVersion: 1, departmentVersion: 0, permissionsReady: true,
      permCodes: ['sys:ai:workflow:list', 'sys:ai:workflow:create', 'sys:ai:workflow:edit', 'sys:ai:workflow:run', 'sys:ai:workflow:delete', 'sys:ai:tool:list', 'sys:ai:task:list'] };
    Object.assign(useAuthStore.getInitialState(), auth);
    useAuthStore.setState(auth);
    const queries = await server.ssrLoadModule('/src/pages/ai/workspaceQueries.ts');
    const render = (Component, props, entries = ['/ai/workflows'], seed = () => {}) => {
      const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
      seed(client);
      try { return renderToStaticMarkup(createElement(QueryClientProvider, { client }, createElement(MemoryRouter, { initialEntries: entries }, createElement(App, null, createElement(Component, props))))); }
      finally { client.clear(); }
    };
    await t.test('流程列表只常显运行编辑，能力目录受工具读取权限约束', async () => {
      const { AIWorkflowsSession } = await server.ssrLoadModule('/src/pages/ai/AIWorkflows.tsx');
      const seed = client => client.setQueryData(queries.workflowListOptions(scope, { page: 1, page_size: 20 }).queryKey, { list: [{ id: 81, name: '我的流程', step_count: 1, enabled: 1, updated_at: 1 }], total: 1 });
      const allowed = render(AIWorkflowsSession, { scope }, undefined, seed);
      assert.ok(allowed.includes('href="/ai/tools"'));
      assert.ok(allowed.includes('我的流程'));
      assert.ok(allowed.includes('更多操作'));
      assert.ok(!allowed.includes('手动运行'));
      assert.ok(!allowed.includes('#81'));
      assert.ok(!allowed.includes('>复制<'));
      assert.ok(!allowed.includes('>删除<'));
      useAuthStore.setState({ permCodes: auth.permCodes.filter(code => code !== 'sys:ai:tool:list') });
      const denied = render(AIWorkflowsSession, { scope }, undefined, seed);
      assert.ok(!denied.includes('href="/ai/tools"'));
      useAuthStore.setState(auth);
    });
    await t.test('记录首屏仅三个常用状态，终态条目不显示步骤进度', async () => {
      const { AITaskSession } = await server.ssrLoadModule('/src/pages/ai/AITasks.tsx');
      const html = render(AITaskSession, { scope }, ['/ai/tasks'], client => {
        client.setQueryData(queries.taskListOptions(scope, { page: 1, page_size: 20 }).queryKey, { total: 2, list: [
          { id: 1, title: '完成的分析', status: 'succeeded', next_step: 3, created_at: 1 },
          { id: 2, title: '进行的分析', status: 'running', next_step: 1, created_at: 1 },
        ] });
      });
      assert.equal((html.match(/role="tab"/g) ?? []).length, 3);
      assert.ok(html.includes('更多状态'));
      assert.ok(!html.includes('已完成 3 步'));
      assert.ok(html.includes('第 2 步'));
    });
    await t.test('任务错误仅在结果区域安全展示一次，技术字段继续保留', async () => {
      const { TaskDetailContent } = await server.ssrLoadModule('/src/pages/ai/TaskDetail.tsx');
      const html = renderToStaticMarkup(createElement(TaskDetailContent, { task: { id: 1, workflow_id: 3, status: 'failed', next_step: 0, input: '{}', result: '', error: '<svg>唯一错误</svg>', approved_by: 0, created_at: 1, completed_at: 2 } }));
      assert.equal((html.match(/唯一错误/g) ?? []).length, 1);
      assert.ok(html.includes('&lt;svg&gt;'));
      assert.ok(html.includes('审批人 ID'));
      assert.ok(html.includes('本次输入'));
    });
    await t.test('参数来源先展示引用摘要，完整选择按需展开且原值保留', async () => {
      const { WorkflowParameters } = await server.ssrLoadModule('/src/pages/ai/WorkflowParameters.tsx');
      const tool = { name: 'text_stats', parameters_schema: '{"type":"object","properties":{"text":{"type":"string"}},"required":["text"]}' };
      const html = render(WorkflowParameters, { value: '{"text":{"$ref":"input.question"}}', tool });
      assert.match(html, /<summary[^>]*>[\s\S]*使用本次运行的问题[\s\S]*更改/);
      assert.match(html, /input.question/);
      assert.ok(!/<details[^>]*open=""[^>]*>[\s\S]*?<summary[^>]*>[\s\S]*?更改/.test(html));
      const unknown = render(WorkflowParameters, { value: '{"text":"旧内容","unknown":{"preserve":1}}', tool });
      assert.ok(unknown.includes('&quot;preserve&quot;:1'));
      assert.ok(unknown.includes('高级参数'));
    });
  } finally { await server.close(); globalThis.window = originalWindow; }
});
