import { queryOptions } from '@tanstack/react-query';
import { fetchAiTask, fetchAiTasks, fetchAiTools, fetchAiWorkflow, fetchAiWorkflows } from '../../api/ai';
import type { AiListTasksReq, PageParams } from '../../api/types';

export type AiWorkspaceScope = readonly [sessionId: string, sessionVersion: number, departmentVersion: number];

export const aiWorkspaceKeys = {
  tools: (scope: AiWorkspaceScope) => ['ai-workspace', ...scope, 'tools'] as const,
  workflows: (scope: AiWorkspaceScope) => ['ai-workspace', ...scope, 'workflows'] as const,
  workflow: (scope: AiWorkspaceScope, id: number) => ['ai-workspace', ...scope, 'workflow', id] as const,
  tasks: (scope: AiWorkspaceScope) => ['ai-workspace', ...scope, 'tasks'] as const,
  task: (scope: AiWorkspaceScope, id: number) => ['ai-workspace', ...scope, 'task', id] as const,
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
    queryFn: ({ signal }) => readActive(signal, fetchAiTools) });
}

export function workflowListOptions(scope: AiWorkspaceScope, params: PageParams) {
  return queryOptions({ queryKey: [...aiWorkspaceKeys.workflows(scope), params], staleTime: 30_000, retry: false,
    queryFn: ({ signal }) => readActive(signal, () => fetchAiWorkflows(params)) });
}

export function workflowDetailOptions(scope: AiWorkspaceScope, id: number) {
  return queryOptions({ queryKey: aiWorkspaceKeys.workflow(scope, id), staleTime: 0, retry: false, refetchOnWindowFocus: false,
    queryFn: ({ signal }) => readActive(signal, () => fetchAiWorkflow(id)) });
}

export function taskListOptions(scope: AiWorkspaceScope, params: AiListTasksReq) {
  return queryOptions({ queryKey: [...aiWorkspaceKeys.tasks(scope), params], staleTime: 0, retry: false,
    queryFn: ({ signal }) => readActive(signal, () => fetchAiTasks(params)) });
}

export function taskDetailOptions(scope: AiWorkspaceScope, id: number) {
  return queryOptions({ queryKey: aiWorkspaceKeys.task(scope, id), staleTime: 0, retry: false, refetchOnWindowFocus: false,
    queryFn: ({ signal }) => readActive(signal, () => fetchAiTask(id)) });
}
