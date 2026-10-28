import { requestClient } from '#/api/request';

export interface AgentTool {
  name: string;
  label: string;
  description: string;
  available: boolean;
}
export interface AgentInfo {
  enabled: boolean;
  configured: boolean;
  provider: string;
  model: string;
  tools: AgentTool[];
  maxInputChars: number;
  maxSteps: number;
  maxRunSeconds: number;
}
export type AgentRunStatus =
  | 'cancelled'
  | 'failed'
  | 'interrupted'
  | 'queued'
  | 'running'
  | 'succeeded';
export interface AgentToolCall {
  name: string;
  status: string;
  error: string;
  summary: string;
}
export interface AgentRun {
  id: string;
  conversationId: string;
  requestId: string;
  status: AgentRunStatus;
  answer: string;
  error: string;
  provider: string;
  model: string;
  inputTokens: number;
  outputTokens: number;
  createdAt: string;
  updatedAt: string;
  toolCalls: AgentToolCall[];
}
export interface AgentConversation {
  id: string;
  title: string;
  createdAt: string;
  updatedAt: string;
}
export interface AgentMessage {
  id: string;
  conversationId: string;
  runId: string;
  role: 'assistant' | 'user';
  content: string;
  createdAt: string;
}
export interface AgentPageQuery {
  pageNo: number;
  pageSize: number;
}
export interface AgentPageResult<T> {
  items: T[];
  total: number;
}
export interface CreateAgentRun {
  conversationId?: string;
  requestId: string;
  message: string;
}
const base = '/v1/ai';
const defaultPage = { pageNo: 1, pageSize: 20 };

export const getAgentInfo = () => requestClient.get<AgentInfo>(`${base}/info`);
export const createAgentRun = (data: CreateAgentRun) =>
  requestClient.post<AgentRun>(`${base}/runs`, data);
export const getAgentRun = (id: string) =>
  requestClient.get<AgentRun>(`${base}/runs/${encodeURIComponent(id)}`);
// A successful stop request is followed by GET Run; the displayed state comes
// from the authoritative read rather than assuming immediate cancellation.
export const cancelAgentRun = (id: string) =>
  requestClient.post<AgentRun>(`${base}/runs/${encodeURIComponent(id)}/cancel`);
export async function getAgentConversations(
  params: AgentPageQuery = defaultPage,
) {
  const result = await requestClient.get<AgentPageResult<AgentConversation>>(
    `${base}/conversations`,
    { params },
  );
  return { ...result, items: result.items ?? [] };
}
export async function getAgentMessages(
  id: string,
  params: AgentPageQuery = defaultPage,
) {
  const result = await requestClient.get<AgentPageResult<AgentMessage>>(
    `${base}/conversations/${encodeURIComponent(id)}/messages`,
    { params },
  );
  return { ...result, items: result.items ?? [] };
}
