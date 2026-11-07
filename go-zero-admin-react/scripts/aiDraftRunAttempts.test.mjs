import test from 'node:test';
import assert from 'node:assert/strict';
import { createServer } from 'vite';

test('草稿运行标识在浏览器存储异常后保留，离页恢复不会重复提交', async t => {
  const server = await createServer({ configFile: false, envFile: false, logLevel: 'error', optimizeDeps: { noDiscovery: true }, server: { middlewareMode: true, watch: null, ws: false } });
  try {
    const attempts = await server.ssrLoadModule('/src/pages/ai/draftRunAttempts.ts');
    await t.test('存储配额不足后返回仍恢复原定义、输入和UUID，并保持身份隔离', () => {
      const storage = { getItem: () => null, setItem() { throw new Error('QuotaExceeded'); }, removeItem() {} };
      const attempt = { name: '原分析', definition: '{"version":2,"steps":[]}', input: '{"kb_id":7,"question":"原问题"}', request_id: '11111111-1111-4111-8111-111111111111' };
      assert.throws(() => attempts.writeDraftRunAttempt(storage, 'scope-a:quota', attempt), /QuotaExceeded/);
      assert.deepEqual(attempts.readDraftRunAttempt(storage, 'scope-a:quota'), attempt);
      assert.equal(attempts.readDraftRunAttempt(storage, 'scope-b:quota'), null);
    });
    await t.test('存储暂不可读仍保留原请求，清理失败也不恢复已确认的旧请求', () => {
      const values = new Map();
      let unreadable = false;
      const storage = { getItem(key) { if (unreadable) throw new Error('storage unavailable'); return values.get(key) ?? null; }, setItem: (key, value) => values.set(key, value), removeItem() { throw new Error('remove unavailable'); } };
      const attempt = { name: '原分析', definition: '{"version":2,"steps":[]}', input: '{"text":"原输入"}', request_id: '22222222-2222-4222-8222-222222222222' };
      attempts.writeDraftRunAttempt(storage, 'scope-a:unreadable', attempt);
      unreadable = true;
      assert.deepEqual(attempts.readDraftRunAttempt(storage, 'scope-a:unreadable'), attempt);
      unreadable = false;
      assert.throws(() => attempts.clearDraftRunAttempt(storage, 'scope-a:unreadable'), /remove unavailable/);
      assert.equal(attempts.readDraftRunAttempt(storage, 'scope-a:unreadable'), null);
    });
  } finally { await server.close(); }
});
