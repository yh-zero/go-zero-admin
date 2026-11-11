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
