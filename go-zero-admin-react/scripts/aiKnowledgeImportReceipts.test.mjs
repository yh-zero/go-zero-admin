import test from 'node:test';
import assert from 'node:assert/strict';
import { createServer } from 'vite';

test('迟到导入响应不能恢复已放弃资料、复活已确认回执或覆盖新批次', async t => {
  const server = await createServer({ configFile: false, envFile: false, logLevel: 'error', optimizeDeps: { noDiscovery: true }, server: { middlewareMode: true, watch: null, ws: false } });
  try {
    const imports = await server.ssrLoadModule('/src/pages/ai/knowledgeImport.ts');
    assert.equal(typeof imports.updateKnowledgeImportAttempt, 'function');
    const makeStorage = () => { const values = new Map(); return { getItem: key => values.get(key) ?? null, setItem: (key, value) => values.set(key, value), removeItem: key => values.delete(key) }; };
    const prepare = () => imports.prepareKnowledgeBatch(Array.from({ length: 4 }, (_, index) => ({ title: `${index}`, content: '正文' })));
    await t.test('导入离页保留已发出未知项，确定失败和未发出项不会混进未知锁', async () => {
      const { ApiError } = await server.ssrLoadModule('/src/utils/response.ts');
      const prepared = prepare();
      let current = true;
      let releaseFirst;
      const first = new Promise(resolve => { releaseFirst = resolve; });
      let attempt = { batchId: prepared.batchId, requestIds: [], unsubmittedIds: prepared.items.map(item => item.id) };
      const sent = [];
      await imports.runKnowledgeImport(prepared.items, async values => {
        sent.push(values.request_id);
        if (values.title === '0') { await first; throw new Error('network outcome unknown'); }
        current = false; releaseFirst(); throw new ApiError(400, '确定未接收');
      }, () => {}, () => current, change => { attempt = imports.advanceKnowledgeImportAttempt(attempt, change); });
      assert.equal(sent.length, 2);
      assert.deepEqual(attempt.requestIds, [prepared.items[0].id]);
      assert.deepEqual(attempt.unsubmittedIds, prepared.items.slice(2).map(item => item.id));
      const storage = makeStorage();
      imports.writeKnowledgeImportAttempt(storage, 'original', attempt);
      assert.deepEqual(imports.readKnowledgeImportAttempt(storage, 'original'), attempt);
    });
    await t.test('未确认导入保留原批次和请求标识，恢复时按身份隔离', () => {
      const batch = prepare();
      const pending = { batchId: batch.batchId, requestIds: batch.items.map(item => item.id) };
      const storage = makeStorage();
      imports.writeKnowledgeImportAttempt(storage, 'scope-a:kb1', pending);
      assert.deepEqual(imports.readKnowledgeImportAttempt(storage, 'scope-a:kb1'), pending);
      assert.equal(imports.readKnowledgeImportAttempt(storage, 'scope-b:kb1'), null);
    });
    async function start(storage, key) {
      const prepared = prepare();
      let current = true;
      const deferred = [];
      imports.writeKnowledgeImportAttempt(storage, key, { batchId: prepared.batchId, requestIds: [], unsubmittedIds: prepared.items.map(item => item.id) });
      const running = imports.runKnowledgeImport(prepared.items, values => new Promise((resolve, reject) => { deferred.push({ id: values.request_id, resolve, reject }); }), () => {}, () => current,
        change => imports.updateKnowledgeImportAttempt(storage, key, prepared.batchId, latest => imports.advanceKnowledgeImportAttempt(latest, change)));
      assert.equal(deferred.length, 2);
      current = false;
      return { prepared, deferred, running };
    }
    await t.test('重开后放弃未发出项，旧请求未知/成功迟到都保留最新放弃决定', async () => {
      const storage = makeStorage();
      const { prepared, deferred, running } = await start(storage, 'same-session-kb');
      const reopened = imports.readKnowledgeImportAttempt(storage, 'same-session-kb');
      assert.deepEqual(reopened.unsubmittedIds, prepared.items.slice(2).map(item => item.id));
      imports.updateKnowledgeImportAttempt(storage, 'same-session-kb', reopened.batchId, latest => ({ ...latest, unsubmittedIds: [] }));
      deferred[0].reject(new Error('network result unknown'));
      deferred[1].resolve({ doc: { id: 9, status: 'pending' } });
      await running;
      const remaining = imports.readKnowledgeImportAttempt(storage, 'same-session-kb');
      assert.deepEqual(remaining.unsubmittedIds, []);
      assert.deepEqual(remaining.requestIds, [prepared.items[0].id]);
    });
    await t.test('GET确认旧批次后启动新批次，旧POST迟到不能覆盖新未知ID', async () => {
      const storage = makeStorage();
      const { prepared, deferred, running } = await start(storage, 'same-session-kb');
      imports.updateKnowledgeImportAttempt(storage, 'same-session-kb', prepared.batchId, latest => ({ ...latest, requestIds: [], unsubmittedIds: [] }));
      assert.equal(imports.readKnowledgeImportAttempt(storage, 'same-session-kb'), null);
      const next = prepare();
      const newer = { batchId: next.batchId, requestIds: [next.items[0].id], unsubmittedIds: [] };
      imports.writeKnowledgeImportAttempt(storage, 'same-session-kb', newer);
      deferred[0].reject(new Error('old network result unknown'));
      deferred[1].resolve({ doc: { id: 9, status: 'pending' } });
      await running;
      assert.deepEqual(imports.readKnowledgeImportAttempt(storage, 'same-session-kb'), newer);
      assert.equal(imports.updateKnowledgeImportAttempt(storage, 'same-session-kb', prepared.batchId, () => { throw new Error('must not update another batch'); }).batchId, newer.batchId);
    });
    await t.test('同批次已有部分ID获GET确认，旧未知响应不能把这些ID重新加入锁', async () => {
      const storage = makeStorage();
      const prepared = prepare();
      imports.writeKnowledgeImportAttempt(storage, 'partially-confirmed', { batchId: prepared.batchId, requestIds: prepared.items.slice(0, 2).map(item => item.id), unsubmittedIds: prepared.items.slice(2).map(item => item.id) });
      imports.updateKnowledgeImportAttempt(storage, 'partially-confirmed', prepared.batchId, latest => ({ ...latest, requestIds: latest.requestIds.filter(id => id !== prepared.items[0].id) }));
      imports.updateKnowledgeImportAttempt(storage, 'partially-confirmed', prepared.batchId, latest => imports.advanceKnowledgeImportAttempt(latest, { id: prepared.items[0].id, state: 'unknown' }));
      assert.deepEqual(imports.readKnowledgeImportAttempt(storage, 'partially-confirmed').requestIds, [prepared.items[1].id]);
    });
    await t.test('空回执不会被旧未知响应复活，存储失败仍使用最新共享内存决定', () => {
      const values = new Map();
      const storage = { getItem: key => values.get(key) ?? null, setItem() { throw new Error('QuotaExceeded'); }, removeItem() { throw new Error('storage unavailable'); } };
      const prepared = prepare();
      const original = { batchId: prepared.batchId, requestIds: [prepared.items[0].id], unsubmittedIds: prepared.items.slice(2).map(item => item.id) };
      assert.throws(() => imports.writeKnowledgeImportAttempt(storage, 'quota', original), /Quota/);
      assert.deepEqual(imports.readKnowledgeImportAttempt(storage, 'quota'), original);
      assert.throws(() => imports.updateKnowledgeImportAttempt(storage, 'quota', original.batchId, latest => ({ ...latest, unsubmittedIds: [] })), /Quota/);
      assert.throws(() => imports.updateKnowledgeImportAttempt(storage, 'quota', original.batchId, latest => imports.advanceKnowledgeImportAttempt(latest, { id: prepared.items[0].id, state: 'unknown' })), /Quota/);
      assert.deepEqual(imports.readKnowledgeImportAttempt(storage, 'quota').unsubmittedIds, []);
      const next = prepare();
      const newer = { batchId: next.batchId, requestIds: [next.items[0].id], unsubmittedIds: [] };
      assert.throws(() => imports.writeKnowledgeImportAttempt(storage, 'quota', newer), /Quota/);
      imports.updateKnowledgeImportAttempt(storage, 'quota', original.batchId, () => { throw new Error('old batch must not replace fallback'); });
      assert.deepEqual(imports.readKnowledgeImportAttempt(storage, 'quota'), newer);
      assert.throws(() => imports.clearKnowledgeImportAttempt(storage, 'quota'), /unavailable/);
      assert.equal(imports.updateKnowledgeImportAttempt(storage, 'quota', next.batchId, () => newer), null);
      assert.equal(imports.readKnowledgeImportAttempt(storage, 'quota'), null);
    });
  } finally { await server.close(); }
});
