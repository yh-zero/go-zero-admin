import test from 'node:test';
import assert from 'node:assert/strict';
import { createServer } from 'vite';
import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';

test('按需运行入口保留恢复信息且不预挂载运行表单', async () => {
  const server = await createServer({ configFile: false, envFile: false, logLevel: 'error', optimizeDeps: { noDiscovery: true }, server: { middlewareMode: true, watch: null, ws: false } });
  try {
    const module = { ...await server.ssrLoadModule('/src/pages/ai/DeferredWorkflowRun.tsx'), ...await server.ssrLoadModule('/src/pages/ai/deferredRunState.ts') };
    assert.equal(typeof module.readDeferredRunState, 'function');
    const storage = raw => ({ getItem: key => key.endsWith(':task') ? raw.task ?? null : raw.attempt ?? null });
    assert.deepEqual(module.readDeferredRunState(storage({}), 'run'), { kind: 'idle', taskId: null });
    assert.deepEqual(module.readDeferredRunState(storage({ task: '6' }), 'run'), { kind: 'result', taskId: 6 });
    assert.equal(module.readDeferredRunState(storage({ attempt: '{bad' }), 'run').kind, 'unknown');
    assert.equal(module.readDeferredRunState({ getItem() { throw new Error('unavailable'); } }, 'run').kind, 'unknown');
    const markup = renderToStaticMarkup(createElement(module.DeferredWorkflowRun, { attemptId: 'test', scope: ['session', 1, 0], guard: { writing: false }, draft: { name: '样例', steps: [] } }));
    assert.match(markup, /试运行/);
    assert.doesNotMatch(markup, /<form|本次文本|本次问题/);
  } finally { await server.close(); }
});
