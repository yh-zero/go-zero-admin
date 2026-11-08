import { queryOptions } from '@tanstack/react-query';
import { fetchAiKbEvaluation, fetchAiKbEvaluationDatasets, fetchAiKbEvaluations } from '../../api/ai';
import type { AiKbEvaluationInfo, AiListKbEvaluationsReq, PageResult } from '../../api/types';
import { aiWorkspaceKeys } from './workspaceQueries';
import type { AiWorkspaceScope } from './workspaceQueries';
import { assertQualityRunInfo, decodeQualityEvaluation, parseQualityDataset } from './knowledgeQuality';
import type { QualityCatalogue, QualityRunView } from './knowledgeQuality';

export function qualityCatalogueOptions(scope: AiWorkspaceScope, kbId: number) {
  return queryOptions<QualityCatalogue[]>({ queryKey: aiWorkspaceKeys.qualityDatasets(scope, kbId), staleTime: 30_000, retry: false,
    queryFn: async ({ signal }) => {
      const result = await fetchAiKbEvaluationDatasets(kbId, signal); signal.throwIfAborted();
      if (!Array.isArray(result.list) || result.list.length > 10 || new Set(result.list.map(item => item.key)).size !== result.list.length) throw new Error('题集列表格式无效');
      const parsed = await Promise.all(result.list.map(parseQualityDataset)); signal.throwIfAborted(); return parsed;
    } });
}
export function qualityListOptions(scope: AiWorkspaceScope, kbId: number, params: AiListKbEvaluationsReq) {
  return queryOptions<PageResult<AiKbEvaluationInfo>>({ queryKey: [...aiWorkspaceKeys.qualityRuns(scope, kbId), params], staleTime: 0, retry: false,
    refetchInterval: query => !query.state.fetchFailureCount && query.state.data?.list.some(run => run.status === 'pending' || run.status === 'running') ? 5000 : false,
    refetchIntervalInBackground: false,
    queryFn: async ({ signal }) => {
      const result = await fetchAiKbEvaluations(kbId, params, signal); signal.throwIfAborted();
      if (!Array.isArray(result.list) || !Number.isSafeInteger(result.total) || result.total < 0 || new Set(result.list.map(r => r.id)).size !== result.list.length) throw new Error('批次列表格式无效');
      for (const run of result.list) assertQualityRunInfo(run, kbId);
      return result;
    } });
}
export function qualityRunOptions(scope: AiWorkspaceScope, kbId: number, id: number) {
  return queryOptions<QualityRunView>({ queryKey: aiWorkspaceKeys.qualityRun(scope, kbId, id), staleTime: 0, retry: false, refetchOnWindowFocus: false,
    refetchInterval: query => !query.state.fetchFailureCount && query.state.data && ['pending', 'running'].includes(query.state.data.run.status) ? 2000 : false,
    refetchIntervalInBackground: false,
    queryFn: async ({ signal }) => {
      const result = await fetchAiKbEvaluation(kbId, id, signal); signal.throwIfAborted();
      if (result.evaluation.id !== id) throw new Error('批次标识不符');
      const parsed = await decodeQualityEvaluation(result.evaluation, kbId); signal.throwIfAborted(); return parsed;
    } });
}
