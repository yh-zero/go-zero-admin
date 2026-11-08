import test from 'node:test';
import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { mkdtemp, readFile, writeFile, rm, access, symlink, unlink } from 'node:fs/promises';
import { dirname, join, resolve, sep } from 'node:path';
import { fileURLToPath } from 'node:url';
import { tmpdir } from 'node:os';
import { execFile } from 'node:child_process';
import { promisify } from 'node:util';
import { loadDataset } from './eval-kb-quality.mjs';

const tool = await import('./merge-kb-feedback.mjs').catch(error => {
  if (error.code === 'ERR_MODULE_NOT_FOUND') return {};
  throw error;
});
const feature = name => { assert.equal(typeof tool[name], 'function', `缺少离线反馈转换能力 ${name}`); return tool[name]; };
const root = resolve(dirname(fileURLToPath(import.meta.url)), '../../..');
const casesFile = resolve(root, 'ai_document/ai-rag/business-cases.json');
const hash = value => createHash('sha256').update(value).digest('hex');
const exec = promisify(execFile);
const kbMap = { project_ops: 701, project_ai: 702 };
async function exportedFeedback() {
  const dataset = await loadDataset(casesFile), doc = dataset.documents.find(row => row.key === 'ai-knowledge');
  const full = await readFile(resolve(root, doc.source), 'utf8');
  const snippet = full.split('\n').find(line => line.startsWith('2. 使用'));
  const source = { index: 1, chunk_id: 931, kb_id: 702, doc_id: 811, doc_title: doc.title, snippet, score: 0.8 };
  const question = '知识库每批最多导入几篇，浏览器最多同时提交几篇？', answer = '每批最多10篇，浏览器同时提交10篇。[1]';
  const analysis = { version: 1, type: 'kb_analysis', status: 'completed', analysis: answer, sources: [source],
    usage: { scope: 'chat', prompt_tokens: 10, completion_tokens: 5, total_tokens: 15, complete: true } };
  const snapshot = { version: 1, kb_id: 702, query: question, step_input: JSON.stringify({ kb_id: 702, query: question }),
    step_output: JSON.stringify(analysis), sources: [source],
    documents: [{ doc_id: 811, kb_id: 702, title: doc.title, version: 1, content_sha256: doc.sha256, status: 'verified' }] };
  return { version: 1, type: 'kb_feedback_regression', feedback: { id: 91, kb_id: 702, task_id: 73, step_index: 0,
    feedback: 'incorrect_answer', note: '数量错误', status: 'accepted', revision: 2, question, answer, snapshot: JSON.stringify(snapshot),
    review: { answerable: true, expected_answer: '每批最多20篇，浏览器最多并发提交2篇。', required_facts: ['20篇', '2篇'],
      evidence: [{ source_index: 1, quote: '每批最多 20 篇' }, { source_index: 1, quote: '浏览器最多并发提交 2 篇' }], reason: '按冻结的导入说明核对两种不同限制。' },
    created_at: 1794000000, updated_at: 1794000001, reviewed_at: 1794000001 } };
}
const merge = async value => feature('mergeFeedback')({ feedbackBytes: Buffer.from(JSON.stringify(value)), casesFile, kbMap });
function editSnapshot(value, edit) {
  const snapshot = JSON.parse(value.feedback.snapshot); edit(snapshot); value.feedback.snapshot = JSON.stringify(snapshot); return value;
}
async function temporary(action) {
  const folder = await mkdtemp(join(root, 'tmp', 'feedback-merge-test-'));
  try { return await action(folder); }
  finally {
    assert.ok(resolve(folder).startsWith(resolve(root, 'tmp') + sep));
    assert.ok(folder.includes('feedback-merge-test-'));
    await rm(folder, { recursive: true, force: true });
  }
}

test('采纳反馈离线转成可运行development题，旧题与语料完整保留并记录原始指纹', async () => {
  const original = await loadDataset(casesFile), exported = await exportedFeedback(), bytes = Buffer.from(JSON.stringify(exported));
  const merged = await feature('mergeFeedback')({ feedbackBytes: bytes, casesFile, kbMap });
  assert.equal(merged.cases.length, 33);
  assert.deepEqual(merged.cases.slice(0, 32), original.cases);
  assert.deepEqual(merged.documents, original.documents);
  const added = merged.cases.at(-1);
  assert.equal(added.id, 'feedback-702-91-r2');
  assert.equal(added.kb, 'project_ai');
  assert.equal(added.split, 'development');
  assert.equal(added.category, 'feedback_regression');
  assert.deepEqual(added.expected_documents, ['ai-knowledge']);
  assert.deepEqual(added.evidence, [{ document: 'ai-knowledge', groups: [['每批最多 20 篇'], ['浏览器最多并发提交 2 篇']] }]);
  assert.deepEqual(added.answer.required_groups, [['20篇'], ['2篇']]);
  assert.equal(added.expected_answer_example, exported.feedback.review.expected_answer);
  assert.equal(added.source_case_id, undefined);
  const provenance = merged.metadata.feedback_regressions[0];
  assert.equal(provenance.export_sha256, hash(bytes));
  assert.equal(provenance.base_dataset_sha256, hash(await readFile(casesFile)));
  assert.equal(provenance.feedback_id, 91);
  assert.equal(provenance.revision, 2);
  assert.equal(provenance.human_identity_verified, false);
});

test('资料不足反馈保留指定KB语料，不伪造证据、事实或旧case来源', async () => {
  const exported = await exportedFeedback();
  exported.feedback.review = { answerable: false, expected_answer: '本次片段资料不足，无法确定。', required_facts: [], evidence: [], reason: '当前资料没有所需业务事实。' };
  const merged = await merge(exported), added = merged.cases.at(-1);
  assert.equal(added.answer.kind, 'unanswerable');
  assert.deepEqual(added.expected_documents, []);
  assert.deepEqual(added.evidence, []);
  assert.deepEqual(added.answer.required_groups, []);
  assert.equal(added.kb, 'project_ai');
});

test('256字节安全截断的历史来源标题可匹配完整中文标题，错误或非工具截断标题拒绝', async () => temporary(async folder => {
  for (const title of ['题'.repeat(100), 'A' + '题'.repeat(100), '🧭'.repeat(100)]) {
    const dataset = await loadDataset(casesFile);
    dataset.documents.find(doc => doc.key === 'ai-knowledge').title = title;
    const longCases = join(folder, 'long-title-cases.json');
    await writeFile(longCases, JSON.stringify(dataset));
    const prefix = Array.from(title).reduce((result, rune) => Buffer.byteLength(result + rune) <= 256 ? result + rune : result, '');
    const value = await exportedFeedback();
    const changeTitle = text => editSnapshot(value, snapshot => {
      snapshot.documents[0].title = title;
      snapshot.sources[0].doc_title = text;
      const output = JSON.parse(snapshot.step_output); output.sources = snapshot.sources; snapshot.step_output = JSON.stringify(output);
    });
    changeTitle(prefix);
    const before = structuredClone(value);
    const merged = await feature('mergeFeedback')({ feedbackBytes: Buffer.from(JSON.stringify(value)), casesFile: longCases, kbMap });
    assert.equal(merged.cases.length, 33);
    assert.equal(merged.documents.find(doc => doc.key === 'ai-knowledge').title, title);
    assert.deepEqual(value, before, '历史来源和完整标题不能被转换工具改写');
    for (const wrong of ['错误标题', Array.from(prefix).slice(0, -1).join(''), title]) {
      changeTitle(wrong);
      await assert.rejects(feature('mergeFeedback')({ feedbackBytes: Buffer.from(JSON.stringify(value)), casesFile: longCases, kbMap }), /标题|片段/u);
    }
  }
}));

test('8来源可从第7和8项采纳并离线转题，第9来源仍拒绝', async () => temporary(async folder => {
  const original = await loadDataset(casesFile), base = original.documents.find(doc => doc.key === 'ai-knowledge');
  const documents = Array.from({ length: 8 }, (_, index) => ({ ...base, key: `source-fixture-${index + 1}`, title: `历史资料${index + 1}` }));
  const fixtureCases = join(folder, 'eight-source-cases.json');
  await writeFile(fixtureCases, JSON.stringify({ version: 1, documents, cases: [{ id: 'fixture-negative', split: 'development', category: 'fixture', kb: 'project_ai', query: '未记录的问题？', rationale: '仅用于8来源转换边界', expected_documents: [], evidence: [], answer: { kind: 'unanswerable', required_groups: [], forbidden_phrases: [] } }] }));
  const value = await exportedFeedback();
  editSnapshot(value, snapshot => {
    const first = snapshot.sources[0];
    snapshot.sources = documents.map((doc, index) => ({ ...first, index: index + 1, chunk_id: 931 + index, doc_id: 811 + index, doc_title: doc.title }));
    snapshot.documents = documents.map((doc, index) => ({ doc_id: 811 + index, kb_id: 702, title: doc.title, version: 1, content_sha256: doc.sha256, status: 'verified' }));
    const output = JSON.parse(snapshot.step_output); output.sources = snapshot.sources; snapshot.step_output = JSON.stringify(output);
  });
  value.feedback.review.evidence[0].source_index = 8;
  value.feedback.review.evidence[1].source_index = 7;
  const merged = await feature('mergeFeedback')({ feedbackBytes: Buffer.from(JSON.stringify(value)), casesFile: fixtureCases, kbMap });
  assert.deepEqual(merged.cases.at(-1).expected_documents, ['source-fixture-8', 'source-fixture-7']);
  const invalid = structuredClone(value);
  editSnapshot(invalid, snapshot => {
    snapshot.sources.push({ ...snapshot.sources[0], index: 9, chunk_id: 999 });
    const output = JSON.parse(snapshot.step_output); output.sources = snapshot.sources; snapshot.step_output = JSON.stringify(output);
  });
  await assert.rejects(feature('mergeFeedback')({ feedbackBytes: Buffer.from(JSON.stringify(invalid)), casesFile: fixtureCases, kbMap }));
}));

test('未采纳、错误版本或复核空缺不会变成回归题', async () => {
  const mutations = [
    value => { value.feedback.status = 'pending'; },
    value => { value.feedback.status = 'dismissed'; },
    value => { value.feedback.revision = 0; },
    value => { value.feedback.step_index = 32; },
    value => { value.feedback.review.expected_answer = ''; },
    value => { value.feedback.review.reason = ''; },
    value => { value.feedback.review.answerable = 'true'; },
    value => { value.feedback.review.required_facts = []; },
    value => { value.feedback.review.evidence = []; },
    value => { value.feedback.review.answerable = false; },
    value => { value.feedback.created_at = value.feedback.updated_at + 1; },
  ];
  for (const mutate of mutations) { const value = await exportedFeedback(); mutate(value); await assert.rejects(merge(value)); }
});

test('映射、历史资料状态、版本、标题和SHA不匹配须拒绝，不静默替换当前文档', async () => {
  const mutations = [
    value => { value.kb_id = 701; },
    value => { value.documents[0].status = 'stale'; },
    value => { value.documents[0].status = 'unavailable'; },
    value => { value.documents[0].version = 0; },
    value => { value.documents[0].content_sha256 = 'a'.repeat(64); },
    value => { value.documents[0].title = '另一个标题'; },
    value => { value.documents[0].doc_id = 999; },
    value => { value.documents.push(structuredClone(value.documents[0])); },
    value => { value.sources[0].snippet = '不是原文里的片段'; },
  ];
  for (const mutate of mutations) await assert.rejects(merge(editSnapshot(await exportedFeedback(), mutate)));
  await assert.rejects(feature('mergeFeedback')({ feedbackBytes: Buffer.from(JSON.stringify(await exportedFeedback())), casesFile, kbMap: { project_ops: 701, project_ai: 999 } }));
});

test('实际步骤、问题、答案和来源必须相互一致，引用摘录和事实须可逐字核对', async () => {
  const mutations = [
    value => { value.feedback.question += '另一个问题'; },
    value => { value.feedback.answer += '伪造答案'; },
    value => { value.feedback.review.evidence[0].source_index = 2; },
    value => { value.feedback.review.evidence[0].quote = '每批最多100篇'; },
    value => { value.feedback.review.required_facts[0] = '100篇'; },
    value => { value.feedback.review.expected_answer = '最多20篇'; },
    value => editSnapshot(value, snapshot => { snapshot.step_input = JSON.stringify({ kb_id: 702, query: '换了查询' }); }),
    value => editSnapshot(value, snapshot => { const output = JSON.parse(snapshot.step_output); output.sources[0].score = 0.7; snapshot.step_output = JSON.stringify(output); }),
  ];
  for (const mutate of mutations) { const value = await exportedFeedback(); mutate(value); await assert.rejects(merge(value)); }
});

test('UTF-8字节上限、非法字节、额外字段与重复JSON键不能绕过校验', async () => {
  for (const mutate of [
    value => { value.feedback.note = '中'.repeat(334); },
    value => { value.feedback.review.evidence[0].quote = '中'.repeat(167); },
    value => { value.feedback.review.required_facts[0] = '中'.repeat(67); },
    value => { value.feedback.review.reason = '中'.repeat(334); },
    value => { value.feedback.review.expected_answer = '中'.repeat(1334); },
    value => { value.feedback.review.extra = '伪造元字段'; },
    value => { value.feedback.note = '\ud800'; },
  ]) { const value = await exportedFeedback(); mutate(value); await assert.rejects(merge(value)); }
  const bytes = Buffer.from(JSON.stringify(await exportedFeedback()).replace('"version":1', '"version":1,"version":1'));
  await assert.rejects(feature('mergeFeedback')({ feedbackBytes: bytes, casesFile, kbMap }));
  await assert.rejects(feature('mergeFeedback')({ feedbackBytes: Buffer.from([0xff, 0xfe]), casesFile, kbMap }));
});

test('真实段落分块的3LF到2LF归一不改变正文SHA，历史quote仍逐字核对', async () => {
  const dataset = await loadDataset(casesFile), doc = dataset.documents.find(row => row.key === 'ai-access');
  const content = await readFile(resolve(root, doc.source), 'utf8'), boundary = content.indexOf('\n\n\n');
  assert.ok(boundary > 0);
  const snippet = content.slice(boundary - 35, boundary + 42).split('\n\n').map(part => part.trim()).join('\n\n').trim();
  assert.equal(content.includes(snippet), false);
  const value = await exportedFeedback();
  value.feedback.question = '自动刷新应该怎样处理共享刷新令牌？';
  value.feedback.answer = '可以直接并发消耗共享刷新令牌。[1]';
  editSnapshot(value, snapshot => {
    snapshot.query = value.feedback.question;
    snapshot.step_input = JSON.stringify({ kb_id: 702, query: value.feedback.question });
    snapshot.sources[0] = { ...snapshot.sources[0], doc_title: doc.title, snippet };
    snapshot.documents[0] = { ...snapshot.documents[0], title: doc.title, content_sha256: doc.sha256 };
    const output = JSON.parse(snapshot.step_output); output.analysis = value.feedback.answer; output.sources = snapshot.sources;
    snapshot.step_output = JSON.stringify(output);
  });
  value.feedback.review = { answerable: true, expected_answer: '不并发消耗共享刷新令牌。', required_facts: ['刷新令牌'],
    evidence: [{ source_index: 1, quote: snippet }], reason: '历史分块仅归一段落边界，数字和正文未改。' };
  const merged = await merge(value);
  assert.deepEqual(merged.cases.at(-1).expected_documents, ['ai-access']);
  assert.equal(merged.cases.at(-1).evidence[0].groups[0][0], snippet);
});

test('段落归一不能掩盖数字篡改，也不把压缩JSON当成原始分块', async () => {
  const numeric = await exportedFeedback();
  editSnapshot(numeric, snapshot => {
    snapshot.sources[0].snippet = snapshot.sources[0].snippet.replace('20 篇', '21 篇');
    const output = JSON.parse(snapshot.step_output); output.sources = snapshot.sources; snapshot.step_output = JSON.stringify(output);
  });
  numeric.feedback.review.expected_answer = '每批最多21篇。'; numeric.feedback.review.required_facts = ['21篇'];
  numeric.feedback.review.evidence = [{ source_index: 1, quote: '每批最多 21 篇' }];
  await assert.rejects(merge(numeric), /片段|正文/u);
  const oldCases = resolve(root, 'ai_document/ai-rag/quality-cases.json'), oldDataset = await loadDataset(oldCases);
  const doc = oldDataset.documents.find(row => row.key === 'stocks-valuation');
  const full = await readFile(resolve(root, doc.source), 'utf8'), compact = JSON.stringify(JSON.parse(full));
  const compressed = await exportedFeedback(); compressed.feedback.kb_id = 701;
  editSnapshot(compressed, snapshot => {
    snapshot.kb_id = 701; snapshot.step_input = JSON.stringify({ kb_id: 701, query: compressed.feedback.question });
    snapshot.sources[0] = { ...snapshot.sources[0], kb_id: 701, doc_title: doc.title, snippet: compact };
    snapshot.documents[0] = { ...snapshot.documents[0], kb_id: 701, title: doc.title, content_sha256: doc.sha256 };
    const output = JSON.parse(snapshot.step_output); output.sources = snapshot.sources; snapshot.step_output = JSON.stringify(output);
  });
  compressed.feedback.review = { answerable: true, expected_answer: 'PB(LF)。', required_facts: ['PB(LF)'],
    evidence: [{ source_index: 1, quote: 'PB(LF)' }], reason: '用于核对非法结构压缩拒绝。' };
  await assert.rejects(feature('mergeFeedback')({ feedbackBytes: Buffer.from(JSON.stringify(compressed)), casesFile: oldCases, kbMap: { stocks: 701, ops: 702 } }), /片段|正文/u);
});

test('仅写入tmp新副本并复验，已存在输出与反馈重复均不覆盖', async () => temporary(async folder => {
  const feedbackFile = join(folder, 'feedback.json'), kbMapFile = join(folder, 'kb-map.json'), output = join(folder, 'merged.json');
  await writeFile(feedbackFile, JSON.stringify(await exportedFeedback())); await writeFile(kbMapFile, JSON.stringify(kbMap));
  const bytesBefore = await readFile(casesFile), result = await feature('convertFeedbackFiles')({ feedbackFile, casesFile, kbMapFile, output });
  assert.equal(result.http_requests, 0);
  assert.equal(result.case_id, 'feedback-702-91-r2');
  assert.equal((await loadDataset(output)).cases.length, 33);
  assert.deepEqual(await readFile(casesFile), bytesBefore);
  const outputBefore = await readFile(output);
  await assert.rejects(feature('convertFeedbackFiles')({ feedbackFile, casesFile, kbMapFile, output }));
  assert.deepEqual(await readFile(output), outputBefore);
  await assert.rejects(feature('convertFeedbackFiles')({ feedbackFile, casesFile: output, kbMapFile, output: join(folder, 'duplicate.json') }));
  await assert.rejects(access(join(folder, 'duplicate.json')));
  await assert.rejects(feature('convertFeedbackFiles')({ feedbackFile, casesFile, kbMapFile, output: join(tmpdir(), 'feedback-must-not-write.json') }));
}));

test('输出父目录的符号链接不能逃出tmp边界', async () => temporary(async folder => {
  const feedbackFile = join(folder, 'feedback.json'), kbMapFile = join(folder, 'kb-map.json'), link = join(folder, 'escape');
  await writeFile(feedbackFile, JSON.stringify(await exportedFeedback())); await writeFile(kbMapFile, JSON.stringify(kbMap));
  await symlink(tmpdir(), link, 'junction');
  try {
    await assert.rejects(feature('convertFeedbackFiles')({ feedbackFile, casesFile, kbMapFile, output: join(link, 'must-not-write.json') }), /越界|tmp/u);
  } finally { await unlink(link); }
}));

test('CLI在阻断fetch时完成真实文件转换，新题集plan也不请求HTTP', async () => temporary(async folder => {
  const feedbackFile = join(folder, 'feedback.json'), kbMapFile = join(folder, 'kb-map.json'), output = join(folder, 'cli-merged.json');
  await writeFile(feedbackFile, JSON.stringify(await exportedFeedback())); await writeFile(kbMapFile, JSON.stringify(kbMap));
  const block = `data:text/javascript,${encodeURIComponent('globalThis.fetch=()=>{throw new Error("离线转换禁止HTTP")};')}`;
  const { stdout } = await exec(process.execPath, ['--import', block, resolve(root, 'go-zero-admin/data/scripts/merge-kb-feedback.mjs'),
    '--feedback', feedbackFile, '--cases', casesFile, '--kb-map', kbMapFile, '--output', output], { cwd: root });
  assert.equal(JSON.parse(stdout).http_requests, 0);
  const plan = await exec(process.execPath, ['--import', block, resolve(root, 'go-zero-admin/data/scripts/eval-kb-quality.mjs'), '--plan', '--cases', output], { cwd: root });
  assert.equal(JSON.parse(plan.stdout).case_count, 33);
  assert.equal(JSON.parse(plan.stdout).http_requests, 0);
}));

test('显式参数无默认网络执行，不接受缺参数、重复参数或run开关', () => {
  const parse = feature('parseArgs');
  assert.deepEqual(parse(['--feedback', 'f.json', '--cases', 'c.json', '--kb-map', 'k.json', '--output', 'tmp/new.json']),
    { feedbackFile: 'f.json', casesFile: 'c.json', kbMapFile: 'k.json', output: 'tmp/new.json' });
  assert.throws(() => parse([]));
  assert.throws(() => parse(['--feedback', 'f', '--feedback', 'f', '--cases', 'c', '--kb-map', 'k', '--output', 'tmp/x']));
  assert.throws(() => parse(['--run']));
  assert.throws(() => parse(['constructor', 'x', '--cases', 'c', '--kb-map', 'k', '--output', 'tmp/x']));
});
