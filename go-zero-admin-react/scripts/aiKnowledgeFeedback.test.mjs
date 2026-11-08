import test from 'node:test';
import assert from 'node:assert/strict';
import { createServer } from 'vite';
import { MutationObserver, QueryClient } from '@tanstack/react-query';
import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';

const snapshot = {
  version: 1, kb_id: 7, query: '如何发布？', step_input: '{"kb_id":7,"query":"如何发布？"}', step_output: '{}',
  sources: [{ index: 1, kb_id: 7, doc_id: 9, chunk_id: 11, doc_title: '发布操作', snippet: '发布前必须备份数据库。验收通过后才可切换流量。', score: 0.9 }],
  documents: [{ doc_id: 9, kb_id: 7, title: '发布操作', version: 2, content_sha256: 'a'.repeat(64), status: 'verified' }],
};
const record = { id: 3, kb_id: 7, task_id: 8, step_index: 0, feedback: 'incorrect_answer', note: '遗漏了备份', status: 'accepted', revision: 2,
  question: '如何发布？', answer: '先发布', snapshot: JSON.stringify(snapshot), review: { answerable: true, expected_answer: '先备份数据库', required_facts: ['备份数据库'],
    evidence: [{ source_index: 1, quote: '发布前必须备份数据库。' }], reason: '依据原文修正' }, created_at: 1, updated_at: 2, reviewed_at: 2 };

function storage() { const values = new Map(); return { getItem: key => values.get(key) ?? null, setItem: (key, value) => values.set(key, value), removeItem: key => values.delete(key) }; }

test('知识库分析反馈与复核边界', async t => {
  const originalWindow = globalThis.window;
  globalThis.window = { localStorage: storage(), sessionStorage: storage(), addEventListener() {} };
  const server = await createServer({ configFile: false, envFile: false, logLevel: 'error', server: { middlewareMode: true, watch: null, ws: false } });
  try {
    const helpers = await server.ssrLoadModule('/src/pages/ai/knowledgeFeedback.ts').catch(() => ({}));
    await t.test('8来源快照与第6至8依据合法，第9来源或6条摘录拒绝', async () => {
      const expanded = count => ({ ...snapshot,
        sources: Array.from({ length: count }, (_, i) => ({ ...snapshot.sources[0], index: i + 1, chunk_id: 11 + i, doc_id: 9 + i, doc_title: `历史资料${i + 1}`, snippet: `事实${i + 1}` })),
        documents: Array.from({ length: count }, (_, i) => ({ ...snapshot.documents[0], doc_id: 9 + i, title: `历史资料${i + 1}` })),
      });
      for (const count of [5, 8]) assert.equal(helpers.parseFeedbackSnapshot(JSON.stringify(expanded(count))).sources.length, count);
      assert.equal(helpers.parseFeedbackSnapshot(JSON.stringify(expanded(9))), null);
      assert.equal(helpers.parseFeedbackSnapshot(JSON.stringify(snapshot) + ' '.repeat(256 * 1024)), null);
      const current = expanded(8), evidence = [6, 7, 8].map(index => ({ source_index: index, quote: `事实${index}` }));
      const values = { status: 'accepted', answerable: true, expected_answer: '事实6、事实7、事实8', required_facts: '事实6\n事实7\n事实8', evidence, reason: '核对第6至8历史来源' };
      assert.deepEqual(helpers.reviewValuesToRequest(values, current, 2).review.evidence, evidence);
      assert.throws(() => helpers.reviewValuesToRequest({ ...values, expected_answer: '事实9', required_facts: '事实9', evidence: [{ source_index: 9, quote: '事实9' }] }, expanded(9), 2));
      assert.throws(() => helpers.reviewValuesToRequest({ ...values, evidence: Array.from({ length: 6 }, (_, i) => ({ source_index: i + 1, quote: `事实${i + 1}` })) }, current, 2));
      const { FeedbackSourceSnapshot } = await server.ssrLoadModule('/src/pages/ai/KnowledgeFeedbackPanel.tsx');
      const html = renderToStaticMarkup(createElement(FeedbackSourceSnapshot, { snapshot: current }));
      for (const index of [6, 7, 8]) assert.ok(html.includes(`历史资料${index}`) && html.includes(`事实${index}`));
    });
    await t.test('8来源在JSON字节边界的快照与导出仍落在256和512KiB总门禁内', async step => {
      const sourceList = Array.from({ length: 8 }, (_, i) => ({ ...snapshot.sources[0], index: i + 1, chunk_id: 11 + i, doc_id: 9 + i, doc_title: '🧭'.repeat(64), snippet: `事实${i + 1}` + '\\'.repeat(200) }));
      for (const source of sourceList) {
        const room = Math.floor((10000 - Buffer.byteLength(JSON.stringify(sourceList))) / 2);
        source.snippet += '\\'.repeat(Math.min(500 - Array.from(source.snippet).length, Math.max(0, room)));
      }
      const gap = 10000 - Buffer.byteLength(JSON.stringify(sourceList));
      sourceList.at(-1).snippet += 'x'.repeat(gap);
      assert.equal(Buffer.byteLength(JSON.stringify(sourceList)), 10000);
      const output = { version: 1, type: 'kb_analysis', status: 'completed', analysis: 'x'.repeat(9000) + '\n'.repeat(1999), sources: sourceList, usage: { scope: 'chat', prompt_tokens: 2, completion_tokens: 1, total_tokens: 3, complete: true } };
      assert.equal(Buffer.byteLength(output.analysis), 10999);
      assert.equal(Buffer.byteLength(JSON.stringify(output.analysis)), 13000);
      const outputBytes = Buffer.byteLength(JSON.stringify(output));
      assert.ok(outputBytes > 23000 && outputBytes <= 24000);
      const query = '\0'.repeat(1999) + 'Q';
      const bounded = { ...snapshot, query, step_input: JSON.stringify({ kb_id: 7, query }), step_output: JSON.stringify(output), sources: sourceList,
        documents: sourceList.map(source => ({ ...snapshot.documents[0], doc_id: source.doc_id, title: '🧭'.repeat(128) })) };
      assert.ok(Buffer.byteLength(bounded.step_input) <= 16000);
      assert.ok(sourceList.every(source => Array.from(source.snippet).length <= 500));
      const raw = JSON.stringify(bounded);
      assert.ok(Buffer.byteLength(raw) < 256 * 1024);
      assert.equal(helpers.parseFeedbackSnapshot(raw).sources.length, 8);
      const accepted = { ...record, question: query, answer: output.analysis, snapshot: raw, review: { answerable: true, expected_answer: '事实8', required_facts: ['事实8'], evidence: [{ source_index: 8, quote: '事实8' }], reason: '仅测试合法字节边界，不代表真实语义答案' } };
      const exported = helpers.feedbackRegressionJson(accepted);
      assert.ok(Buffer.byteLength(exported) < 512 * 1024);
      assert.throws(() => helpers.feedbackRegressionJson({ ...accepted, untrusted_extra: 'x'.repeat(512 * 1024) }), /512/);
      step.diagnostic(JSON.stringify({ analysis_raw_bytes: 10999, analysis_json_bytes: 13000, sources_json_bytes: 10000, output_json_bytes: outputBytes, snapshot_bytes: Buffer.byteLength(raw), export_bytes: Buffer.byteLength(exported) }));
    });
    await t.test('评价备注按UTF-8字节校验，历史快照保留原片段和版本', () => {
      assert.equal(typeof helpers.feedbackNoteError, 'function');
      assert.equal(helpers.feedbackNoteError('中'.repeat(333) + 'a'), undefined);
      assert.ok(helpers.feedbackNoteError('中'.repeat(334)));
      assert.deepEqual(helpers.parseFeedbackSnapshot(JSON.stringify(snapshot)), snapshot);
      for (const value of ['', '{}', JSON.stringify({ ...snapshot, version: 2 }), JSON.stringify({ ...snapshot, kb_id: 0 }),
        JSON.stringify({ ...snapshot, sources: [{ ...snapshot.sources[0], index: 0 }] }),
        JSON.stringify({ ...snapshot, documents: [{ ...snapshot.documents[0], status: 'current' }] })]) assert.equal(helpers.parseFeedbackSnapshot(value), null);
    });
    await t.test('采纳必须人工填写预期答案、检查项和逐字历史摘录；不足题不带伪造依据', () => {
      assert.equal(typeof helpers.reviewValuesToRequest, 'function');
      const values = { status: 'accepted', answerable: true, expected_answer: '先备份数据库，验收通过后切换流量', required_facts: '备份数据库\n切换流量',
        evidence: [{ source_index: 1, quote: snapshot.sources[0].snippet }], reason: '依据原文修正' };
      const request = helpers.reviewValuesToRequest(values, snapshot, 2);
      assert.deepEqual(request, { expected_revision: 2, status: 'accepted', review: { ...record.review, expected_answer: values.expected_answer,
        required_facts: ['备份数据库', '切换流量'], evidence: values.evidence } });
      for (const override of [{ expected_answer: '' }, { required_facts: '' }, { reason: ' ' }, { evidence: [] },
        { evidence: [{ source_index: 1, quote: '并不存在的摘录' }] }, { evidence: [{ source_index: 2, quote: '发布前' }] },
        { expected_answer: '中'.repeat(1334) }, { required_facts: Array.from({ length: 11 }, (_, i) => `事实${i}`).join('\n') }])
        assert.throws(() => helpers.reviewValuesToRequest({ ...values, ...override }, snapshot, 2));
      assert.deepEqual(helpers.reviewValuesToRequest({ ...values, answerable: false, required_facts: '', evidence: [], expected_answer: '资料不足，无法回答' }, snapshot, 2).review,
        { answerable: false, expected_answer: '资料不足，无法回答', required_facts: [], evidence: [], reason: '依据原文修正' });
      assert.throws(() => helpers.reviewValuesToRequest({ ...values, answerable: false }, snapshot, 2), /不足/);
      assert.deepEqual(helpers.reviewValuesToRequest({ status: 'dismissed', reason: '重复反馈' }, snapshot, 2), { expected_revision: 2, status: 'dismissed',
        review: { answerable: false, expected_answer: '', required_facts: [], evidence: [], reason: '重复反馈' } });
    });
    await t.test('页面采纳和离线合并使用相同事实/摘录归一规则，拒绝遗漏和重复项', () => {
      const values = { status: 'accepted', answerable: true, expected_answer: '先备份数据库，再切换流量', required_facts: '备份数据库\n切换流量',
        evidence: [{ source_index: 1, quote: snapshot.sources[0].snippet }], reason: '核对两项事实' };
      for (const override of [{ expected_answer: '先备份数据库' }, { evidence: [{ source_index: 1, quote: '发布前必须备份数据库。' }] },
        { required_facts: '备份数据库\n备份 数据库' }, { evidence: [values.evidence[0], values.evidence[0]] }])
        assert.throws(() => helpers.reviewValuesToRequest({ ...values, ...override }, snapshot, 2), /关键词|重复/);
      const numericSnapshot = { ...snapshot, sources: [{ ...snapshot.sources[0], snippet: '每批最多20 篇资料。' }] };
      assert.deepEqual(helpers.reviewValuesToRequest({ ...values, expected_answer: '每批最多20篇', required_facts: '２０ 篇',
        evidence: [{ source_index: 1, quote: '每批最多20 篇资料。' }] }, numericSnapshot, 2).review.required_facts, ['２０ 篇']);
      assert.throws(() => helpers.feedbackRegressionJson({ ...record, review: { ...record.review, expected_answer: values.expected_answer,
        required_facts: ['备份数据库', '切换流量'] } }), /关键词/);
    });
    await t.test('只导出已采纳完整原记录，不把待复核或摘要当回归样本', () => {
      assert.equal(typeof helpers.feedbackRegressionJson, 'function');
      assert.deepEqual(JSON.parse(helpers.feedbackRegressionJson(record)), { version: 1, type: 'kb_feedback_regression', feedback: record });
      for (const value of [{ ...record, status: 'pending' }, { ...record, review: null }, { ...record, snapshot: '' },
        { ...record, kb_id: 99 }, { ...record, review: { ...record.review, evidence: [{ source_index: 1, quote: '伪造' }] } }]) assert.throws(() => helpers.feedbackRegressionJson(value));
    });

    const api = await server.ssrLoadModule('/src/api/ai.ts');
    const { request, setRequestErrorReporter } = await server.ssrLoadModule('/src/utils/request.ts');
    const { useAuthStore } = await server.ssrLoadModule('/src/stores/auth.ts');
    const scope = ['local-kb-feedback-session', 1, 1];
    const authorize = () => useAuthStore.setState({ token: 'local-kb-feedback-test', sessionId: scope[0], sessionVersion: 1, departmentVersion: 1,
      expiresAt: Date.now() / 1000 + 3600, permissionsReady: true, isLoggingOut: false,
      permCodes: ['sys:ai:task:list', 'sys:ai:task:feedback', 'sys:ai:kb:list', 'sys:ai:kb:review'] });
    const response = (config, result, code = 200) => ({ data: { code, result, msg: code === 409 ? '记录已更新' : '' }, status: 200, statusText: 'OK', headers: {}, config });
    authorize();
    await t.test('五个统一API保持0起步骤、revision和过滤分页，不发送模型请求', async () => {
      for (const name of ['fetchAiTaskFeedback', 'setAiTaskFeedback', 'fetchAiKbFeedbackList', 'fetchAiKbFeedback', 'reviewAiKbFeedback']) assert.equal(typeof api[name], 'function', name);
      const calls = [];
      request.defaults.adapter = async config => { calls.push(config); return response(config, { feedback: record, list: [record], total: 1 }); };
      await api.fetchAiTaskFeedback(8);
      await api.setAiTaskFeedback(8, { step_index: 0, expected_revision: 0, feedback: 'incorrect_answer', note: '问题' });
      await api.fetchAiKbFeedbackList(7, { page: 2, page_size: 10, status: 'accepted' });
      await api.fetchAiKbFeedback(7, 3);
      await api.reviewAiKbFeedback(7, 3, { expected_revision: 2, status: 'dismissed', review: { answerable: false, expected_answer: '', required_facts: [], evidence: [], reason: '重复' } });
      assert.deepEqual(calls.map(call => `${call.method} ${call.url}`), ['get /sys/ai/tasks/8/feedback', 'put /sys/ai/tasks/8/feedback',
        'get /sys/ai/kbs/7/feedback', 'get /sys/ai/kbs/7/feedback/3', 'put /sys/ai/kbs/7/feedback/3/review']);
      assert.equal(JSON.parse(calls[1].data).step_index, 0);
      assert.deepEqual(calls[2].params, { page: 2, page_size: 10, status: 'accepted' });
      assert.ok(calls.every(call => call.headers.Authorization === 'Bearer local-kb-feedback-test'));
    });
    await t.test('反馈Mutation不重试409，切换身份和撤销权限阻止迟到结果', async () => {
      assert.equal(typeof helpers.taskFeedbackMutationOptions, 'function');
      assert.equal(typeof helpers.kbReviewMutationOptions, 'function');
      const client = new QueryClient();
      let calls = 0;
      setRequestErrorReporter(() => {});
      try {
        authorize();
        request.defaults.adapter = async config => { calls++; return response(config, {}, 409); };
        const mutation = new MutationObserver(client, helpers.taskFeedbackMutationOptions(scope, 8));
        await assert.rejects(mutation.mutate({ step_index: 0, expected_revision: 1, feedback: 'incorrect_answer', note: '草稿' }), error => error.code === 409);
        assert.equal(calls, 1);
        useAuthStore.setState({ permCodes: ['sys:ai:task:list', 'sys:ai:kb:list'] });
        await assert.rejects(mutation.mutate({ step_index: 0, expected_revision: 1, feedback: 'helpful', note: '' }), /权限|工作区/);
        assert.equal(calls, 1);
        authorize();
        request.defaults.adapter = async config => { calls++; useAuthStore.setState({ departmentVersion: 2 }); return response(config, { feedback: record }); };
        await assert.rejects(mutation.mutate({ step_index: 0, expected_revision: 1, feedback: 'helpful', note: '' }), /工作区|部门已切换/);
        authorize();
        const review = new MutationObserver(client, helpers.kbReviewMutationOptions(scope, 7, 3));
        request.defaults.adapter = async config => { calls++; useAuthStore.setState({ permCodes: ['sys:ai:kb:list'] }); return response(config, { feedback: record }); };
        await assert.rejects(review.mutate({ expected_revision: 2, status: 'accepted', review: record.review }), /权限|工作区/);
      } finally { client.clear(); setRequestErrorReporter(undefined); authorize(); }
    });
    await t.test('列表和详情查询按身份与筛选隔离，取消后迟到响应不能写入有效结果', async () => {
      assert.equal(typeof helpers.kbFeedbackListOptions, 'function');
      const client = new QueryClient();
      const pending = helpers.kbFeedbackListOptions(scope, 7, { page: 2, page_size: 10, status: 'pending' });
      const accepted = helpers.kbFeedbackListOptions(scope, 7, { page: 1, page_size: 10, status: 'accepted' });
      assert.notDeepEqual(pending.queryKey, accepted.queryKey);
      assert.notDeepEqual(pending.queryKey, helpers.kbFeedbackListOptions(['another-session', 1, 1], 7, { page: 2, page_size: 10, status: 'pending' }).queryKey);
      let release, started;
      const called = new Promise(resolve => { started = resolve; });
      request.defaults.adapter = async config => { started(); await new Promise(resolve => { release = resolve; }); return response(config, { list: [record], total: 1 }); };
      const result = client.fetchQuery(pending);
      await called;
      await client.cancelQueries({ queryKey: pending.queryKey, exact: true });
      release();
      await assert.rejects(result);
      assert.equal(client.getQueryData(pending.queryKey), undefined);
      client.clear();
    });
    await t.test('已加载详情刷新失败保留原data及失败版本，恢复成功不抹去需人工核对标记', async () => {
      const client = new QueryClient();
      const options = helpers.kbFeedbackDetailOptions(scope, 7, 3);
      let fail = false;
      setRequestErrorReporter(() => {});
      request.defaults.adapter = async config => response(config, fail ? {} : { feedback: record }, fail ? 500 : 200);
      try {
        await client.fetchQuery(options);
        fail = true;
        await assert.rejects(client.fetchQuery(options), error => error.code === 500);
        const failed = client.getQueryState(options.queryKey);
        assert.equal(failed.status, 'error');
        assert.deepEqual(failed.data, { feedback: record });
        assert.equal(failed.errorUpdateCount, 1);
        fail = false;
        await client.fetchQuery(options);
        const recovered = client.getQueryState(options.queryKey);
        assert.equal(recovered.status, 'success');
        assert.equal(recovered.errorUpdateCount, 1);
      } finally { client.clear(); setRequestErrorReporter(undefined); }
    });
    await t.test('共用任务详情按实际步骤挂评价，快照HTML只展示文字', async () => {
      const { TaskDetailContent } = await server.ssrLoadModule('/src/pages/ai/TaskDetail.tsx');
      const { FeedbackSourceSnapshot } = await server.ssrLoadModule('/src/pages/ai/KnowledgeFeedbackPanel.tsx');
      const analysis = { version: 1, type: 'kb_analysis', status: 'completed', analysis: '发布步骤', sources: snapshot.sources,
        usage: { scope: 'chat', prompt_tokens: 1, completion_tokens: 1, total_tokens: 2, complete: true } };
      const task = { id: 8, status: 'failed', title: '发布分析', result: `[1/2] 知识库分析: ${JSON.stringify(analysis)}`, next_step: 1, steps: [] };
      const steps = [];
      const html = renderToStaticMarkup(createElement(TaskDetailContent, { task, renderFeedback: value => { steps.push(value.step); return createElement('p', null, '评价入口'); } }));
      assert.deepEqual(steps, [1]);
      assert.ok(html.includes('评价入口'));
      const unsafe = { ...snapshot, sources: [{ ...snapshot.sources[0], doc_title: '<img src=x onerror=evil()>', snippet: '<script>evil()</script>' }] };
      const sourceHtml = renderToStaticMarkup(createElement(FeedbackSourceSnapshot, { snapshot: unsafe }));
      assert.ok(sourceHtml.includes('&lt;script&gt;'));
      assert.ok(sourceHtml.includes('&lt;img'));
      assert.ok(!sourceHtml.includes('<script>'));
      assert.ok(!sourceHtml.includes('<img src=x'));
    });
  } finally { await server.close(); globalThis.window = originalWindow; }
});
