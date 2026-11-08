import { queryOptions } from '@tanstack/react-query';
import { fetchAiKbDoc, fetchAiKbDocs, fetchAiKbFeedback } from '../../api/ai';
import type { AiKbDocDetailResp, AiKbFeedbackInfo } from '../../api/types';
import { feedbackRegressionJson, parseFeedbackSnapshot } from './knowledgeFeedback';
import { importQualityDataset, qualitySha256 } from './knowledgeQuality';
import type { QualityCase, QualityDocument } from './knowledgeQuality';
import { isAiWorkspaceCurrent } from './useAiWorkspaceGuard';
import type { AiWorkspaceScope } from './workspaceQueries';
import { aiWorkspaceKeys } from './workspaceQueries';

const bytes = (value: string) => new TextEncoder().encode(value).length;
const normalized = (value: string) => value.normalize('NFKC').replace(/\s+/gu, '');

export async function buildFeedbackDataset(record: AiKbFeedbackInfo, current: AiKbDocDetailResp[], unboundConfirmed: boolean) {
  const exported = feedbackRegressionJson(record), snapshot = parseFeedbackSnapshot(record.snapshot)!;
  const review = record.review!;
  if (!review.answerable && !unboundConfirmed) throw new Error('请先明确核对当前知识库资料，原反馈不证明全库无答案');
  if (!current.length || current.length > 100) throw new Error('需要1..100篇当前就绪资料才能冻结题集');
  const documents: QualityDocument[] = [], seen = new Set<number>();
  for (const { doc, content } of current) {
    if (doc.kb_id !== record.kb_id || !Number.isSafeInteger(doc.id) || doc.id < 1 || seen.has(doc.id) || doc.status !== 'ready' || !Number.isSafeInteger(doc.version) || doc.version < 1 || !doc.title.trim() || !content.trim()) throw new Error('当前资料ID、版本、状态或正文无法核对');
    if (bytes(content) > 262_144) throw new Error('冻结题集超过256 KiB大小限制，请缩小资料范围');
    const sha256 = await qualitySha256(content);
    if (review.answerable) {
      const previous = snapshot.documents.find(item => item.doc_id === doc.id);
      if (!previous || previous.status !== 'verified' || previous.kb_id !== doc.kb_id || previous.version !== doc.version || previous.title !== doc.title || previous.content_sha256 !== sha256) throw new Error('来源ID、版本、标题或正文指纹已变化，请重新复核反馈');
    }
    const key = `doc-${doc.id}`;
    documents.push({ key, kb: `kb-${record.kb_id}`, source: `inline:${key}`, title: doc.title, sha256, source_doc_id: doc.id, source_doc_version: doc.version, content });
    seen.add(doc.id);
  }
  const evidence = new Map<string, string[][]>();
  for (const row of review.evidence) {
    const source = snapshot.sources.find(item => item.index === row.source_index);
    const document = source && documents.find(item => item.source_doc_id === source.doc_id);
    if (!source || !document || source.doc_title !== document.title || !source.snippet.includes(row.quote) || !document.content!.includes(row.quote)) throw new Error('已采纳摘录无法逐字对应当前原文，请重新复核');
    const groups = evidence.get(document.key) ?? [];
    if (!groups.some(group => group[0] === row.quote)) groups.push([row.quote]);
    evidence.set(document.key, groups);
  }
  if (review.required_facts.some(fact => !normalized(review.expected_answer).includes(normalized(fact)) || !review.evidence.some(row => normalized(row.quote).includes(normalized(fact))))) throw new Error('事实检查项无法对应采纳结论与摘录');
  const id = `feedback-${record.kb_id}-${record.id}-r${record.revision}`;
  const item: QualityCase = { id, kb: `kb-${record.kb_id}`, split: 'development', category: 'feedback_regression', query: record.question,
    rationale: review.reason, expected_documents: [...evidence.keys()], evidence: [...evidence].map(([document, groups]) => ({ document, groups })),
    answer: { kind: review.answerable ? 'answerable' : 'unanswerable', required_groups: review.required_facts.map(fact => [fact]), forbidden_phrases: [] },
    expected_answer_example: review.expected_answer,
    checks: review.required_facts.map((fact, required_group) => ({ label: fact, kind: 'fact', required_group })) };
  return importQualityDataset(JSON.stringify({ version: 1,
    description: `反馈 #${record.id} 版本 ${record.revision} 的开发题。采纳反馈不等于质量签署；规则仅做词法筛查，仍需人工核对。${review.answerable ? '' : '资料不足结论由用户明确核对本次冻结的当前就绪资料，不由原no_matches推定。'}`,
    documents, cases: [item], metadata: { feedback_regressions: [{ case_id: id, feedback_id: record.id, revision: record.revision, kb_id: record.kb_id, task_id: record.task_id, step_index: record.step_index,
      reviewed_at: record.reviewed_at, export_sha256: await qualitySha256(exported), snapshot_sha256: await qualitySha256(record.snapshot), review_origin: 'exported_review_record', human_identity_verified: false,
      note: '采纳记录只说明本次版本与来源核对，不证明审核者身份或语义质量。' }] } }, null, 2) + '\n');
}

export function feedbackDatasetOptions(scope: AiWorkspaceScope, kbId: number, expected: AiKbFeedbackInfo, unboundConfirmed: boolean) {
  return queryOptions({ queryKey: [...aiWorkspaceKeys.kbFeedback(scope, kbId, expected.id), 'dataset', expected.revision, unboundConfirmed], staleTime: 0, retry: false, gcTime: 0,
    queryFn: async ({ signal }) => {
      const assertCurrent = () => { signal.throwIfAborted(); if (!isAiWorkspaceCurrent(scope, 'sys:ai:kb:list')) throw new Error('当前工作区或读取权限已变化'); };
      const read = async <T,>(fetch: () => Promise<T>) => { assertCurrent(); const result = await fetch(); assertCurrent(); return result; };
      if (expected.kb_id !== kbId) throw new Error('反馈知识库归属不符');
      const first = await read(() => fetchAiKbFeedback(kbId, expected.id, signal));
      const same = (value: AiKbFeedbackInfo) => value.id === expected.id && value.kb_id === kbId && value.revision === expected.revision && value.status === 'accepted' && value.snapshot === expected.snapshot && JSON.stringify(value.review) === JSON.stringify(expected.review);
      if (!same(first.feedback)) throw new Error('反馈版本或采纳结论已变化，请刷新后重新核对');
      feedbackRegressionJson(first.feedback);
      const snapshot = parseFeedbackSnapshot(first.feedback.snapshot)!;
      let ids: number[];
      if (first.feedback.review!.answerable) {
        if (snapshot.documents.some(doc => doc.status !== 'verified' || doc.version < 1) || new Set(snapshot.documents.map(doc => doc.doc_id)).size !== snapshot.documents.length
          || snapshot.sources.some(source => !snapshot.documents.some(doc => doc.doc_id === source.doc_id && doc.title === source.doc_title))) throw new Error('历史来源资料无法完整核对，请重新复核');
        ids = snapshot.documents.map(doc => doc.doc_id);
      } else {
        if (!unboundConfirmed) throw new Error('请先明确核对当前知识库资料');
        const list = await read(() => fetchAiKbDocs(kbId, { page: 1, page_size: 100, status: 'ready' }, signal));
        if (!Array.isArray(list.list) || list.total !== list.list.length || list.total < 1 || list.total > 100 || list.list.some(doc => doc.kb_id !== kbId || doc.status !== 'ready')) throw new Error('当前就绪资料为空、超出100篇或清单不完整，无法冻结');
        ids = list.list.map(doc => doc.id);
      }
      const documents: AiKbDocDetailResp[] = [];
      // 顺序读并及时限制体积，避免大库一次并发拉取全文。
      let total = 0;
      for (const id of ids) {
        const detail = await read(() => fetchAiKbDoc(kbId, id, false, signal));
        if (detail.doc.id !== id) throw new Error('资料读取ID不符');
        total += bytes(detail.content); if (total > 262_144) throw new Error('冻结题集超过256 KiB大小限制');
        documents.push(detail);
      }
      const last = await read(() => fetchAiKbFeedback(kbId, expected.id, signal));
      if (!same(last.feedback)) throw new Error('读取期间反馈版本变化，请重新核对');
      const result = await buildFeedbackDataset(last.feedback, documents, unboundConfirmed); assertCurrent(); return result;
    } });
}

export function downloadQualityDataset(raw: string, filename: string) {
  const url = URL.createObjectURL(new Blob([raw], { type: 'application/json;charset=utf-8' }));
  const link = document.createElement('a'); link.href = url; link.download = filename; document.body.append(link); link.click(); link.remove();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}
