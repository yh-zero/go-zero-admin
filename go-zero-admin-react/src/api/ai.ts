import { apiDelete, apiGet, apiPost, apiPut } from '../utils/request';
import type {
  AiApproveTaskReq, AiConversationInfo, AiConversationResp, AiCreateWorkflowReq, AiGetConversationReq,
  AiGetConversationResp, AiListTasksReq, AiRejectTaskReq, AiRunWorkflowResp, AiSendMessageReq,
  AiSendMessageResp, AiTaskInfo, AiTaskResp, AiToolInfo, AiUpdateWorkflowReq, AiWorkflowInfo,
  AiWorkflowResp, PageParams, PageResult,
} from './types';

const base = '/sys/ai';
export const createAiConversation = (title?: string) => apiPost<AiConversationResp>(`${base}/conversations`, { title });
export const fetchAiConversations = (params: PageParams = {}) => apiGet<PageResult<AiConversationInfo>>(`${base}/conversations`, params);
export const fetchAiConversation = (id: number, params: AiGetConversationReq = {}) => apiGet<AiGetConversationResp>(`${base}/conversations/${id}`, params);
export const deleteAiConversation = (id: number) => apiDelete(`${base}/conversations/${id}`);
// 等待120秒AI HTTP/RPC上限及代理返回，保留统一鉴权、刷新和取消处理。
export const sendAiMessage = (data: AiSendMessageReq) => apiPost<AiSendMessageResp>(`${base}/messages`, data, { timeout: 130_000 });
export const fetchAiTools = () => apiGet<{ list: AiToolInfo[] }>(`${base}/tools`);
export const createAiWorkflow = (data: AiCreateWorkflowReq) => apiPost<AiWorkflowResp>(`${base}/workflows`, data);
export const fetchAiWorkflows = (params: PageParams = {}) => apiGet<PageResult<AiWorkflowInfo>>(`${base}/workflows`, params);
export const fetchAiWorkflow = (id: number) => apiGet<AiWorkflowResp>(`${base}/workflows/${id}`);
export const updateAiWorkflow = (id: number, data: AiUpdateWorkflowReq) => apiPut(`${base}/workflows/${id}`, data);
export const deleteAiWorkflow = (id: number) => apiDelete(`${base}/workflows/${id}`);
export const runAiWorkflow = (id: number, input?: string) => apiPost<AiRunWorkflowResp>(`${base}/workflows/${id}/run`, { input });
export const fetchAiTasks = (params: AiListTasksReq = {}) => apiGet<PageResult<AiTaskInfo>>(`${base}/tasks`, params);
export const fetchAiTask = (id: number) => apiGet<AiTaskResp>(`${base}/tasks/${id}`);
export const approveAiTask = (id: number, data: AiApproveTaskReq) => apiPost(`${base}/tasks/${id}/approve`, data);
export const rejectAiTask = (id: number, data: AiRejectTaskReq) => apiPost(`${base}/tasks/${id}/reject`, data);
