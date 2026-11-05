import test from 'node:test';
import assert from 'node:assert/strict';
import { createServer } from 'vite';
import { QueryClient } from '@tanstack/react-query';
import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';

function memoryStorage() {
  const values = new Map();
  return { getItem: key => values.get(key) ?? null, setItem: (key, value) => values.set(key, value), removeItem: key => values.delete(key) };
}

test('AI 工作区接入边界', async t => {
  const originalWindow = globalThis.window;
  globalThis.window = { localStorage: memoryStorage(), sessionStorage: memoryStorage(), addEventListener() {} };
  const server = await createServer({ configFile: false, envFile: false, logLevel: 'error', server: { middlewareMode: true, watch: null, ws: false } });
  try {
    const { requiredWritePermission } = await server.ssrLoadModule('/src/utils/permissions.ts');
    await t.test('六项写入逐一映射操作码，读取不冒充写入', () => {
      for (const [method, path, permission] of [
        ['POST', '/sys/ai/workflows', 'sys:ai:workflow:create'],
        ['PUT', '/sys/ai/workflows/7', 'sys:ai:workflow:edit'],
        ['DELETE', '/sys/ai/workflows/7', 'sys:ai:workflow:delete'],
        ['POST', '/sys/ai/workflows/7/run', 'sys:ai:workflow:run'],
        ['POST', '/sys/ai/tasks/8/approve', 'sys:ai:task:approve'],
        ['POST', '/sys/ai/tasks/8/reject', 'sys:ai:task:reject'],
      ]) assert.equal(requiredWritePermission(method, path), permission);
      assert.equal(requiredWritePermission('GET', '/sys/ai/workflows/7'), undefined);
      assert.equal(requiredWritePermission('POST', '/sys/ai/workflows/7'), undefined);
    });

    const queries = await server.ssrLoadModule('/src/pages/ai/workspaceQueries.ts');
    const { AIWorkspaceAccessView } = await server.ssrLoadModule('/src/pages/ai/AIWorkspaceAccess.tsx');
    const { isAiWorkspaceCurrent } = await server.ssrLoadModule('/src/pages/ai/useAiWorkspaceGuard.ts');
    const { useAuthStore } = await server.ssrLoadModule('/src/stores/auth.ts');
    const { request } = await server.ssrLoadModule('/src/utils/request.ts');
    const api = await server.ssrLoadModule('/src/api/ai.ts');
    const readCode = 'sys:ai:workflow:list';
    const scope = ['workspace-session', 1, 1];
    function authorize(codes = [readCode]) {
      useAuthStore.setState({ token: 'local-test-access', sessionId: scope[0], sessionVersion: scope[1], departmentVersion: scope[2],
        expiresAt: Date.now() / 1000 + 3600, isLoggingOut: false, permissionsReady: true, permCodes: codes });
    }
    authorize();
    await t.test('所有列表详情按身份部门分开；任务筛选参与缓存键', () => {
      for (const key of ['tools', 'workflows', 'tasks']) {
        assert.notDeepEqual(queries.aiWorkspaceKeys[key](scope), queries.aiWorkspaceKeys[key](['other', 1, 1]));
        assert.notDeepEqual(queries.aiWorkspaceKeys[key](scope), queries.aiWorkspaceKeys[key](['workspace-session', 1, 2]));
      }
      assert.notDeepEqual(queries.aiWorkspaceKeys.task(scope, 1), queries.aiWorkspaceKeys.task(scope, 2));
      assert.notDeepEqual(queries.taskListOptions(scope, { status: 'failed' }).queryKey, queries.taskListOptions(scope, { status: 'running' }).queryKey);
      assert.notDeepEqual(queries.workflowListOptions(scope, { page: 1 }).queryKey, queries.workflowListOptions(scope, { page: 2 }).queryKey);
    });
    await t.test('权限未就绪与被拒绝时不挂载业务查询，撤权与退出使旧回调失效', () => {
      let mounted = 0;
      const render = () => renderToStaticMarkup(createElement(AIWorkspaceAccessView, { code: readCode, title: '工作流管理', scope,
        permissionsReady: useAuthStore.getState().permissionsReady, permCodes: useAuthStore.getState().permCodes },
        () => { mounted++; return createElement('p', null, '业务内容'); }));
      useAuthStore.setState({ permissionsReady: false });
      assert.match(render(), /加载/);
      assert.equal(mounted, 0);
      useAuthStore.setState({ permissionsReady: true, permCodes: [] });
      assert.match(render(), /无权/);
      assert.equal(mounted, 0);
      assert.equal(isAiWorkspaceCurrent(scope, readCode), false);
      authorize();
      assert.match(render(), /业务内容/);
      assert.equal(isAiWorkspaceCurrent(scope, readCode), true);
      useAuthStore.setState({ departmentVersion: 2 });
      assert.equal(isAiWorkspaceCurrent(scope, readCode), false);
      authorize();
      useAuthStore.setState({ isLoggingOut: true });
      assert.equal(isAiWorkspaceCurrent(scope, readCode), false);
      authorize();
    });
    await t.test('取消已关闭详情后，迟到响应不能恢复缓存', async () => {
      const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
      let release;
      const response = new Promise(resolve => { release = resolve; });
      request.defaults.adapter = async config => {
        await response;
        return { data: { code: 200, result: { task: { id: 8, next_step: 0 } } }, status: 200, statusText: 'OK', headers: {}, config };
      };
      const options = queries.taskDetailOptions(scope, 8);
      const pending = client.fetchQuery(options).catch(() => undefined);
      await Promise.resolve();
      await client.cancelQueries({ queryKey: options.queryKey });
      client.removeQueries({ queryKey: options.queryKey });
      release();
      await pending;
      assert.equal(client.getQueryData(options.queryKey), undefined);
      client.clear();
    });
    await t.test('统一请求层拒绝无权限写入；审批显式发送零断点且不更换请求', async () => {
      let calls = 0;
      let payload;
      request.defaults.adapter = async config => {
        calls++;
        payload = JSON.parse(config.data);
        return { data: { code: 200, result: {} }, status: 200, statusText: 'OK', headers: {}, config };
      };
      authorize([]);
      await assert.rejects(api.approveAiTask(8, { expected_step: 0 }), error => error.code === 403);
      assert.equal(calls, 0);
      authorize(['sys:ai:task:approve']);
      await api.approveAiTask(8, { expected_step: 0 });
      assert.equal(calls, 1);
      assert.deepEqual(payload, { expected_step: 0 });
    });
  } finally {
    await server.close();
    globalThis.window = originalWindow;
  }
});
