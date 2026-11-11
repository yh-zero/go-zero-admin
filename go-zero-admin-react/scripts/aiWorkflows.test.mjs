import test from 'node:test';
import assert from 'node:assert/strict';
import { existsSync } from 'node:fs';
import { createServer } from 'vite';
import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';

const tools = [
  { name: 'text_stats', description: '文本统计', parameters_schema: '{"type":"object","properties":{"text":{"type":"string"}},"required":["text"]}' },
  { name: 'get_current_time', description: '当前时间', parameters_schema: '{"type":"object","properties":{}}' },
];
const step = { key: 'local-only', name: '复核', tool: 'text_stats', input: '{"text":"原文"}', approval: true };

test('工作流 DSL 与只读呈现边界', async t => {
  assert.ok(existsSync(new URL('../src/pages/ai/workflowDefinition.ts', import.meta.url)), '尚未实现工作流 DSL 校验');
  const server = await createServer({ configFile: false, envFile: false, logLevel: 'error', server: { middlewareMode: true, watch: null, ws: false } });
  try {
    const dsl = await server.ssrLoadModule('/src/pages/ai/workflowDefinition.ts');
    await t.test('同一抽屉只采纳首次成功的草稿种子，后台重查与失败不替换它', () => {
      assert.equal(typeof dsl.retainWorkflowDraftSeed, 'function', '尚未实现按抽屉保留编辑草稿');
      const original = { name: '未保存名称', enabled: true, steps: [step] };
      const first = dsl.retainWorkflowDraftSeed(null, 'panel-one', original);
      assert.equal(first.draft, original);
      assert.equal(dsl.retainWorkflowDraftSeed(first, 'panel-one', undefined), first, '后台重查没有可用定义时仍保留草稿');
      const changedRemote = { name: '远端新名称', enabled: false, steps: [{ ...step, input: '{"text":"远端参数"}' }] };
      assert.equal(dsl.retainWorkflowDraftSeed(first, 'panel-one', changedRemote), first, '服务端或工具刷新不能重建当前表单');
      assert.equal(dsl.retainWorkflowDraftSeed(first, 'panel-two', undefined), null, '新抽屉不能复用旧草稿');
      const reopened = dsl.retainWorkflowDraftSeed(first, 'panel-two', changedRemote);
      assert.equal(reopened.key, 'panel-two');
      assert.equal(reopened.draft, changedRemote, '重新打开时才采用最新已保存定义');
    });
    await t.test('本地步骤标识不进入 DSL，审批和固定参数原样保留', () => {
      assert.equal(dsl.serializeWorkflow([step], tools), '{"steps":[{"name":"复核","tool":"text_stats","input":{"text":"原文"},"approval":true}]}');
      const decoded = dsl.parseWorkflow(dsl.serializeWorkflow([step], tools), tools);
      assert.deepEqual(decoded.map(({ key, ...rest }) => { assert.ok(key); return rest; }), [{ name: '复核', tool: 'text_stats', input: '{\n  "text": "原文"\n}', approval: true }]);
      assert.notEqual(dsl.newWorkflowStep('text_stats').key, dsl.newWorkflowStep('text_stats').key);
    });
    await t.test('工作流与步骤名称按 Unicode 字符计数', () => {
      assert.equal(dsl.validateWorkflowName('😀'.repeat(128)), undefined);
      assert.match(dsl.validateWorkflowName('😀'.repeat(129)), /128/);
      assert.match(dsl.validateWorkflowName(' \n'), /名称/);
      assert.throws(() => dsl.serializeWorkflow([{ ...step, name: '😀'.repeat(129) }], tools), /128/);
    });
    await t.test('顺序步骤的上下移动保持稳定 key，限制为 1..32 步', () => {
      const steps = [step, { ...step, key: 'second', name: '第二步' }];
      assert.deepEqual(dsl.moveWorkflowStep(steps, 'second', -1).map(item => item.key), ['second', 'local-only']);
      assert.deepEqual(steps.map(item => item.key), ['local-only', 'second']);
      assert.deepEqual(dsl.moveWorkflowStep(steps, 'local-only', -1), steps);
      assert.throws(() => dsl.serializeWorkflow([], tools), /1..32/);
      assert.throws(() => dsl.serializeWorkflow(Array(33).fill(step), tools), /1..32/);
      assert.equal(JSON.parse(dsl.serializeWorkflow(Array(32).fill(step), tools)).steps.length, 32);
    });
    await t.test('拒绝 DSL 未知字段、未注册工具及非对象参数', () => {
      const raw = JSON.parse(dsl.serializeWorkflow([step], tools));
      assert.throws(() => dsl.parseWorkflow(JSON.stringify({ ...raw, parallel: true }), tools), /未知字段/);
      assert.throws(() => dsl.parseWorkflow(JSON.stringify({ steps: [{ ...raw.steps[0], next: 'other' }] }), tools), /未知字段/);
      for (const input of ['null', '[]', '"x"', 'false', '{oops']) assert.throws(() => dsl.serializeWorkflow([{ ...step, input }], tools), /JSON 对象/);
      assert.throws(() => dsl.serializeWorkflow([{ ...step, tool: 'unregistered' }], tools), /注册/);
      assert.throws(() => dsl.serializeWorkflow([{ ...step, input: '{"text":"ok","unknown":true}' }], tools), /未知字段/);
      assert.throws(() => dsl.serializeWorkflow([{ ...step, tool: 'get_current_time', input: '{"text":"x"}' }], tools), /未知字段/);
      assert.throws(() => dsl.serializeWorkflow([{ ...step, input: '{}' }], tools), /text/);
    });
    await t.test('text、definition 与运行输入执行 UTF8 字节边界', () => {
      dsl.serializeWorkflow([{ ...step, input: JSON.stringify({ text: '中'.repeat(5333) + 'a' }) }], tools);
      assert.throws(() => dsl.serializeWorkflow([{ ...step, input: JSON.stringify({ text: '中'.repeat(5334) }) }], tools), /16000/);
      assert.throws(() => dsl.serializeWorkflow(Array(5).fill({ ...step, input: JSON.stringify({ text: 'a'.repeat(16000) }) }), tools), /65536/);
      assert.throws(() => dsl.parseWorkflow(' '.repeat(65537)), /65536/);
      assert.equal(dsl.validateRunInput('{"text":"' + '中'.repeat(5329) + 'aa"}'), undefined);
      assert.match(dsl.validateRunInput('"' + '中'.repeat(5333) + '"'), /16000/);
      assert.equal(dsl.validateRunInput(''), undefined);
      assert.match(dsl.validateRunInput('[1,true,null]'), /对象/);
      assert.match(dsl.validateRunInput('{oops'), /JSON/);
    });
    await t.test('无工具读取权限时已存定义仍可只读解析', () => {
      const raw = '{"steps":[{"name":"历史步骤","tool":"legacy_tool","input":{},"approval":false}]}';
      assert.equal(dsl.parseWorkflow(raw)[0].tool, 'legacy_tool');
      assert.throws(() => dsl.parseWorkflow(raw, tools), /注册/);
    });
    await t.test('只读工作流名称与原始定义保持文本转义', async () => {
      const { WorkflowDefinitionView } = await server.ssrLoadModule('/src/pages/ai/WorkflowReadOnly.tsx');
      const html = renderToStaticMarkup(createElement(WorkflowDefinitionView, { definition: '<img src=x onerror=alert(1)>', name: '<script>bad()</script>' }));
      assert.ok(!html.includes('<img src=x'));
      assert.ok(!html.includes('<script>bad()'));
      assert.ok(html.includes('&lt;img'));
      assert.ok(html.includes('&lt;script&gt;'));
    });
    await t.test('后台核对期间保留可编辑草稿，只禁用保存入口', async () => {
      const { WorkflowEditor } = await server.ssrLoadModule('/src/pages/ai/WorkflowEditor.tsx');
      const html = renderToStaticMarkup(createElement(WorkflowEditor, { initial: { name: '未保存草稿', enabled: true, steps: [step] }, tools, existing: true, saving: false, saveDisabled: true, onSave: async () => {}, onCancel() {} }));
      assert.ok(html.includes('value="未保存草稿"'));
      assert.ok(html.includes('原文'));
      const saveButton = [...html.matchAll(/<button\b[^>]*>[\s\S]*?<\/button>/g)].find(match => match[0].includes('保存为常用流程'))?.[0];
      assert.ok(saveButton);
      assert.match(saveButton, /disabled/);
    });
    await t.test('运行成功但无任务查看权限时显示任务标识与用户状态，不产生无权链接', async () => {
      const { WorkflowRunResultNotice } = await server.ssrLoadModule('/src/pages/ai/WorkflowReadOnly.tsx');
      assert.equal(typeof WorkflowRunResultNotice, 'function', '尚未提供独立任务权限下的运行结果反馈');
      const html = renderToStaticMarkup(createElement(WorkflowRunResultNotice, { result: { task_id: 246, status: 'running' } }));
      assert.match(html, /246/);
      assert.match(html, /执行中/);
      assert.match(html, /管理员/);
      assert.ok(!html.includes('/ai/tasks'));
    });
  } finally { await server.close(); }
});
