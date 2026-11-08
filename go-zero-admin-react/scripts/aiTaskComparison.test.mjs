import test from 'node:test';
import assert from 'node:assert/strict';
import { createServer } from 'vite';
import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';

const step = (id, index, extra = {}) => ({ id, index, name: `步骤${id}`, tool: 'text_stats', status: 'succeeded', input: '{"text":"原文"}', output: '{"characters":2}', error: '', started_at: 100, completed_at: 124, ...extra });
const task = (id, steps, extra = {}) => ({ id, workflow_id: 7, conversation_id: 0, status: 'succeeded', input: '{"question":"本次问题"}', result: '', error: '', approved_by: 0,
  created_at: 1, updated_at: 3, completed_at: 3, next_step: steps.length, steps, request_id: `fixture-${id}`, title: `运行${id}`, workflow_name: '原流程快照', input_summary: '本次问题',
  definition: JSON.stringify({ version: 2, steps: steps.map(item => ({ id: item.id, name: item.name, tool: item.tool, input: {}, approval: false })) }), ...extra });

test('普通运行对照按实际历史记录配对并安全展示', async t => {
  const server = await createServer({ configFile: false, envFile: false, logLevel: 'error', server: { middlewareMode: true, watch: null, ws: false } });
  try {
    const comparison = await server.ssrLoadModule('/src/pages/ai/taskComparisonData.ts');
    await t.test('同流程v2稳定ID在步骤重排后仍配对同一步', () => {
      const paired = comparison.pairTaskSteps(task(1, [step('a', 0), step('b', 1)]), task(2, [step('b', 0), step('a', 1, { input: '{"text":"新输入"}' })]));
      assert.equal(paired.mode, 'stable_id');
      assert.deepEqual(paired.rows.map(row => [row.left?.id, row.right?.id]), [['a', 'a'], ['b', 'b']]);
      assert.deepEqual(paired.rows.map(row => row.moved), [true, true]);
      assert.deepEqual(paired.rows.map(row => row.inputChanged), [true, false]);
    });
    await t.test('新增或缺失记录保留单侧位置，不把未执行步骤补造为结果', () => {
      const paired = comparison.pairTaskSteps(task(1, [step('a', 0), step('b', 1)]), task(2, [step('b', 0), step('c', 1)]));
      assert.deepEqual(paired.rows.map(row => [row.left?.id, row.right?.id]), [['a', undefined], ['b', 'b'], [undefined, 'c']]);
      assert.equal(new Set(paired.rows.map(row => row.key)).size, 3);
    });
    await t.test('旧DSL自动补的ID和不同流程不能冒充稳定身份', () => {
      const left = task(1, [step('a', 0), step('b', 1)]);
      const reordered = task(2, [step('b', 0), step('a', 1)]);
      const legacy = comparison.pairTaskSteps({ ...left, definition: '{"steps":[]}' }, reordered);
      assert.equal(legacy.mode, 'order');
      assert.match(legacy.explanation, /旧|快照/);
      assert.deepEqual(legacy.rows.map(row => [row.left?.id, row.right?.id]), [['a', 'b'], ['b', 'a']]);
      assert.equal(comparison.pairTaskSteps(left, { ...reordered, workflow_id: 8 }).mode, 'order');
      assert.equal(comparison.pairTaskSteps({ ...left, workflow_id: 0 }, { ...reordered, workflow_id: 0 }).mode, 'order');
      assert.equal(comparison.pairTaskSteps(left, { ...reordered, definition: '' }).mode, 'order');
    });
    await t.test('缺失、倒序或非法计时标未记录，有效同一时刻确实为0', () => {
      assert.equal(comparison.recordedElapsed(0, 124), null);
      assert.equal(comparison.recordedElapsed(100, 99), null);
      assert.equal(comparison.recordedElapsed(100, Infinity), null);
      assert.equal(comparison.recordedElapsed(100, 100), 0);
      assert.equal(comparison.recordedElapsed(100, 124), 24);
      assert.equal(comparison.isTerminalTask('pending_approval'), false);
      for (const status of ['succeeded', 'failed', 'rejected', 'cancelled']) assert.equal(comparison.isTerminalTask(status), true);
    });
    const { TaskComparisonContent } = await server.ssrLoadModule('/src/pages/ai/TaskComparison.tsx');
    await t.test('实际结果和来源沿用结构化安全展示，文字变化不声称质量提升', () => {
      const analysis = { version: 1, type: 'kb_analysis', status: 'completed', analysis: '带来源的分析[1]',
        sources: [{ index: 1, chunk_id: 11, kb_id: 7, doc_id: 9, doc_title: '历史原文甲', snippet: '真实历史片段', score: 0.8 }],
        usage: { scope: 'chat', prompt_tokens: 10, completion_tokens: 5, total_tokens: 15, complete: true } };
      const left = task(1, [step('analysis', 0, { name: '<script>恶意步骤</script>', tool: 'kb_analyze', output: JSON.stringify(analysis), started_at: 0 })], { result: `[1/1] 分析: ${JSON.stringify(analysis)}` });
      const right = task(2, [step('analysis', 0, { tool: 'kb_analyze', output: '其他结果' })], { result: '不同原文', input: '<img src=x onerror=alert(1)>' });
      const html = renderToStaticMarkup(createElement(TaskComparisonContent, { left, right }));
      for (const expected of ['带来源的分析', '历史原文甲', '真实历史片段', '未记录', '输出文字有差异', '差异不代表质量提升', '&lt;script&gt;']) assert.ok(html.includes(expected), expected);
      assert.ok(html.includes('用量：未记录'));
      assert.ok(!html.includes('<script>恶意步骤'));
      assert.ok(!html.includes('<img src=x'));
      assert.ok(html.includes('/ai/knowledge/7?doc_id=9'));
    });
    await t.test('非终态或相同任务不展示对照结果，旧任务无步骤仍可读', () => {
      const old = task(1, [], { definition: '', result: '旧版运行原文' });
      const html = renderToStaticMarkup(createElement(TaskComparisonContent, { left: old, right: task(2, [], { definition: '' }) }));
      assert.ok(html.includes('旧版运行原文'));
      assert.ok(html.includes('未记录步骤'));
      const blocked = renderToStaticMarkup(createElement(TaskComparisonContent, { left: old, right: { ...old, status: 'running' } }));
      assert.ok(blocked.includes('仅可对照两条不同的终态运行'));
      assert.ok(!blocked.includes('旧版运行原文'));
    });
  } finally { await server.close(); }
});
