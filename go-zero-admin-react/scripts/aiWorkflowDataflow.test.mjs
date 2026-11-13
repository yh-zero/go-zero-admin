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
    const knowledgeTools = [...tools, { name: 'kb_analyze', parameters_schema: JSON.stringify({ type: 'object', properties: { kb_id: { type: 'integer', minimum: 1 }, query: { type: 'string', minLength: 1 } }, required: ['kb_id', 'query'] }) }];
    await t.test('默认分析仅执行有实际用途的一步，并在每次运行选择知识库和问题', () => {
      const draft = dsl.workflowTemplate('kb_analyze', knowledgeTools);
      assert.equal(draft.steps.length, 1);
      assert.deepEqual(JSON.parse(draft.steps[0].input), { kb_id: { $ref: 'input.kb_id' }, query: { $ref: 'input.question' } });
      const definition = dsl.serializeWorkflow(draft.steps, knowledgeTools);
      assert.deepEqual(dsl.workflowRunFields(definition), { question: true, text: false, kb_id: true });
    });
    await t.test('JSON起步模板按格式整理命名，并明确使用本次文本', () => {
      const draft = dsl.workflowTemplate('json_formatter', tools);
      assert.match(draft.name, /JSON/);
      assert.deepEqual(JSON.parse(draft.steps[0].input), { text: { $ref: 'input.text' } });
      assert.doesNotThrow(() => dsl.serializeWorkflow(draft.steps, tools));
    });
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
      assert.deepEqual(JSON.parse(dsl.prepareWorkflowRunInput({ input: '{"kb_id":2,"extra":7}', kb_id: 13, question: '本次分析' })), { kb_id: 13, extra: 7, question: '本次分析' });
      for (const kb_id of [0, -1, 1.5, Number.MAX_SAFE_INTEGER + 1]) assert.throws(() => dsl.prepareWorkflowRunInput({ kb_id }), /知识库/);
      assert.throws(() => dsl.prepareWorkflowRunInput({ kb_id: 1, question: '中'.repeat(6000) }), /16000/);
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
    async function renderStep(step) {
      const { TaskDetailContent } = await server.ssrLoadModule('/src/pages/ai/TaskDetail.tsx');
      return renderToStaticMarkup(createElement(TaskDetailContent, { task: {
        id: 1, workflow_id: 3, conversation_id: 0, status: step.status ?? 'failed', input: '{}', result: '',
        error: '模型答案未通过安全校验', next_step: 0, created_at: 1, updated_at: 2, completed_at: 3, approved_by: 0,
        request_id: '11111111-1111-4111-8111-111111111111', title: '本次分析', workflow_name: '资料分析', input_summary: '本次问题', definition: '{}',
        steps: [{ index: 0, id: 'step_analysis', name: '分析资料', tool: 'kb_analyze', input: '{"query":"本次问题"}',
          status: 'failed', output: '', error: '模型答案未通过安全校验', started_at: 100, completed_at: 124, ...step }],
      } }));
    }
    await t.test('失败步骤仍展示已知聊天用量，不把计量变成已接受的分析答案', async () => {
      const markup = await renderStep({ usage: { scope: 'chat', prompt_tokens: 31, completion_tokens: 7, total_tokens: 38, complete: false } });
      for (const expected of ['已知聊天用量', '输入 31', '输出 7', '合计 38 tokens', '用量不完整（实际费用可能更高）', '模型答案未通过安全校验', '执行失败']) assert.ok(markup.includes(expected), expected);
      assert.ok(!markup.includes('分析内容'));
    });
    await t.test('零值且用量不完整时仍明确提示可能有更多费用', async () => {
      const markup = await renderStep({ usage: { scope: 'chat', prompt_tokens: 0, completion_tokens: 0, total_tokens: 0, complete: false } });
      assert.ok(markup.includes('合计 0 tokens'));
      assert.ok(markup.includes('用量不完整（实际费用可能更高）'));
    });
    await t.test('分项与合计一致的完整聊天用量正常展示，旧步骤缺字段保持可读', async () => {
      const complete = await renderStep({ status: 'succeeded', error: '', usage: { scope: 'chat', prompt_tokens: 10, completion_tokens: 5, total_tokens: 15, complete: true } });
      for (const expected of ['已知聊天用量', '输入 10', '输出 5', '合计 15 tokens', '已完成']) assert.ok(complete.includes(expected), expected);
      assert.ok(!complete.includes('用量不完整（实际费用可能更高）'));
      const legacy = await renderStep({});
      assert.ok(legacy.includes('分析资料'));
      assert.ok(legacy.includes('本次实际输入'));
      assert.ok(!legacy.includes('已知聊天用量'));
      assert.ok(!legacy.includes('合计 0 tokens'));
    });
    await t.test('其他scope或非法数值不能冒充聊天用量，合计不一致不能标完整', async () => {
      for (const usage of [
        { scope: 'embedding', prompt_tokens: 10, completion_tokens: 5, total_tokens: 15, complete: true },
        { scope: 'chat', prompt_tokens: -1, completion_tokens: 5, total_tokens: 4, complete: false },
        { scope: 'chat', prompt_tokens: 10, completion_tokens: NaN, total_tokens: 10, complete: false },
      ]) assert.ok(!(await renderStep({ usage })).includes('已知聊天用量'));
      const inconsistent = await renderStep({ usage: { scope: 'chat', prompt_tokens: 10, completion_tokens: 5, total_tokens: 20, complete: true } });
      assert.ok(inconsistent.includes('合计 20 tokens'));
      assert.ok(inconsistent.includes('用量不完整（实际费用可能更高）'));
    });
  } finally { await server.close(); }
});
