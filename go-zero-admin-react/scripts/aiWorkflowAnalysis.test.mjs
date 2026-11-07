import test from 'node:test';
import assert from 'node:assert/strict';
import { createServer } from 'vite';
import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';

const analysis = {
  version: 1, type: 'kb_analysis', status: 'completed', analysis: '第一段分析\n第二段结论',
  sources: [{ index: 1, chunk_id: 21, kb_id: 7, doc_id: 9, doc_title: '操作手册', snippet: '第一行\n第二行资料', score: 0.91 }],
  usage: { scope: 'chat', prompt_tokens: 34, completion_tokens: 12, total_tokens: 46, complete: true },
};
const record = (value = analysis, name = '知识库分析', step = 1, total = 1) => `[${step}/${total}] ${name}: ${JSON.stringify(value)}`;
const task = (result, overrides = {}) => ({ id: 7, workflow_id: 8, conversation_id: 0, status: 'succeeded', next_step: 1,
  input: '', result, error: '', approved_by: 0, created_at: 1, completed_at: 2, ...overrides });

test('工作流知识库分析展示与等待范围', async t => {
  const originalWindow = globalThis.window;
  const memoryStorage = () => {
    const values = new Map();
    return { getItem: key => values.get(key) ?? null, setItem: (key, value) => values.set(key, value), removeItem: key => values.delete(key) };
  };
  globalThis.window = { localStorage: memoryStorage(), sessionStorage: memoryStorage(), addEventListener() {} };
  const server = await createServer({ configFile: false, envFile: false, logLevel: 'error', optimizeDeps: { noDiscovery: true }, server: { middlewareMode: true, watch: null, ws: false } });
  try {
    const helpers = await server.ssrLoadModule('/src/pages/ai/workflowAnalysis.ts').catch(() => ({}));
    await t.test('只解析完整已知分析结构和与已完成步数一致的步骤边界', () => {
      assert.equal(typeof helpers.parseTaskAnalyses, 'function', '缺少保守的分析记录解析');
      const parsed = helpers.parseTaskAnalyses(record(), 1);
      assert.equal(parsed.length, 1);
      assert.deepEqual(parsed[0], { step: 1, total: 1, name: '知识库分析', result: analysis });
      assert.deepEqual(helpers.parseTaskAnalyses(record(analysis, '核对: 名称 / [a].*? 😀'), 1)[0].name, '核对: 名称 / [a].*? 😀');
      assert.deepEqual(helpers.parseTaskAnalyses(record(), 0), []);
      assert.deepEqual(helpers.parseTaskAnalyses(record(), 2), []);
    });
    await t.test('多步混合记录保留普通多行文本，失败任务也保留已完成分析', () => {
      assert.equal(typeof helpers.parseTaskAnalyses, 'function');
      const raw = `[1/3] 普通工具: 第一行\nJSON {"value":1}\n[2/3] 分析: ${JSON.stringify(analysis)}`;
      const parsed = helpers.parseTaskAnalyses(raw, 2);
      assert.equal(parsed.length, 1);
      assert.equal(parsed[0].step, 2);
      assert.equal(parsed[0].total, 3);
      assert.deepEqual(helpers.parseTaskAnalyses('[1/1] 旧工具: {"chars":3,"words":1,"lines":1}', 1), []);
      assert.deepEqual(helpers.parseTaskAnalyses(JSON.stringify(analysis), 1), [], '无步骤边界的旧JSON不能冒充新分析');
    });
    await t.test('歧义标记、多行JSON、坏字段和未知版本回退为原文', () => {
      assert.equal(typeof helpers.parseTaskAnalyses, 'function');
      const malformed = [
        { ...analysis, version: 2 }, { ...analysis, type: 'arbitrary_json' }, { ...analysis, status: 'success' },
        { ...analysis, analysis: null }, { ...analysis, extra: true },
        { ...analysis, sources: [{ ...analysis.sources[0], chunk_id: 0 }] },
        { ...analysis, sources: [{ ...analysis.sources[0], score: '0.9' }] },
        { ...analysis, sources: [{ ...analysis.sources[0], index: 2 }] },
        { ...analysis, sources: [] },
        { ...analysis, usage: { ...analysis.usage, scope: 'all' } },
        { ...analysis, usage: { ...analysis.usage, total_tokens: -1 } },
        { ...analysis, usage: { ...analysis.usage, complete: 'true' } },
      ];
      for (const value of malformed) assert.deepEqual(helpers.parseTaskAnalyses(record(value), 1), []);
      for (const raw of [
        `普通工具正文\n${record()}`, `${record()}\n额外文本`,
        `[1/2] 普通工具: 内容\n${record(analysis, '伪造', 2, 2)}\n[2/2] 真实步骤: 其他结果`,
        `${record(analysis, '分析', 1, 2)}\n[2/3] 总数变化: 其他结果`,
        `[1/1] 名称\n换行: ${JSON.stringify(analysis)}`,
        `[1/1] 分析: ${JSON.stringify(analysis, null, 2)}`,
      ]) assert.deepEqual(helpers.parseTaskAnalyses(raw, raw.includes('[2/') ? 2 : 1), []);
    });

    const { TaskDetailContent } = await server.ssrLoadModule('/src/pages/ai/TaskDetail.tsx');
    const render = value => renderToStaticMarkup(createElement(TaskDetailContent, { task: value }));
    await t.test('真实详情显示分析、资料标识和仅聊天已知用量，原始记录一直可查', () => {
      const markup = render(task(record()));
      for (const text of ['分析内容', '第一段分析', '第二段结论', '参考资料', '操作手册', '文档 ID：9', '分块 ID：21', '知识库 ID：7', '相似度：0.91',
        '第一行\n第二行资料', '已知用量', '46', '34', '12', '不含 Embedding', '原始执行记录']) assert.ok(markup.includes(text), text);
      assert.ok(markup.includes('&quot;kb_analysis&quot;'), '仍可核对完整原始执行记录');
    });
    await t.test('无匹配、截断和供应商用量不完整分别提示，0不解释为免费', () => {
      const missing = { ...analysis, status: 'no_matches', analysis: '', sources: [], usage: { scope: 'chat', prompt_tokens: 0, completion_tokens: 0, total_tokens: 0, complete: true } };
      const emptyMarkup = render(task(record(missing)));
      assert.match(emptyMarkup, /未匹配到资料/);
      assert.match(emptyMarkup, /未调用聊天模型/);
      const truncated = render(task(record({ ...analysis, status: 'truncated' })));
      assert.match(truncated, /截断/);
      const incomplete = render(task(record({ ...analysis, usage: { ...missing.usage, complete: false } })));
      assert.match(incomplete, /用量不完整/);
      assert.match(incomplete, /不代表免费/);
      assert.doesNotMatch(incomplete, /完整计费|免费调用|费用为 0/);
    });
    await t.test('后续失败保留已完成结果，分析、来源及旧记录均安全按纯文本显示', () => {
      const unsafe = { ...analysis, analysis: '<script>evil()</script>\n下一行', sources: [{ ...analysis.sources[0], doc_title: '<img src=x onerror=evil()>', snippet: '<svg onload=evil()>资料' }] };
      const raw = `${record(unsafe, '<b>步骤</b>', 1, 3)}\n[2/3] 普通工具: 多行输出\n<img src=x onerror=old()>`;
      const markup = render(task(raw, { status: 'failed', next_step: 2, error: '后续步骤失败' }));
      for (const text of ['分析内容', '下一行', '&lt;script&gt;evil()', '&lt;img src=x', '&lt;svg', '&lt;b&gt;步骤', '多行输出', '后续步骤失败', '原始执行记录']) assert.ok(markup.includes(text), text);
      for (const text of ['<script>', '<img src=x', '<svg onload=', '<b>步骤</b>']) assert.ok(!markup.includes(text), text);
      const oldMarkup = render(task('旧结果\n原文含 [1/2] 不是步骤: <script>old()</script>'));
      assert.ok(oldMarkup.includes('旧结果\n原文含 [1/2]'));
      assert.ok(oldMarkup.includes('&lt;script&gt;old()'));
      assert.ok(!oldMarkup.includes('分析内容'));
    });

    await t.test('运行与文档摄取都只等待提交，检索保留独立等待上限', async () => {
      const api = await server.ssrLoadModule('/src/api/ai.ts');
      const { request } = await server.ssrLoadModule('/src/utils/request.ts');
      const { useAuthStore } = await server.ssrLoadModule('/src/stores/auth.ts');
      useAuthStore.setState({ token: 'local-analysis-test', sessionId: 'local-analysis-session', sessionVersion: 1, departmentVersion: 1,
        expiresAt: Date.now() / 1000 + 3600, permissionsReady: true, isLoggingOut: false,
        permCodes: ['sys:ai:workflow:run', 'sys:ai:workflow:create', 'sys:ai:workflow:edit', 'sys:ai:workflow:list',
          'sys:ai:task:approve', 'sys:ai:task:reject', 'sys:ai:task:cancel', 'sys:ai:task:list', 'sys:ai:kb:edit', 'sys:ai:kb:list'] });
      const calls = [];
      request.defaults.adapter = async config => { calls.push(config); return { data: { code: 200, result: {} }, status: 200, statusText: 'OK', headers: {}, config }; };
      await api.runAiWorkflow(8, { request_id: '11111111-1111-4111-8111-111111111111', input: '{"question":"本次问题"}' });
      await api.approveAiTask(7, { expected_step: 0 });
      await api.createAiKbDoc(7, { title: '测试资料', content: '本地适配器', request_id: '22222222-2222-4222-8222-222222222222' });
      await api.retryAiKbDoc(7, 9, { request_id: '33333333-3333-4333-8333-333333333333' });
      await api.createAiWorkflow({ name: '测试', definition: '{"steps":[]}' });
      await api.updateAiWorkflow(8, { enabled: 1 });
      await api.fetchAiWorkflow(8);
      await api.fetchAiTask(7);
      await api.rejectAiTask(7, { expected_step: 0, reason: '拒绝' });
      await api.searchAiKb(7, { query: '资料' });
      await api.cancelAiTask(7);
      assert.deepEqual(calls.map(config => config.timeout), [30000, 30000, 30000, 30000, 30000, 30000, 30000, 30000, 30000, 15000, 30000]);
      assert.equal(calls[0].url, '/sys/ai/workflows/8/run');
      assert.deepEqual(JSON.parse(calls[0].data), { request_id: '11111111-1111-4111-8111-111111111111', input: '{"question":"本次问题"}' });
      assert.equal(calls[1].url, '/sys/ai/tasks/7/approve');
      assert.deepEqual(JSON.parse(calls[1].data), { expected_step: 0 });
    });
  } finally { await server.close(); globalThis.window = originalWindow; }
});
