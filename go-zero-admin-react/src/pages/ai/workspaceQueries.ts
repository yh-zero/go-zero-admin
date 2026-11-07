import { queryOptions } from '@tanstack/react-query';
import { fetchAiKb, fetchAiKbDocs, fetchAiKbImport, fetchAiKbImports, fetchAiKbs, fetchAiTask, fetchAiTasks, fetchAiTools, fetchAiWorkflow, fetchAiWorkflows } from '../../api/ai';
import type { AiListKbDocsReq, AiListKbDocsResp, AiListKbsReq, AiListTasksReq, AiListWorkflowsReq, AiTaskInfo, AiTaskResp, PageParams } from '../../api/types';

export type AiWorkspaceScope = readonly [sessionId: string, sessionVersion: number, departmentVersion: number];

export const aiWorkspaceKeys = {
  tools: (scope: AiWorkspaceScope) => ['ai-workspace', ...scope, 'tools'] as const,
  workflows: (scope: AiWorkspaceScope) => ['ai-workspace', ...scope, 'workflows'] as const,
  workflow: (scope: AiWorkspaceScope, id: number) => ['ai-workspace', ...scope, 'workflow', id] as const,
  tasks: (scope: AiWorkspaceScope) => ['ai-workspace', ...scope, 'tasks'] as const,
  task: (scope: AiWorkspaceScope, id: number) => ['ai-workspace', ...scope, 'task', id] as const,
  kbs: (scope: AiWorkspaceScope) => ['ai-workspace', ...scope, 'kbs'] as const,
  kbDocs: (scope: AiWorkspaceScope, kbId: number) => ['ai-workspace', ...scope, 'kb-docs', kbId] as const,
  kb: (scope: AiWorkspaceScope, kbId: number) => ['ai-workspace', ...scope, 'kb', kbId] as const,
  imports: (scope: AiWorkspaceScope, kbId: number) => ['ai-workspace', ...scope, 'kb-imports', kbId] as const,
  import: (scope: AiWorkspaceScope, kbId: number, batchId: string) => ['ai-workspace', ...scope, 'kb-import', kbId, batchId] as const,
};

// 身份取消由统一请求层处理；Query 取消另行阻止已关闭详情的迟到响应回填。
async function readActive<T>(signal: AbortSignal, read: () => Promise<T>): Promise<T> {
  signal.throwIfAborted();
  const data = await read();
  signal.throwIfAborted();
  return data;
}

export function toolListOptions(scope: AiWorkspaceScope) {
  return queryOptions({ queryKey: aiWorkspaceKeys.tools(scope), staleTime: 30_000, retry: false,
    queryFn: ({ signal }) => readActive(signal, () => fetchAiTools(signal)) });
}

export function workflowListOptions(scope: AiWorkspaceScope, params: AiListWorkflowsReq) {
  return queryOptions({ queryKey: [...aiWorkspaceKeys.workflows(scope), params], staleTime: 30_000, retry: false,
    queryFn: ({ signal }) => readActive(signal, () => fetchAiWorkflows(params, signal)) });
}

export function workflowDetailOptions(scope: AiWorkspaceScope, id: number) {
  return queryOptions({ queryKey: aiWorkspaceKeys.workflow(scope, id), staleTime: 0, retry: false, refetchOnWindowFocus: false,
    queryFn: ({ signal }) => readActive(signal, () => fetchAiWorkflow(id, signal)) });
}

export function taskListOptions(scope: AiWorkspaceScope, params: AiListTasksReq) {
  return queryOptions({ queryKey: [...aiWorkspaceKeys.tasks(scope), params], staleTime: 0, retry: false,
    queryFn: ({ signal }) => readActive(signal, () => fetchAiTasks(params, signal)) });
}

export function taskDetailOptions(scope: AiWorkspaceScope, id: number) {
  return queryOptions({ queryKey: aiWorkspaceKeys.task(scope, id), staleTime: 0, retry: false, refetchOnWindowFocus: false,
    queryFn: ({ signal }) => readActive(signal, () => fetchAiTask(id, {}, signal)) });
}

export function taskMetadataOptions(scope: AiWorkspaceScope, id: number, task?: AiTaskInfo, parentMetadata?: AiTaskInfo) {
  return queryOptions<AiTaskResp>({ queryKey: [...aiWorkspaceKeys.task(scope, id), 'meta'], staleTime: 0, retry: false, refetchOnWindowFocus: false,
    enabled: !parentMetadata && (task?.status === 'pending' || task?.status === 'running'),
    queryFn: ({ signal }) => fetchAiTask(id, { meta_only: true }, signal) });
}

export function kbListOptions(scope: AiWorkspaceScope, params: AiListKbsReq) {
  return queryOptions({ queryKey: [...aiWorkspaceKeys.kbs(scope), params], staleTime: 30_000, retry: false,
    queryFn: ({ signal }) => readActive(signal, () => fetchAiKbs(params, signal)) });
}

export function kbDocListOptions(scope: AiWorkspaceScope, kbId: number, params: AiListKbDocsReq) {
  return queryOptions<AiListKbDocsResp>({ queryKey: [...aiWorkspaceKeys.kbDocs(scope, kbId), params], staleTime: 0, retry: false,
    refetchInterval: query => query.state.data?.list.some(doc => doc.status === 'pending') ? 3000 : false,
    refetchIntervalInBackground: false,
    queryFn: ({ signal }) => readActive(signal, () => fetchAiKbDocs(kbId, params, signal)) });
}

export function kbDetailOptions(scope: AiWorkspaceScope, kbId: number) {
  return queryOptions({ queryKey: aiWorkspaceKeys.kb(scope, kbId), staleTime: 0, retry: false,
    queryFn: ({ signal }) => fetchAiKb(kbId, signal) });
}

export function kbImportListOptions(scope: AiWorkspaceScope, kbId: number, params: PageParams) {
  return queryOptions({ queryKey: [...aiWorkspaceKeys.imports(scope, kbId), params], staleTime: 0, retry: false,
    queryFn: ({ signal }) => fetchAiKbImports(kbId, params, signal) });
}

export function kbImportOptions(scope: AiWorkspaceScope, kbId: number, batchId: string) {
  return queryOptions({ queryKey: aiWorkspaceKeys.import(scope, kbId, batchId), staleTime: 0, retry: false,
    queryFn: ({ signal }) => fetchAiKbImport(kbId, batchId, signal) });
}
