import { searchAiKb } from '../../api/ai';
import type { AiSearchKbReq } from '../../api/types';
import { isAiWorkspaceCurrent } from './useAiWorkspaceGuard';
import type { AiWorkspaceScope } from './workspaceQueries';

export const KB_SEARCH_MAX_BYTES = 2000;

export function kbSearchQueryBytes(query: string): number {
  return new TextEncoder().encode(query).byteLength;
}

export function validateKbSearchQuery(query: string): string | undefined {
  if (!query.trim()) return '查询不能为空白';
  if (kbSearchQueryBytes(query) > KB_SEARCH_MAX_BYTES) return '查询最多 2000 字节（UTF-8）';
}

export function kbSearchOptions(scope: AiWorkspaceScope, kbId: number) {
  const assertCurrent = () => {
    if (!isAiWorkspaceCurrent(scope, 'sys:ai:kb:list')) throw new Error('当前工作区已变化，请重新打开召回测试');
  };
  return { retry: false as const, gcTime: 0, mutationFn: async (values: AiSearchKbReq) => {
    assertCurrent();
    const error = validateKbSearchQuery(values.query);
    if (error) throw new Error(error);
    const result = await searchAiKb(kbId, values);
    // 撤权不一定更改会话版本，响应后仍须复核当前查看权限。
    assertCurrent();
    return result;
  } };
}
