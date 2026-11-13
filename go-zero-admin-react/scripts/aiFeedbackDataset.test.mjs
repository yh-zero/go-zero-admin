import test from 'node:test';
import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { createServer } from 'vite';
import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { IsRestoringProvider, QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import { readFile } from 'node:fs/promises';
import { invalidCustomSamples, validCustomVariants } from '../../go-zero-admin/data/scripts/kb-custom-dataset.test-fixtures.mjs';

const hash = raw => createHash('sha256').update(raw).digest('hex');
const content = '发布前必须备份数据库。验收通过后才可切换流量。';
const doc = { id: 9, kb_id: 7, title: '发布资料', version: 2, status: 'ready', embedding_model: 'fixture' };
const snapshot = { version: 1, kb_id: 7, query: '如何发布？', step_input: '{}', step_output: '{}', sources: [{ index: 1, kb_id: 7, doc_id: 9, chunk_id: 11, doc_title: doc.title, snippet: content, score: 0.8 }], documents: [{ doc_id: 9, kb_id: 7, title: doc.title, version: 2, status: 'verified', content_sha256: hash(content) }] };
const record = { id: 3, kb_id: 7, task_id: 8, step_index: 0, status: 'accepted', revision: 4, question: snapshot.query, snapshot: JSON.stringify(snapshot), review: { answerable: true, expected_answer: '先备份数据库，再切换流量。', required_facts: ['备份数据库', '切换流量'], evidence: [{ source_index: 1, quote: content }], reason: '逐字核对' } };

test('采纳反馈生成可校验的冻结题集并保留原提交字节', async t => {
  const originalWindow = globalThis.window, data = new Map();
  const storage = { getItem: k => data.get(k) ?? null, setItem: (k, v) => data.set(k, v), removeItem: k => data.delete(k) };
  globalThis.window = { localStorage: storage, sessionStorage: storage, addEventListener() {} };
  const server = await createServer({ configFile: false, envFile: false, logLevel: 'error', server: { middlewareMode: true, watch: null, ws: false } });
  try {
    const helper = await server.ssrLoadModule('/src/pages/ai/feedbackDataset.ts');
    const quality = await server.ssrLoadModule('/src/pages/ai/knowledgeQuality.ts');
    const frozen = await helper.buildFeedbackDataset(record, [{ doc, content }], false);
    await t.test('导入与Go共同合法样本及坏样本保持一致', async t => {
      for (const [index, raw] of validCustomVariants().entries()) await t.test(`合法${index}`, async () => assert.equal((await quality.importQualityDataset(raw)).raw, raw));
      for (const [name, raw] of invalidCustomSamples()) await t.test(name, async () => assert.rejects(quality.importQualityDataset(raw)));
    });
    await t.test('文件导入保留原始BOM并拒绝，不静默改写指纹', async () => {
      const source = await readFile(new URL('../src/pages/ai/KnowledgeQualityPage.tsx', import.meta.url), 'utf8');
      assert.match(source, /decodeCustomDataset\(await file\.arrayBuffer\(\)\)/u);
    });
    await t.test('正例冻结原文/ID/版本并保持词法评分与预期答案，所有题为开发题', () => {
      const item = frozen.dataset.cases[0], source = frozen.dataset.documents[0];
      assert.equal(frozen.key, `custom-${hash(frozen.raw).slice(0, 32)}`);
      assert.equal(source.content, content); assert.equal(source.source_doc_id, 9); assert.equal(source.source_doc_version, 2);
      assert.equal(source.sha256, hash(content)); assert.equal(source.source, `inline:${source.key}`);
      assert.deepEqual(item.answer.required_groups, [['备份数据库'], ['切换流量']]);
      assert.equal(item.expected_answer_example, record.review.expected_answer); assert.equal(item.split, 'development');
      assert.match(frozen.dataset.description, /采纳.*不等于.*签署/);
    });
    await t.test('来源版本/ID/状态/标题/SHA或逐字摘录变化均拒绝，不套旧结论', async () => {
      for (const changed of [{ doc: { ...doc, id: 10 }, content }, { doc: { ...doc, version: 3 }, content }, { doc: { ...doc, status: 'pending' }, content }, { doc: { ...doc, title: '改名' }, content }, { doc, content: content + '改动' }])
        await assert.rejects(helper.buildFeedbackDataset(record, [changed], false));
      await assert.rejects(helper.buildFeedbackDataset({ ...record, review: { ...record.review, evidence: [{ source_index: 1, quote: '伪造原文' }] } }, [{ doc, content }], false));
    });
    await t.test('负例须明确核对当前库，空库/体积超限不能生成，原no_matches不作全库结论', async () => {
      const negative = { ...record, review: { answerable: false, expected_answer: '当前资料无法确定。', required_facts: [], evidence: [], reason: '人工核对' }, snapshot: JSON.stringify({ ...snapshot, sources: [], documents: [] }) };
      await assert.rejects(helper.buildFeedbackDataset(negative, [{ doc, content }], false), /核对/);
      await assert.rejects(helper.buildFeedbackDataset(negative, [], true), /资料/);
      const result = await helper.buildFeedbackDataset(negative, [{ doc, content }], true);
      assert.equal(result.dataset.cases[0].answer.kind, 'unanswerable');
      assert.deepEqual(result.dataset.cases[0].expected_documents, []);
      await assert.rejects(helper.buildFeedbackDataset(negative, [{ doc, content: '中'.repeat(100_000) }], true), /256|大小/);
    });
    await t.test('导入校验大小、严格字段、重复键及内联SHA，原字节不格式化', async () => {
      const raw = ` \n${frozen.raw}\n`;
      assert.equal((await quality.importQualityDataset(raw)).raw, raw);
      await assert.rejects(quality.importQualityDataset(raw.replace('"content":', '"Content":')));
      await assert.rejects(quality.importQualityDataset(raw.replace('"version": 1,', '"version": 1, "version":1,')));
      await assert.rejects(quality.importQualityDataset(raw.replace(content, '篡改正文')));
      await assert.rejects(quality.importQualityDataset(raw.replace('"version": 1,', '"version": 1, "description":"\\ud800",')));
      const external = JSON.parse(raw); delete external.documents[0].content;
      await assert.rejects(quality.importQualityDataset(JSON.stringify(external)), /内联|正文/);
      for (const bad of [{ ...JSON.parse(raw), description: {} }, { ...JSON.parse(raw), cases: [{ ...frozen.dataset.cases[0], expected_answer_example: {} }] }]) await assert.rejects(quality.importQualityDataset(JSON.stringify(bad)));
    });
    await t.test('未知提交恢复/同UUID核对保留完整raw，拒绝快照不符响应', () => {
      const attempt = quality.createQualityAttempt(frozen, frozen.dataset.cases[0].kb, [frozen.dataset.cases[0].id], '177654b5-56f1-4a44-afb4-f39ccb5e261f');
      assert.equal(attempt.dataset_snapshot, frozen.raw);
      const stored = new Map(), storage = { getItem: k => stored.get(k) ?? null, setItem: (k, v) => stored.set(k, v) };
      quality.writeQualityAttempt(storage, 'attempt', attempt);
      assert.equal(quality.readQualityAttempt(storage, 'attempt').dataset_snapshot, frozen.raw);
      assert.throws(() => quality.assertQualityAttemptResult({ ...attempt, snapshot: frozen.raw + ' ', case_count: 1, observations: '' }, attempt), /快照|原提交/);
    });
    await t.test('真实读取模块先后复核revision，scope/权限/取消拒绝迟到结果，全程只有GET', async () => {
      const { request } = await server.ssrLoadModule('/src/utils/request.ts');
      const { useAuthStore } = await server.ssrLoadModule('/src/stores/auth.ts');
      const scope = ['feedback-dataset-fixture', 1, 1];
      const authorize = () => useAuthStore.setState({ token: 'fixture', sessionId: scope[0], sessionVersion: 1, departmentVersion: 1, expiresAt: Date.now() / 1000 + 3600, permissionsReady: true, isLoggingOut: false, permCodes: ['sys:ai:kb:list'] });
      const calls = [];
      let changed = false, revoke = false, cancelled;
      request.defaults.adapter = async config => {
        calls.push(config); const feedbackRead = config.url.endsWith('/feedback/3');
        if (!feedbackRead && revoke) useAuthStore.setState({ permCodes: [] });
        if (!feedbackRead && cancelled) cancelled.abort();
        return { data: { code: 200, result: feedbackRead ? { feedback: { ...record, revision: changed && calls.filter(c => c.url.endsWith('/feedback/3')).length > 1 ? 5 : 4 } } : { doc, content } }, status: 200, statusText: 'OK', headers: {}, config };
      };
      const read = signal => helper.feedbackDatasetOptions(scope, 7, record, false).queryFn({ signal });
      authorize(); assert.equal((await read(new AbortController().signal)).dataset.cases.length, 1);
      assert.equal(calls.length, 3); assert.ok(calls.every(c => c.method === 'get'));
      calls.length = 0; changed = true; await assert.rejects(read(new AbortController().signal), /版本|变化/); changed = false;
      assert.equal(calls.length, 3, '全文读完后必须再次确认反馈revision');
      authorize(); revoke = true; await assert.rejects(read(new AbortController().signal), /权限|工作区/); revoke = false;
      authorize(); cancelled = new AbortController(); await assert.rejects(read(cancelled.signal));
      authorize(); useAuthStore.setState({ sessionVersion: 2 }); const before = calls.length; await assert.rejects(read(new AbortController().signal)); assert.equal(calls.length, before);
    });
    await t.test('真实预览只显示安全文本、来源与检查项，不将恶意内容当HTML', async () => {
      const { FeedbackDatasetPreview } = await server.ssrLoadModule('/src/pages/ai/FeedbackDatasetPreview.tsx');
      const value = { ...frozen, dataset: { ...frozen.dataset, cases: [{ ...frozen.dataset.cases[0], query: '<img src=x onerror=alert(1)>' }] } };
      const html = renderToStaticMarkup(createElement(FeedbackDatasetPreview, { catalogue: value }));
      for (const text of ['备份数据库', '切换流量', '发布资料', '开发题', '预期答案', '&lt;img']) assert.ok(html.includes(text), text);
      assert.ok(!html.includes('<img src=x'));
    });
    await t.test('题集预览把显式前往新建评测作为主操作，下载收进更多导出', async () => {
      const { FeedbackDatasetGenerator } = await server.ssrLoadModule('/src/pages/ai/FeedbackDatasetGenerator.tsx');
      const scope = ['feedback-preview-fixture', 1, 1], client = new QueryClient();
      client.setQueryData(helper.feedbackDatasetOptions(scope, 7, record, false).queryKey, frozen);
      const guard = { can: () => true, isCurrent: () => true, writing: false };
      try {
        const html = renderToStaticMarkup(createElement(QueryClientProvider, { client }, createElement(IsRestoringProvider, { value: true }, createElement(MemoryRouter, null,
          createElement(FeedbackDatasetGenerator, { scope, kbId: 7, record, guard, onClose() {} })))));
        assert.match(html, /ant-btn-primary[^>]*>[\s\S]*?确认题集，前往新建评测/u);
        assert.match(html, /href="\/ai\/knowledge\/7\/quality"/u);
        assert.ok(html.includes('只有点击“开始评测”才执行模型'));
        assert.ok(html.includes('更多导出'));
        assert.ok(!html.includes('下载冻结题集'), '下载操作应在打开菜单后显示');
        assert.ok(!html.includes('导入质量验收页'), '不能继续把下载与移交并列为无主次动作');
      } finally { client.clear(); }
    });
  } finally { await server.close(); globalThis.window = originalWindow; }
});
