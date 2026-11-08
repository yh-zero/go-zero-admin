import test from 'node:test';
import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { decodeCustomDataset, parseCustomDataset } from './kb-custom-dataset.mjs';
import { customRaw, invalidCustomSamples, validCustomVariants } from './kb-custom-dataset.test-fixtures.mjs';

test('共享解析器保留原文且对同一批合法及坏样本执行严格校验', async () => {
  for (const raw of validCustomVariants()) assert.deepEqual(await parseCustomDataset(raw), JSON.parse(raw));
  for (const [, raw] of invalidCustomSamples()) await assert.rejects(parseCustomDataset(raw));
  const raw = ` \r\n${customRaw}\r\n`;
  assert.equal(decodeCustomDataset(new TextEncoder().encode(raw)), raw);
  assert.deepEqual(await parseCustomDataset(raw), JSON.parse(customRaw));
});

test('字节入口拒绝BOM与坏UTF8，字符串入口也不能悄悄替换孤立代理项', async () => {
  assert.throws(() => decodeCustomDataset(Buffer.concat([Buffer.from([0xef, 0xbb, 0xbf]), Buffer.from(customRaw)])), /BOM/u);
  assert.throws(() => decodeCustomDataset(new Uint8Array([0xff])), /UTF-8/u);
  await assert.rejects(parseCustomDataset('\ufeff' + customRaw));
  await assert.rejects(parseCustomDataset(customRaw.replace('预算和保存期是什么？', '\ud800')));
  const paired = customRaw.replace('预算和保存期是什么？', '资料😀是什么？');
  assert.equal((await parseCustomDataset(paired)).cases[0].query, '资料😀是什么？');
});

test('快照256KiB、100篇与100题、正文64KiB的上限本身可用', async () => {
  const padded = bytes => customRaw + ' '.repeat(bytes - Buffer.byteLength(customRaw));
  await parseCustomDataset(padded(262144));
  await assert.rejects(parseCustomDataset(padded(262145)));
  const value = JSON.parse(customRaw);
  value.documents = [value.documents[0], ...Array.from({ length: 99 }, (_, i) => ({ ...value.documents[0], key: `d${i}`, source: `inline:d${i}`, source_doc_id: i + 100 }))];
  value.cases = Array.from({ length: 100 }, (_, i) => ({ ...value.cases[0], id: `c${i}` }));
  assert.equal((await parseCustomDataset(JSON.stringify(value))).documents.length, 100);
  value.cases.push({ ...value.cases[0], id: 'extra' });
  await assert.rejects(parseCustomDataset(JSON.stringify(value)));
  const document = JSON.parse(customRaw); document.documents[0].content = document.documents[0].content.padEnd(65536 - Buffer.byteLength(document.documents[0].content) + document.documents[0].content.length, 'x');
  document.documents[0].sha256 = createHash('sha256').update(document.documents[0].content).digest('hex');
  assert.equal(Buffer.byteLength((await parseCustomDataset(JSON.stringify(document))).documents[0].content), 65536);
});

test('Go按root为0计算深度，metadata数字不增加业务整数约束', async () => {
  const value = JSON.parse(customRaw); let nested = true;
  for (let i = 0; i < 62; i++) nested = [nested];
  value.metadata = { nested };
  await parseCustomDataset(JSON.stringify(value));
  value.metadata.nested = [nested];
  await assert.rejects(parseCustomDataset(JSON.stringify(value)));
  const large = customRaw.replace('"metadata": {', '"metadata": {"unbounded_number":1e10000,');
  assert.equal((await parseCustomDataset(large)).metadata.unbounded_number, Infinity);
});
