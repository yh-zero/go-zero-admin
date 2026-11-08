import test from 'node:test';
import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { mkdtemp, writeFile, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';
import * as quality from './eval-kb-quality.mjs';
import { customRaw, invalidCustomSamples, validCustomVariants } from './kb-custom-dataset.test-fixtures.mjs';

const digest = value => createHash('sha256').update(value).digest('hex');
function fixture() {
  const content = '文档甲：变更前备份，并核对权限。';
  return { version: 1, description: '反馈冻结回归', documents: [{ key: 'doc_7', kb: 'feedback_5', source: 'inline:doc_7', source_doc_id: 7, source_doc_version: 2, title: '同名文档', content, sha256: digest(content) }],
    cases: [{ id: 'feedback_9_r2', kb: 'feedback_5', split: 'development', category: 'feedback', query: '变更前做什么？', rationale: '采纳反馈生成，不代表质量签署', expected_documents: ['doc_7'], evidence: [{ document: 'doc_7', groups: [['变更前备份']] }], answer: { kind: 'answerable', required_groups: [['备份']], forbidden_phrases: [] } }] };
}
async function withDataset(value, action) {
  const folder = await mkdtemp(join(tmpdir(), 'gozero-inline-dataset-'));
  try { const filename = join(folder, 'cases.json'); await writeFile(filename, JSON.stringify(value)); return await action(filename); }
  finally { const target = resolve(folder); if (!target.startsWith(join(resolve(tmpdir()), 'gozero-inline-dataset-'))) throw new Error('临时目录越界'); await rm(target, { recursive: true, force: true }); }
}
test('CLI可验证内联原文，不把标记当文件路径读取', async () => {
  const value = fixture();
  await withDataset(value, async path => assert.deepEqual(await quality.loadDataset(path), value));
});
test('内联SHA、文件标记、ID版本和非法Unicode错误拒绝', async () => {
  const bad = [doc => { doc.sha256 = '0'.repeat(64); }, doc => { doc.source = '../../private.txt'; }, doc => { doc.source_doc_id = 0; }, doc => { doc.source_doc_version = 0; }, doc => { doc.content = '\ud800'; doc.sha256 = digest(doc.content); }];
  for (const mutate of bad) { const value = fixture(); mutate(value.documents[0]); await withDataset(value, path => assert.rejects(quality.loadDataset(path))); }
});
test('相同标题按显式文档ID及版本绑定，旧资料继续按唯一标题', () => {
  const rows = [{ id: 7, title: '同名文档', version: 2 }, { id: 8, title: '同名文档', version: 2 }];
  assert.deepEqual(quality.matchSourceDocuments?.(fixture().documents[0], rows), [rows[0]]);
  assert.deepEqual(quality.matchSourceDocuments?.({ title: '同名文档' }, rows), rows);
  assert.deepEqual(quality.matchSourceDocuments?.({ ...fixture().documents[0], source_doc_version: 1 }, rows), []);
});
test('绑定仅保留资料标识，不复制内联全文；错误冻结ID在读取前拒绝', async () => {
  const source = fixture().documents[0], listed = { id: 7, kb_id: 5, title: source.title, version: 2, embedding_model: 'fixture' };
  let reads = 0;
  const client = async () => { reads++; return { doc: { ...listed, status: 'ready' }, content: source.content }; };
  const binding = await quality.readBoundDocument(source, listed, client);
  assert.equal(Object.hasOwn(binding, 'content'), false);
  assert.equal(binding.content_sha256, source.sha256);
  await assert.rejects(quality.readBoundDocument(source, { ...listed, id: 8 }, client));
  assert.equal(reads, 1);
});

test('CLI自定义快照与Go共同合法样本及坏样本保持一致', async t => {
  for (const [index, raw] of validCustomVariants().entries()) await t.test(`合法${index}`, async () => {
    await withDataset(JSON.parse(customRaw), async filename => { await writeFile(filename, raw); assert.deepEqual(await quality.loadDataset(filename), JSON.parse(raw)); });
  });
  for (const [name, raw] of invalidCustomSamples()) await t.test(name, async () => {
    await withDataset(JSON.parse(customRaw), async filename => { await writeFile(filename, raw); await assert.rejects(quality.loadDataset(filename)); });
  });
  await t.test('非法原始UTF8', async () => {
    await withDataset(JSON.parse(customRaw), async filename => { await writeFile(filename, Buffer.from(customRaw.replace('预算和保存期是什么？', 'broken\ufffdquery')).map(byte => byte === 0xef ? 0xff : byte)); await assert.rejects(quality.loadDataset(filename)); });
  });
});
