import { useEffect, useRef, useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { sendAiMessageStream } from '../../api/aiStream';
import type { AiSendMessageReq, AiSendMessageResult, AiStreamMeta } from '../../api/types';
import { applyAiSendResult, recoverAiSend } from './chatQueries';
import type { AiChatScope } from './chatQueries';
import { createMessageSender } from './chatState';
import type { MessageAttempt } from './chatState';

type Feedback = { request: AiSendMessageReq; state: 'sending' | 'unknown' | 'succeeded'; error?: string; meta?: AiStreamMeta };
interface Options {
  conversationId: number; scope: AiChatScope; isCurrent: () => boolean;
  beginWrite: (permission: string) => boolean; finishWrite: () => void;
  onCompleted?: (result: AiSendMessageResult, request: AiSendMessageReq) => void;
}

// 两个作曲家只管理自己的表单；请求身份、草稿帧、取消与缓存结算在同一生命周期内。
export function useMessageSend({ conversationId, scope, isCurrent, beginWrite, finishWrite, onCompleted }: Options) {
  const client = useQueryClient();
  const [sender] = useState(() => createMessageSender(sendAiMessageStream));
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
    onMeta: (meta: AiStreamMeta) => { if (current()) setFeedback(state => state ? { ...state, meta } : state); },
    onDelta: (content: string) => { if (current()) setStreamState(state => state ? { ...state, content: state.content + content } : state); },
    onTool: (name: string, status: 'start' | 'done') => { if (current()) setStreamState(state => state ? { ...state, tools: [...state.tools, { key: state.tools.length, name, status }] } : state); },
  };
  async function settle(attempt: MessageAttempt, onSuccess?: (request: AiSendMessageReq) => void) {
    setFeedback({ request: attempt.request, state: 'sending', meta: attempt.meta });
    setStreamState({ content: '', tools: [] });
    try {
      const result = await mutation.mutateAsync(attempt);
      if (!current()) return;
      await applyAiSendResult(client, scope, result, { request: attempt.request, startedAt: attempt.startedAt, isCurrent: current });
      if (!current()) return;
      setFeedback({ request: attempt.request, state: 'succeeded', meta: attempt.meta });
      onSuccess?.(attempt.request);
      onCompleted?.(result, attempt.request);
    } catch (error) {
      if (!current() || attempt.controller.signal.aborted || (error instanceof DOMException && error.name === 'AbortError')) return;
      setFeedback({ request: attempt.request, state: 'unknown', meta: attempt.meta,
        error: error instanceof Error ? error.message : '请求未完成，请重查本次结果' });
      setStreamState(null);
      await recoverAiSend(client, scope, attempt.meta?.conversation_id ?? attempt.request.conversation_id, current);
    } finally {
      if (current()) setStreamState(null);
      finishWrite();
    }
  }
  return {
    feedback, streamState, sending: mutation.isPending || streamState !== null,
    start(content: string, onSuccess?: (request: AiSendMessageReq) => void) {
      if (!beginWrite('sys:ai:message:send')) return;
      const attempt = sender.start(conversationId, content, handlers);
      if (!attempt) { finishWrite(); return; }
      void settle(attempt, onSuccess);
    },
    recheck() {
      if (!beginWrite('sys:ai:message:send')) return;
      const attempt = sender.recheck(handlers);
      if (!attempt) { finishWrite(); return; }
      void settle(attempt);
    },
  };
}
