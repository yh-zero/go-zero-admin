export interface QualityAnswerScreen {
  label: string; flagged: boolean; requires_human_review: boolean; reasons: string[];
  required_groups_matched: boolean[]; forbidden_phrases_matched: string[]; citation_indices: number[];
  cited_evidence_groups_matched: boolean[];
}
export interface QualityRetrieval {
  kind: 'positive' | 'negative'; candidate_count?: number; max_score?: number | null;
  hit_at?: Record<number, number>; recall_at?: Record<number, number>; mrr_at?: Record<number, number>;
  evidence_at?: Record<number, { covered: number; total: number; coverage: number; complete: boolean; matched_groups: boolean[] }>;
}
export interface QualityLatency { n: number; p50_ms: number | null; p95_ms: number | null; min_ms: number | null; max_ms: number | null; method: string }
export interface QualitySummary {
  n: number; positive_count: number; negative_count: number;
  positive: { hit_at: Record<number, number | null>; recall_at: Record<number, number | null>; mrr_at: Record<number, number | null>; evidence_coverage_at: Record<number, number | null>; complete_evidence_rate_at: Record<number, number | null> };
  answer_rule_screen: { flagged_count: number; requires_human_review_count: number; reasons: Record<string, number> };
  latency: { search_client: QualityLatency | null; search_server: QualityLatency | null; analysis_submit_to_terminal: QualityLatency | null };
  usage: { scope: 'chat'; known_prompt_tokens: number; known_completion_tokens: number; known_total_tokens: number; complete_cases: number; incomplete_cases: number; note: string };
}
export const ks: readonly number[];
export function normalized(value: string): string;
export function validateAnalysis<T>(value: T): T;
export function evaluateRetrieval(item: unknown, list: readonly unknown[], bindings: Map<string, unknown>): QualityRetrieval;
export function screenAnswer(item: unknown, analysis: unknown, kbID: number, knownDocuments: Map<number, unknown>, bindings?: Map<string, unknown>): QualityAnswerScreen;
export function empiricalLatency(values: number[]): QualityLatency | null;
export function summarize(results: readonly unknown[], calls?: Record<string, unknown>): QualitySummary;
