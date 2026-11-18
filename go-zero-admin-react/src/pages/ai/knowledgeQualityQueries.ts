import { queryOptions } from '@tanstack/react-query';
import { fetchAiKbEvaluation, fetchAiKbEvaluationDatasets, fetchAiKbEvaluations } from '../../api/ai';
import type { AiKbEvaluationInfo, AiListKbEvaluationsReq, PageResult } from '../../api/types';
import { aiWorkspaceKeys } from './workspaceQueries';
import type { AiWorkspaceScope } from './workspaceQueries';
import { assertQualityRunInfo, decodeQualityEvaluation, parseQualityDataset } from './knowledgeQuality';
import type { QualityCatalogue, QualityRunView } from './knowledgeQuality';

type CatalogueEntry = Pick<QualityCatalogue, 'key' | 'name' | 'sha256'>;

export function qualityCatalogueOptions(scope: AiWorkspaceScope, kbId: number) {
  return queryOptions<CatalogueEntry[]>({ queryKey: aiWorkspaceKeys.qualityDatasets(scope, kbId), staleTime: 30_000, retry: false,
    queryFn: async ({ signal }) => {
      const result = await fetchAiKbEvaluationDatasets(kbId, signal, { meta_only: true }); signal.throwIfAborted();
      if (!Array.isArray(result.list) || result.list.length > 10 || new Set(result.list.map(item => item.key)).size !== result.list.length) throw new Error('题集列表格式无效');
      return result.list.map(({ key, name, sha256 }) => {
        if (typeof key !== 'string' || !key.trim() || typeof name !== 'string' || !name.trim() || !/^[a-f0-9]{64}$/.test(sha256)) throw new Error('题集目录信息无效');
        return { key, name, sha256 };
      });
    } });
}
export function qualityDatasetOptions(scope: AiWorkspaceScope, kbId: number, entry?: CatalogueEntry) {
  return queryOptions<QualityCatalogue>({ queryKey: [...aiWorkspaceKeys.qualityDatasets(scope, kbId), entry?.key, entry?.sha256], staleTime: Infinity, retry: false,
    queryFn: async ({ signal }) => {
      if (!entry) throw new Error('请选择题集');
      const result = await fetchAiKbEvaluationDatasets(kbId, signal, { dataset_key: entry.key }); signal.throwIfAborted();
      const info = result.list.find(value => value.key === entry.key);
      if (!info || info.sha256 !== entry.sha256) throw new Error('题集版本已变化，请刷新目录');
      const parsed = await parseQualityDataset(info); signal.throwIfAborted(); return parsed;
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
  const queryKey = aiWorkspaceKeys.qualityRun(scope, kbId, id);
  return queryOptions<QualityRunView>({ queryKey, staleTime: 0, retry: false, refetchOnWindowFocus: false,
    refetchInterval: query => !query.state.fetchFailureCount && query.state.data && ['pending', 'running'].includes(query.state.data.run.status) ? 2000 : false,
    refetchIntervalInBackground: false,
    queryFn: async ({ signal, client }) => {
      const previous = client.getQueryData<QualityRunView>(queryKey);
      const prior = previous?.run, state = client.getQueryState(queryKey);
      let next: AiKbEvaluationInfo | undefined;
      if (previous && prior && ['pending', 'running'].includes(prior.status) && !state?.isInvalidated && state?.status !== 'error') {
        const progress = (await fetchAiKbEvaluation(kbId, id, signal, { progress_only: true, known_observations_sha256: prior.observations_sha256 || undefined })).evaluation;
        signal.throwIfAborted();
        assertQualityRunInfo(progress, kbId);
        if (progress.id !== id || progress.request_id !== prior.request_id) throw new Error('批次标识不符');
        if (progress.snapshot) next = progress; // 兼容返回完整记录的旧服务。
        else if (['pending', 'running'].includes(progress.status) && progress.dataset_key === prior.dataset_key && progress.dataset_sha256 === prior.dataset_sha256
          && progress.group_key === prior.group_key && progress.case_count === prior.case_count && /^[a-f0-9]{64}$/.test(progress.observations_sha256 ?? '')) {
          next = { ...progress, snapshot: prior.snapshot, bindings: prior.bindings, context: prior.context, observations: progress.observations || prior.observations };
          if (!progress.observations) {
            if (progress.observations_sha256 === prior.observations_sha256 && prior.completed_count === progress.completed_count && prior.failed_count === progress.failed_count) return { ...previous, run: next };
            next = undefined;
          }
        }
      }
      next ??= (await fetchAiKbEvaluation(kbId, id, signal)).evaluation;
      signal.throwIfAborted();
      if (next.id !== id) throw new Error('批次标识不符');
      const parsed = await decodeQualityEvaluation(next, kbId, previous); signal.throwIfAborted(); return parsed;
    } });
}
