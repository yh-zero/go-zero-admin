import test from 'node:test';
import assert from 'node:assert/strict';
import { createServer } from 'vite';
import { QueryClient, QueryObserver } from '@tanstack/react-query';

const storage = () => { const data = new Map(); return { getItem: key => data.get(key) ?? null, setItem: (key, value) => data.set(key, value), removeItem: key => data.delete(key) }; };
const message = (id, role, content = role) => ({ id, role, content, tool_calls: '', tokens: role === 'assistant' ? 7 : 0, status: 'succeeded', error: '', created_at: id });
const conversation = (id, count = 2, updated = 100) => ({ id, title: `会话 ${id}`, user_id: 1, message_count: count, created_at: 1, updated_at: updated });
const result = (id = 7) => ({ conversation_id: id, conversation: conversation(id), user_message: message(71, 'user', '原始消息'), ai_message: message(72, 'assistant', '权威答案') });
const meta = id => ({ conversation_id: id, user_message_id: 71, ai_message_id: 72 });
const frame = (event, data) => `event: ${event}\ndata: ${JSON.stringify(data)}\n\n`;
const stream = text => new Response(text, { headers: { 'Content-Type': 'text/event-stream' } });

test('AI流式权威结果与精确缓存回填', async t => {
  const originalWindow = globalThis.window;
  const originalFetch = globalThis.fetch;
  globalThis.window = { localStorage: storage(), sessionStorage: storage(), addEventListener() {} };
  const server = await createServer({ configFile: false, envFile: false, logLevel: 'error', server: { middlewareMode: true, watch: null, ws: false } });
  let client;
  try {
    const { useAuthStore } = await server.ssrLoadModule('/src/stores/auth.ts');
    const login = () => {
      useAuthStore.getState().setSession({ access_token: 'test-access', refresh_token: 'test-refresh', expires_at: Date.now() / 1000 + 3600, refresh_expires_at: Date.now() / 1000 + 7200, session_id: 'test-session' });
      useAuthStore.getState().setPermCodes(['sys:ai:message:send', 'sys:ai:conversation:list']);
    };
    login();
    const { sendAiMessageStream } = await server.ssrLoadModule('/src/api/aiStream.ts');
    const queries = await server.ssrLoadModule('/src/pages/ai/chatQueries.ts');
    const helpers = await server.ssrLoadModule('/src/pages/ai/chatState.ts');
    const input = { request_id: '00000000-0000-4000-8000-000000000001', conversation_id: 7, content: '原始消息' };
    const scope = ['test-session', 1, 1];
    const success = value => frame('meta', meta(value.conversation_id)) + frame('delta', { content: '临时增量' }) + frame('done', { tokens: 7, usage_complete: true, result: value });

    await t.test('done返回权威消息对，忽略增量草稿并释放流reader', async () => {
      const expected = result();
      let cancelled = false;
      globalThis.fetch = async () => new Response(new ReadableStream({ start(controller) { controller.enqueue(new TextEncoder().encode(success(expected))); }, cancel() { cancelled = true; } }), { headers: { 'Content-Type': 'text/event-stream' } });
      assert.deepEqual(await sendAiMessageStream(input, {}), expected);
      assert.equal(cancelled, true);
    });
    await t.test('error、缺done及终帧ID错配不能当成功或触发onDone', async () => {
      const invalid = [
        frame('meta', meta(7)) + frame('delta', { content: '不可信片段' }) + frame('error', { code: 503, msg: '模型不可用' }),
        frame('meta', meta(7)) + frame('delta', { content: '未完成' }),
        frame('meta', meta(7)) + frame('done', { tokens: 7, usage_complete: true }),
        frame('meta', meta(8)) + frame('done', { tokens: 7, usage_complete: true, result: result(7) }),
        frame('meta', meta(7)) + frame('done', { tokens: 7, usage_complete: true, result: { ...result(), ai_message: { ...message(73, 'assistant'), status: 'running' } } }),
      ];
      for (const text of invalid) {
        let called = false;
        globalThis.fetch = async () => stream(text);
        await assert.rejects(sendAiMessageStream(input, { onDone() { called = true; } }));
        assert.equal(called, false);
      }
    });
    await t.test('身份切换后的迟到终帧被拒绝', async () => {
      globalThis.fetch = async () => { login(); return stream(success(result())); };
      await assert.rejects(sendAiMessageStream(input, {}));
    });
    await t.test('三页历史发送只一次POST，保留三页游标并按ID替换重放', async () => {
      client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
      const key = queries.aiChatKeys.messages(scope, 7);
      client.setQueryData(key, { pages: [
        { conversation: conversation(7, 60), messages: [message(69, 'user'), message(70, 'assistant')], has_more: true, next_before_id: 69 },
        { conversation: conversation(7, 60), messages: [message(49, 'user'), message(50, 'assistant')], has_more: true, next_before_id: 49 },
        { conversation: conversation(7, 60), messages: [message(29, 'user'), message(30, 'assistant')], has_more: true, next_before_id: 29 },
      ], pageParams: [0, 69, 49] });
      client.setQueryData([...queries.aiChatKeys.lists(scope), 1], { list: [conversation(6, 2, 99), conversation(7, 60, 90)], total: 2 });
      const calls = [];
      globalThis.fetch = async (url, options) => { calls.push({ url, ...options }); return stream(success({ ...result(), conversation: conversation(7, 62) })); };
      const response = await sendAiMessageStream(input, {});
      assert.equal(typeof queries.applyAiSendResult, 'function');
      await queries.applyAiSendResult(client, scope, response, { request: input, isCurrent: () => true });
      await queries.applyAiSendResult(client, scope, response, { request: input, isCurrent: () => true });
      const data = client.getQueryData(key);
      assert.deepEqual(data.pageParams, [0, 69, 49]);
      assert.deepEqual(data.pages.map(page => page.next_before_id), [69, 49, 29]);
      assert.deepEqual(helpers.mergeMessagePages(data.pages).map(row => row.id), [29, 30, 49, 50, 69, 70, 71, 72]);
      assert.equal(data.pages[0].messages.at(-1).content, '权威答案');
      assert.deepEqual(client.getQueryData([...queries.aiChatKeys.lists(scope), 1]).list.map(row => row.id), [7, 6]);
      assert.equal(client.getQueryData([...queries.aiChatKeys.lists(scope), 1]).total, 2);
      assert.equal(calls.length, 1);
      assert.equal(calls[0].method, 'POST');
      client.clear();
    });
    await t.test('首次发送只一次POST，完整详情直接命中缓存；重查不重复累计total', async () => {
      client = new QueryClient();
      client.setQueryData([...queries.aiChatKeys.lists(scope), 1], { list: [], total: 0 });
      let calls = 0;
      globalThis.fetch = async () => { calls += 1; return stream(success(result())); };
      const request = { ...input, conversation_id: 0 };
      const response = await sendAiMessageStream(request, {});
      assert.equal(typeof queries.applyAiSendResult, 'function');
      await queries.applyAiSendResult(client, scope, response, { request, isCurrent: () => true });
      await queries.applyAiSendResult(client, scope, response, { request, isCurrent: () => true });
      const detail = await client.fetchInfiniteQuery(queries.conversationMessagesOptions(scope, 7));
      assert.equal(detail.pages[0].messages[1].content, '权威答案');
      assert.equal(detail.pages[0].has_more, false);
      assert.equal(client.getQueryData([...queries.aiChatKeys.lists(scope), 1]).total, 1);
      assert.equal(calls, 1);
      client.clear();
    });
    await t.test('已删除或身份失效结果不恢复消息缓存', async () => {
      client = new QueryClient();
      assert.equal(typeof queries.applyAiSendResult, 'function');
      await queries.applyAiSendResult(client, scope, result(), { request: input, isCurrent: () => false });
      assert.equal(client.getQueryData(queries.aiChatKeys.messages(scope, 7)), undefined);
      client.setQueryData(queries.aiChatKeys.messages(scope, 7), { pages: [], pageParams: [] });
      await queries.applyAiSendResult(client, scope, { ...result(), conversation: null }, { request: input, isCurrent: () => true });
      assert.equal(client.getQueryData(queries.aiChatKeys.messages(scope, 7)), undefined);
      client.clear();
    });
    await t.test('侧栏连续分页重新排序保留边界项目与真实total，重复回填不重复插入', async () => {
      client = new QueryClient();
      const listKey = queries.aiChatKeys.lists(scope);
      const rows = Array.from({ length: 40 }, (_, index) => conversation(40 - index, 2, 80 - index));
      client.setQueryData([...listKey, 1], { list: rows.slice(0, 20), total: 40 });
      client.setQueryData([...listKey, 2], { list: rows.slice(20), total: 40 });
      await queries.applyAiSendResult(client, scope, { ...result(5), conversation: conversation(5, 4, 100) }, { request: { ...input, conversation_id: 5 }, isCurrent: () => true });
      assert.deepEqual(client.getQueryData([...listKey, 1]).list.map(row => row.id), [5, 40, 39, 38, 37, 36, 35, 34, 33, 32, 31, 30, 29, 28, 27, 26, 25, 24, 23, 22]);
      assert.equal(client.getQueryData([...listKey, 2]).list[0].id, 21);
      const fresh = { ...result(41), conversation: conversation(41, 2, 101) };
      await queries.applyAiSendResult(client, scope, fresh, { request: { ...input, conversation_id: 0 }, isCurrent: () => true });
      await queries.applyAiSendResult(client, scope, fresh, { request: { ...input, conversation_id: 0 }, isCurrent: () => true });
      const cached = [1, 2, 3].flatMap(page => client.getQueryData([...listKey, page]).list);
      assert.equal(cached.length, 41);
      assert.equal(new Set(cached.map(row => row.id)).size, 41);
      assert.equal(client.getQueryData([...listKey, 3]).total, 41);
      assert.equal(client.getQueryData([...listKey, 3]).list[0].id, 1);
      client.clear();
    });
    await t.test('新会话断流后重查沿用原UUID、conversation_id=0及正文，meta保留且无自动重发', async () => {
      let calls = 0;
      const requests = [];
      globalThis.fetch = async (_url, options) => {
        calls += 1;
        requests.push(JSON.parse(options.body));
        return stream(calls === 1 ? frame('meta', meta(7)) + frame('delta', { content: '半截回复' }) : success(result()));
      };
      const sender = helpers.createMessageSender(sendAiMessageStream, () => input.request_id);
      const first = sender.start(0, input.content);
      assert.equal(sender.start(0, '双击'), null);
      await assert.rejects(first.promise);
      assert.equal(first.meta.conversation_id, 7);
      assert.equal(calls, 1);
      const recheck = sender.recheck();
      assert.equal(recheck.meta.conversation_id, 7);
      assert.deepEqual(await recheck.promise, result());
      assert.equal(calls, 2);
      assert.deepEqual(requests, [
        { request_id: input.request_id, conversation_id: 0, content: '原始消息' },
        { request_id: input.request_id, conversation_id: 0, content: '原始消息' },
      ]);
    });
    await t.test('从其他标签页删除后收到null摘要不会触发onDone之外的伪造会话', async () => {
      const deleted = { ...result(), conversation: null };
      globalThis.fetch = async () => stream(success(deleted));
      assert.deepEqual(await sendAiMessageStream(input, {}), deleted);
    });
    await t.test('无详情缓存的历史重放读取最新一页，不跳过原请求之后的新消息', async () => {
      client = new QueryClient();
      const { request } = await server.ssrLoadModule('/src/utils/request.ts');
      let reads = 0;
      request.defaults.adapter = async config => {
        reads += 1;
        return { data: { code: 200, result: { conversation: conversation(7, 4),
          messages: [message(71, 'user', input.content), message(72, 'assistant', '权威答案'), message(73, 'user', '后续问题'), message(74, 'assistant', '后续答案')], has_more: false, next_before_id: 0 } }, status: 200, statusText: 'OK', headers: {}, config };
      };
      await queries.applyAiSendResult(client, scope, { ...result(), conversation: conversation(7, 4) }, { request: { ...input, conversation_id: 0 }, isCurrent: () => true });
      const data = client.getQueryData(queries.aiChatKeys.messages(scope, 7));
      assert.deepEqual(helpers.mergeMessagePages(data.pages).map(row => row.id), [71, 72, 73, 74]);
      assert.equal(data.pages[0].has_more, false);
      assert.equal(reads, 1);
      client.clear();
    });
    await t.test('首条成功后保留新草稿，再次发送归属已创建会话，原请求重查仍保持0指纹', async () => {
      const requests = [];
      globalThis.fetch = async (_url, options) => {
        const request = JSON.parse(options.body);
        requests.push(request);
        const value = { ...result(), user_message: { ...result().user_message, content: request.content } };
        return stream(success(value));
      };
      let serial = 0;
      const sender = helpers.createMessageSender(sendAiMessageStream, () => `00000000-0000-4000-8000-${String(++serial).padStart(12, '0')}`);
      await sender.start(0, input.content).promise;
      await sender.recheck().promise;
      await sender.start(0, '等待期间编辑的新草稿').promise;
      assert.equal(requests[0].conversation_id, 0);
      assert.deepEqual(requests[1], requests[0]);
      assert.equal(requests[2].conversation_id, 7);
      assert.notEqual(requests[2].request_id, requests[0].request_id);
      assert.equal(requests[2].content, '等待期间编辑的新草稿');
    });
    await t.test('新建会话已收到meta但失败后，新意图使用已知会话而非另建会话', async () => {
      const requests = [];
      globalThis.fetch = async (_url, options) => {
        const request = JSON.parse(options.body);
        requests.push(request);
        return stream(frame('meta', meta(7)) + frame('error', { code: 503, msg: '调用失败' }));
      };
      const sender = helpers.createMessageSender(sendAiMessageStream);
      await assert.rejects(sender.start(0, input.content).promise);
      await assert.rejects(sender.start(0, '新的发送意图').promise);
      assert.equal(requests[1].conversation_id, 7);
      assert.notEqual(requests[0].request_id, requests[1].request_id);
    });
    await t.test('跨标签页消息数量跳增只查最新页补齐未知消息，已有连续历史游标保留', async () => {
      client = new QueryClient();
      const key = queries.aiChatKeys.messages(scope, 7);
      client.setQueryData(key, { pageParams: [0], pages: [{ conversation: conversation(7, 20),
        messages: Array.from({ length: 20 }, (_, index) => message(index + 1, index % 2 ? 'assistant' : 'user')),
        has_more: false, next_before_id: 0 }] });
      const { request } = await server.ssrLoadModule('/src/utils/request.ts');
      let reads = 0;
      request.defaults.adapter = async config => {
        reads += 1;
        return { data: { code: 200, result: { conversation: conversation(7, 24),
          messages: Array.from({ length: 20 }, (_, index) => message(index + 5, index % 2 ? 'assistant' : 'user')),
          has_more: true, next_before_id: 5 } }, status: 200, statusText: 'OK', headers: {}, config };
      };
      const response = { conversation_id: 7, conversation: conversation(7, 24), user_message: message(23, 'user'), ai_message: message(24, 'assistant') };
      await queries.applyAiSendResult(client, scope, response, { request: input, isCurrent: () => true });
      const data = client.getQueryData(key);
      assert.deepEqual(helpers.mergeMessagePages(data.pages).map(row => row.id), Array.from({ length: 24 }, (_, index) => index + 1));
      assert.equal(data.pages[0].has_more, false, '已有完整历史与最新页重叠，可保持完整覆盖');
      assert.equal(reads, 1);
      client.clear();
    });
    await t.test('跨标签页大跨度无法证明连续时重建最新页，游标允许重新加载缺口与旧历史', async () => {
      client = new QueryClient();
      const key = queries.aiChatKeys.messages(scope, 7);
      client.setQueryData(key, { pageParams: [0, 11], pages: [
        { conversation: conversation(7, 20), messages: Array.from({ length: 10 }, (_, index) => message(index + 11, 'user')), has_more: true, next_before_id: 11 },
        { conversation: conversation(7, 20), messages: Array.from({ length: 10 }, (_, index) => message(index + 1, 'user')), has_more: false, next_before_id: 0 },
      ] });
      const { request } = await server.ssrLoadModule('/src/utils/request.ts');
      let reads = 0;
      request.defaults.adapter = async config => {
        reads += 1;
        return { data: { code: 200, result: { conversation: conversation(7, 100), messages: Array.from({ length: 20 }, (_, index) => message(index + 81, 'user')),
          has_more: true, next_before_id: 81 } }, status: 200, statusText: 'OK', headers: {}, config };
      };
      await queries.applyAiSendResult(client, scope, { conversation_id: 7, conversation: conversation(7, 100), user_message: message(99, 'user'), ai_message: message(100, 'assistant') }, { request: input, isCurrent: () => true });
      const data = client.getQueryData(key);
      assert.equal(data.pages.length, 1);
      assert.deepEqual(data.pageParams, [0]);
      assert.equal(data.pages[0].next_before_id, 81);
      assert.equal(data.pages[0].has_more, true);
      assert.equal(reads, 1);
      client.clear();
    });
    await t.test('发送前已在途的侧栏读取不能覆盖成功摘要', async () => {
      client = new QueryClient();
      const key = [...queries.aiChatKeys.lists(scope), 1];
      client.setQueryData(key, { list: [conversation(7, 2, 90)], total: 1 });
      let release;
      const pending = client.fetchQuery({ queryKey: key, queryFn: () => new Promise(resolve => { release = resolve; }) }).catch(() => undefined);
      await Promise.resolve();
      await queries.applyAiSendResult(client, scope, { ...result(), conversation: conversation(7, 2, 100) }, { request: input, isCurrent: () => true });
      release({ list: [conversation(7, 0, 1)], total: 1 });
      await pending;
      assert.equal(client.getQueryData(key).list[0].updated_at, 100);
      assert.equal(client.getQueryData(key).list[0].message_count, 2);
      client.clear();
    });
    await t.test('发送后切回侧栏第一页直接使用已更新缓存，不立即GET列表', async () => {
      client = new QueryClient();
      const key = [...queries.aiChatKeys.lists(scope), 1];
      client.setQueryData(key, { list: [conversation(7)], total: 1 });
      const { request } = await server.ssrLoadModule('/src/utils/request.ts');
      let reads = 0;
      request.defaults.adapter = async config => {
        reads += 1;
        return { data: { code: 200, result: { list: [conversation(7)], total: 1 } }, status: 200, statusText: 'OK', headers: {}, config };
      };
      await queries.applyAiSendResult(client, scope, result(), { request: input, isCurrent: () => true });
      await client.fetchQuery(queries.conversationListOptions(scope, 1));
      assert.equal(reads, 0);
      client.clear();
    });
    await t.test('重查新增会话成员关系不确定时保留服务端total并标记其他侧栏页待刷新', async () => {
      client = new QueryClient();
      const prefix = queries.aiChatKeys.lists(scope);
      client.setQueryData([...prefix, 1], { list: Array.from({ length: 20 }, (_, index) => conversation(index + 100)), total: 60 });
      client.setQueryData([...prefix, 3], { list: Array.from({ length: 20 }, (_, index) => conversation(index + 200)), total: 60 });
      await queries.applyAiSendResult(client, scope, result(), { request: { ...input, conversation_id: 0 }, isCurrent: () => true });
      assert.equal(client.getQueryData([...prefix, 1]).total, 60);
      assert.equal(client.getQueryState([...prefix, 1]).isInvalidated, true);
      assert.equal(client.getQueryState([...prefix, 3]).isInvalidated, true);
      client.clear();
    });
    await t.test('已知会话失败恢复同步侧栏消息数，只读取一次详情', async () => {
      client = new QueryClient();
      const listKey = [...queries.aiChatKeys.lists(scope), 1];
      client.setQueryData(listKey, { list: [conversation(7, 4, 90)], total: 1 });
      const { request } = await server.ssrLoadModule('/src/utils/request.ts');
      const calls = [];
      request.defaults.adapter = async config => {
        calls.push(config.url);
        return { data: { code: 200, result: { conversation: conversation(7, 6, 101),
          messages: [message(71, 'user'), message(72, 'assistant'), { ...message(73, 'user'), status: 'failed', error: '调用失败' },
            { ...message(74, 'assistant', ''), status: 'failed', error: '调用失败' }], has_more: false, next_before_id: 0 } }, status: 200, statusText: 'OK', headers: {}, config };
      };
      await queries.recoverAiSend(client, scope, 7, () => true);
      assert.equal(client.getQueryData(listKey).list[0].message_count, 6);
      assert.equal(client.getQueryData(listKey).list[0].updated_at, 101);
      assert.equal(client.getQueryData(listKey).total, 1);
      assert.equal(client.getQueryData(queries.aiChatKeys.messages(scope, 7)).pages[0].messages.at(-1).status, 'failed');
      assert.deepEqual(calls, ['/sys/ai/conversations/7']);
      client.clear();
    });
    await t.test('未知失败新会话只定点读取第一页确认total并出现在侧栏', async () => {
      client = new QueryClient();
      const listKey = [...queries.aiChatKeys.lists(scope), 1];
      client.setQueryData(listKey, { list: [], total: 0 });
      const observer = new QueryObserver(client, queries.conversationListOptions(scope, 1));
      const unsubscribe = observer.subscribe(() => undefined);
      try {
        const { request } = await server.ssrLoadModule('/src/utils/request.ts');
        const calls = [];
        request.defaults.adapter = async config => {
          calls.push(config.url);
          const data = config.url === '/sys/ai/conversations/7'
            ? { conversation: conversation(7, 2), messages: [{ ...message(71, 'user'), status: 'failed', error: '调用失败' },
              { ...message(72, 'assistant', ''), status: 'failed', error: '调用失败' }], has_more: false, next_before_id: 0 }
            : { list: [conversation(7, 2)], total: 1 };
          return { data: { code: 200, result: data }, status: 200, statusText: 'OK', headers: {}, config };
        };
        await queries.recoverAiSend(client, scope, 7, () => true);
        assert.equal(client.getQueryData(listKey).list[0]?.id, 7);
        assert.equal(client.getQueryData(listKey).total, 1);
        assert.deepEqual(calls, ['/sys/ai/conversations/7', '/sys/ai/conversations']);
      } finally { unsubscribe(); client.clear(); }
    });
  } finally {
    client?.clear();
    await server.close();
    globalThis.window = originalWindow;
    globalThis.fetch = originalFetch;
  }
});
