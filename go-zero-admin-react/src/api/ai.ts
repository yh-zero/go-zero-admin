import { apiDelete, apiGet, apiPost, apiPut } from '../utils/request';
import type { AiRestoreWorkflowRevisionReq, AiWorkflowRevisionInfo, AiWorkflowRevisionResp } from './types';
import type { AiKbEvaluationDatasetsResp, AiKbEvaluationInfo, AiKbEvaluationResp, AiListKbEvaluationsReq, AiStartKbEvaluationReq } from './types';
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
export const fetchAiKbEvaluationDatasets = (kbId: number, signal?: AbortSignal) => apiGet<AiKbEvaluationDatasetsResp>(`${base}/kbs/${kbId}/evaluation-datasets`, undefined, { signal });
export const startAiKbEvaluation = (kbId: number, data: AiStartKbEvaluationReq) => apiPost<AiKbEvaluationResp>(`${base}/kbs/${kbId}/evaluations`, data);
export const fetchAiKbEvaluations = (kbId: number, params: AiListKbEvaluationsReq = {}, signal?: AbortSignal) => apiGet<PageResult<AiKbEvaluationInfo>>(`${base}/kbs/${kbId}/evaluations`, params, { signal });
export const fetchAiKbEvaluation = (kbId: number, id: number, signal?: AbortSignal) => apiGet<AiKbEvaluationResp>(`${base}/kbs/${kbId}/evaluations/${id}`, undefined, { signal });
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
