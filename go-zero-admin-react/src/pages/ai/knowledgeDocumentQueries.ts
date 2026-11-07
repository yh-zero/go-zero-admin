import { queryOptions } from '@tanstack/react-query';
import { fetchAiKbDoc } from '../../api/ai';
import type { AiKbDocDetailResp } from '../../api/types';
import type { AiWorkspaceScope } from './workspaceQueries';

export const knowledgeDocumentKeys = {
  all: (scope: AiWorkspaceScope, kbId: number) => ['ai-workspace', ...scope, 'kb-document', kbId] as const,
  detail: (scope: AiWorkspaceScope, kbId: number, docId: number, metaOnly: boolean) => [...knowledgeDocumentKeys.all(scope, kbId), docId, metaOnly] as const,
};

export function knowledgeDocumentOptions(scope: AiWorkspaceScope, kbId: number, docId: number, metaOnly = false) {
  return queryOptions<AiKbDocDetailResp>({ queryKey: knowledgeDocumentKeys.detail(scope, kbId, docId, metaOnly), staleTime: 0, retry: false,
    refetchOnWindowFocus: false,
    refetchInterval: query => metaOnly && query.state.data?.doc.status === 'pending' ? 3000 : false,
    refetchIntervalInBackground: false,
    queryFn: async ({ signal }) => {
      signal.throwIfAborted();
      const result = await fetchAiKbDoc(kbId, docId, metaOnly);
      signal.throwIfAborted();
      return result;
    },
  });
}
