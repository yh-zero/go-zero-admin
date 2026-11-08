import { createHash } from 'node:crypto';
import { open, readFile, realpath, stat } from 'node:fs/promises';
import { dirname, isAbsolute, relative, resolve } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { performance } from 'node:perf_hooks';
import { createApiClient, loadDataset, runEvaluation, validateContext } from './eval-kb-quality.mjs';

const repositoryRoot = resolve(dirname(fileURLToPath(import.meta.url)), '../../..');
const defaultCases = resolve(repositoryRoot, 'ai_document/ai-rag/quality-cases.json');
const levels = [1, 3, 5];
const sha = value => createHash('sha256').update(value).digest('hex');
const positiveID = value => Number.isSafeInteger(value) && value > 0;
const within = (root, filename) => { const path = relative(root, filename); return !!path && !isAbsolute(path) && path !== '..' && !path.startsWith('../') && !path.startsWith('..\\'); };
const laneKey = ({ concurrency, lane }) => `${concurrency}:${lane}`;

export function parseBenchArgs(args) {
  const options = { run: false, base: 'http://127.0.0.1:7001', cases: defaultCases };
  const values = { '--base': 'base', '--cases': 'cases', '--case-id': 'caseID', '--auth-file': 'authFile', '--kb-map': 'kbMapFile', '--context-file': 'contextFile', '--output': 'output' };
  const seen = new Set();
  for (let index = 0; index < args.length; index++) {
    const key = args[index]; if (seen.has(key)) throw new Error('参数重复'); seen.add(key);
    if (key === '--run') options.run = true;
    else if (key === '--plan') options.plan = true;
    else {
      if (!values[key] || !args[index + 1] || args[index + 1].startsWith('--')) throw new Error('未知参数或参数缺值');
      options[values[key]] = args[++index];
    }
  }
  if (options.run && options.plan) throw new Error('--run与--plan不能混用');
  if (options.caseID && (!options.caseID.trim() || /[,\s]/u.test(options.caseID))) throw new Error('case-id必须是单一非空题目ID');
  let base; try { base = new URL(options.base); } catch { throw new Error('base不是有效本机HTTP地址'); }
  if (base.protocol !== 'http:' || !['127.0.0.1', 'localhost', '[::1]'].includes(base.hostname) || base.username || base.password || base.pathname !== '/' || base.search || base.hash) throw new Error('base仅允许本机HTTP根地址');
  options.base = base.origin;
  if (options.run && ['caseID', 'authFile', 'kbMapFile', 'contextFile', 'output'].some(key => !options[key])) throw new Error('--run需要case-id、认证auth-file、kb-map、context-file和新output');
  if (options.output) {
    options.output = resolve(options.output);
    if (!within(resolve(repositoryRoot, 'tmp'), options.output) || !options.output.endsWith('.jsonl')) throw new Error('output必须是仓库忽略的tmp目录下新.jsonl文件');
  }
  return options;
}

export function performanceLatency(values) {
  if (!Array.isArray(values) || values.some(value => typeof value !== 'number' || !Number.isFinite(value) || value < 0)) throw new Error('耗时必须是非负有限数');
  const sorted = [...values].sort((left, right) => left - right);
  const quantile = probability => sorted[Math.max(0, Math.ceil(probability * sorted.length) - 1)] ?? null;
  return { n: sorted.length, p50_ms: quantile(0.5), p95_ms: quantile(0.95), min_ms: sorted[0] ?? null, max_ms: sorted.at(-1) ?? null, method: 'nearest-rank' };
}

function inputCase({ dataset, datasetSha256, caseID, kbMap, context }) {
  validateContext(context);
  if (typeof datasetSha256 !== 'string' || !/^[a-f0-9]{64}$/u.test(datasetSha256)) throw new Error('需要完整冻结题集SHA256');
  if (!dataset || dataset.version !== 1 || !Array.isArray(dataset.documents) || !dataset.documents.length || !Array.isArray(dataset.cases)) throw new Error('题集结构无效');
  const matches = dataset.cases.filter(item => item.id === caseID), item = matches[0];
  if (matches.length !== 1 || item.split !== 'development' || item.answer?.kind !== 'answerable' || typeof item.query !== 'string' || !item.query.trim()
    || !Array.isArray(item.expected_documents) || !item.expected_documents.length || item.expected_documents.some(key => !dataset.documents.some(doc => doc.key === key && doc.kb === item.kb))) throw new Error('性能基线必须显式选择唯一development正例，不能消耗保留集');
  if (!kbMap || !positiveID(kbMap[item.kb])) throw new Error('缺少有效知识库映射');
  return item;
}

function rendezvous(count) {
  let ready = 0, cancelled = false, release;
  const waiting = new Promise(resolvePromise => { release = resolvePromise; });
  return {
    arrive: async () => { ready++; if (ready === count) release(); await waiting; if (cancelled) throw new Error('同波次预检失败，取消尚未放行的工作'); },
    cancel: () => { cancelled = true; release(); },
  };
}

const emptyHTTP = () => ({ total: 0, get: 0, post: 0, search: 0, analysis_submissions: 0, task_reads: 0, failed: 0 });
function aggregateLanes(lanes) {
  const http = emptyHTTP();
  for (const lane of lanes) for (const key of Object.keys(http)) http[key] += lane.http[key] ?? 0;
  return {
    attempted_lanes: lanes.length, succeeded_lanes: lanes.filter(lane => lane.completed).length, failed_lanes: lanes.filter(lane => !lane.completed).length,
    unique_cases: lanes.length ? 1 : 0, search_requests: http.search, analysis_submissions: http.analysis_submissions,
    accepted_task_responses: lanes.filter(lane => lane.task_id !== null).length,
    unique_task_ids: new Set(lanes.filter(lane => lane.task_id !== null).map(lane => lane.task_id)).size, task_reads: http.task_reads,
    metadata_http_requests: http.total - http.search - http.analysis_submissions - http.task_reads,
    known_chat_calls: lanes.reduce((sum, lane) => sum + (lane.known_chat_calls ?? 0), 0),
    known_chat_total_tokens: lanes.reduce((sum, lane) => sum + (lane.known_chat_total_tokens ?? 0), 0),
    observed_analyses: lanes.filter(lane => lane.analysis_observed).length,
    rule_flagged_lanes: lanes.filter(lane => lane.completed && lane.rule_flags.length).length,
    truncated_lanes: lanes.filter(lane => lane.completed && lane.answer_status === 'truncated').length,
    no_match_lanes: lanes.filter(lane => lane.completed && lane.answer_status === 'no_matches').length,
    unknown_usage_lanes: lanes.filter(lane => lane.http.analysis_submissions > 0
      && (!lane.analysis_observed || (lane.answer_status !== 'no_matches' && !lane.usage_complete))).length, http,
    note: 'HTTP计数为客户端实际尝试次数；故障时供应商调用与用量未知。已知聊天tokens不含Embedding费用；9次重复只算1个独立问题。',
  };
}

const notes = [
  '固定同一开发正例按1、3、5顺序各运行一波，共9次重复；不是9个独立问题，且后续波次可能受缓存与供应商预热影响。',
  '目标并发是同时放行的客户端工作数；峰值为客户端观察，不能证明服务端实际模型并发数。',
  '检索和提交到观察终态分别计时，后者包括服务端排队、模型处理及轮询误差；没有精确内部排队/模型阶段计时。',
  '每lane单独保留严格串行v1质量日志，性能manifest不能作为串行质量日志使用；各lane均核对语料前后哈希、版本与库存。',
  '元数据核对HTTP次数另列；workload_wall_ms从所有lane语料预检完成后放行至最后结果/失败，full_wave_wall_ms另含语料核对及日志开销。',
  'P50/P95为所注明N的nearest-rank经验分位数；N=1/3/5很小，不代表生产容量、稳定尾延迟或并发上限。',
  '有错误时保留原UUID、载荷及已知任务ID，等待同波次已开始lane收束，停止下一波；不自动重发、取消或清理服务端任务。',
  '已观察结果中的已知聊天用量独立于语料/日志复核是否通过保留；证据失败的lane不进入成功耗时或质量统计，用量未知与证据无效分别记录。',
  '模型/构建/配置标签由操作者核对实际运行环境后提供；工具不独立证明服务端配置，规则筛查也不是语义正确率。',
];

export async function runBenchmark({ dataset, datasetSha256, caseID, kbMap, context, createClient, openLane,
  onManifest = async () => {}, pollIntervalMs = 750, taskTimeoutMs = 100_000, now = () => performance.now(), provenance = {} }) {
  const item = inputCase({ dataset, datasetSha256, caseID, kbMap, context });
  if (typeof createClient !== 'function' || typeof openLane !== 'function') throw new Error('缺少客户端或独立lane日志');
  if (!(taskTimeoutMs > 0 && taskTimeoutMs <= 100_000) || !(pollIntervalMs >= 0 && pollIntervalMs <= 5000)) throw new Error('任务超时或轮询间隔无效');
  const selectedDataset = { ...dataset, cases: [item] }, taskIDs = new Set(), requestIDs = new Set(), waves = [];
  const meta = { type: 'benchmark_meta', version: 1, started_at: new Date().toISOString(), levels, case_id: item.id,
    dataset_sha256: datasetSha256, kb_map: { [item.kb]: kbMap[item.kb] }, context, provenance, planned_analysis_submissions: 9, planned_logical_query_embeddings: 18, notes };
  await onManifest(meta);
  let corpusFingerprint;
  for (const concurrency of levels) {
    const gate = rendezvous(concurrency), waveStarted = now();
    let firstWorkloadStart = null, lastWorkloadEnd = null, activeLanes = 0, peakActiveLanes = 0, acceptedWaiters = 0, peakAcceptedWaiters = 0;
    await onManifest({ type: 'wave_started', target_concurrency: concurrency, started_at: new Date().toISOString() });
    const lanes = await Promise.all(Array.from({ length: concurrency }, async (_, index) => {
      const lane = index + 1, descriptor = { concurrency, lane }, events = [];
      const state = { lane, completed: false, outcome: 'not_submitted', evidence_status: 'pending', analysis_observed: false,
        request_id: null, task_id: null, status: null, retry: false, log_integrity: true };
      let log, client, active = false, waitingForTerminal = false;
      const append = async event => { try { await log.append(event); events.push(event); } catch (error) { state.log_integrity = false; throw error; } };
      function stopWorkload() {
        if (active) { active = false; activeLanes--; lastWorkloadEnd = now(); }
        if (waitingForTerminal) { waitingForTerminal = false; acceptedWaiters--; }
      }
      try {
        log = await openLane(descriptor);
        state.log_path = log.path;
        await append({ type: 'meta', version: 1, phase: 'baseline', split: 'development', started_at: new Date().toISOString(),
          dataset_sha256: datasetSha256, case_ids: [item.id], kb_map: meta.kb_map, context });
        client = createClient(descriptor);
        if (!client || typeof client !== 'function' || !client.counts) throw new Error('客户端缺少真实HTTP计数');
        const result = await runEvaluation({ dataset: selectedDataset, kbMap: meta.kb_map, client, pollIntervalMs, taskTimeoutMs, now, onProgress: async event => {
          if (event.type === 'result') {
            stopWorkload();
            Object.assign(state, { analysis_observed: true, outcome: 'succeeded', status: 'succeeded', search_client_ms: event.search_client_ms,
              search_server_ms: event.search_server_ms, analysis_elapsed_ms: event.analysis_elapsed_ms, answer_status: event.analysis.status,
              rule_flags: [...event.answer_screen.reasons], usage: { ...event.analysis.usage }, usage_complete: event.analysis.usage.complete,
              known_chat_calls: event.analysis.status === 'no_matches' ? 0 : 1, known_chat_total_tokens: event.analysis.usage.total_tokens });
          }
          if (event.type === 'analysis_requested') { state.request_id = event.request_id; state.outcome = 'unknown'; }
          if (event.type === 'analysis_accepted') { state.task_id = event.task_id; state.status = event.status; }
          if (event.type === 'analysis_terminal') { state.status = event.status; state.outcome = event.status === 'succeeded' ? 'succeeded' : 'failed'; }
          await append(event);
          if (event.type === 'bindings') {
            const fingerprint = JSON.stringify({ documents: event.documents, inventory: event.inventory });
            if (corpusFingerprint !== undefined && fingerprint !== corpusFingerprint) throw new Error('不同lane或波次语料哈希、版本、状态或库存变化');
            corpusFingerprint ??= fingerprint;
          }
          if (event.type === 'case_started') {
            await gate.arrive();
            active = true; activeLanes++; peakActiveLanes = Math.max(peakActiveLanes, activeLanes); firstWorkloadStart ??= now();
          }
          if (event.type === 'analysis_requested') {
            if (requestIDs.has(event.request_id)) throw new Error('不同lane原UUID重复，禁止作为独立请求');
            requestIDs.add(event.request_id);
          }
          if (event.type === 'analysis_accepted') {
            if (['failed', 'cancelled', 'rejected'].includes(event.status)) state.outcome = 'failed';
            if (taskIDs.has(event.task_id)) throw new Error('不同lane收到重复任务ID，禁止作为独立样本');
            taskIDs.add(event.task_id);
            waitingForTerminal = true; acceptedWaiters++; peakAcceptedWaiters = Math.max(peakAcceptedWaiters, acceptedWaiters);
          }
          if (event.type === 'analysis_terminal') {
            if (waitingForTerminal) { waitingForTerminal = false; acceptedWaiters--; }
            if (event.status !== 'succeeded') state.outcome = 'failed';
          }
        } });
        await append({ type: 'summary', completed: true, completed_at: new Date().toISOString(), ...result.summary });
        state.completed = true; state.evidence_status = 'verified';
      } catch (error) {
        gate.cancel(); stopWorkload(); state.error = error.message; state.evidence_status = 'invalid';
        if (log) {
          try { await append({ type: 'error', case_id: item.id, stopped_at: new Date().toISOString(), message: error.message,
            retry: false, comparable: false, calls: { ...(client?.counts ?? emptyHTTP()) } }); }
          catch { state.log_error = '失败记录也未能持久化，请核对manifest中的原UUID与已知任务ID'; }
        }
      } finally {
        stopWorkload();
        if (log) {
          try { await log.close(); }
          catch { gate.cancel(); Object.assign(state, { completed: false, evidence_status: 'invalid', log_integrity: false, log_error: '独立lane日志关闭失败' }); state.error ??= state.log_error; }
        }
      }
      state.http = { ...(client?.counts ?? emptyHTTP()) };
      state.log_sha256 = state.log_integrity ? sha(events.map(event => JSON.stringify(event) + '\n').join('')) : null;
      return state;
    }));
    const successes = lanes.filter(lane => lane.completed), wave = { target_concurrency: concurrency, lanes,
      completed: successes.length === concurrency, observed_active_lanes_peak: peakActiveLanes, observed_accepted_waiters_peak: peakAcceptedWaiters,
      full_wave_wall_ms: now() - waveStarted, workload_wall_ms: firstWorkloadStart === null ? null : lastWorkloadEnd - firstWorkloadStart,
      latency: { search_client: performanceLatency(successes.map(lane => lane.search_client_ms)), search_server: performanceLatency(successes.map(lane => lane.search_server_ms)),
        analysis_submit_to_terminal: performanceLatency(successes.map(lane => lane.analysis_elapsed_ms)) }, summary: aggregateLanes(lanes) };
    waves.push(wave);
    await onManifest({ type: 'wave_finished', ...wave });
    if (!wave.completed) break;
  }
  const result = { completed: waves.length === levels.length && waves.every(wave => wave.completed), meta, waves,
    summary: aggregateLanes(waves.flatMap(wave => wave.lanes)), notes };
  await onManifest({ type: 'benchmark_summary', completed: result.completed, completed_at: new Date().toISOString(), ...result.summary });
  return result;
}

export async function reserveBenchmarkOutputs(output) {
  const tmpRoot = await realpath(resolve(repositoryRoot, 'tmp')), filename = resolve(output), parent = await realpath(dirname(filename));
  if (!within(resolve(repositoryRoot, 'tmp'), filename) || !filename.endsWith('.jsonl') || (parent !== tmpRoot && !within(tmpRoot, parent))) throw new Error('输出必须位于真实tmp目录，不能经链接越界');
  const stem = filename.slice(0, -'.jsonl'.length), paths = [filename, `${stem}.md`, ...levels.flatMap(concurrency => Array.from({ length: concurrency }, (_, index) => `${stem}.c${concurrency}.l${index + 1}.jsonl`))];
  for (const path of paths) {
    try { await stat(path); throw new Error('结果文件已存在，拒绝覆盖历史结果'); }
    catch (error) { if (error.code !== 'ENOENT') throw error; }
  }
  const handles = [], reports = [];
  try {
    for (const path of paths) {
      const handle = await open(path, 'wx', 0o600); handles.push(handle);
      reports.push({ path: relative(repositoryRoot, path).replaceAll('\\', '/'), append: async value => { await handle.writeFile(JSON.stringify(value) + '\n'); await handle.sync(); },
        write: async value => { await handle.writeFile(value); await handle.sync(); }, close: async () => {} });
    }
  } catch (error) { await Promise.allSettled(handles.map(handle => handle.close())); throw new Error(error.code === 'EEXIST' ? '结果文件已存在，拒绝覆盖历史结果' : '无法独占创建全部结果文件；本轮未发HTTP，请保留已创建空文件核对'); }
  return { manifest: reports[0], markdown: reports[1], lanes: new Map(levels.flatMap(concurrency => Array.from({ length: concurrency }, (_, index) => ({ concurrency, lane: index + 1 })))
    .map((descriptor, index) => [laneKey(descriptor), reports[index + 2]])), close: async () => {
      const results = await Promise.allSettled(handles.map(handle => handle.close()));
      const failure = results.find(result => result.status === 'rejected'); if (failure) throw failure.reason;
    } };
}

export function renderBenchmarkReport(result) {
  const display = value => value === null ? '—' : Number(value.toFixed(2));
  const markdownText = value => String(value).replace(/[\r\n]/gu, ' ').replace(/[\\`*_{}\[\]()#+\-.!|>]/gu, '\\$&');
  const lines = ['# 知识库小规模并发基线', '', result.completed ? '本轮计划执行完成；以下为观测结果，不代表生产容量。' : '本轮失败或不完整；停止后续波次，保留已知ID，不计算性能提升。', '',
    `- 题目：${markdownText(result.meta.case_id)}（development，同题重复，独立问题数1）`, `- 题集SHA256：${result.meta.dataset_sha256}`, `- 构建SHA256：${result.meta.context.binary_sha}`, '',
    '| 目标并发 | 成功/尝试 | 检索客户端P50/P95 ms | 提交到观察终态P50/P95 ms | 活动lane峰值 | 元数据HTTP |', '|---:|---:|---:|---:|---:|---:|'];
  for (const wave of result.waves) {
    const search = wave.latency.search_client, analysisLatency = wave.latency.analysis_submit_to_terminal;
    lines.push(`| ${wave.target_concurrency} | ${wave.summary.succeeded_lanes}/${wave.summary.attempted_lanes} | ${display(search.p50_ms)}/${display(search.p95_ms)}（N=${search.n}） | ${display(analysisLatency.p50_ms)}/${display(analysisLatency.p95_ms)}（N=${analysisLatency.n}） | ${wave.observed_active_lanes_peak} | ${wave.summary.metadata_http_requests} |`);
  }
  lines.push('', '## 调用与用量', '', '```json', JSON.stringify(result.summary, null, 2), '```', '', '## 原始记录', '', '| 并发/lane | 原UUID | 任务ID | 任务结果 / 证据核验 | 日志 |', '|---|---|---:|---|---|');
  for (const wave of result.waves) for (const lane of wave.lanes) lines.push(`| ${wave.target_concurrency}/${lane.lane} | ${lane.request_id ?? '—'} | ${lane.task_id ?? '—'} | ${lane.outcome} / ${lane.evidence_status} | ${markdownText(lane.log_path ?? '未创建')} |`);
  lines.push('', '## 测量边界', '', ...result.notes.map(note => `- ${note}`), '');
  return lines.join('\n');
}

async function readJSON(filename, label) {
  try { return JSON.parse(await readFile(filename, 'utf8')); } catch { throw new Error(`${label}不可读或不是有效JSON`); }
}

async function main() {
  const options = parseBenchArgs(process.argv.slice(2)), dataset = await loadDataset(options.cases);
  const selected = dataset.cases.find(item => item.id === options.caseID);
  if (options.caseID && (!selected || selected.split !== 'development' || selected.answer.kind !== 'answerable')) throw new Error('case-id必须是题集中唯一development正例');
  if (!options.run) {
    console.log(JSON.stringify({ mode: 'plan', http_requests: 0, levels, case_id: selected?.id ?? null,
      eligible_development_cases: dataset.cases.filter(item => item.split === 'development' && item.answer.kind === 'answerable').map(item => item.id),
      max_analysis_submissions: 9, max_logical_query_embeddings: 18, independent_case_count: 1,
      usage: '--run --cases <冻结题集> --case-id <单一开发正例> --auth-file <正常登录result> --kb-map <JSON> --context-file <消毒运行标签JSON> --output tmp/<新路径>.jsonl', notes }, null, 2)); return;
  }
  const auth = await readJSON(options.authFile, '认证文件'), context = await readJSON(options.contextFile, 'context'), kbMap = await readJSON(options.kbMapFile, 'kb-map');
  if (typeof auth.access_token !== 'string' || !auth.access_token.trim()) throw new Error('认证令牌缺失');
  if (auth.expires_at !== undefined && (!positiveID(auth.expires_at) || auth.expires_at * 1000 <= Date.now() + 1000)) throw new Error('认证已过期，请正常登录后显式执行');
  const datasetSha256 = sha(await readFile(options.cases));
  inputCase({ dataset, datasetSha256, caseID: options.caseID, kbMap, context });
  const reports = await reserveBenchmarkOutputs(options.output);
  try {
    const result = await runBenchmark({ dataset, datasetSha256, caseID: options.caseID, kbMap, context,
      provenance: { base_url: options.base, benchmark_runner_sha256: sha(await readFile(fileURLToPath(import.meta.url))),
        quality_runner_sha256: sha(await readFile(new URL('./eval-kb-quality.mjs', import.meta.url))) },
      createClient: () => createApiClient({ token: auth.access_token, base: options.base }), openLane: descriptor => reports.lanes.get(laneKey(descriptor)),
      onManifest: event => reports.manifest.append(event) });
    await reports.markdown.write(renderBenchmarkReport(result));
    console.log(JSON.stringify({ completed: result.completed, output: relative(repositoryRoot, options.output), ...result.summary }));
    if (!result.completed) process.exitCode = 1;
  } finally { await reports.close(); }
}

if (process.argv[1] && pathToFileURL(resolve(process.argv[1])).href === import.meta.url) main().catch(error => { console.error(error.message); process.exitCode = 1; });
