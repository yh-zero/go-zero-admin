import test from 'node:test';
import assert from 'node:assert/strict';
import { createServer } from 'vite';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';

function deferred() {
  let resolve;
  let reject;
  const promise = new Promise((yes, no) => { resolve = yes; reject = no; });
  return { promise, resolve, reject };
}

function memoryStorage() {
  const values = new Map();
  return { getItem: key => values.get(key) ?? null, setItem: (key, value) => values.set(key, value), removeItem: key => values.delete(key) };
}

test('AI对话关键边界', async t => {
  const originalWindow = globalThis.window;
  globalThis.window = { localStorage: memoryStorage(), sessionStorage: memoryStorage(), addEventListener() {} };
  const server = await createServer({ configFile: false, envFile: false, logLevel: 'error', server: { middlewareMode: true, watch: null, ws: false } });
  try {
    const helpers = await server.ssrLoadModule('/src/pages/ai/chatState.ts');
    await t.test('消息限制按UTF8字节且保持原始正文，空白不能发送', () => {
      assert.equal(helpers.validateMessageContent('中'.repeat(5333) + 'a'), undefined);
      assert.match(helpers.validateMessageContent('中'.repeat(5334)), /16000/);
      assert.match(helpers.validateMessageContent(' \n\t'), /内容/);
    });

    await t.test('双击和Enter同时点击只发一次；重查沿用UUID；新意图使用新UUID', async () => {
      const first = deferred();
      const calls = [];
      let serial = 0;
      const sender = helpers.createMessageSender(request => {
        calls.push(request);
        return calls.length === 1 ? first.promise : Promise.resolve({ conversation_id: 8 });
      }, () => `00000000-0000-4000-8000-${String(++serial).padStart(12, '0')}`);
      const initial = sender.start(8, '  原始正文  ');
      assert.ok(initial);
      assert.equal(sender.start(8, '  原始正文  '), null);
      assert.equal(sender.recheck(), null);
      await Promise.resolve();
      assert.equal(calls.length, 1);
      assert.equal(calls[0].content, '  原始正文  ');
      first.reject(new Error('网络结果未知'));
      await assert.rejects(initial.promise);
      const recheck = sender.recheck();
      await recheck.promise;
      assert.deepEqual(calls[1], calls[0]);
      await sender.start(8, '  原始正文  ').promise;
      assert.notEqual(calls[2].request_id, calls[0].request_id);
      assert.equal(calls.length, 3, '失败不自动重试');
    });

    await t.test('资料范围跟随原请求重查，知识问答按2000 UTF8字节限制', async () => {
      const calls = [];
      let serial = 0;
      const sender = helpers.createMessageSender(async request => {
        calls.push(request);
        if (calls.length === 1) throw new Error('结果未知');
        return { conversation_id: 8, conversation: { id: 8 } };
      }, () => `request-${++serial}`);
      assert.equal(sender.start(0, '中'.repeat(667), {}, { kb_id: 2, grounded: true }), null);
      assert.equal(sender.start(0, '问题', {}, { kb_id: 0, grounded: true }), null);
      assert.equal(serial, 0, '超限或缺少资料范围时不创建请求');
      const content = '中'.repeat(666) + 'aa';
      const first = sender.start(0, content, {}, { kb_id: 2, grounded: true });
      assert.ok(first, '正好2000 UTF8字节的问题可以发送');
      await assert.rejects(first.promise, /结果未知/);
      await sender.recheck().promise;
      assert.deepEqual(calls[1], calls[0]);
      assert.equal(calls[1].content, content);
      assert.equal(calls[1].kb_id, 2);
      assert.equal(calls[1].grounded, true);
      await sender.start(8, '问题', {}, { kb_id: 3, grounded: false }).promise;
      assert.equal(calls[2].kb_id, 3);
      assert.equal(calls[2].grounded, false);
      assert.notEqual(calls[0].request_id, calls[2].request_id);
    });

    await t.test('知识库选择器按资料范围显示约束并保留读取与权限边界', async t => {
      const { ChatKnowledgePicker } = await server.ssrLoadModule('/src/pages/ai/ChatKnowledgePicker.tsx');
      const { kbListOptions } = await server.ssrLoadModule('/src/pages/ai/workspaceQueries.ts');
      const { useAuthStore } = await server.ssrLoadModule('/src/stores/auth.ts');
      // Zustand 的 SSR 读取初始快照，逐例设置权限后恢复，避免把权限分支替换成桩。
      const snapshot = useAuthStore.getInitialState();
      const originalPermissions = { permissionsReady: snapshot.permissionsReady, permCodes: snapshot.permCodes };
      const scope = ['picker-session', 1, 1];
      const client = new QueryClient({ defaultOptions: { queries: { retry: false, retryOnMount: false } } });
      const listKey = kbListOptions(scope, { page: 1, page_size: 20 }).queryKey;
      const selectedKey = ['ai-chat', ...scope, 'selected-kb', 27];
      const kb = { id: 2, name: '团队资料', description: '', owner_id: 1, doc_count: 3, chunk_count: 6,
        ready_count: 3, pending_count: 0, failed_count: 0, created_at: 1, updated_at: 1 };
      const capabilities = { retrieval_configured: true, analysis_configured: false, ingestion_mode: 'async' };
      const reset = () => {
        client.clear();
        Object.assign(snapshot, { permissionsReady: true, permCodes: ['sys:ai:kb:list'] });
        client.setQueryData(listKey, { list: [kb], total: 1, capabilities });
      };
      const render = (value, disabled = false) => renderToStaticMarkup(createElement(QueryClientProvider, { client },
        createElement(ChatKnowledgePicker, { scope, value, disabled, onChange() { assert.fail('渲染不应触发选择变更'); } })));
      const readSwitch = markup => {
        const control = markup.match(/<button\b[^>]*\brole="switch"[^>]*>/)?.[0];
        assert.ok(control, '指定知识库时显示资料约束开关');
        return control;
      };
      try {
        await t.test('自动检索范围只显示资料选择，不渲染禁用的约束开关或标签', () => {
          reset();
          const markup = render({ kb_id: 0, grounded: false });
          assert.match(markup, /自动检索我的知识库/);
          assert.doesNotMatch(markup, /role="switch"/);
          assert.doesNotMatch(markup, /仅依据资料回答/);
        });

        await t.test('选定知识库后显示原约束值，开关仅在写入时禁用', () => {
          reset();
          for (const grounded of [false, true]) {
            for (const writing of [false, true]) {
              const markup = render({ kb_id: 2, grounded }, writing);
              const control = readSwitch(markup);
              assert.match(control, /aria-label="仅依据所选资料回答"/);
              assert.match(control, new RegExp(`aria-checked="${grounded}"`));
              assert.equal(/\bdisabled=""/.test(control), writing);
              assert.match(markup, /仅依据资料回答/);
              assert.equal(markup.includes('2000 UTF-8 字节'), grounded);
            }
          }
        });

        await t.test('已选资料不在当前页时仍显示资料名和分页', () => {
          reset();
          client.setQueryData(listKey, { list: [kb], total: 21, capabilities });
          client.setQueryData(selectedKey, { ...kb, id: 27, name: '分页外资料' });
          const markup = render({ kb_id: 27, grounded: true });
          assert.match(markup, /分页外资料/);
          assert.match(markup, /ant-pagination/);
          assert.match(readSwitch(markup), /aria-checked="true"/);
          assert.match(markup, /2000 UTF-8 字节/);
        });

        await t.test('列表或已选资料读取失败仍显示恢复提示', () => {
          for (const failingKey of [listKey, selectedKey]) {
            reset();
            client.getQueryCache().build(client, { queryKey: failingKey }).setState({
              status: 'error', error: new Error('资料读取失败'), fetchStatus: 'idle',
            });
            const markup = render({ kb_id: 27, grounded: true });
            assert.match(markup, /知识库读取失败，请刷新后重试/);
            assert.match(markup, /知识库 #27/);
          }
        });

        await t.test('权限未就绪或无知识库查看权限时不显示选择器且禁止两类查询', () => {
          for (const permissions of [
            { permissionsReady: false, permCodes: ['sys:ai:kb:list'] },
            { permissionsReady: true, permCodes: ['sys:ai:conversation:list'] },
          ]) {
            reset();
            Object.assign(snapshot, permissions);
            assert.equal(render({ kb_id: 27, grounded: true }), '');
            const queries = client.getQueryCache().getAll();
            assert.equal(queries.length, 2, '列表和分页外已选资料均受权限约束');
            for (const query of queries) {
              assert.equal(query.options.enabled, false);
              assert.equal(query.state.fetchStatus, 'idle');
            }
          }
        });
      } finally {
        Object.assign(snapshot, originalPermissions);
        client.clear();
      }
    });

    await t.test('流式通道保存meta和原请求，卸载中止后不接受迟到结果', async () => {
      const pending = deferred();
      let signal;
      const sender = helpers.createMessageSender((request, handlers, abortSignal) => {
        signal = abortSignal;
        handlers.onMeta({ conversation_id: 9, user_message_id: 1, ai_message_id: 2 });
        return pending.promise;
      });
      const attempt = sender.start(0, '首条内容');
      assert.equal(sender.start(0, '第二次'), null);
      await Promise.resolve();
      assert.equal(attempt.meta.conversation_id, 9);
      sender.abort();
      assert.equal(signal.aborted, true);
      pending.resolve({ conversation_id: 9 });
      await assert.rejects(attempt.promise, error => error.name === 'AbortError');
    });

    await t.test('游标页去重按时间正序，重复或倒退无效的游标终止', () => {
      const message = (id, created_at, content = '') => ({ id, created_at, content });
      const pages = [
        { messages: [message(3, 30, '最新'), message(4, 40)] },
        { messages: [message(1, 10), message(3, 30, '旧快照'), message(2, 20)] },
      ];
      const messages = helpers.mergeMessagePages(pages);
      assert.deepEqual(messages.map(row => row.id), [1, 2, 3, 4]);
      assert.equal(messages[2].content, '最新');
      assert.equal(helpers.nextMessageCursor({ has_more: true, next_before_id: 12 }, 0), 12);
      assert.equal(helpers.nextMessageCursor({ has_more: true, next_before_id: 12 }, 12), undefined);
      assert.equal(helpers.nextMessageCursor({ has_more: true, next_before_id: 13 }, 12), undefined);
      assert.equal(helpers.nextMessageCursor({ has_more: true, next_before_id: 0 }, 0), undefined);
    });

    await t.test('完整导出逐页到起点，坏游标和身份变化不下载残缺结果', async () => {
      const read = [];
      const message = id => ({ id, created_at: id });
      const result = await helpers.readEntireConversation(async before => {
        read.push(before);
        return { conversation: { title: '整个会话' }, messages: before ? [message(1), message(2)] : [message(3), message(4)], has_more: before === 0, next_before_id: before ? 0 : 3 };
      }, () => true, new AbortController().signal);
      assert.deepEqual(read, [0, 3]);
      assert.deepEqual(result.messages.map(row => row.id), [1, 2, 3, 4]);
      assert.equal(result.has_more, false);
      await assert.rejects(helpers.readEntireConversation(async () => ({ messages: [], has_more: true, next_before_id: 0 }), () => true, new AbortController().signal), /游标/);
      const stopped = new AbortController();
      stopped.abort();
      await assert.rejects(helpers.readEntireConversation(async () => { throw new Error('不应读取'); }, () => true, stopped.signal), { name: 'AbortError' });
      await assert.rejects(helpers.readEntireConversation(async () => ({ messages: [], has_more: false }), () => false, new AbortController().signal), { name: 'AbortError' });
      let current = true;
      await assert.rejects(helpers.readEntireConversation(async () => {
        current = false;
        return { messages: [message(1)], has_more: false, next_before_id: 0 };
      }, () => current, new AbortController().signal), { name: 'AbortError' });
    });

    await t.test('消息缓存按身份与会话隔离，删除时取消的迟到响应不能恢复缓存', async () => {
      const { conversationMessagesOptions, aiChatKeys } = await server.ssrLoadModule('/src/pages/ai/chatQueries.ts');
      const { request } = await server.ssrLoadModule('/src/utils/request.ts');
      const { useAuthStore } = await server.ssrLoadModule('/src/stores/auth.ts');
      useAuthStore.getState().setSession({ access_token: 'test-access', refresh_token: 'test-refresh', expires_at: Date.now() / 1000 + 3600, refresh_expires_at: Date.now() / 1000 + 7200, session_id: 'test-session' });
      const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
      const response = deferred();
      request.defaults.adapter = async config => {
        await response.promise;
        return { data: { code: 200, result: { conversation: { id: 7 }, messages: [{ id: 9 }], has_more: false, next_before_id: 0 } }, status: 200, statusText: 'OK', headers: {}, config };
      };
      const scope = ['test-session', 1, 1];
      const options = conversationMessagesOptions(scope, 7);
      const pending = client.fetchInfiniteQuery(options).catch(() => undefined);
      await Promise.resolve();
      await client.cancelQueries({ queryKey: options.queryKey });
      client.removeQueries({ queryKey: options.queryKey });
      response.resolve();
      await pending;
      assert.equal(client.getQueryData(options.queryKey), undefined);
      assert.notDeepEqual(aiChatKeys.messages(scope, 7), aiChatKeys.messages(scope, 8));
      assert.notDeepEqual(aiChatKeys.messages(scope, 7), aiChatKeys.messages(['other-session', 1, 1], 7));
      assert.notDeepEqual(aiChatKeys.messages(scope, 7), aiChatKeys.messages(['test-session', 1, 2], 7));
      client.clear();
    });

    await t.test('LLM正文与工具参数作为安全文本，缺失用量不声称免费或完整计费', async () => {
      const { MessageRecord, MessageDiagnostics } = await server.ssrLoadModule('/src/pages/ai/MessageRecord.tsx');
      const message = {
        id: 1, role: 'assistant', content: '<img src=x onerror=alert(1)>',
        tool_calls: JSON.stringify([{ id: 'tool-1', type: 'function', function: { name: '<script>evil</script>', arguments: '<svg onload=alert(1)>' } }]),
        tokens: 0, status: 'failed', error: '模型调用失败', created_at: 1,
      };
      const markup = renderToStaticMarkup(createElement(MessageRecord, { message })) + renderToStaticMarkup(createElement(MessageDiagnostics, { message }));
      assert.ok(markup.includes('&lt;img'));
      assert.ok(markup.includes('&lt;script&gt;evil'));
      assert.ok(markup.includes('&lt;svg'));
      assert.ok(!markup.includes('<img src=x'));
      assert.match(markup, /已知 token/);
      assert.match(markup, /用量可能不完整/);
      assert.match(markup, /模型调用失败/);
      assert.ok(!markup.includes('免费'));
    });

    await t.test('回答实际来源可展开且标题正文均转义', async () => {
      const { MessageRecord } = await server.ssrLoadModule('/src/pages/ai/MessageRecord.tsx');
      const markup = renderToStaticMarkup(createElement(MessageRecord, { message: { id: 1, role: 'assistant', status: 'succeeded', content: '**结论** [1]', tool_calls: '[]', tokens: 5, created_at: 1, error: '', feedback: '', feedback_note: '', sources: [{ chunk_id: 9, kb_id: 2, doc_id: 4, doc_title: '<script>标题</script>', snippet: '<img src=x onerror=evil()>原文', score: 0.7 }] } }));
      for (const text of ['参考资料', '查看原文', '本次引用片段', '<details>', '<strong>结论</strong>', '&lt;script&gt;标题', '&lt;img src=x', '分块 ID：9', '复制回答']) assert.ok(markup.includes(text), text);
      assert.doesNotMatch(markup, /<script>|<img src=x/);
      assert.doesNotMatch(markup, /已完成|已知 token/);
      assert.match(markup, /用量与调用详情/);
    });

    await t.test('刷新入口放在会话列表标题栏，不再占用独立页面标题区', async () => {
      const { AIChatAccess } = await server.ssrLoadModule('/src/pages/ai/AIChat.tsx');
      const { conversationListOptions } = await server.ssrLoadModule('/src/pages/ai/chatQueries.ts');
      const { useAuthStore } = await server.ssrLoadModule('/src/stores/auth.ts');
      const scope = ['test-session', 1, 1];
      const permCodes = ['sys:ai:conversation:list', 'sys:ai:conversation:create'];
      useAuthStore.getState().setPermCodes(permCodes);
      const client = new QueryClient();
      try {
        client.setQueryData(conversationListOptions(scope, 1).queryKey, { list: [], total: 0 });
        const markup = renderToStaticMarkup(createElement(QueryClientProvider, { client }, createElement(AIChatAccess,
          { scope, permissionsReady: true, permCodes })));
        assert.ok(!markup.includes('与 AI 助手对话，查看消息'), '不再渲染独立页面标题区');
        const title = markup.indexOf('我的会话');
        const refresh = markup.indexOf('aria-label="刷新会话"');
        assert.ok(title >= 0 && refresh > title && refresh < markup.indexOf('ant-card-body', title), '刷新按钮应位于会话列表标题栏');
      } finally { client.clear(); }
    });

    await t.test('权限未就绪或缺少查看码时页面不创建会话查询，授权不足显示403', async () => {
      const { AIChatAccess } = await server.ssrLoadModule('/src/pages/ai/AIChat.tsx');
      const client = new QueryClient();
      const render = permissionsReady => renderToStaticMarkup(createElement(QueryClientProvider, { client }, createElement(AIChatAccess,
        { scope: ['test-session', 1, 1], permissionsReady, permCodes: ['sys:ai:conversation:create'] })));
      assert.match(render(false), /AI 对话/);
      assert.equal(client.getQueryCache().getAll().length, 0);
      assert.match(render(true), /403/);
      assert.equal(client.getQueryCache().getAll().length, 0);
      client.clear();
    });

    await t.test('AI写请求在权限就绪前及撤权后被统一请求层拒绝', async () => {
      const { request } = await server.ssrLoadModule('/src/utils/request.ts');
      const { useAuthStore } = await server.ssrLoadModule('/src/stores/auth.ts');
      const { createAiConversation, deleteAiConversation } = await server.ssrLoadModule('/src/api/ai.ts');
      const { sendAiMessageStream } = await server.ssrLoadModule('/src/api/aiStream.ts');
      let calls = 0;
      request.defaults.adapter = async config => { calls += 1; return { data: { code: 200, result: {} }, status: 200, statusText: 'OK', headers: {}, config }; };
      useAuthStore.getState().clearPermissions();
      await assert.rejects(createAiConversation(), error => error.code === 403);
      useAuthStore.getState().setPermCodes(['sys:ai:conversation:create']);
      await createAiConversation();
      await assert.rejects(deleteAiConversation(7), error => error.code === 403);
      await assert.rejects(sendAiMessageStream({ request_id: '00000000-0000-4000-8000-000000000001', conversation_id: 7, content: 'test' }, {}), error => error.code === 403);
      assert.equal(calls, 1);
    });
  } finally {
    await server.close();
    globalThis.window = originalWindow;
  }
});
