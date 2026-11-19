import { useEffect, useRef, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { sendAiMessageStream } from '../../api/aiStream';
import type { AiSendMessageReq, AiSendMessageResult, AiStreamMeta } from '../../api/types';
import { applyAiSendResult, recoverAiSend } from './chatQueries';
import type { AiChatScope } from './chatQueries';
import { createMessageSender } from './chatState';
import { useAuthStore } from '../../stores/auth';
import { ApiError } from '../../utils/response';
import type { MessageAttempt } from './chatState';
import type { ChatKnowledgeSelection } from './chatKnowledge';
import { confirmChatKnowledge } from './chatKnowledge';
import { knowledgeDocumentKeys, knowledgeDocumentOptions } from './knowledgeDocumentQueries';

type Feedback = { request: AiSendMessageReq; state: 'sending' | 'unknown' | 'succeeded'; error?: string; meta?: AiStreamMeta };
interface Options {
  conversationId: number; scope: AiChatScope; isCurrent: () => boolean;
  knowledge: ChatKnowledgeSelection; knowledgeRevision?: number;
  beginWrite: (permission: string) => boolean; finishWrite: () => void;
  onCompleted?: (result: AiSendMessageResult, request: AiSendMessageReq) => void;
  onConversationAssigned?: (id: number) => void;
}

// 两个作曲家只管理自己的表单；请求身份、草稿帧、取消与缓存结算在同一生命周期内。
export function useMessageSend({ conversationId, scope, knowledge, knowledgeRevision = 0, isCurrent, beginWrite, finishWrite, onCompleted, onConversationAssigned }: Options) {
  const client = useQueryClient();
  const canReadKnowledge = useAuthStore(state => state.permissionsReady && state.permCodes.includes('sys:ai:kb:list') && state.permCodes.includes('sys:ai:message:send'));
  const canUseScope = (kbId?: number) => !kbId || canReadKnowledge;
  const scopedDoc = useQuery({ ...knowledgeDocumentOptions(scope, knowledge.kb_id, knowledge.doc_id ?? 0, true),
    enabled: !!knowledge.doc_id && canReadKnowledge && isCurrent() });
  const doc = scopedDoc.data?.doc;
  const matches = doc?.id === knowledge.doc_id && doc?.kb_id === knowledge.kb_id;
  const canSendSelection = canUseScope(knowledge.kb_id) && (!knowledge.doc_id || (matches && doc?.status === 'ready' && scopedDoc.isSuccess && !scopedDoc.isFetching));
  const documentScope = knowledge.doc_id ? {
    title: canReadKnowledge && matches && scopedDoc.isSuccess ? doc?.title : undefined,
    loading: scopedDoc.isFetching,
    problem: scopedDoc.isError || (scopedDoc.isSuccess && !matches) ? '无法读取这篇资料，请核对权限或重试。'
      : scopedDoc.isSuccess && doc?.status !== 'ready' ? '这篇资料尚未就绪，暂时无法提问。' : undefined,
    retry: () => { if (canReadKnowledge && isCurrent()) void scopedDoc.refetch(); },
  } : undefined;
  const [sender] = useState(() => createMessageSender(sendAiMessageStream));
  const submittedRevision = useRef(0);
  const mounted = useRef(true);
  const [feedback, setFeedback] = useState<Feedback | null>(null);
  const [streamState, setStreamState] = useState<{ content: string; tools: { key: number; name: string; status: string }[] } | null>(null);
  const mutation = useMutation({ mutationFn: (attempt: MessageAttempt) => attempt.promise, retry: false });
  useEffect(() => {
    mounted.current = true;
    return () => { mounted.current = false; sender.abort(); };
  }, [sender]);
  const current = () => mounted.current && isCurrent();
  const handlers = {
    onMeta: (meta: AiStreamMeta) => {
      if (!current()) return;
      onConversationAssigned?.(meta.conversation_id);
      setFeedback(state => state ? { ...state, meta } : state);
    },
    onDelta: (content: string) => { if (current()) setStreamState(state => state ? { ...state, content: state.content + content } : state); },
    onTool: (name: string, status: 'start' | 'done') => { if (current()) setStreamState(state => state ? { ...state, tools: [...state.tools, { key: state.tools.length, name, status }] } : state); },
  };
  async function settle(attempt: MessageAttempt, onSuccess?: (request: AiSendMessageReq) => void, preserveNewDraft = false) {
    setFeedback({ request: attempt.request, state: 'sending', meta: attempt.meta });
    setStreamState({ content: '', tools: [] });
    try {
      const result = await mutation.mutateAsync(attempt);
      if (!current()) return;
      await applyAiSendResult(client, scope, result, { request: attempt.request, startedAt: attempt.startedAt, isCurrent: current });
      if (!current()) return;
      if (result.conversation) confirmChatKnowledge(scope[0], useAuthStore.getState().selectedDeptId, result.conversation_id,
        attempt.request, Math.max(result.user_message.id, result.ai_message.id), preserveNewDraft);
      setFeedback({ request: attempt.request, state: 'succeeded', meta: attempt.meta });
      onSuccess?.(attempt.request);
      onCompleted?.(result, attempt.request);
    } catch (error) {
      if (!current() || attempt.controller.signal.aborted || (error instanceof DOMException && error.name === 'AbortError')) return;
      setFeedback({ request: attempt.request, state: 'unknown', meta: attempt.meta,
        error: error instanceof Error ? error.message : '请求未完成，请重查本次结果' });
      setStreamState(null);
      if (attempt.request.kb_id && attempt.request.doc_id && error instanceof ApiError && [403, 404, 409].includes(error.code)) {
        void client.invalidateQueries({ queryKey: knowledgeDocumentKeys.detail(scope, attempt.request.kb_id, attempt.request.doc_id, true), exact: true });
      }
      await recoverAiSend(client, scope, attempt.meta?.conversation_id ?? attempt.request.conversation_id, current);
    } finally {
      if (current()) setStreamState(null);
      finishWrite();
    }
  }
  return {
    feedback, streamState, canUseScope, canSendSelection, documentScope, sending: mutation.isPending || streamState !== null,
    start(content: string, onSuccess?: (request: AiSendMessageReq) => void) {
      if (!canSendSelection || !beginWrite('sys:ai:message:send')) return;
      const attempt = sender.start(conversationId, content, handlers, knowledge);
      if (!attempt) { finishWrite(); return; }
      submittedRevision.current = knowledgeRevision;
      void settle(attempt, onSuccess);
    },
    recheck() {
      if (!canUseScope(feedback?.request.kb_id) || !beginWrite('sys:ai:message:send')) return;
      const attempt = sender.recheck(handlers);
      if (!attempt) { finishWrite(); return; }
      // 用户改回相同的 kb/doc/grounded 也是新的选择，重查仍只确认发送时那一次选择。
      void settle(attempt, undefined, submittedRevision.current !== knowledgeRevision);
    },
  };
}
