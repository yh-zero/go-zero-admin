import test from 'node:test';
import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { existsSync } from 'node:fs';
import { readFile, stat } from 'node:fs/promises';
import { execFile } from 'node:child_process';
import { promisify } from 'node:util';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { loadDataset } from './eval-kb-quality.mjs';
import { screenAnswer } from './eval-kb-quality-metrics.mjs';
import { validateDatasetChecks } from './sync-kb-evaluation-data.mjs';

const exec = promisify(execFile);
const root = resolve(dirname(fileURLToPath(import.meta.url)), '../../..');
const source = resolve(root, 'ai_document/ai-rag/business-evidence-cases.json');
const parent = resolve(root, 'ai_document/ai-rag/business-acceptance-cases.json');
const parentSHA = '176e20b4563d8c892b375fe45af89a4a27b78ee1d62c9686e9b49b3956696760';
const calibratedSHA = '1b51cb93a575876002bd9877c4ffa2ef3ab95f43785bd5ea4a20292c8b6eb5ca';
const changed = ['acceptance-dev-backup-checksum-gap', 'acceptance-dev-refresh-advice', 'acceptance-val-captcha-runtime-gap', 'acceptance-dev-isolated-cleanup', 'acceptance-dev-text-limits', 'acceptance-dev-capability-config'];
const sha = bytes => createHash('sha256').update(bytes).digest('hex');
const blockedFetch = 'globalThis.fetch=()=>{throw new Error("离线校准禁止HTTP")};';
const offline = args => exec(process.execPath, ['--import', `data:text/javascript,${encodeURIComponent(blockedFetch)}`, ...args], { cwd: root });
const valueAt = (value, path) => path.split('.').reduce((current, key) => current?.[key], value);

async function datasets() {
  assert.ok(existsSync(source), '缺少独立的24题证据覆盖校准派生版');
  return { calibrated: await loadDataset(source), original: await loadDataset(parent) };
}

test('校准版冻结原题与来源，逐字段记录六处校准而保留真实召回失败', async () => {
  const { calibrated, original } = await datasets();
  assert.equal(sha(await readFile(parent)), parentSHA);
  assert.equal(sha(await readFile(source)), calibratedSHA);
  assert.equal(calibrated.annotation.parent_dataset_sha256, parentSHA);
  assert.equal(calibrated.annotation.parent_dataset, 'business-acceptance-cases.json');
  assert.equal(calibrated.annotation.authoring, 'ai_assisted');
  assert.equal(calibrated.annotation.human_signed, false);
  assert.equal(calibrated.annotation.actual_user_history, false);
  assert.equal(calibrated.annotation.all_cases_previously_used, true);
  assert.equal(calibrated.cases.length, 24);
  assert.deepEqual(calibrated.documents, original.documents);
  assert.deepEqual(calibrated.annotation.calibration_changes.map(row => row.case_id), changed);
  for (const previous of original.cases) {
    const current = calibrated.cases.find(row => row.id === previous.id);
    for (const field of ['id', 'query', 'split', 'category', 'kb', 'rationale']) assert.deepEqual(current[field], previous[field], `${previous.id}:${field}`);
    if (!changed.includes(previous.id)) { assert.deepEqual(current, previous); continue; }
    const audit = calibrated.annotation.calibration_changes.find(row => row.case_id === previous.id);
    assert.ok(audit.basis.length > 20 && audit.fields.length > 0);
    const rebuilt = structuredClone(previous);
    for (const entry of audit.fields) {
      assert.deepEqual(valueAt(previous, entry.path), entry.before);
      assert.deepEqual(valueAt(current, entry.path) ?? null, entry.after);
      const parts = entry.path.split('.'), key = parts.pop();
      const target = parts.reduce((value, part) => value[part], rebuilt);
      if (entry.operation === 'remove') delete target[key]; else target[key] = entry.after;
    }
    assert.deepEqual(rebuilt, current, previous.id);
  }
  for (const id of ['acceptance-dev-code-and-env', 'acceptance-val-config-selection']) assert.deepEqual(calibrated.cases.find(row => row.id === id), original.cases.find(row => row.id === id));
  validateDatasetChecks(calibrated);
});

test('原问之外的补充细节不再强制必答，明确要求和未知实例保留', async () => {
  const { calibrated, original } = await datasets();
  const find = id => calibrated.cases.find(row => row.id === id);
  const backup = find(changed[0]), access = find(changed[1]), captcha = find(changed[2]), cleanup = find(changed[3]);
  assert.equal(backup.answer.required_groups.length, 4);
  assert.equal(backup.evidence[0].groups.length, 4);
  assert.equal(backup.answer.absence_policy, 'required');
  assert.equal(access.answer.required_groups.length, 4);
  assert.ok(access.checks.some(row => row.label === '知识库按创建者隔离'));
  assert.equal(captcha.answer.required_groups.length, 3);
  assert.equal(captcha.answer.absence_policy, 'required');
  assert.deepEqual(cleanup.expected_documents, ['ops-cleanup']);
  assert.equal(cleanup.answer.required_groups.length, 7);
  assert.equal(cleanup.evidence.length, 1);
  assert.ok(original.cases.find(row => row.id === changed[0]).answer.required_groups[0].includes('完整原始 SQL'));
  assert.ok(original.cases.find(row => row.id === changed[1]).answer.required_groups[2].includes('返回404'));
  assert.equal(find(changed[5]).answer.absence_policy, undefined);
});

async function screen(item, answer, dataset) {
  const known = new Map(), bindings = new Map(), sources = [];
  for (const [index, document] of dataset.documents.entries()) {
    known.set(index + 1, { ...document, id: index + 1, title: document.title, status: 'ready' });
    bindings.set(document.key, { doc_id: index + 1 });
    if (item.expected_documents.includes(document.key)) sources.push({ index: sources.length + 1, doc_id: index + 1, chunk_id: index + 101, kb_id: 1, doc_title: document.title, snippet: await readFile(resolve(root, document.source), 'utf8'), score: 0.8 });
  }
  const text = `${answer} ${sources.map(source => `[${source.index}]`).join('')}`;
  return screenAnswer(item, { version: 1, type: 'kb_analysis', status: 'completed', analysis: text, sources, usage: { scope: 'chat', complete: true, prompt_tokens: 2, completion_tokens: 1, total_tokens: 3 } }, 1, known, bindings);
}

test('校准只改变原问判读范围，缺必答或漏真实未知仍提示', async () => {
  const { calibrated } = await datasets();
  const find = id => calibrated.cases.find(row => row.id === id);
  const reconstructed = [
    [changed[0], '记录来源、导出时间与文件校验值；在独立环境恢复验证，不导回现有业务库，不启动会调用外部服务的业务进程。未能确定本实例实际SHA256。'],
    [changed[1], '知识库按创建者隔离；服务端权限校验。刷新令牌一次性轮换，退出、撤销后旧访问与刷新令牌失效。'],
    [changed[2], '本机使用admin-rpc.ai-dev.local.yaml，Captcha.Enabled改为false；账号密码和会话/权限仍正常校验。本实例当前实际值资料未提供。'],
    [changed[3], '准确 PID/任务 ID、容器名、路径与临时数据 ID；不按进程名/通配符批量删除。保留用户要求运行的项目栈，恢复本轮临时配置，确认测试端口释放；回归测试、用户数据及既有备份保留，不执行checkout/reset。'],
    [changed[5], '配置状态未知也只是客户端配置；不做供应商实时健康探测，实际请求仍可能失败。会话管理和本地工具仍可用，发送消息返回降级提示。检索与聊天模型均已配置才能分析。'],
  ];
  for (const [id, answer] of reconstructed) assert.equal((await screen(find(id), answer, calibrated)).flagged, false, id);
  const backup = find(changed[0]);
  assert.ok((await screen(backup, '记录来源、导出时间与文件校验值。在独立环境恢复验证。未能确定实例SHA。', calibrated)).reasons.includes('missing_required_facts'));
  assert.ok((await screen(backup, backup.answer.required_groups.map(group => group[0]).join('；'), calibrated)).reasons.includes('missing_scoped_insufficiency'));
  const capability = find(changed[5]);
  assert.ok((await screen(capability, '配置状态未知，实际请求仍可能失败。', calibrated)).reasons.includes('missing_required_facts'));
});

test('正文超字节拒绝接受有范围的同义表述，裸否定或允许超限仍提示', async () => {
  const { calibrated } = await datasets();
  const item = calibrated.cases.find(row => row.id === changed[4]);
  const otherFacts = item.answer.required_groups.filter((_, index) => index !== 2).map(group => group[0]).join('；');
  for (const phrase of ['正文超限不能提交', '正文超过65536个UTF-8字节不可提交', 'UTF-8字节超限时不能提交', '65536 UTF-8字节），超过即超出限制']) {
    assert.equal((await screen(item, `${otherFacts}；${phrase}`, calibrated)).flagged, false, phrase);
  }
  for (const phrase of ['可以提交，即使正文超出65536个UTF-8字节', '不能']) {
    assert.ok((await screen(item, `${otherFacts}；${phrase}`, calibrated)).reasons.includes('missing_required_facts'), phrase);
  }
});

test('校准版24题plan及四份同步保持0HTTP，check只读且逐字幂等', async () => {
  await datasets();
  const planned = JSON.parse((await offline([resolve(root, 'go-zero-admin/data/scripts/eval-kb-quality.mjs'), '--plan', '--cases', source])).stdout);
  assert.equal(planned.http_requests, 0);
  assert.equal(planned.case_count, 24);
  const syncPath = resolve(root, 'go-zero-admin/data/scripts/sync-kb-evaluation-data.mjs');
  const synced = JSON.parse((await offline([syncPath])).stdout);
  assert.equal(synced.files.length, 4);
  assert.equal(synced.http_requests, 0);
  const embedded = resolve(root, 'go-zero-admin/application/ai/rpc/internal/logic/aiservice/evaluation_data/business-evidence-cases.json');
  assert.deepEqual(await readFile(embedded), await readFile(source));
  const before = (await stat(embedded)).mtimeMs;
  for (const args of [[syncPath, '--check'], [syncPath]]) {
    const report = JSON.parse((await offline(args)).stdout);
    assert.equal(report.http_requests, 0);
    assert.ok(report.files.every(row => row.written === false));
    assert.equal((await stat(embedded)).mtimeMs, before);
  }
});
