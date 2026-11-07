import test from 'node:test';
import assert from 'node:assert/strict';
import { createServer } from 'vite';
import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';

test('对话展示与知识范围', async t => {
  const server = await createServer({ configFile: false, envFile: false, logLevel: 'error', optimizeDeps: { noDiscovery: true }, server: { middlewareMode: true, watch: null, ws: false } });
  try {
    const { SafeMarkdown } = await server.ssrLoadModule('/src/pages/ai/SafeMarkdown.tsx');
    const render = content => renderToStaticMarkup(createElement(SafeMarkdown, { content }));
    await t.test('可读表格、粗体、代码与外链', () => {
      const markup = render('# 结论\n**重点**\n| 名称 | 数量 |\n| --- | --- |\n| 实例 | 2 |\n```js\nconst text = "<img>";\n```\n[资料](https://example.test/doc)');
      for (const text of ['<table>', '<th>名称</th>', '<td>2</td>', '<strong>重点</strong>', '复制代码', '&lt;img&gt;', 'rel="noopener noreferrer"']) assert.ok(markup.includes(text), text);
    });
    await t.test('恶意HTML、图片和危险链接不会成为活动内容', () => {
      const markup = render('<script>alert(1)</script>\n<img src=x onerror=evil()>\n[危险](javascript:evil)\n![远程图片](https://example.test/spy)\n[危险](data:text/html,evil)');
      assert.match(markup, /&lt;script&gt;/);
      assert.doesNotMatch(markup, /<script|<img|href="javascript:|href="data:|href="https:\/\/example.test\/spy/);
    });
    const helpers = await server.ssrLoadModule('/src/pages/ai/chatState.ts');
    await t.test('资料范围跟随原请求重查，超限问题不创建请求', async () => {
      const calls = [];
      let serial = 0;
      const sender = helpers.createMessageSender(async request => {
        calls.push(request);
        if (calls.length === 1) throw new Error('结果未知');
        return { conversation_id: 8, conversation: { id: 8 } };
      }, () => `request-${++serial}`);
      assert.equal(sender.start(0, '中'.repeat(667), {}, { kb_id: 2, grounded: true }), null);
      assert.equal(sender.start(0, '问题', {}, { kb_id: 0, grounded: true }), null);
      const first = sender.start(0, '问题', {}, { kb_id: 2, grounded: true });
      await assert.rejects(first.promise);
      await sender.recheck().promise;
      assert.deepEqual(calls[1], calls[0]);
      assert.equal(calls[1].kb_id, 2);
      assert.equal(calls[1].grounded, true);
      await sender.start(8, '问题', {}, { kb_id: 3, grounded: false }).promise;
      assert.equal(calls[2].kb_id, 3);
      assert.notEqual(calls[0].request_id, calls[2].request_id);
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
      const stopped = new AbortController(); stopped.abort();
      await assert.rejects(helpers.readEntireConversation(async () => { throw new Error('不应读取'); }, () => true, stopped.signal), { name: 'AbortError' });
      await assert.rejects(helpers.readEntireConversation(async () => ({ messages: [], has_more: false }), () => false, new AbortController().signal), { name: 'AbortError' });
    });
    await t.test('回答实际来源可展开且标题正文均转义', async () => {
      const { MessageRecord } = await server.ssrLoadModule('/src/pages/ai/MessageRecord.tsx');
      const markup = renderToStaticMarkup(createElement(MessageRecord, { message: { id: 1, role: 'assistant', status: 'succeeded', content: '**结论** [1]', tool_calls: '[]', tokens: 5, created_at: 1, error: '', feedback: '', feedback_note: '', sources: [{ chunk_id: 9, kb_id: 2, doc_id: 4, doc_title: '<script>标题</script>', snippet: '<img src=x onerror=evil()>原文', score: 0.7 }] } }));
      for (const text of ['参考资料（实际检索片段）', '<details>', '<strong>结论</strong>', '&lt;script&gt;标题', '&lt;img src=x', '分块 ID：9', '复制回答']) assert.ok(markup.includes(text), text);
      assert.doesNotMatch(markup, /<script>|<img src=x/);
    });
  } finally { await server.close(); }
});
