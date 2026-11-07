import test from 'node:test';
import assert from 'node:assert/strict';
import { createServer } from 'vite';
import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { MemoryRouter } from 'react-router-dom';

test('工具按用途选择并如实展示配置状态和来源版本', async t => {
  const originalWindow = globalThis.window;
  globalThis.window = { localStorage: { getItem() { return null; } }, sessionStorage: { getItem() { return null; } }, addEventListener() {} };
  const server = await createServer({ configFile: false, envFile: false, logLevel: 'error', optimizeDeps: { noDiscovery: true }, server: { middlewareMode: true, watch: null, ws: false } });
  try {
    const guides = await server.ssrLoadModule('/src/pages/ai/toolGuides.ts');
    const { ToolCard, ToolDetailContent } = await server.ssrLoadModule('/src/pages/ai/AITools.tsx');
    const renderCard = props => renderToStaticMarkup(createElement(MemoryRouter, null, createElement(ToolCard, props)));
    const tool = { name: 'kb_analyze', description: '知识库分析', parameters_schema: '{"type":"object"}' };
    const capabilities = { retrieval_configured: true, analysis_configured: true, ingestion_mode: 'background' };

    await t.test('中文用途搜索包含分类，优先呈现资料分析和查找资料', () => {
      assert.equal(typeof guides.filterToolsByPurpose, 'function');
      const tools = [{ name: 'get_current_time', description: '' }, { name: 'kb_search', description: '' }, tool];
      assert.deepEqual(guides.filterToolsByPurpose(tools, '').map(item => item.name), ['kb_analyze', 'kb_search', 'get_current_time']);
      assert.deepEqual(guides.filterToolsByPurpose(tools, '知识库').map(item => item.name), ['kb_analyze', 'kb_search']);
      assert.deepEqual(guides.filterToolsByPurpose(tools, '分析资料').map(item => item.name), ['kb_analyze']);
      assert.deepEqual(guides.filterToolsByPurpose([{ name: 'json_formatter', description: '' }], '整理 JSON').map(item => item.name), ['json_formatter']);
      assert.equal(guides.filterToolsByPurpose(tools, 'not-found').length, 0);
    });
    await t.test('配置缺失、未知和已配置分别显示，工具注册不等于模型可用', () => {
      const unknown = renderCard({ tool, canUse: true, onView() {} });
      assert.match(unknown, /配置状态未知/);
      const missing = renderCard({ tool, capabilities: { ...capabilities, analysis_configured: false }, canUse: true, onView() {} });
      assert.match(missing, /分析模型未配置/);
      assert.doesNotMatch(missing, /资料分析已配置/);
      const noRetrieval = renderCard({ tool, capabilities: { ...capabilities, retrieval_configured: false }, canUse: true, onView() {} });
      assert.match(noRetrieval, /检索能力未配置/);
      const ready = renderCard({ tool, capabilities, canUse: true, onView() {} });
      assert.match(ready, /资料分析已配置/);
      assert.match(ready, /知识库/);
    });
    await t.test('详情先展示输入输出，参数Schema默认折叠且保持原始坏Schema转义', () => {
      const markup = renderToStaticMarkup(createElement(ToolDetailContent, { tool: { ...tool, parameters_schema: '{bad <svg onload=evil()>' }, capabilities }));
      assert.match(markup, /示例输入/);
      assert.match(markup, /示例输出/);
      assert.match(markup, /<details[^>]*><summary>参数 Schema/);
      assert.doesNotMatch(markup, /<details[^>]*\bopen(?:[=>\s])/);
      assert.match(markup, /\{bad &lt;svg/);
      assert.doesNotMatch(markup, /<svg onload/);
    });
    await t.test('能力卡支持加入当前流程，兼容原创建链接并保持权限边界', () => {
      const embedded = renderCard({ tool, capabilities, canUse: true, onView() {}, onSelect() {} });
      assert.match(embedded, /加入当前流程/);
      assert.doesNotMatch(embedded, /\/ai\/workflows\/new/);
      const catalog = renderCard({ tool, capabilities, canUse: true, onView() {} });
      assert.match(catalog, /\/ai\/workflows\/new\?tool=kb_analyze/);
      const denied = renderCard({ tool, capabilities, canUse: false, onView() {}, onSelect() {} });
      assert.doesNotMatch(denied, /加入当前流程|\/ai\/workflows\/new/);
    });
    await t.test('历史分析保留当时摘录，打开当前原文不冒充当时版本', async () => {
      const { WorkflowAnalysisResult } = await server.ssrLoadModule('/src/pages/ai/WorkflowAnalysisResult.tsx');
      const markup = renderToStaticMarkup(createElement(WorkflowAnalysisResult, { analysis: {
        step: 1, total: 1, name: '资料分析', result: {
          version: 1, type: 'kb_analysis', status: 'completed', analysis: '已保存的分析',
          sources: [{ index: 1, kb_id: 7, doc_id: 9, chunk_id: 21, doc_title: '原资料', snippet: '执行当时的摘录 <script>unsafe()</script>', score: 0.9 }],
          usage: { scope: 'chat', complete: true, prompt_tokens: 1, completion_tokens: 1, total_tokens: 2 },
        },
      } }));
      assert.match(markup, /href="\/ai\/knowledge\/7\?doc_id=9"/);
      assert.match(markup, /打开当前原文/);
      assert.match(markup, /执行当时的摘录 &lt;script&gt;/);
      assert.match(markup, /当前原文可能已更新/);
      assert.doesNotMatch(markup, /<script>unsafe/);
    });
  } finally {
    await server.close();
    globalThis.window = originalWindow;
  }
});
