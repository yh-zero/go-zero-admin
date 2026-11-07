import test from 'node:test';
import assert from 'node:assert/strict';
import { createServer } from 'vite';
import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';

test('工作流变量、运行表单和步骤快照', async t => {
  const server = await createServer({ configFile: false, envFile: false, logLevel: 'error', server: { middlewareMode: true, watch: null, ws: false } });
  try {
    const dsl = await server.ssrLoadModule('/src/pages/ai/workflowDefinition.ts');
    const attempts = await server.ssrLoadModule('/src/pages/ai/workflowRunAttempts.ts');
    const tools = [{ name: 'text_stats', parameters_schema: '{"type":"object","properties":{"text":{"type":"string"}},"required":["text"]}' }, { name: 'json_formatter', parameters_schema: '{"type":"object","properties":{"text":{"type":"string"}},"required":["text"]}' }];
    await t.test('v2稳定ID在刷新与移动后保留，只允许前序引用', () => {
      const first = { ...dsl.newWorkflowStep('json_formatter'), input: '{"text":{"$ref":"input.text"}}' };
      const next = { ...dsl.newWorkflowStep('text_stats'), input: JSON.stringify({ text: { $ref: `steps.${first.key}` } }) };
      const raw = dsl.serializeWorkflow([first, next], tools);
      assert.equal(JSON.parse(raw).version, 2);
      const parsed = dsl.parseWorkflow(raw, tools);
      assert.equal(parsed[0].id, first.key);
      assert.equal(parsed[1].key, next.key);
      assert.deepEqual(JSON.parse(dsl.serializeWorkflow(parsed, tools)), JSON.parse(raw));
      assert.throws(() => dsl.serializeWorkflow([next, first], tools), /前序/);
      for (const ref of ['input.__proto__', 'steps.future', 'input.constructor']) assert.throws(() => dsl.serializeWorkflow([{ ...first, input: JSON.stringify({ text: { $ref: ref } }) }], tools), /变量|前序/);
    });
    await t.test('普通运行字段从引用推导，合并高级参数并检查总字节', () => {
      const raw = dsl.serializeWorkflow([{ ...dsl.newWorkflowStep('text_stats'), input: '{"text":{"$ref":"input.question"}}' }], tools);
      assert.deepEqual(dsl.workflowRunFields(raw), { question: true, text: false, kb_id: false });
      assert.deepEqual(JSON.parse(dsl.prepareWorkflowRunInput({ input: '{"extra":7,"question":"旧"}', question: '本次问题' })), { extra: 7, question: '本次问题' });
      assert.throws(() => dsl.prepareWorkflowRunInput({ input: '[]' }), /对象/);
      assert.throws(() => dsl.prepareWorkflowRunInput({ text: '中'.repeat(6000) }), /16000/);
    });
    await t.test('复制步骤保留参数，身份独立；完整模板连接输入与后序步骤', () => {
      const draft = dsl.workflowTemplate('text_stats', tools);
      assert.equal(draft.steps.length, 2);
      assert.doesNotThrow(() => dsl.serializeWorkflow(draft.steps, tools));
      const copy = dsl.copyWorkflowStep(draft.steps[0]);
      assert.notEqual(copy.key, draft.steps[0].key);
      assert.equal(copy.input, draft.steps[0].input);
      assert.match(copy.name, /副本/);
    });
    await t.test('存储配额失败后关闭重开仍复用内存中的原请求，身份和工作流隔离', () => {
      const memory = new Map();
      const storage = { setItem() { throw new Error('QuotaExceeded'); }, getItem() { return null; }, removeItem() {} };
      const original = { workflowId: 7, request_id: '11111111-1111-4111-8111-111111111111', input: '{"text":"原文本"}' };
      assert.throws(() => attempts.saveWorkflowRunAttempt(storage, memory, 'session-a:7', original), /Quota/);
      assert.deepEqual(attempts.readWorkflowRunAttempt(storage, memory, 'session-a:7', 7), original);
      assert.equal(attempts.readWorkflowRunAttempt(storage, memory, 'session-a:8', 8), null);
      assert.equal(attempts.readWorkflowRunAttempt(storage, memory, 'session-b:7', 7), null);
      attempts.clearWorkflowRunAttempt(storage, memory, 'session-a:7');
      assert.equal(attempts.readWorkflowRunAttempt(storage, memory, 'session-a:7', 7), null);
      assert.throws(() => attempts.readWorkflowRunAttempt({ getItem: () => '{"request_id":"bad"}' }, memory, 'bad', 7), /异常/);
    });
    await t.test('任务显示实际步骤输入、耗时与终态，恶意内容仍转义', async () => {
      const { TaskDetailContent } = await server.ssrLoadModule('/src/pages/ai/TaskDetail.tsx');
      const markup = renderToStaticMarkup(createElement(TaskDetailContent, { task: { id: 1, workflow_id: 3, status: 'cancelled', input: '{}', result: '', error: '', next_step: 0, created_at: 1, updated_at: 2, completed_at: 3, approved_by: 0,
        steps: [{ index: 0, id: 'step_a', name: '<script>恶意步骤</script>', tool: 'text_stats', input: '{"text":"本次问题"}', status: 'cancelled', output: '', error: '工具结果可能未知', started_at: 100, completed_at: 124 }] } }));
      for (const expected of ['步骤执行过程', '本次实际输入', '本次问题', '24 毫秒', '已取消', '工具结果可能未知', '&lt;script&gt;']) assert.ok(markup.includes(expected), expected);
      assert.ok(!markup.includes('<script>恶意步骤'));
    });
  } finally { await server.close(); }
});
