import test from 'node:test';
import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { existsSync } from 'node:fs';
import { readFile, stat } from 'node:fs/promises';
import { execFile } from 'node:child_process';
import { promisify } from 'node:util';
import { dirname, resolve } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { loadDataset } from './eval-kb-quality.mjs';

const exec = promisify(execFile);
const root = resolve(dirname(fileURLToPath(import.meta.url)), '../../..');
const casesPath = resolve(root, 'ai_document/ai-rag/business-acceptance-cases.json');
const syncPath = resolve(root, 'go-zero-admin/data/scripts/sync-kb-evaluation-data.mjs');
const embedded = resolve(root, 'go-zero-admin/application/ai/rpc/internal/logic/aiservice/evaluation_data');
const sha = value => createHash('sha256').update(value).digest('hex');
const frozen = {
  'business-cases.json': 'f3a4016eabc10a974e3acffc6d7347477976d00923c196c608de55a6db99538b',
  'business-failure-regression-cases.json': '6f6fbf2a4881751d1a9a77f2cde4571c2176cd3cca26fdab310f3a6eed0c9b87',
  'quality-cases.json': '538f6de8322da47a77bb51db7bea76e2aaa9a3ae8de34767824a61698dcb5b4c',
  'answer-regression-cases.json': '844a999cfa03179bba7370299a6bf8e3ad2fd6dc463611c8d980b5104e586ee4',
  'business-failure-fixtures.json': 'd5f265325d099f9b8d10ea3849f9d6ec29ae6aa29f47eb6d62f0c076fbfd55ea',
};
const blockFetch = 'globalThis.fetch=()=>{throw new Error("禁止离线题集操作调用HTTP")};';
const offline = args => exec(process.execPath, ['--import', `data:text/javascript,${encodeURIComponent(blockFetch)}`, ...args], { cwd: root });

test('24个项目场景题保持16开发8初始预留及原文边界', async () => {
  assert.ok(existsSync(casesPath), '缺少独立24题验收题集');
  const dataset = await loadDataset(casesPath);
  assert.equal(dataset.cases.length, 24);
  assert.equal(dataset.annotation.authoring, 'ai_assisted');
  assert.equal(dataset.annotation.human_signed, false);
  assert.equal(dataset.annotation.actual_user_history, false);
  assert.equal(dataset.annotation.checks_version, 1);
  assert.match(dataset.description, /AI辅助/u);
  assert.match(dataset.description, /不是真实用户历史问题/u);
  assert.equal(dataset.documents.length, 8);
  const old = await loadDataset(resolve(root, 'ai_document/ai-rag/business-cases.json'));
  assert.deepEqual(dataset.documents, old.documents);
  for (const kb of ['project_ops', 'project_ai']) {
    const rows = dataset.cases.filter(row => row.kb === kb);
    assert.equal(rows.length, 12);
    assert.equal(rows.filter(row => row.split === 'development').length, 8);
    assert.equal(rows.filter(row => row.split === 'validation').length, 4);
  }
  assert.equal(dataset.cases.filter(row => row.answer.kind === 'answerable').length, 20);
  assert.equal(dataset.cases.filter(row => row.answer.kind === 'unanswerable').length, 4);
  assert.equal(dataset.cases.filter(row => row.category === 'partial_unknown').length, 4);
  assert.equal(new Set(dataset.cases.map(row => row.query)).size, 24);
  assert.ok(dataset.cases.every(row => !old.cases.some(previous => previous.query === row.query)));
  for (const [file, expected] of Object.entries(frozen)) {
    assert.equal(sha(await readFile(resolve(root, 'ai_document/ai-rag', file))), expected, file);
  }
});

test('中文检查项逐一映射事实组，关键风险与实际未知分别标注', async () => {
  const dataset = await loadDataset(casesPath);
  for (const row of dataset.cases) {
    assert.ok(Array.isArray(row.checks), row.id);
    assert.equal(row.checks.length, row.answer.required_groups.length, row.id);
    assert.deepEqual(row.checks.map(check => check.required_group).sort((a, b) => a - b), row.answer.required_groups.map((_, index) => index), row.id);
    assert.ok(row.checks.every(check => typeof check.label === 'string' && check.label.trim() === check.label && check.label.length > 0 && ['fact', 'condition', 'risk', 'unknown'].includes(check.kind)), row.id);
    assert.equal(row.answer.absence_policy, ['partial_unknown', 'unanswerable'].includes(row.category) ? 'required' : 'forbidden', row.id);
    if (row.answer.kind === 'unanswerable') assert.deepEqual(row.checks, [], row.id);
  }
  const destructive = dataset.cases.find(row => row.id === 'acceptance-dev-init-destructive');
  assert.match(destructive.query, /初始化SQL会做什么/u);
  assert.ok(destructive.checks.some(check => check.kind === 'risk' && destructive.answer.required_groups[check.required_group].some(value => /DROP|删除/u.test(value))));
  assert.ok(destructive.checks.some(check => check.kind === 'risk' && destructive.answer.required_groups[check.required_group].some(value => /重建|重新创建/u.test(value))));
  for (const row of dataset.cases.filter(row => row.category === 'partial_unknown')) {
    assert.equal(row.answer.kind, 'answerable');
    assert.ok(row.expected_documents.length > 0);
    assert.match(row.rationale, /未知|不足/u);
  }
});

test('24题及开发16和初始预留8的plan在阻断fetch时仍为0HTTP', async () => {
  for (const [split, count] of [['all', 24], ['development', 16], ['validation', 8]]) {
    const { stdout } = await offline([resolve(root, 'go-zero-admin/data/scripts/eval-kb-quality.mjs'), '--plan', '--cases', casesPath, '--split', split]);
    const plan = JSON.parse(stdout);
    assert.equal(plan.mode, 'plan');
    assert.equal(plan.http_requests, 0);
    assert.equal(plan.case_count, count);
    assert.equal(plan.max_analysis_submissions, count);
  }
});

test('同步只复制四份逐字校验题集，check只读且重复同步不改写', async () => {
  assert.ok(existsSync(syncPath), '缺少评测内嵌数据同步入口');
  const { stdout } = await offline([syncPath]);
  const synced = JSON.parse(stdout);
  assert.equal(synced.mode, 'sync');
  assert.equal(synced.http_requests, 0);
  const names = ['business-acceptance-cases.json', 'business-cases.json', 'business-failure-regression-cases.json', 'business-evidence-cases.json'];
  assert.deepEqual(synced.files.map(row => row.name), names);
  const before = new Map();
  for (const name of names) {
    const source = await readFile(resolve(root, 'ai_document/ai-rag', name));
    const target = resolve(embedded, name);
    assert.deepEqual(await readFile(target), source, name);
    before.set(name, (await stat(target)).mtimeMs);
  }
  for (const args of [[syncPath, '--check'], [syncPath]]) {
    const { stdout: again } = await offline(args);
    const report = JSON.parse(again);
    assert.equal(report.http_requests, 0);
    assert.ok(report.files.every(row => row.written === false));
    for (const name of names) assert.equal((await stat(resolve(embedded, name))).mtimeMs, before.get(name), name);
  }
});

test('检查项拒绝重复映射、未知类型及缺失标签，旧题可保留无标签格式', async () => {
  assert.ok(existsSync(syncPath), '缺少评测数据校验入口');
  const { validateDatasetChecks } = await import(pathToFileURL(syncPath).href);
  const dataset = await loadDataset(casesPath);
  const mutate = change => { const copy = structuredClone(dataset); change(copy.cases[0]); return copy; };
  assert.throws(() => validateDatasetChecks(mutate(row => { row.checks[1].required_group = 0; })), /检查项/u);
  assert.throws(() => validateDatasetChecks(mutate(row => { row.checks[0].kind = 'score'; })), /检查项/u);
  assert.throws(() => validateDatasetChecks(mutate(row => { row.checks[0].label = ' '; })), /检查项/u);
  assert.throws(() => validateDatasetChecks(mutate(row => { row.checks.pop(); })), /检查项/u);
  assert.throws(() => validateDatasetChecks(mutate(row => { delete row.checks; })), /检查项/u);
  validateDatasetChecks(await loadDataset(resolve(root, 'ai_document/ai-rag/business-cases.json')));
});

test('同步CLI拒绝额外路径选项，不支持写出到任意目录', async () => {
  assert.ok(existsSync(syncPath), '缺少评测数据同步入口');
  await assert.rejects(offline([syncPath, '--output', resolve(root, 'tmp/not-authorized-evaluation-output.json')]), /未知参数|只支持/u);
});
