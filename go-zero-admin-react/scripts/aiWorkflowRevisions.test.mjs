import test from 'node:test';
import assert from 'node:assert/strict';
import { createServer } from 'vite';

test('工作流版本基线和历史请求使用真实模块', async t => {
  const previousWindow = globalThis.window;
  const values = new Map();
  const storage = { getItem: key => values.get(key) ?? null, setItem: (key, value) => values.set(key, value), removeItem: key => values.delete(key) };
  globalThis.window = { localStorage: storage, sessionStorage: storage, addEventListener() {} };
  const server = await createServer({ configFile: false, envFile: false, logLevel: 'error', server: { middlewareMode: true, watch: null, ws: false } });
  try {
    const drafts = await server.ssrLoadModule('/src/pages/ai/workflowDraft.ts');
    const workflow = { id: 7, name: '原名称', definition: '{"steps":[{"name":"时间","tool":"get_current_time","input":{}}]}', interval_seconds: 0, enabled: 1, next_run_at: 0, created_by: 1, created_at: 1, updated_at: 9, step_count: 1, revision: 1 };
    await t.test('同一秒内即使恢复相同内容，版本变化也必须触发编辑冲突', () => {
      const original = drafts.workflowSnapshot(workflow);
      assert.notEqual(drafts.workflowSnapshot({ ...workflow, revision: 3 }), original);
    });
    await t.test('保存只能使用草稿原基线版本，旧草稿不能自动采用远端新版本', () => {
      const snapshot = drafts.workflowSnapshot(workflow);
      const draft = { name: '  我的草稿  ', enabled: false, steps: [] };
      assert.deepEqual(drafts.workflowUpdateRequest(draft, workflow.definition, snapshot), { expected_revision: 1, name: '我的草稿', definition: workflow.definition, enabled: 0 });
      assert.throws(() => drafts.workflowUpdateRequest(draft, workflow.definition, JSON.stringify({ name: '旧记录', updatedAt: 9 })), /版本|核对/);
      assert.throws(() => drafts.workflowUpdateRequest(draft, workflow.definition, null), /版本|核对/);
      assert.throws(() => drafts.workflowUpdateRequest(draft, workflow.definition, '{broken'), /版本|核对/);
    });
    await t.test('历史读取与条件恢复经过统一请求层，不把恢复变成运行任务', async () => {
      const api = await server.ssrLoadModule('/src/api/ai.ts');
      const { request } = await server.ssrLoadModule('/src/utils/request.ts');
      const { useAuthStore } = await server.ssrLoadModule('/src/stores/auth.ts');
      useAuthStore.getState().setSession({ access_token: 'workflow-revision-fixture', refresh_token: 'workflow-revision-refresh', expires_at: Date.now() / 1000 + 3600, refresh_expires_at: Date.now() / 1000 + 7200, session_id: 'workflow-revision-session' });
      useAuthStore.getState().setPermCodes(['sys:ai:workflow:list', 'sys:ai:workflow:edit']);
      const calls = [];
      request.defaults.adapter = async config => {
        calls.push({ method: config.method, url: config.url, params: config.params, data: config.data ? JSON.parse(config.data) : undefined, authorization: config.headers.get('Authorization') });
        const result = config.url.endsWith('/restore') ? { workflow: { ...workflow, revision: 3 } } : config.url.endsWith('/revisions/1') ? { revision: { workflow_id: 7, revision: 1, definition: workflow.definition } } : { list: [], total: 0 };
        return { data: { code: 200, result }, status: 200, statusText: 'OK', headers: {}, config };
      };
      await api.fetchAiWorkflowRevisions(7, { page: 2, page_size: 10 });
      await api.fetchAiWorkflowRevision(7, 1);
      const restored = await api.restoreAiWorkflowRevision(7, 1, { expected_revision: 2 });
      assert.equal(restored.workflow.revision, 3);
      assert.deepEqual(calls.map(({ method, url, params, data }) => ({ method, url, params, data })), [
        { method: 'get', url: '/sys/ai/workflows/7/revisions', params: { page: 2, page_size: 10 }, data: undefined },
        { method: 'get', url: '/sys/ai/workflows/7/revisions/1', params: undefined, data: undefined },
        { method: 'post', url: '/sys/ai/workflows/7/revisions/1/restore', params: undefined, data: { expected_revision: 2 } },
      ]);
      assert.ok(calls.every(call => call.authorization === 'Bearer workflow-revision-fixture'));
    });
    await t.test('恢复结果未知时保留原目标和expected版本，存储失败也不能变成新恢复', async () => {
      const attempts = await server.ssrLoadModule('/src/pages/ai/workflowRestoreAttempt.ts');
      const storage = { getItem: () => null, setItem() { throw new Error('QuotaExceeded'); }, removeItem() { throw new Error('storage unavailable'); } };
      const key = attempts.workflowRestoreAttemptKey('session-a', 7, 12);
      const original = { revision: 1, expected_revision: 3 };
      assert.throws(() => attempts.writeWorkflowRestoreAttempt(storage, key, original), /Quota/);
      assert.deepEqual(attempts.readWorkflowRestoreAttempt(storage, key), { revision: 1, expected_revision: 3 });
      assert.equal(attempts.readWorkflowRestoreAttempt(storage, attempts.workflowRestoreAttemptKey('session-b', 7, 12)), null);
      assert.equal(attempts.readWorkflowRestoreAttempt(storage, attempts.workflowRestoreAttemptKey('session-a', 8, 12)), null);
      assert.throws(() => attempts.clearWorkflowRestoreAttempt(storage, key), /storage/);
      assert.equal(attempts.readWorkflowRestoreAttempt(storage, key), null);
    });
  } finally { await server.close(); globalThis.window = previousWindow; }
});
