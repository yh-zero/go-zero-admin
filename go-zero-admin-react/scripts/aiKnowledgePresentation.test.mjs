import test from 'node:test';
import assert from 'node:assert/strict';
import { createServer } from 'vite';
import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';

test('知识库以资料和历史为默认任务，次要入口按需展开', async t => {
  const previousWindow = globalThis.window, previousStorage = globalThis.sessionStorage;
  const storage = () => { const values = new Map(); return { getItem: key => values.get(key) ?? null, setItem: (key, value) => values.set(key, value), removeItem: key => values.delete(key) }; };
  globalThis.window = { localStorage: storage(), sessionStorage: storage(), addEventListener() {} };
  globalThis.sessionStorage = globalThis.window.sessionStorage;
  const server = await createServer({ configFile: false, envFile: false, logLevel: 'error', optimizeDeps: { noDiscovery: true }, server: { middlewareMode: true, watch: null, ws: false } });
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  try {
    const { useAuthStore } = await server.ssrLoadModule('/src/stores/auth.ts');
    const scope = ['knowledge-presentation', 1, 1];
    useAuthStore.setState({ token: 'local-fixture', sessionId: scope[0], sessionVersion: 1, departmentVersion: 1, permissionsReady: true, isLoggingOut: false,
      permCodes: ['sys:ai:kb:list', 'sys:ai:kb:edit', 'sys:ai:kb:delete', 'sys:ai:kb:evaluate'] });
    // SSR 读取测试当前身份快照；实际权限判断仍走真实 guard。
    Object.assign(useAuthStore.getInitialState(), useAuthStore.getState());
    const { aiWorkspaceKeys } = await server.ssrLoadModule('/src/pages/ai/workspaceQueries.ts');
    const kb = { id: 7, name: '发布手册', description: '', owner_id: 1, doc_count: 1, chunk_count: 0, created_at: 1, updated_at: 1, ready_count: 0, pending_count: 0, failed_count: 1 };
    const capabilities = { analysis_configured: true, retrieval_configured: true };
    client.setQueryData([...aiWorkspaceKeys.kbs(scope), { page: 1, page_size: 20, query: '' }], { list: [kb], total: 1, capabilities });
    client.setQueryData(aiWorkspaceKeys.kb(scope, 7), { kb, capabilities });
    client.setQueryData([...aiWorkspaceKeys.kbDocs(scope, 7), { page: 1, page_size: 10 }], { list: [{ id: 8, kb_id: 7, title: '故障指南', status: 'failed', stage: 'failed', version: 1, error: '处理失败', chunk_count: 0, embedding_model: '', created_at: 1, updated_at: 1 }], total: 1, ready_count: 0, pending_count: 0, failed_count: 1 });
    client.setQueryData(aiWorkspaceKeys.qualityDatasets(scope, 7), [{ key: 'fixture', name: '回归题集', sha256: 'a'.repeat(64), raw: '{}', dataset: { version: 1, documents: [], cases: [] } }]);
    client.setQueryData([...aiWorkspaceKeys.qualityRuns(scope, 7), { page: 1, page_size: 10 }], { list: [], total: 0 });
    const render = Component => renderToStaticMarkup(createElement(QueryClientProvider, { client }, createElement(MemoryRouter, null, createElement(Component, { scope, kbId: 7, kb, onChanged() {} }))));
    await t.test('名称是唯一知识库入口，编辑删除默认收起', async () => {
      const { AIKnowledgeSession } = await server.ssrLoadModule('/src/pages/ai/AIKnowledge.tsx');
      const html = render(AIKnowledgeSession);
      assert.match(html, /发布手册/);
      assert.doesNotMatch(html, /进入工作区/);
      assert.match(html, /更多/);
      assert.doesNotMatch(html, />编辑<|>删除</);
    });
    await t.test('资料名称打开原文，失败重试明确且添加方式收起', async () => {
      const { KnowledgeDocumentsPanel } = await server.ssrLoadModule('/src/pages/ai/KnowledgeDocumentsDrawer.tsx');
      const html = render(KnowledgeDocumentsPanel);
      assert.match(html, /故障指南/);
      assert.match(html, />重试处理</);
      assert.match(html, /添加资料/);
      assert.match(html, /查看导入批次/);
      assert.doesNotMatch(html, />原文<|>导入 TXT \/ Markdown<|>粘贴资料<|>删除</);
    });
    await t.test('质量页默认历史，创建表单及题集导入不挂载', async () => {
      const { KnowledgeQualitySession } = await server.ssrLoadModule('/src/pages/ai/KnowledgeQualityPage.tsx');
      const html = render(KnowledgeQualitySession);
      assert.match(html, /历史批次/);
      assert.match(html, /新建评测/);
      assert.doesNotMatch(html, /选择评测题集|开始一批评测|导入冻结题集JSON/);
      assert.match(html, /调用.*用量/);
      const executedDetailModules = [...server.environments.ssr.moduleGraph.idToModuleMap.values()]
        .filter(module => module.ssrModule && /\/(?:KnowledgeQualityDetails|TaskDetail)\.tsx$/.test(module.id ?? ''));
      assert.deepEqual(executedDetailModules.map(module => module.id), [], '历史首屏不应执行结果或任务详情模块');
    });
    await t.test('新建区收起时仍可核对原评测标识，避免未知提交被藏起', async () => {
      const { qualityAttemptKey, writeQualityAttempt } = await server.ssrLoadModule('/src/pages/ai/knowledgeQuality.ts');
      const key = qualityAttemptKey(scope, null, 7);
      writeQualityAttempt(sessionStorage, key, { request_id: '11111111-1111-4111-8111-111111111111', dataset_key: 'fixture', dataset_sha256: 'a'.repeat(64), group_key: 'project_ai', case_ids: ['one'] });
      const { KnowledgeQualitySession } = await server.ssrLoadModule('/src/pages/ai/KnowledgeQualityPage.tsx');
      const html = render(KnowledgeQualitySession);
      for (const text of ['上次提交结果待确认', '11111111-1111-4111-8111-111111111111', '核对原请求', '使用原标识继续', '历史批次']) assert.ok(html.includes(text), text);
      assert.doesNotMatch(html, /选择评测题集/);
      sessionStorage.removeItem(key);
    });
    await t.test('只读身份仍能打开资料名称，编辑删除和添加入口均不可见', async () => {
      useAuthStore.setState({ permCodes: ['sys:ai:kb:list'] });
      Object.assign(useAuthStore.getInitialState(), useAuthStore.getState());
      const { KnowledgeDocumentsPanel } = await server.ssrLoadModule('/src/pages/ai/KnowledgeDocumentsDrawer.tsx');
      const html = render(KnowledgeDocumentsPanel);
      assert.match(html, /故障指南/);
      assert.match(html, /查看导入批次/);
      assert.doesNotMatch(html, /添加资料|更多：|重试故障指南|>删除</);
    });
  } finally { client.clear(); await server.close(); globalThis.window = previousWindow; globalThis.sessionStorage = previousStorage; }
});
