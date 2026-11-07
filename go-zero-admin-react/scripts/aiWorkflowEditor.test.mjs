import test from 'node:test';
import assert from 'node:assert/strict';
import { createServer } from 'vite';
import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';

test('独立工作流编辑：参数、草稿及菜单访问边界', async t => {
  const server = await createServer({ configFile: false, envFile: false, logLevel: 'error', server: { middlewareMode: true, watch: null, ws: false } });
  try {
    const dsl = await server.ssrLoadModule('/src/pages/ai/workflowDefinition.ts');
    const tool = { name: 'kb_analyze', description: '知识库分析', parameters_schema: JSON.stringify({ type: 'object', properties: { kb_id: { type: 'integer' }, query: { type: 'string' } }, required: ['kb_id', 'query'] }) };
    await t.test('知识库参数拒绝空白、非正整数和UTF8超长，并返回具体字段', () => {
      for (const kb_id of [0, -1, 1.5, '1']) assert.throws(() => dsl.parseStepInput(JSON.stringify({ kb_id, query: '分析' }), tool), /知识库|kb_id/);
      assert.throws(() => dsl.parseStepInput('{"kb_id":1,"query":"  "}', tool), /分析目标|query/);
      assert.throws(() => dsl.parseStepInput(JSON.stringify({ kb_id: 1, query: '中'.repeat(667) }), tool), /2000/);
      assert.deepEqual(dsl.parseStepInput('{"kb_id":5,"query":"总结"}', tool), { kb_id: 5, query: '总结' });
    });
    await t.test('基础字段更新保留其他参数，复杂Schema保持高级输入且不接受未知约束', () => {
      assert.deepEqual(JSON.parse(dsl.updateStepInputField('{"text":"旧","custom":{"x":1}}', 'text', '新')), { text: '新', custom: { x: 1 } });
      assert.throws(() => dsl.updateStepInputField('{oops', 'text', '新'), /JSON/);
      assert.equal(dsl.simpleToolFields(tool).length, 2);
      assert.equal(dsl.simpleToolFields({ ...tool, parameters_schema: '{"type":"object","oneOf":[]}' }), null);
      assert.throws(() => dsl.parseStepInput('{}', { ...tool, parameters_schema: '{"type":"object","oneOf":[]}' }), /Schema|支持/);
      assert.throws(() => dsl.parseStepInput('{}', { ...tool, parameters_schema: '{"type":"object","properties":{"optional":{"type":"string","pattern":"x"}}}' }), /Schema|支持/, '省略字段也不能跳过未知Schema约束');
    });
    await t.test('未知工具和未知DSL字段安全拒绝，已知旧定义往返保留参数与审批', () => {
      const raw = '{"steps":[{"name":"分析","tool":"kb_analyze","input":{"kb_id":5,"query":"总结"},"approval":true}]}';
      assert.deepEqual(JSON.parse(dsl.serializeWorkflow(dsl.parseWorkflow(raw, [tool]), [tool])), JSON.parse(raw));
      assert.throws(() => dsl.parseWorkflow(raw, []), /注册/);
      assert.throws(() => dsl.parseWorkflow(raw.replace('"approval":true', '"approval":true,"next":1'), [tool]), /未知/);
    });
    await t.test('切换工具后再切回来保留原始高级参数，UI缓存不进入服务端定义', () => {
      const original = { key: 'stable-key', name: '原步骤', tool: 'text_stats', input: '{"text":"原文","unrecognized":{"x":1}}', approval: true };
      const switched = dsl.changeWorkflowStepTool(original, 'get_current_time');
      assert.equal(switched.key, original.key);
      assert.equal(switched.input, '{}');
      const restored = dsl.changeWorkflowStepTool(switched, 'text_stats');
      assert.equal(restored.input, original.input);
      assert.equal(restored.approval, true);
      const raw = JSON.parse(dsl.serializeWorkflow([switched], [{ name: 'get_current_time', parameters_schema: '{"type":"object","properties":{}}' }]));
      assert.deepEqual(raw.steps[0], { name: '原步骤', tool: 'get_current_time', input: {}, approval: true });
      assert.throws(() => dsl.serializeWorkflow([restored], [{ name: 'text_stats', parameters_schema: '{"type":"object","properties":{"text":{"type":"string"}},"required":["text"]}' }]), /未知/);
    });
    const drafts = await server.ssrLoadModule('/src/pages/ai/workflowDraft.ts');
    await t.test('已保存定义的工具下线不能挡住有效本地草稿恢复，草稿自己的未知工具也原样保留', () => {
      const remote = { id: 7, name: '服务端分析', enabled: 1, updated_at: 9, definition: '{"steps":[{"name":"分析","tool":"kb_analyze","input":{"kb_id":1,"query":"总结"},"approval":false}]}' };
      const local = { name: '未保存文本', enabled: true, steps: [dsl.newWorkflowStep('text_stats')] };
      const tools = [{ name: 'text_stats', parameters_schema: '{"type":"object","properties":{"text":{"type":"string"}},"required":["text"]}' }];
      const record = drafts.makeWorkflowDraft(local, 'old-baseline');
      const recovered = drafts.resolveWorkflowEditorDraft(remote, record, tools, local);
      assert.equal(recovered.draft, local);
      assert.equal(recovered.baseSnapshot, 'old-baseline');
      const unavailable = { ...local, steps: [dsl.newWorkflowStep('removed_tool')] };
      assert.equal(drafts.resolveWorkflowEditorDraft(remote, drafts.makeWorkflowDraft(unavailable, 'old-baseline'), tools, local).draft, unavailable);
      assert.throws(() => drafts.resolveWorkflowEditorDraft(remote, null, tools, local), /注册/);
    });
    await t.test('本地草稿工具下线后仍显示原文，给出步骤提示并禁保存', async () => {
      const { WorkflowEditor } = await server.ssrLoadModule('/src/pages/ai/WorkflowEditor.tsx');
      const html = renderToStaticMarkup(createElement(WorkflowEditor, { initial: { name: '有效本地草稿', enabled: true, steps: [{ key: 'local-key', name: '保留的步骤', tool: 'removed_tool', input: '{"old":"原始参数"}', approval: false }] }, tools: [tool], existing: true, saving: false, onSave: async () => {}, onCancel() {} }));
      assert.match(html, /有效本地草稿/);
      assert.match(html, /原始参数/);
      assert.match(html, /第 1 步的类型已不可用/);
      const save = [...html.matchAll(/<button\b[^>]*>[\s\S]*?<\/button>/g)].find(match => match[0].includes('保存工作流'))?.[0];
      assert.match(save, /disabled/);
    });
    await t.test('草稿刷新可恢复，身份、部门和新建编辑隔离，提交中恢复为未知', () => {
      const storage = new Map();
      const adapter = { getItem: key => storage.get(key) ?? null, setItem: (key, value) => storage.set(key, value), removeItem: key => storage.delete(key) };
      const key = drafts.workflowDraftKey('session-a', 7, 'new');
      const draft = { name: '草稿', enabled: true, steps: [dsl.newWorkflowStep('kb_analyze')] };
      const record = drafts.makeWorkflowDraft(draft, 'baseline', 'submitting');
      drafts.writeWorkflowDraft(adapter, key, record);
      assert.equal(drafts.readWorkflowDraft(adapter, key).submissionState, 'unknown');
      assert.equal(drafts.readWorkflowDraft(adapter, drafts.workflowDraftKey('session-b', 7, 'new')), null);
      assert.equal(drafts.readWorkflowDraft(adapter, drafts.workflowDraftKey('session-a', 8, 'new')), null);
      assert.equal(drafts.readWorkflowDraft(adapter, drafts.workflowDraftKey('session-a', 7, 1)), null);
      assert.equal(drafts.readWorkflowDraft(adapter, key).draft.name, '草稿');
      drafts.clearWorkflowDraft(adapter, key);
      assert.equal(drafts.readWorkflowDraft(adapter, key), null);
    });
    await t.test('存储异常明确失败，不改调用者当前草稿；不兼容版本不被加载', () => {
      const draft = { name: '当前值', enabled: true, steps: [dsl.newWorkflowStep()] };
      assert.throws(() => drafts.writeWorkflowDraft({ setItem() { throw new Error('QuotaExceeded'); } }, 'key', drafts.makeWorkflowDraft(draft, null)), /Quota/);
      assert.equal(draft.name, '当前值');
      assert.equal(drafts.readWorkflowDraft({ getItem: () => '{"schemaVersion":999}' }, 'key'), null);
    });
    const routes = await server.ssrLoadModule('/src/router/menuRoutes.ts');
    await t.test('仅精确登记的新建编辑子页映射父菜单，不能放行任意前缀', () => {
      assert.equal(routes.pageMenuPath('/ai/workflows/new'), '/ai/workflows');
      assert.equal(routes.pageMenuPath('/ai/workflows/12/edit'), '/ai/workflows');
      for (const path of ['/ai/workflows/12/delete', '/ai/workflows/new/extra', '/ai/workflows/abc/edit', '/ai/workflows/0/edit']) assert.notEqual(routes.pageMenuPath(path), '/ai/workflows');
      assert.equal(routes.isPageAccessible('/ai/workflows/new', new Set(['/ai/workflows'])), true);
      assert.equal(routes.isPageAccessible('/ai/workflows/new', new Set(['/dashboard'])), false);
    });
  } finally { await server.close(); }
});
