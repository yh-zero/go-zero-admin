import test from 'node:test';
import assert from 'node:assert/strict';
import { createServer } from 'vite';
import { MutationObserver, QueryClient } from '@tanstack/react-query';
import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';

function memoryStorage() {
  const values = new Map();
  return { getItem: key => values.get(key) ?? null, setItem: (key, value) => values.set(key, value), removeItem: key => values.delete(key) };
}

test('知识库直接召回测试', async t => {
  const originalWindow = globalThis.window;
  globalThis.window = { localStorage: memoryStorage(), sessionStorage: memoryStorage(), addEventListener() {} };
  const server = await createServer({ configFile: false, envFile: false, logLevel: 'error', server: { middlewareMode: true, watch: null, ws: false } });
  try {
    const search = await server.ssrLoadModule('/src/pages/ai/knowledgeSearch.ts').catch(() => ({}));
    await t.test('查询校验使用UTF-8的2000字节边界，拒绝空白而非截断查询', () => {
      assert.equal(typeof search.validateKbSearchQuery, 'function', '缺少召回查询校验');
      assert.equal(search.validateKbSearchQuery('a'.repeat(2000)), undefined);
      assert.equal(search.validateKbSearchQuery('中'.repeat(666) + 'aa'), undefined);
      assert.equal(search.validateKbSearchQuery('🙂'.repeat(500)), undefined);
      assert.equal(search.kbSearchQueryBytes('🙂'), 4);
      assert.ok(search.validateKbSearchQuery('a'.repeat(2001)));
      assert.ok(search.validateKbSearchQuery('中'.repeat(667)));
      assert.ok(search.validateKbSearchQuery('🙂'.repeat(501)));
      assert.ok(search.validateKbSearchQuery(' \n\t '));
    });

    const api = await server.ssrLoadModule('/src/api/ai.ts');
    const { request, setRequestErrorReporter } = await server.ssrLoadModule('/src/utils/request.ts');
    const { useAuthStore } = await server.ssrLoadModule('/src/stores/auth.ts');
    const scope = ['local-recall-session', 1, 1];
    function authorize() {
      useAuthStore.setState({ token: 'local-recall-test', sessionId: scope[0], sessionVersion: scope[1], departmentVersion: scope[2],
        expiresAt: Date.now() / 1000 + 3600, permissionsReady: true, permCodes: ['sys:ai:kb:list'], isLoggingOut: false });
    }
    authorize();
    const hits = [
      { chunk_id: 21, kb_id: 7, doc_id: 9, doc_title: '操作手册', snippet: '第一行\n第二行<script>unsafe</script>', score: 0.91 },
      { chunk_id: 10, kb_id: 7, doc_id: 8, doc_title: '发布说明', snippet: '检索片段', score: 0.8 },
    ];
    const success = { list: hits, elapsed_ms: 123 };
    const respond = (config, code, result, msg = '') => ({ data: { code, result, msg }, status: 200, statusText: 'OK', headers: {}, config });

    await t.test('统一API只发送召回请求并保留顺序、字段及15秒超时', async () => {
      assert.equal(typeof api.searchAiKb, 'function', '缺少直接召回API');
      const calls = [];
      request.defaults.adapter = async config => { calls.push(config); return respond(config, 200, success); };
      assert.deepEqual(await api.searchAiKb(7, { query: '  如何发布？  ' }), success);
      assert.equal(calls.length, 1);
      assert.equal(calls[0].method, 'post');
      assert.equal(calls[0].url, '/sys/ai/kbs/7/search');
      assert.equal(calls[0].timeout, 15_000);
      assert.deepEqual(JSON.parse(calls[0].data), { query: '  如何发布？  ' });
      assert.equal(calls[0].headers.Authorization, 'Bearer local-recall-test');
    });

    await t.test('Mutation区分结果、空召回和503失败，新请求前可清除旧结果且不自动重试', async () => {
      assert.equal(typeof search.kbSearchOptions, 'function', '缺少受身份保护的召回Mutation');
      const client = new QueryClient();
      const calls = [];
      const errors = [];
      const responses = [{ code: 200, result: success }, { code: 200, result: { list: [], elapsed_ms: 42 } }, { code: 503, msg: '检索暂不可用' }];
      setRequestErrorReporter(message => errors.push(message));
      request.defaults.adapter = async config => { calls.push(config.url); const next = responses.shift(); return respond(config, next.code, next.result, next.msg); };
      try {
        const mutation = new MutationObserver(client, search.kbSearchOptions(scope, 7));
        await mutation.mutate({ query: '发布流程' });
        assert.deepEqual(mutation.getCurrentResult().data, success);
        mutation.reset();
        assert.equal(mutation.getCurrentResult().data, undefined);
        assert.deepEqual(await mutation.mutate({ query: '无资料问题' }), { list: [], elapsed_ms: 42 });
        mutation.reset();
        await assert.rejects(mutation.mutate({ query: '服务失败' }), error => error.code === 503);
        assert.equal(mutation.getCurrentResult().data, undefined);
        assert.equal(mutation.getCurrentResult().isError, true);
        assert.equal(calls.length, 3, '503不得自动重复请求');
        assert.deepEqual(errors, ['检索暂不可用'], '由统一请求层只反馈一次');
        await assert.rejects(mutation.mutate({ query: '\n ' }), /空白/);
        assert.equal(calls.length, 3, '非法查询不发送API');
      } finally { client.clear(); setRequestErrorReporter(undefined); }
    });

    await t.test('旧身份不能发起检索，身份或权限改变后的迟到响应不能变成有效结果', async () => {
      assert.equal(typeof search.kbSearchOptions, 'function', '缺少受身份保护的召回Mutation');
      const client = new QueryClient();
      let calls = 0;
      try {
        for (const change of [{ departmentVersion: 2 }, { sessionVersion: 2 }, { permCodes: [] }]) {
          authorize();
          let started;
          let release;
          const pending = new Promise(resolve => { release = resolve; });
          const sent = new Promise(resolve => { started = resolve; });
          request.defaults.adapter = async config => { calls++; started(); await pending; return respond(config, 200, success); };
          const mutation = new MutationObserver(client, search.kbSearchOptions(scope, 7));
          const result = mutation.mutate({ query: '旧身份查询' });
          await sent;
          useAuthStore.setState(change);
          release();
          await assert.rejects(result);
          assert.equal(mutation.getCurrentResult().data, undefined);
          await assert.rejects(mutation.mutate({ query: '失效后查询' }));
        }
        assert.equal(calls, 3, '作用域失效后不能新增请求');
      } finally { authorize(); client.clear(); }
    });

    await t.test('结果按服务端排名展示ID与相似度，片段以纯文本保留换行，空结果单独展示', async () => {
      const view = await server.ssrLoadModule('/src/pages/ai/KbSearchModal.tsx').catch(() => ({}));
      assert.equal(typeof view.KbSearchResults, 'function', '缺少召回结果展示');
      const markup = renderToStaticMarkup(createElement(view.KbSearchResults, { result: success }));
      assert.ok(markup.indexOf('操作手册') < markup.indexOf('发布说明'));
      for (const text of ['文档 ID：9', '分块 ID：21', '相似度：0.91', '123', '第一行\n第二行']) assert.ok(markup.includes(text), text);
      assert.ok(markup.includes('&lt;script&gt;unsafe&lt;/script&gt;'));
      assert.ok(!markup.includes('<script>unsafe</script>'));
      const empty = renderToStaticMarkup(createElement(view.KbSearchResults, { result: { list: [], elapsed_ms: 42 } }));
      assert.match(empty, /未召回片段/);
      assert.ok(empty.includes('42'));
      assert.doesNotMatch(empty, /召回失败/);
    });
  } finally { await server.close(); globalThis.window = originalWindow; }
});
