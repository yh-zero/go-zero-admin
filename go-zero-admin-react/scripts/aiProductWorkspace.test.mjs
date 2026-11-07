import test from 'node:test';
import assert from 'node:assert/strict';
import { createServer } from 'vite';
import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';

test('后台排队和处理中资料不会因等待时长而重复提交', async () => {
  const server = await createServer({ configFile: false, envFile: false, logLevel: 'error', optimizeDeps: { noDiscovery: true }, server: { middlewareMode: true, watch: null, ws: false } });
  try {
    const { canRetryDocument } = await server.ssrLoadModule('/src/pages/ai/knowledgeDocuments.ts');
    for (const stage of ['queued', 'embedding', 'indexing']) assert.equal(canRetryDocument({ status: 'pending', stage, updated_at: 1 }, 999999999), false);
    assert.equal(canRetryDocument({ status: 'failed', stage: 'failed', updated_at: 1 }), true);
  } finally { await server.close(); }
});

test('导入离页保留已发出未知项，确定失败和未发出项不会混进未知锁', async () => {
  const server = await createServer({ configFile: false, envFile: false, logLevel: 'error', optimizeDeps: { noDiscovery: true }, server: { middlewareMode: true, watch: null, ws: false } });
  try {
    const imports = await server.ssrLoadModule('/src/pages/ai/knowledgeImport.ts');
    const { ApiError } = await server.ssrLoadModule('/src/utils/response.ts');
    assert.equal(typeof imports.advanceKnowledgeImportAttempt, 'function');
    const prepared = imports.prepareKnowledgeBatch(Array.from({ length: 4 }, (_, index) => ({ title: `${index}`, content: '正文' })));
    let current = true;
    let releaseFirst;
    const first = new Promise(resolve => { releaseFirst = resolve; });
    let attempt = { batchId: prepared.batchId, requestIds: [], unsubmittedIds: prepared.items.map(item => item.id) };
    const sent = [];
    await imports.runKnowledgeImport(prepared.items, async values => {
      sent.push(values.request_id);
      if (values.title === '0') { await first; throw new Error('network outcome unknown'); }
      current = false; releaseFirst(); throw new ApiError(400, '确定未接收');
    }, () => {}, () => current, change => { attempt = imports.advanceKnowledgeImportAttempt(attempt, change); });
    assert.equal(sent.length, 2);
    assert.deepEqual(attempt.requestIds, [prepared.items[0].id]);
    assert.deepEqual(attempt.unsubmittedIds, prepared.items.slice(2).map(item => item.id));
    const memory = new Map();
    const storage = { getItem: key => memory.get(key) ?? null, setItem: (key, value) => memory.set(key, value) };
    imports.writeKnowledgeImportAttempt(storage, 'original', attempt);
    assert.deepEqual(imports.readKnowledgeImportAttempt(storage, 'original'), attempt);
  } finally { await server.close(); }
});

test('资料分析工作区的可复用运行参数', async t => {
  const originalWindow = globalThis.window;
  const memoryStorage = () => { const values = new Map(); return { getItem: key => values.get(key) ?? null, setItem: (key, value) => values.set(key, value), removeItem: key => values.delete(key) }; };
  globalThis.window = { localStorage: memoryStorage(), sessionStorage: memoryStorage(), addEventListener() {} };
  const server = await createServer({ configFile: false, envFile: false, logLevel: 'error', optimizeDeps: { noDiscovery: true }, server: { middlewareMode: true, watch: null, ws: false } });
  try {
    const dsl = await server.ssrLoadModule('/src/pages/ai/workflowDefinition.ts');
    const tools = [
      { name: 'kb_analyze', parameters_schema: JSON.stringify({ type: 'object', properties: { kb_id: { type: 'integer', minimum: 1 }, query: { type: 'string', minLength: 1 } }, required: ['kb_id', 'query'] }) },
      { name: 'text_stats', parameters_schema: JSON.stringify({ type: 'object', properties: { text: { type: 'string' } }, required: ['text'] }) },
      { name: 'json_formatter', parameters_schema: JSON.stringify({ type: 'object', properties: { text: { type: 'string' } }, required: ['text'] }) },
    ];
    await t.test('默认分析仅执行有实际用途的一步，并在每次运行选择知识库和问题', () => {
      const draft = dsl.workflowTemplate('kb_analyze', tools);
      assert.equal(draft.steps.length, 1);
      assert.deepEqual(JSON.parse(draft.steps[0].input), { kb_id: { $ref: 'input.kb_id' }, query: { $ref: 'input.question' } });
      const definition = dsl.serializeWorkflow(draft.steps, tools);
      assert.deepEqual(dsl.workflowRunFields(definition), { question: true, text: false, kb_id: true });
    });
    await t.test('知识库运行输入覆盖同名高级值，并拒绝无效标识与超限正文', () => {
      assert.deepEqual(JSON.parse(dsl.prepareWorkflowRunInput({ input: '{"kb_id":2,"extra":7}', kb_id: 13, question: '本次分析' })), { kb_id: 13, extra: 7, question: '本次分析' });
      for (const kb_id of [0, -1, 1.5, Number.MAX_SAFE_INTEGER + 1]) assert.throws(() => dsl.prepareWorkflowRunInput({ kb_id }), /知识库/);
      assert.throws(() => dsl.prepareWorkflowRunInput({ kb_id: 1, question: '中'.repeat(6000) }), /16000/);
    });
    await t.test('JSON起步模板按格式整理命名，并明确使用本次文本', () => {
      const draft = dsl.workflowTemplate('json_formatter', tools);
      assert.match(draft.name, /JSON/);
      assert.deepEqual(JSON.parse(draft.steps[0].input), { text: { $ref: 'input.text' } });
      assert.doesNotThrow(() => dsl.serializeWorkflow(draft.steps, tools));
    });
    await t.test('未确认的草稿运行保留原始定义、输入与请求标识，关闭返回不变更', async () => {
      const attempts = await server.ssrLoadModule('/src/pages/ai/draftRunAttempts.ts').catch(() => ({}));
      assert.equal(typeof attempts.readDraftRunAttempt, 'function');
      const state = new Map();
      const storage = { getItem: key => state.get(key) ?? null, setItem: (key, value) => state.set(key, value), removeItem: key => state.delete(key) };
      const attempt = { name: '风险分析', definition: '{"steps":[]}', input: '{"kb_id":7}', request_id: '11111111-1111-4111-8111-111111111111' };
      attempts.writeDraftRunAttempt(storage, 'scope-a:new', attempt);
      assert.deepEqual(attempts.readDraftRunAttempt(storage, 'scope-a:new'), attempt);
      assert.equal(attempts.readDraftRunAttempt(storage, 'scope-b:new'), null);
      attempts.clearDraftRunAttempt(storage, 'scope-a:new');
      assert.equal(attempts.readDraftRunAttempt(storage, 'scope-a:new'), null);
    });
    await t.test('运行结果先于输入和执行过程，名称及问题摘要直接可读', async () => {
      const { TaskDetailContent } = await server.ssrLoadModule('/src/pages/ai/TaskDetail.tsx');
      const html = renderToStaticMarkup(createElement(TaskDetailContent, { task: { id: 1, workflow_id: 0, title: '资料风险分析', workflow_name: '一次性分析', input_summary: '本次需要核对什么', status: 'succeeded', next_step: 1, input: '{}', result: '本次分析结果', error: '', created_at: 1, updated_at: 2, completed_at: 2, approved_by: 0, steps: [{ id: 'one', index: 0, name: '资料分析', tool: 'kb_analyze', status: 'succeeded', input: '{}', output: '结果', error: '', started_at: 1, completed_at: 2 }] } }));
      assert.ok(html.includes('资料风险分析'));
      assert.ok(html.includes('本次需要核对什么'));
      assert.ok(html.indexOf('执行结果') < html.indexOf('本次输入'));
      assert.ok(html.indexOf('执行结果') < html.indexOf('步骤执行过程'));
    });
    await t.test('导入批次在提交前分配唯一请求标识，终态停止轮询', async () => {
      const imports = await server.ssrLoadModule('/src/pages/ai/knowledgeImport.ts');
      assert.equal(typeof imports.prepareKnowledgeBatch, 'function');
      const batch = imports.prepareKnowledgeBatch([{ title: '甲', content: '正文' }, { title: '乙', content: '正文' }]);
      assert.equal(new Set(batch.items.map(item => item.values.request_id)).size, 2);
      assert.ok(batch.items.every(item => item.values.batch_id === batch.batchId));
      assert.equal(imports.knowledgeBatchPollInterval({ queued: 1, running: 0 }, true), 3000);
      assert.equal(imports.knowledgeBatchPollInterval({ queued: 0, running: 0 }, true), false);
      assert.equal(imports.knowledgeBatchPollInterval({ queued: 1, running: 1 }, false), false);
      assert.throws(() => imports.prepareKnowledgeBatch(Array.from({ length: 21 }, () => ({ title: '甲', content: '正文' }))), /20/);
      const pending = { batchId: batch.batchId, requestIds: batch.items.map(item => item.id) };
      const memory = new Map();
      const storage = { getItem: key => memory.get(key) ?? null, setItem: (key, value) => memory.set(key, value), removeItem: key => memory.delete(key) };
      assert.equal(typeof imports.writeKnowledgeImportAttempt, 'function');
      imports.writeKnowledgeImportAttempt(storage, 'scope-a:kb1', pending);
      assert.deepEqual(imports.readKnowledgeImportAttempt(storage, 'scope-a:kb1'), pending);
      assert.equal(imports.readKnowledgeImportAttempt(storage, 'scope-b:kb1'), null);
    });
  } finally { await server.close(); globalThis.window = originalWindow; }
});
