import { apiDelete, apiGet, apiPost, apiPut } from '../utils/request';
import { ApiError } from '../utils/response';
import type { AiRestoreWorkflowRevisionReq, AiWorkflowRevisionInfo, AiWorkflowRevisionResp } from './types';
import type { AiKbChoice, AiKbEvaluationDatasetsResp, AiKbEvaluationInfo, AiKbEvaluationResp, AiListKbEvaluationsReq, AiStartKbEvaluationReq } from './types';
import type { AiKbDocDetailResp, AiUpdateKbDocReq } from './types';
import type { AiKbFeedbackInfo, AiKbFeedbackResp, AiListKbFeedbackReq, AiReviewKbFeedbackReq, AiSetTaskFeedbackReq, AiTaskFeedbackResp } from './types';
import type { AiGetTaskReq, AiKbImportResp, AiKbIngestBatchInfo, AiKbIngestReq, AiListKbDocsReq, AiListKbDocsResp, AiListKbsReq, AiListKbsResp, AiListToolsResp, AiListWorkflowsReq, AiRunDraftReq } from './types';
import type {
  AiApproveTaskReq, AiConversationInfo, AiConversationResp, AiCreateKbDocReq, AiCreateKbReq,
  AiCreateWorkflowReq, AiGetConversationReq, AiGetConversationResp, AiKbDocResp,
  AiKbResp, AiListTasksReq, AiRejectTaskReq, AiRunWorkflowReq, AiRunWorkflowResp, AiSearchKbReq, AiSearchKbResp,
  AiTaskInfo, AiTaskResp, AiUpdateKbReq, AiUpdateWorkflowReq, AiWorkflowInfo,
  AiWorkflowResp, AiListConversationsReq, AiUpdateConversationReq, AiMessageFeedbackReq, PageParams, PageResult,
} from './types';

const base = '/sys/ai';
export const fetchAiKbEvaluationDatasets = (kbId: number, signal?: AbortSignal, params?: { meta_only?: boolean; dataset_key?: string }) => apiGet<AiKbEvaluationDatasetsResp>(`${base}/kbs/${kbId}/evaluation-datasets`, params, { signal });
export const startAiKbEvaluation = (kbId: number, data: AiStartKbEvaluationReq) => apiPost<AiKbEvaluationResp>(`${base}/kbs/${kbId}/evaluations`, data);
export const fetchAiKbEvaluations = (kbId: number, params: AiListKbEvaluationsReq = {}, signal?: AbortSignal) => apiGet<PageResult<AiKbEvaluationInfo>>(`${base}/kbs/${kbId}/evaluations`, params, { signal });
export const fetchAiKbEvaluation = (kbId: number, id: number, signal?: AbortSignal, params?: { progress_only: true; known_observations_sha256?: string }) => apiGet<AiKbEvaluationResp>(`${base}/kbs/${kbId}/evaluations/${id}`, params, { signal });
export const stopAiKbEvaluation = (kbId: number, id: number) => apiPost<AiKbEvaluationResp>(`${base}/kbs/${kbId}/evaluations/${id}/stop`);
export const createAiConversation = (title?: string) => apiPost<AiConversationResp>(`${base}/conversations`, { title });
export const fetchAiConversations = (params: AiListConversationsReq = {}) => apiGet<PageResult<AiConversationInfo>>(`${base}/conversations`, params);
export const fetchAiConversation = (id: number, params: AiGetConversationReq = {}) => apiGet<AiGetConversationResp>(`${base}/conversations/${id}`, params);
export const deleteAiConversation = (id: number) => apiDelete(`${base}/conversations/${id}`);
export const updateAiConversation = (id: number, data: AiUpdateConversationReq) => apiPut<AiConversationResp>(`${base}/conversations/${id}`, data);
export const setAiMessageFeedback = (id: number, messageId: number, data: AiMessageFeedbackReq) => apiPut(`${base}/conversations/${id}/messages/${messageId}/feedback`, data);
export const fetchAiTools = (signal?: AbortSignal) => apiGet<AiListToolsResp>(`${base}/tools`, undefined, { signal });
export const createAiWorkflow = (data: AiCreateWorkflowReq) => apiPost<AiWorkflowResp>(`${base}/workflows`, data);
export const fetchAiWorkflows = (params: AiListWorkflowsReq = {}, signal?: AbortSignal) => apiGet<PageResult<AiWorkflowInfo>>(`${base}/workflows`, params, { signal });
export const fetchAiWorkflow = (id: number, signal?: AbortSignal) => apiGet<AiWorkflowResp>(`${base}/workflows/${id}`, undefined, { signal });
export const updateAiWorkflow = (id: number, data: AiUpdateWorkflowReq) => apiPut<AiWorkflowResp>(`${base}/workflows/${id}`, data);
export const fetchAiWorkflowRevisions = (id: number, params: PageParams = {}, signal?: AbortSignal) => apiGet<PageResult<AiWorkflowRevisionInfo>>(`${base}/workflows/${id}/revisions`, params, { signal });
export const fetchAiWorkflowRevision = (id: number, revision: number, signal?: AbortSignal) => apiGet<AiWorkflowRevisionResp>(`${base}/workflows/${id}/revisions/${revision}`, undefined, { signal });
export const restoreAiWorkflowRevision = (id: number, revision: number, data: AiRestoreWorkflowRevisionReq) => apiPost<AiWorkflowResp>(`${base}/workflows/${id}/revisions/${revision}/restore`, data);
export const deleteAiWorkflow = (id: number) => apiDelete(`${base}/workflows/${id}`);
export const runAiWorkflow = (id: number, data: AiRunWorkflowReq) => apiPost<AiRunWorkflowResp>(`${base}/workflows/${id}/run`, data);
export const runAiDraft = (data: AiRunDraftReq) => apiPost<AiRunWorkflowResp>(`${base}/tasks/run`, data);
export const fetchAiTasks = (params: AiListTasksReq = {}, signal?: AbortSignal) => apiGet<PageResult<AiTaskInfo>>(`${base}/tasks`, params, { signal });
export const fetchAiTask = (id: number, params: AiGetTaskReq = {}, signal?: AbortSignal) => apiGet<AiTaskResp>(`${base}/tasks/${id}`, params, { signal });
export const approveAiTask = (id: number, data: AiApproveTaskReq) => apiPost(`${base}/tasks/${id}/approve`, data);
export const rejectAiTask = (id: number, data: AiRejectTaskReq) => apiPost(`${base}/tasks/${id}/reject`, data);
export const cancelAiTask = (id: number) => apiPost(`${base}/tasks/${id}/cancel`, {});
export const fetchAiTaskFeedback = (id: number, signal?: AbortSignal) => apiGet<AiTaskFeedbackResp>(`${base}/tasks/${id}/feedback`, undefined, { signal });
export const setAiTaskFeedback = (id: number, data: AiSetTaskFeedbackReq) => apiPut<AiKbFeedbackResp>(`${base}/tasks/${id}/feedback`, data);
export const fetchAiKbFeedbackList = (kbId: number, params: AiListKbFeedbackReq = {}, signal?: AbortSignal) => apiGet<PageResult<AiKbFeedbackInfo>>(`${base}/kbs/${kbId}/feedback`, params, { signal });
export const fetchAiKbFeedback = (kbId: number, id: number, signal?: AbortSignal) => apiGet<AiKbFeedbackResp>(`${base}/kbs/${kbId}/feedback/${id}`, undefined, { signal });
export const reviewAiKbFeedback = (kbId: number, id: number, data: AiReviewKbFeedbackReq) => apiPut<AiKbFeedbackResp>(`${base}/kbs/${kbId}/feedback/${id}/review`, data);
export const createAiKb = (data: AiCreateKbReq) => apiPost<AiKbResp>(`${base}/kbs`, data);
export const fetchAiKbs = (params: AiListKbsReq = {}, signal?: AbortSignal) => apiGet<AiListKbsResp>(`${base}/kbs`, params, { signal });
export async function fetchAiKbChoices(params: AiListKbsReq & { id?: number } = {}, signal?: AbortSignal) {
  const filteredId = params.query?.trim() && params.id;
  const pageSize = 100;
  const lookup = filteredId ? { ...params, page: 1, page_size: pageSize } : params;
  const result = await apiGet<{ choices?: AiKbChoice[]; list?: AiKbChoice[]; total: number }>(`${base}/kbs`, { ...lookup, choices_only: true }, { signal });
  if (Array.isArray(result.choices)) return { list: result.choices, total: result.total };
  // 旧服务忽略投影参数时，仅保留选项字段。历史选库可能位于当前列表页之外。
  if (!Array.isArray(result.list)) throw new Error('知识库选项目录不可用');
  const choices = result.list.map(({ id, name }) => ({ id, name }));
  if (!params.id) return { list: choices, total: result.total };
  const selected = choices.find(kb => kb.id === params.id);
  if (selected) return { list: [selected], total: 1 };
  if (filteredId) {
    // 旧服务按 id 降序分页。组合筛选沿用数据库的名称/描述匹配和排序规则。
    let first = 2, last = Math.ceil(result.total / pageSize);
    if (!choices.length || filteredId >= choices[choices.length - 1].id) return { list: [], total: 0 };
    while (first <= last) {
      const page = Math.floor((first + last) / 2);
      const { list } = await fetchAiKbs({ query: params.query, page, page_size: pageSize }, signal);
      const match = list.find(kb => kb.id === filteredId);
      if (match) return { list: [{ id: match.id, name: match.name }], total: 1 };
      if (!list.length || filteredId > list[0].id) last = page - 1;
      else if (filteredId < list[list.length - 1].id) first = page + 1;
      else break;
    }
    return { list: [], total: 0 };
  }
  try {
    const { kb } = await fetchAiKb(params.id, signal);
    if (kb.id !== params.id) throw new Error('知识库标识不符');
    return { list: [{ id: kb.id, name: kb.name }], total: 1 };
  } catch (error) {
    if (error instanceof ApiError && error.code === 404) return { list: [], total: 0 };
    throw error;
  }
}
export const fetchAiKb = (id: number, signal?: AbortSignal) => apiGet<AiKbResp>(`${base}/kbs/${id}`, undefined, { signal });
export const updateAiKb = (id: number, data: AiUpdateKbReq) => apiPut(`${base}/kbs/${id}`, data);
export const deleteAiKb = (id: number) => apiDelete(`${base}/kbs/${id}`);
export const searchAiKb = (id: number, data: AiSearchKbReq) => apiPost<AiSearchKbResp>(`${base}/kbs/${id}/search`, data, { timeout: 15_000 });
export const createAiKbDoc = (kbId: number, data: AiCreateKbDocReq) => apiPost<AiKbDocResp>(`${base}/kbs/${kbId}/docs`, data);
export const fetchAiKbDocs = (kbId: number, params: AiListKbDocsReq = {}, signal?: AbortSignal) => apiGet<AiListKbDocsResp>(`${base}/kbs/${kbId}/docs`, params, { signal });
export const fetchAiKbDoc = (kbId: number, docId: number, metaOnly = false, signal?: AbortSignal) => apiGet<AiKbDocDetailResp>(`${base}/kbs/${kbId}/docs/${docId}`, { meta_only: metaOnly }, { signal });
export const updateAiKbDoc = (kbId: number, docId: number, data: AiUpdateKbDocReq) => apiPut<AiKbDocResp>(`${base}/kbs/${kbId}/docs/${docId}`, data);
export const deleteAiKbDoc = (kbId: number, docId: number) => apiDelete(`${base}/kbs/${kbId}/docs/${docId}`);
export const retryAiKbDoc = (kbId: number, docId: number, data: AiKbIngestReq) => apiPost<AiKbDocResp>(`${base}/kbs/${kbId}/docs/${docId}/retry`, data);
export const fetchAiKbImports = (kbId: number, params: PageParams = {}, signal?: AbortSignal) => apiGet<PageResult<AiKbIngestBatchInfo>>(`${base}/kbs/${kbId}/imports`, params, { signal });
export const fetchAiKbImport = (kbId: number, batchId: string, signal?: AbortSignal) => apiGet<AiKbImportResp>(`${base}/kbs/${kbId}/imports/${encodeURIComponent(batchId)}`, undefined, { signal });
export const stopAiKbImport = (kbId: number, batchId: string) => apiPost<AiKbImportResp>(`${base}/kbs/${kbId}/imports/${encodeURIComponent(batchId)}/stop`, {});
