import test from 'node:test';
import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { existsSync } from 'node:fs';
import { readFile } from 'node:fs/promises';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { loadDataset } from './eval-kb-quality.mjs';
import { evaluateRetrieval, screenAnswer, validateAnalysis } from './eval-kb-quality-metrics.mjs';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '../../..');
const sourceCaseID = 'business-dev-existing-database';
const sha = value => createHash('sha256').update(value).digest('hex');
const fixture = JSON.parse(await readFile(resolve(root, 'ai_document/ai-rag/business-omission-fixture.json'), 'utf8'));
const dataset = await loadDataset(resolve(root, 'ai_document/ai-rag/business-cases.json'));
const item = dataset.cases.find(item => item.id === sourceCaseID);
const example = fixture.examples[0];
const observed = example.observed;
const bindings = new Map(fixture.bound_documents.map(document => [document.key, document]));
const knownDocuments = new Map(fixture.bound_documents.map(document => [document.doc_id, document]));

test('261遗漏夹具只保存真实外层观测，实际来源已包含关键后果', async () => {
  assert.equal(fixture.version, 1);
  assert.equal(fixture.examples.length, 1);
  assert.equal(fixture.provenance.raw_provider_structure_observed, false);
  assert.equal(fixture.provenance.baseline_context.binary_sha, '72ff8f9c90c401178860c53093989dc0de63b9be5e7bbca55970573ac7225125');
  assert.equal(fixture.provenance.base_dataset_sha256, 'f3a4016eabc10a974e3acffc6d7347477976d00923c196c608de55a6db99538b');
  assert.equal(example.source_case_id, sourceCaseID);
  assert.equal(observed.case_id, sourceCaseID);
  assert.equal(observed.task_id, 261);
  assert.equal(observed.query, item.query);
  assert.equal(observed.kb_id, 5);
  assert.equal(example.observed_sha256, sha(JSON.stringify(observed)));
  assert.equal(example.review.human_signed, false);
  assert.equal(example.review.retrieval_failure, false);
  assert.deepEqual(example.review.actual_support, { source_index: 1, chunk_id: 27, doc_id: 18 });
  assert.equal(example.review.missing_required_group_index, 2);
  assert.equal(validateAnalysis(observed.analysis), observed.analysis);
  const source = observed.analysis.sources.find(source => source.index === 1);
  assert.equal(source.chunk_id, 27);
  assert.equal(source.doc_id, 18);
  assert.match(source.snippet, /初始化 SQL 会 DROP 并重建业务表/u);
  assert.doesNotMatch(observed.analysis.analysis, /DROP|删除并重建|删除重建/u);
  for (const document of fixture.bound_documents) {
    const data = await readFile(resolve(root, document.source));
    assert.equal(sha(data), document.sha256);
    assert.equal(document.content_sha256, document.sha256);
    assert.equal(document.status, 'ready');
  }
  const original = JSON.parse(await readFile(resolve(root, 'ai_document/ai-rag/business-cases.json'), 'utf8'));
  assert.deepEqual(original.cases.find(item => item.id === sourceCaseID), item);
});

test('既有真实回答仍检出第三事实遗漏，完整预期对照仅验证评分敏感性', () => {
  const retrieval = evaluateRetrieval(item, observed.list, bindings);
  const actual = screenAnswer(item, observed.analysis, observed.kb_id, knownDocuments, bindings);
  assert.equal(retrieval.recall_at[5], 1);
  assert.equal(retrieval.evidence_at[5].complete, true);
  assert.deepEqual(actual.required_groups_matched, [true, true, false]);
  assert.ok(actual.reasons.includes('missing_required_facts'));
  assert.deepEqual(actual.cited_evidence_groups_matched, [true, true, true]);
  assert.deepEqual(actual, observed.answer_screen);

  // 人工构造的评分对照，仅替换答案文本；不是原始内部JSON或修复后的真实模型运行。
  const complete = structuredClone(observed.analysis);
  complete.analysis = '已有数据卷启动后会自动更新数据库表吗？：已有卷不自动覆盖或更新结构。 [1]\n\n生成model时能重新导入mysql-all.sql清空现有业务库吗？：不能。初始化 SQL 会 DROP 并重建业务表，不是已有库的升级脚本；生成 model 时使用新开发库，不为生成代码清空业务库。 [1]';
  const control = screenAnswer(item, complete, observed.kb_id, knownDocuments, bindings);
  assert.deepEqual(control.required_groups_matched, [true, true, true]);
  assert.equal(control.flagged, false);
  assert.equal(observed.analysis.usage.total_tokens, 1985);
  assert.equal(example.observed_sha256, sha(JSON.stringify(observed)));
});

test('有本地原日志时独立核对来源记录，无原日志不冒充重新读取', async t => {
  const path = resolve(root, fixture.provenance.source_log);
  if (!existsSync(path)) {
    t.diagnostic('原始本地日志未随仓库分发；只复验自包含夹具与冻结正文，不宣称重新读取原日志。');
    return;
  }
  const raw = await readFile(path);
  assert.equal(sha(raw), fixture.provenance.source_log_sha256);
  const record = raw.toString('utf8').trim().split('\n').map(JSON.parse).find(row => row.type === 'result' && row.case_id === sourceCaseID);
  assert.ok(record);
  assert.equal(sha(JSON.stringify(record)), example.source_record_sha256);
  assert.deepEqual(Object.fromEntries(Object.keys(observed).map(key => [key, record[key]])), observed);
});
