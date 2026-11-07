import { apiDelete, apiGet, apiPost, apiPut } from '../utils/request';
import type { AiKbDocDetailResp, AiUpdateKbDocReq } from './types';
import type {
  AiApproveTaskReq, AiConversationInfo, AiConversationResp, AiCreateKbDocReq, AiCreateKbReq,
  AiCreateWorkflowReq, AiGetConversationReq, AiGetConversationResp, AiKbDocInfo, AiKbDocResp,
  AiKbInfo, AiKbResp, AiListTasksReq, AiRejectTaskReq, AiRunWorkflowReq, AiRunWorkflowResp, AiSearchKbReq, AiSearchKbResp,
  AiTaskInfo, AiTaskResp, AiToolInfo, AiUpdateKbReq, AiUpdateWorkflowReq, AiWorkflowInfo,
  AiWorkflowResp, AiListConversationsReq, AiUpdateConversationReq, AiMessageFeedbackReq, PageParams, PageResult,
} from './types';

const base = '/sys/ai';
// 同步摄取业务最多 90 秒；等待覆盖 RPC/HTTP 的结束与失败状态回传窗口。
const kbIngestTimeout = 125_000;
export const createAiConversation = (title?: string) => apiPost<AiConversationResp>(`${base}/conversations`, { title });
export const fetchAiConversations = (params: AiListConversationsReq = {}) => apiGet<PageResult<AiConversationInfo>>(`${base}/conversations`, params);
export const fetchAiConversation = (id: number, params: AiGetConversationReq = {}) => apiGet<AiGetConversationResp>(`${base}/conversations/${id}`, params);
export const deleteAiConversation = (id: number) => apiDelete(`${base}/conversations/${id}`);
export const updateAiConversation = (id: number, data: AiUpdateConversationReq) => apiPut<AiConversationResp>(`${base}/conversations/${id}`, data);
export const setAiMessageFeedback = (id: number, messageId: number, data: AiMessageFeedbackReq) => apiPut(`${base}/conversations/${id}/messages/${messageId}/feedback`, data);
export const fetchAiTools = () => apiGet<{ list: AiToolInfo[] }>(`${base}/tools`);
export const createAiWorkflow = (data: AiCreateWorkflowReq) => apiPost<AiWorkflowResp>(`${base}/workflows`, data);
export const fetchAiWorkflows = (params: PageParams = {}) => apiGet<PageResult<AiWorkflowInfo>>(`${base}/workflows`, params);
export const fetchAiWorkflow = (id: number) => apiGet<AiWorkflowResp>(`${base}/workflows/${id}`);
export const updateAiWorkflow = (id: number, data: AiUpdateWorkflowReq) => apiPut(`${base}/workflows/${id}`, data);
export const deleteAiWorkflow = (id: number) => apiDelete(`${base}/workflows/${id}`);
export const runAiWorkflow = (id: number, data: AiRunWorkflowReq) => apiPost<AiRunWorkflowResp>(`${base}/workflows/${id}/run`, data);
export const fetchAiTasks = (params: AiListTasksReq = {}) => apiGet<PageResult<AiTaskInfo>>(`${base}/tasks`, params);
export const fetchAiTask = (id: number) => apiGet<AiTaskResp>(`${base}/tasks/${id}`);
export const approveAiTask = (id: number, data: AiApproveTaskReq) => apiPost(`${base}/tasks/${id}/approve`, data);
export const rejectAiTask = (id: number, data: AiRejectTaskReq) => apiPost(`${base}/tasks/${id}/reject`, data);
export const cancelAiTask = (id: number) => apiPost(`${base}/tasks/${id}/cancel`, {});
export const createAiKb = (data: AiCreateKbReq) => apiPost<AiKbResp>(`${base}/kbs`, data);
export const fetchAiKbs = (params: PageParams = {}) => apiGet<PageResult<AiKbInfo>>(`${base}/kbs`, params);
export const fetchAiKb = (id: number) => apiGet<AiKbResp>(`${base}/kbs/${id}`);
export const updateAiKb = (id: number, data: AiUpdateKbReq) => apiPut(`${base}/kbs/${id}`, data);
export const deleteAiKb = (id: number) => apiDelete(`${base}/kbs/${id}`);
export const searchAiKb = (id: number, data: AiSearchKbReq) => apiPost<AiSearchKbResp>(`${base}/kbs/${id}/search`, data, { timeout: 15_000 });
export const createAiKbDoc = (kbId: number, data: AiCreateKbDocReq) => apiPost<AiKbDocResp>(`${base}/kbs/${kbId}/docs`, data, { timeout: kbIngestTimeout });
export const fetchAiKbDocs = (kbId: number, params: PageParams = {}) => apiGet<PageResult<AiKbDocInfo>>(`${base}/kbs/${kbId}/docs`, params);
export const fetchAiKbDoc = (kbId: number, docId: number, metaOnly = false) => apiGet<AiKbDocDetailResp>(`${base}/kbs/${kbId}/docs/${docId}`, { meta_only: metaOnly });
export const updateAiKbDoc = (kbId: number, docId: number, data: AiUpdateKbDocReq) => apiPut<AiKbDocResp>(`${base}/kbs/${kbId}/docs/${docId}`, data, { timeout: kbIngestTimeout });
export const deleteAiKbDoc = (kbId: number, docId: number) => apiDelete(`${base}/kbs/${kbId}/docs/${docId}`);
export const retryAiKbDoc = (kbId: number, docId: number) => apiPost<AiKbDocResp>(`${base}/kbs/${kbId}/docs/${docId}/retry`, {}, { timeout: kbIngestTimeout });
