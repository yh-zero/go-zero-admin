export const ks = [1, 3, 5];
export const normalized = value => value.normalize('NFKC').replace(/\s+/gu, '');
const mean = values => values.length ? values.reduce((sum, value) => sum + value, 0) / values.length : null;
const matchGroups = (groups, texts) => groups.map(group => group.some(value => texts.some(text => normalized(text).includes(normalized(value)))));
const absenceClaimPattern = /(未提供|未提及|未说明|未记载|未给出|未涉及|未包含|未能确定|无法确定|无法确认|无法判断|不能确定|不能确认|资料不足|信息不足|证据不足|未知|没有提供|没有说明)/u;

export function validateAnalysis(result) {
  const nonnegative = value => Number.isSafeInteger(value) && value >= 0, positiveID = value => Number.isSafeInteger(value) && value > 0;
  if (!result || result.version !== 1 || result.type !== 'kb_analysis' || !['completed', 'no_matches', 'truncated'].includes(result.status) || typeof result.analysis !== 'string' || !result.analysis.trim()
    || !Array.isArray(result.sources) || result.sources.length > 8 || !result.usage || result.usage.scope !== 'chat' || typeof result.usage.complete !== 'boolean'
    || ['prompt_tokens', 'completion_tokens', 'total_tokens'].some(key => !nonnegative(result.usage[key]))) throw new Error('分析输出或已知用量字段无效');
  const encoder = new TextEncoder();
  if (encoder.encode(result.analysis).length > 12000 || encoder.encode(JSON.stringify(result.analysis)).length > 13000) throw new Error('分析正文超过原始或编码字节上限');
  if (encoder.encode(JSON.stringify(result.sources)).length > 10000 || encoder.encode(JSON.stringify(result)).length > 24000) throw new Error('分析或来源JSON超过字节上限');
  for (const source of result.sources) if (!positiveID(source.index) || source.index > 8 || !positiveID(source.chunk_id) || !positiveID(source.doc_id) || !positiveID(source.kb_id)
    || typeof source.doc_title !== 'string' || typeof source.snippet !== 'string' || !source.snippet.trim() || typeof source.score !== 'number' || !Number.isFinite(source.score)) throw new Error('分析来源字段无效');
  if ((result.status === 'no_matches' && result.sources.length) || (result.status !== 'no_matches' && !result.sources.length)
    || (result.status === 'no_matches' && (result.usage.complete || ['prompt_tokens', 'completion_tokens', 'total_tokens'].some(key => result.usage[key] !== 0)))
    || (result.usage.complete && (!result.usage.prompt_tokens || !result.usage.completion_tokens || result.usage.total_tokens !== result.usage.prompt_tokens + result.usage.completion_tokens))) throw new Error('分析状态、来源或用量不一致');
  return result;
}

export function evaluateRetrieval(item, list, bindings) {
  if (item.answer.kind === 'unanswerable') return { kind: 'negative', candidate_count: list.length,
    max_score: list.length ? Math.max(...list.map(hit => hit.score)) : null };
  const expected = new Set(item.expected_documents.map(key => bindings.get(key).doc_id));
  const result = { kind: 'positive', expected_document_count: expected.size, hit_at: {}, recall_at: {}, mrr_at: {}, evidence_at: {} };
  for (const k of ks) {
    const top = list.slice(0, k), found = new Set(top.filter(hit => expected.has(hit.doc_id)).map(hit => hit.doc_id));
    const rank = top.findIndex(hit => expected.has(hit.doc_id));
    const groups = item.evidence.flatMap(evidence => matchGroups(evidence.groups,
      top.filter(hit => hit.doc_id === bindings.get(evidence.document).doc_id).map(hit => hit.snippet)));
    result.hit_at[k] = Number(found.size > 0);
    result.recall_at[k] = found.size / expected.size;
    result.mrr_at[k] = rank < 0 ? 0 : 1 / (rank + 1);
    result.evidence_at[k] = { covered: groups.filter(Boolean).length, total: groups.length,
      coverage: groups.filter(Boolean).length / groups.length, complete: groups.every(Boolean), matched_groups: groups };
  }
  return result;
}

export function screenAnswer(item, analysis, kbID, knownDocuments, bindings) {
  const reasons = [], answer = analysis.analysis, sources = analysis.sources;
  const matched = matchGroups(item.answer.required_groups, [answer]);
  if (matched.some(value => !value)) reasons.push(item.answer.kind === 'unanswerable' ? 'missing_insufficiency' : 'missing_required_facts');
  // 仅显式标注的新题启用；词法无法区分引用、否定或缺口范围，命中只能提示人工复核。
  if (item.answer.absence_policy !== undefined) {
    const hasAbsenceClaim = absenceClaimPattern.test(normalized(answer));
    if (item.answer.absence_policy === 'forbidden' && hasAbsenceClaim) reasons.push('additional_absence_claim_needs_review');
    if (item.answer.absence_policy === 'required' && !hasAbsenceClaim) reasons.push('missing_scoped_insufficiency');
  }
  // 资料不足措辞和禁答内容独立核对，避免“资料不足，但是编造结论”通过筛查。
  if (item.answer.kind === 'unanswerable' && !/(资料不足|无法确定|无法回答|未提供|未检索到|没有.*资料|未.*说明|不能确定|无.*依据)/u.test(answer)) reasons.push('missing_insufficiency');
  const forbidden = item.answer.forbidden_phrases.filter(phrase => normalized(answer).includes(normalized(phrase)));
  if (forbidden.length) reasons.push('forbidden_phrase');
  const indices = new Set(sources.map(source => source.index));
  const cited = [...answer.matchAll(/\[(\d+)\]/gu)].map(match => Number(match[1]));
  if (cited.some(index => !indices.has(index) || index < 1)) reasons.push('invalid_citation');
  if (item.answer.kind === 'answerable' && !cited.length) reasons.push('missing_citation');
  if (/\[[^\]\n]*\d[^\]\n]*\]/u.test(answer.replace(/\[\d+\]/gu, '')) || /\[\s*\d/u.test(answer.replace(/\[[1-9]\d*\]/gu, '')) || /[【［]\s*\d/u.test(answer)) reasons.push('malformed_citation');
  if (sources.some((source, offset) => {
    const known = knownDocuments.get(source.doc_id);
    return source.index !== offset + 1 || !known || known.status !== 'ready' || source.kb_id !== kbID || source.doc_title !== known.title;
  })) reasons.push('source_scope_mismatch');
  if (new Set(sources.map(source => source.chunk_id)).size !== sources.length) reasons.push('duplicate_source_chunk');
  const boundIDs = bindings ? new Set([...bindings.values()].map(binding => binding.doc_id)) : null;
  if (boundIDs && sources.some(source => !boundIDs.has(source.doc_id))) reasons.push('unbound_source');
  if (item.answer.kind === 'answerable' && !sources.length) reasons.push('missing_sources');
  let citedEvidence = [];
  if (item.answer.kind === 'answerable' && bindings) {
    const citedSources = sources.filter(source => cited.includes(source.index) && source.kb_id === kbID && boundIDs.has(source.doc_id) && knownDocuments.get(source.doc_id)?.title === source.doc_title);
    if (item.expected_documents.some(key => !citedSources.some(source => source.doc_id === bindings.get(key)?.doc_id))) reasons.push('missing_expected_citations');
    citedEvidence = item.evidence.flatMap(evidence => matchGroups(evidence.groups, citedSources.filter(source => source.doc_id === bindings.get(evidence.document)?.doc_id).map(source => source.snippet)));
    if (citedEvidence.some(value => !value)) reasons.push('cited_evidence_incomplete');
  }
  if (analysis.status === 'truncated') reasons.push('truncated_answer');
  if (item.answer.kind === 'answerable' && analysis.status === 'no_matches') reasons.push('no_matches_for_answerable');
  return { label: '规则筛查，非语义准确率或人工评审', flagged: reasons.length > 0, requires_human_review: true,
    reasons: [...new Set(reasons)], required_groups_matched: matched, forbidden_phrases_matched: forbidden,
    citation_indices: cited, cited_evidence_groups_matched: citedEvidence };
}

export function empiricalLatency(values) {
  if (!values.length) return null;
  const sorted = [...values].sort((a, b) => a - b);
  const quantile = p => sorted[Math.max(0, Math.ceil(p * sorted.length) - 1)];
  return { n: sorted.length, p50_ms: quantile(0.5), p95_ms: quantile(0.95), min_ms: sorted[0], max_ms: sorted.at(-1),
    method: '串行样本经验分位数，nearest-rank；非生产并发或容量指标' };
}

function summarizeGroup(results) {
  const positives = results.filter(row => row.retrieval.kind === 'positive'), negatives = results.filter(row => row.retrieval.kind === 'negative');
  return { n: results.length, positive_count: positives.length, negative_count: negatives.length,
    positive: { hit_at: Object.fromEntries(ks.map(k => [k, mean(positives.map(row => row.retrieval.hit_at[k]))])),
      recall_at: Object.fromEntries(ks.map(k => [k, mean(positives.map(row => row.retrieval.recall_at[k]))])),
      mrr_at: Object.fromEntries(ks.map(k => [k, mean(positives.map(row => row.retrieval.mrr_at[k]))])),
      evidence_coverage_at: Object.fromEntries(ks.map(k => [k, mean(positives.map(row => row.retrieval.evidence_at[k].coverage))])),
      complete_evidence_rate_at: Object.fromEntries(ks.map(k => [k, mean(positives.map(row => Number(row.retrieval.evidence_at[k].complete)))])) },
    negative: { candidate_return_rate: mean(negatives.map(row => Number(row.retrieval.candidate_count > 0))),
      cases: negatives.map(row => ({ case_id: row.case_id, ...row.retrieval, rule_reasons: row.answer_screen.reasons })) },
    answer_rule_screen: { flagged_count: results.filter(row => row.answer_screen.flagged).length, requires_human_review_count: results.length,
      reasons: Object.fromEntries([...new Set(results.flatMap(row => row.answer_screen.reasons))].sort().map(reason => [reason, results.filter(row => row.answer_screen.reasons.includes(reason)).length])) },
    latency: { search_client: empiricalLatency(results.map(row => row.search_client_ms)), search_server: empiricalLatency(results.map(row => row.search_server_ms)),
      analysis_submit_to_terminal: empiricalLatency(results.map(row => row.analysis_elapsed_ms)) } };
}

export function summarize(results, calls = {}) {
  const groupBy = field => Object.fromEntries([...new Set(results.map(row => row[field]))].sort().map(key => [key, summarizeGroup(results.filter(row => row[field] === key))]));
  const analyses = results.map(row => row.analysis);
  return { ...summarizeGroup(results), by_split: groupBy('split'), by_category: groupBy('category'), calls,
    usage: { scope: 'chat', known_prompt_tokens: analyses.reduce((sum, row) => sum + row.usage.prompt_tokens, 0),
      known_completion_tokens: analyses.reduce((sum, row) => sum + row.usage.completion_tokens, 0),
      known_total_tokens: analyses.reduce((sum, row) => sum + row.usage.total_tokens, 0),
      complete_cases: analyses.filter(row => row.usage.complete).length, incomplete_cases: analyses.filter(row => !row.usage.complete).length,
      note: '仅供应商已知聊天用量，不含Embedding费用；complete=false不能按零费用解释' },
    notes: ['K为原始前K片段，文档Recall先按doc_id去重，MRR为任一相关文档首次出现的截断倒数排名',
      '证据覆盖与回答筛查仅做NFKC/空白规范化字符串规则匹配，全部答案仍需人工复核',
      '负例非空候选不等于有答案；相似度不代表置信度，不用小样本选择通用阈值',
      '串行经验P50/P95注明N；失败题不进入成功题指标分母，失败/不完整运行不可比较'] };
}
