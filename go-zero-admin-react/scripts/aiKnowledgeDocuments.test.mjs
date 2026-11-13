import test from 'node:test';
import assert from 'node:assert/strict';
import { createServer } from 'vite';

test('批量资料导入保留原文并限制类型、UTF-8字节和并发', async t => {
  const server = await createServer({ configFile: false, envFile: false, logLevel: 'error', server: { middlewareMode: true, watch: null, ws: false } });
  try {
    const files = await server.ssrLoadModule('/src/pages/ai/knowledgeImport.ts').catch(() => ({}));
    await t.test('导入批次在提交前分配唯一请求标识，最多20篇且终态停止轮询', () => {
      const batch = files.prepareKnowledgeBatch([{ title: '甲', content: '正文' }, { title: '乙', content: '正文' }]);
      assert.equal(new Set(batch.items.map(item => item.values.request_id)).size, 2);
      assert.ok(batch.items.every(item => item.values.batch_id === batch.batchId));
      assert.equal(files.knowledgeBatchPollInterval({ queued: 1, running: 0 }, true), 3000);
      assert.equal(files.knowledgeBatchPollInterval({ queued: 0, running: 0 }, true), false);
      assert.equal(files.knowledgeBatchPollInterval({ queued: 1, running: 1 }, false), false);
      assert.throws(() => files.prepareKnowledgeBatch(Array.from({ length: 21 }, () => ({ title: '甲', content: '正文' }))), /20/);
    });
    await t.test('TXT/Markdown原始CRLF和BOM保留，标题使用文件名', async () => {
      assert.equal(typeof files.readKnowledgeFile, 'function');
      const body = '\uFEFF第一行\r\n第二行\r\n';
      const parsed = await files.readKnowledgeFile(new File([body], '发布手册.md'));
      assert.deepEqual(parsed, { title: '发布手册', content: body });
      assert.equal((await files.readKnowledgeFile(new File(['a'.repeat(65536)], '边界.TXT'))).content.length, 65536);
    });
    await t.test('超字节、非UTF8、空白、不支持格式和过长标题在提交前拒绝', async () => {
      assert.equal(typeof files.readKnowledgeFile, 'function');
      for (const file of [new File(['中'.repeat(21846)], '超限.txt'), new File([new Uint8Array([0xff, 0xfe])], '非UTF8.txt'),
        new File([' \r\n\t'], '空白.md'), new File(['PDF'], '文档.pdf'), new File(['正文'], `${'长'.repeat(129)}.md`)]) {
        await assert.rejects(files.readKnowledgeFile(file));
      }
    });
    await t.test('业务400/403/404是确定失败，网络和5xx保留未知状态，均不自动重放', async () => {
      const documents = await server.ssrLoadModule('/src/pages/ai/knowledgeDocuments.ts');
      const { ApiError } = await server.ssrLoadModule('/src/utils/response.ts');
      for (const code of [400, 403, 404]) assert.deepEqual(documents.documentWriteFailure(new ApiError(code, '请刷新版本')), { uncertain: false, message: '请刷新版本' });
      assert.equal(documents.documentWriteFailure(new ApiError(503, '暂时失败')).uncertain, true);
      assert.equal(documents.documentWriteFailure(new Error('network')).uncertain, true);
      const changes = [];
      let calls = 0;
      await files.runKnowledgeImport([{ id: 'one', values: { title: '标题', content: '正文' } }], async () => {
        calls++; throw new ApiError(400, '文件参数不符合要求');
      }, change => changes.push(change), () => true);
      assert.equal(calls, 1);
      assert.equal(changes.at(-1).state, 'failed');
      assert.equal(changes.at(-1).error, '文件参数不符合要求');
    });
    await t.test('最多两次并行，失败不重放且每项结果独立，旧身份停止新提交', async () => {
      assert.equal(typeof files.runKnowledgeImport, 'function');
      const rows = Array.from({ length: 6 }, (_, i) => ({ id: `${i}`, values: { title: `${i}`, content: '正文' } }));
      let active = 0; let maxActive = 0; let current = true;
      const calls = []; const changes = [];
      await files.runKnowledgeImport(rows, async values => {
        active++; maxActive = Math.max(maxActive, active); calls.push(values.title);
        await new Promise(resolve => setTimeout(resolve, 5));
        active--;
        if (values.title === '1') throw new Error('network outcome unknown');
        if (values.title === '2') current = false;
        return { doc: { id: Number(values.title) + 1, status: 'ready' } };
      }, change => changes.push(change), () => current);
      assert.equal(maxActive, 2);
      assert.equal(calls.filter(value => value === '1').length, 1);
      assert.ok(calls.length < 6);
      assert.ok(changes.some(value => value.id === '1' && value.state === 'unknown'));
    });
  } finally { await server.close(); }
});
