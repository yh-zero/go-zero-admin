import test from 'node:test';
import assert from 'node:assert/strict';
import { validateAnalysis } from './eval-kb-quality-metrics.mjs';
import { validateHits } from './eval-kb-recall.mjs';

const analysis = count => ({ version: 1, type: 'kb_analysis', status: 'completed', analysis: '事实[1]',
  sources: Array.from({ length: count }, (_, i) => ({ index: i + 1, chunk_id: i + 1, kb_id: 7, doc_id: i + 11, doc_title: `资料${i + 1}`, snippet: `事实${i + 1}`, score: .9 - i / 100 })),
  usage: { scope: 'chat', prompt_tokens: 10, completion_tokens: 5, total_tokens: 15, complete: true } });

test('分析来源上限8且旧5兼容，第9来源和越界编号拒绝', () => {
  for (const count of [5, 8]) assert.equal(validateAnalysis(analysis(count)).sources.length, count);
  assert.throws(() => validateAnalysis(analysis(9)), /分析/);
  const invalid = analysis(1); invalid.sources[0].index = 9;
  assert.throws(() => validateAnalysis(invalid), /来源/);
});

test('普通检索仍最多5个来源，扩展分析不能放宽directSearch边界', () => {
  const known = new Map(analysis(6).sources.map(source => [source.doc_id, { ...source, id: source.doc_id, title: source.doc_title, status: 'ready' }]));
  const hits = analysis(6).sources.map(({ index: _index, ...hit }) => hit);
  assert.equal(validateHits(hits.slice(0, 5), 7, known).length, 5);
  assert.throws(() => validateHits(hits, 7, known), /数量/);
});

test('8来源仍受10000来源JSON和24000总输出字节约束', () => {
  for (const count of [5, 8]) {
    const crowded = analysis(count); crowded.sources.forEach(source => { source.snippet = 'x'.repeat(count === 5 ? 2100 : 1300); });
    assert.throws(() => validateAnalysis(crowded), /来源|字节/);
    const oversized = analysis(count); oversized.analysis = 'x'.repeat(24000);
    assert.throws(() => validateAnalysis(oversized), /分析|字节/);
  }
});

test('分析正文保留12000原字节及13000编码字节双限', () => {
  const accepted = analysis(1); accepted.analysis = 'x'.repeat(9000) + '\n'.repeat(1999);
  assert.equal(Buffer.byteLength(JSON.stringify(accepted.analysis)), 13000);
  assert.equal(validateAnalysis(accepted), accepted);
  for (const text of ['x'.repeat(12001), '\n'.repeat(6500)]) {
    const invalid = analysis(1); invalid.analysis = text;
    assert.throws(() => validateAnalysis(invalid), /正文|分析|字节/);
  }
});
