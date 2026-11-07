import { apiDelete, apiGet, apiPost, apiPut } from '../utils/request';
import type { AiKbDocDetailResp, AiUpdateKbDocReq } from './types';
import type { AiGetTaskReq, AiKbImportResp, AiKbIngestBatchInfo, AiKbIngestReq, AiListKbDocsReq, AiListKbDocsResp, AiListKbsReq, AiListKbsResp, AiListToolsResp, AiListWorkflowsReq, AiRunDraftReq } from './types';
import type {
  AiApproveTaskReq, AiConversationInfo, AiConversationResp, AiCreateKbDocReq, AiCreateKbReq,
  AiCreateWorkflowReq, AiGetConversationReq, AiGetConversationResp, AiKbDocResp,
  AiKbResp, AiListTasksReq, AiRejectTaskReq, AiRunWorkflowReq, AiRunWorkflowResp, AiSearchKbReq, AiSearchKbResp,
  AiTaskInfo, AiTaskResp, AiUpdateKbReq, AiUpdateWorkflowReq, AiWorkflowInfo,
  AiWorkflowResp, AiListConversationsReq, AiUpdateConversationReq, AiMessageFeedbackReq, PageParams, PageResult,
} from './types';

const base = '/sys/ai';
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
export const updateAiWorkflow = (id: number, data: AiUpdateWorkflowReq) => apiPut(`${base}/workflows/${id}`, data);
export const deleteAiWorkflow = (id: number) => apiDelete(`${base}/workflows/${id}`);
export const runAiWorkflow = (id: number, data: AiRunWorkflowReq) => apiPost<AiRunWorkflowResp>(`${base}/workflows/${id}/run`, data);
export const runAiDraft = (data: AiRunDraftReq) => apiPost<AiRunWorkflowResp>(`${base}/tasks/run`, data);
export const fetchAiTasks = (params: AiListTasksReq = {}, signal?: AbortSignal) => apiGet<PageResult<AiTaskInfo>>(`${base}/tasks`, params, { signal });
export const fetchAiTask = (id: number, params: AiGetTaskReq = {}, signal?: AbortSignal) => apiGet<AiTaskResp>(`${base}/tasks/${id}`, params, { signal });
export const approveAiTask = (id: number, data: AiApproveTaskReq) => apiPost(`${base}/tasks/${id}/approve`, data);
export const rejectAiTask = (id: number, data: AiRejectTaskReq) => apiPost(`${base}/tasks/${id}/reject`, data);
export const cancelAiTask = (id: number) => apiPost(`${base}/tasks/${id}/cancel`, {});
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
