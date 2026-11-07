import type { AiSearchKbHit } from '../../api/types';

export interface WorkflowAnalysisResult {
  version: 1;
  type: 'kb_analysis';
  status: 'completed' | 'no_matches' | 'truncated';
  analysis: string;
  sources: (AiSearchKbHit & { index: number })[];
  usage: { scope: 'chat'; prompt_tokens: number; completion_tokens: number; total_tokens: number; complete: boolean };
}

export interface TaskAnalysis {
  step: number;
  total: number;
  name: string;
  result: WorkflowAnalysisResult;
}

function hasFields(value: unknown, fields: string[]): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
    && Object.keys(value).length === fields.length && fields.every(field => Object.hasOwn(value, field));
}

function isCount(value: unknown, minimum = 0): value is number {
  return typeof value === 'number' && Number.isSafeInteger(value) && value >= minimum;
}

function isAnalysis(value: unknown): value is WorkflowAnalysisResult {
  if (!hasFields(value, ['version', 'type', 'status', 'analysis', 'sources', 'usage'])
    || value.version !== 1 || value.type !== 'kb_analysis'
    || !['completed', 'no_matches', 'truncated'].includes(String(value.status))
    || typeof value.analysis !== 'string' || !Array.isArray(value.sources)) return false;
  if (value.status === 'no_matches' ? value.sources.length !== 0 : !value.sources.length || !value.analysis.trim()) return false;
  if (!value.sources.every((source: unknown, index: number) => hasFields(source, ['index', 'chunk_id', 'kb_id', 'doc_id', 'doc_title', 'snippet', 'score'])
    && source.index === index + 1 && isCount(source.chunk_id, 1) && isCount(source.kb_id, 1) && isCount(source.doc_id, 1)
    && typeof source.doc_title === 'string' && typeof source.snippet === 'string' && !!source.snippet.trim()
    && typeof source.score === 'number' && Number.isFinite(source.score))) return false;
  return hasFields(value.usage, ['scope', 'prompt_tokens', 'completion_tokens', 'total_tokens', 'complete'])
    && value.usage.scope === 'chat' && isCount(value.usage.prompt_tokens) && isCount(value.usage.completion_tokens)
    && isCount(value.usage.total_tokens) && typeof value.usage.complete === 'boolean';
}

function parseAnalysisRecord(body: string): Pick<TaskAnalysis, 'name' | 'result'> | null {
  // 新工具只输出单行 JSON；多行旧工具内容不能可靠区分名称与执行记录，保留原文。
  if (body.includes('\n') || body.includes('\r')) return null;
  let parsed: Pick<TaskAnalysis, 'name' | 'result'> | null = null;
  for (let separator = body.indexOf(': {'); separator !== -1; separator = body.indexOf(': {', separator + 1)) {
    const name = body.slice(0, separator);
    if (!name.trim() || Array.from(name).length > 128) continue;
    try {
      const result: unknown = JSON.parse(body.slice(separator + 2));
      if (!isAnalysis(result)) continue;
      if (parsed) return null;
      parsed = { name, result };
    } catch { /* 旧文本或步骤名可能含冒号，不能据此拆掉原文。 */ }
  }
  return parsed;
}

export function parseTaskAnalyses(raw: string, completedSteps: number): TaskAnalysis[] {
  if (!isCount(completedSteps, 1) || completedSteps > 32) return [];
  const headers = [...raw.matchAll(/^\[([1-9]\d*)\/([1-9]\d*)\] /gm)];
  if (headers.length !== completedSteps || headers[0]?.index !== 0) return [];
  const total = Number(headers[0][2]);
  if (total < completedSteps || total > 32 || headers.some((header, index) => Number(header[1]) !== index + 1 || Number(header[2]) !== total)) return [];
  const analyses: TaskAnalysis[] = [];
  for (const [index, header] of headers.entries()) {
    const end = index + 1 < headers.length ? headers[index + 1].index! - 1 : raw.length;
    const body = raw.slice(header.index! + header[0].length, end);
    const parsed = parseAnalysisRecord(body);
    if (parsed) analyses.push({ step: index + 1, total, ...parsed });
  }
  return analyses;
}
