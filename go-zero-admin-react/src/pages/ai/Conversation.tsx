import { useEffect, useMemo, useRef, useState } from 'react';
import { Alert, Button, Card, Dropdown, Empty, Form, Input, Space, Tag, Typography } from 'antd';
import { DownOutlined, DownloadOutlined, SendOutlined } from '@ant-design/icons';
import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { fetchAiConversation } from '../../api/ai';
import type { AiChatMessage, AiSendMessageResult } from '../../api/types';
import { QueryState } from '../../components/QueryState';
import { aiChatKeys, conversationMessagesOptions, readConversationHead } from './chatQueries';
import type { AiChatScope } from './chatQueries';
import { exportConversation, mergeMessagePages, readEntireConversation, validateMessageContent } from './chatState';
import { ChatKnowledgePicker } from './ChatKnowledgePicker';
import type { ChatKnowledgeSelection, StoredChatKnowledge } from './chatKnowledge';
import { hasLegacyKnowledgeConflict, latestConversationKnowledge } from './chatKnowledge';
import { SafeMarkdown } from './SafeMarkdown';
import { useMessageSend } from './useMessageSend';
import { useChatScroll } from './useChatScroll';
import type { ChatScrollPosition } from './useChatScroll';
import { MessageRecord } from './MessageRecord';
import styles from './chat.module.css';

type ComposerValues = { content: string };
export interface ConversationProps {
  id: number; scope: AiChatScope; canRead: boolean; canSend: boolean; writing: boolean;
  scrollPosition: { current: ChatScrollPosition | null };
  knowledge?: StoredChatKnowledge; knowledgeRevision?: number; onKnowledgeChange?: (value?: ChatKnowledgeSelection) => void; readLatestOnOpen?: boolean;
  isCurrent: () => boolean; beginWrite: (permission: string) => boolean; finishWrite: () => void; onCompleted: (result: AiSendMessageResult) => void;
}

export function Conversation({ id, scope, canRead, canSend, writing, scrollPosition, knowledge: selectedKnowledge, knowledgeRevision, onKnowledgeChange, readLatestOnOpen, isCurrent, beginWrite, finishWrite, onCompleted }: ConversationProps) {
  const client = useQueryClient();
  const [readCachedHead] = useState(() => !!readLatestOnOpen && !!client.getQueryData(aiChatKeys.messages(scope, id)));
  const query = useInfiniteQuery({ ...conversationMessagesOptions(scope, id), enabled: canRead, refetchOnMount: false });
  // 重新打开或返回标签时仅核对当前会话最新页。刚发送后命中终帧缓存，无需立即重读。
  const headKey = aiChatKeys.latest(scope, id);
  const head = useQuery({ queryKey: headKey, initialData: 0, gcTime: 0, staleTime: Infinity, retry: false, enabled: canRead,
    refetchOnMount: readCachedHead ? 'always' : false, refetchOnWindowFocus: () => !writing && query.data ? 'always' : false,
    queryFn: async ({ signal }) => (await readConversationHead(client, scope, id, isCurrent, signal)).message_count });
  const headMessages = query.data?.pages[0]?.messages ?? [];
  const legacyConflict = hasLegacyKnowledgeConflict(headMessages, selectedKnowledge);
  const scopeBlocked = head.isFetching || head.isError || legacyConflict;
  const knowledge = latestConversationKnowledge(headMessages, selectedKnowledge);
  const [form] = Form.useForm<ComposerValues>();
  const draft = Form.useWatch('content', form) ?? '';
  const draftRevision = useRef(0);
  const exportController = useRef<AbortController | null>(null);
  useEffect(() => () => exportController.current?.abort(), []);
  const completeExport = useMutation({ mutationFn: async (format: 'md' | 'json') => {
    const controller = new AbortController();
    exportController.current = controller;
    const snapshot = await readEntireConversation(before_id => fetchAiConversation(id, { before_id, page_size: 20 }), isCurrent, controller.signal);
    if (isCurrent() && !controller.signal.aborted) exportConversation(snapshot.conversation.title, snapshot.messages, format);
  }, retry: false });
  const { start, recheck, feedback, streamState, canUseScope, canSendSelection, documentScope, sending } = useMessageSend({ conversationId: id, scope, knowledge, knowledgeRevision, isCurrent, beginWrite, finishWrite, onCompleted });
  // 历史消息只在分页数据变化时重算，避免流式增量触发全量重建。
  const messages = useMemo(() => mergeMessagePages(query.data?.pages ?? []), [query.data?.pages]);
  const hasRunning = messages.some(row => row.status === 'running');
  const scroll = useChatScroll({ sending, firstMessageId: messages[0]?.id, position: scrollPosition });

  function sendMessage(values: ComposerValues) {
    if (query.isError || !query.data || scopeBlocked || client.getQueryState(headKey)?.fetchStatus === 'fetching') return;
    const submittedRevision = draftRevision.current;
    start(values.content, request => {
      // 等待期间编辑的新草稿不随旧请求成功而清空。
      if (submittedRevision === draftRevision.current && form.getFieldValue('content') === request.content) form.setFieldValue('content', '');
    });
  }

  const conversationTitle = query.data?.pages[0]?.conversation.title || 'AI 对话';
  return <Card className={styles.conversation} title={conversationTitle}
    extra={<Dropdown menu={{ items: [
      { key: 'md', label: '整个会话 · Markdown' },
      { key: 'json', label: '整个会话 · JSON' },
      { key: 'loaded-md', label: '已加载消息 · Markdown' },
      { key: 'loaded-json', label: '已加载消息 · JSON' },
    ], onClick: ({ key }) => { if (completeExport.isPending) return; if (key.startsWith('loaded-')) exportConversation(conversationTitle, messages, key.slice(7) as 'md' | 'json'); else completeExport.mutate(key as 'md' | 'json'); } }}>
      <Button size="small" icon={<DownloadOutlined />} loading={completeExport.isPending}>导出</Button>
    </Dropdown>}>
    {completeExport.isPending && <Alert type="info" title="正在读取整个会话" action={<Button size="small" onClick={() => exportController.current?.abort()}>取消导出</Button>} />}
    {completeExport.isError && <Alert type="warning" title="整个会话未导出" description={completeExport.error instanceof Error ? completeExport.error.message : '请稍后重试'} />}
    {query.hasNextPage && <div className={styles.historyAction}><Button loading={query.isFetchingNextPage} disabled={writing || query.isFetching}
      onClick={() => { if (!query.isFetching) { scroll.prepareForHistory(); void query.fetchNextPage().finally(scroll.finishHistory); } }}>加载更早消息</Button></div>}
    <div className={styles.messagesArea}>
    <div className={styles.messages} ref={scroll.viewportRef} onScroll={scroll.onScroll}
      onWheel={scroll.onWheel} onKeyDown={scroll.onKeyDown} onTouchStart={scroll.onTouchStart} onTouchMove={scroll.onTouchMove}
      role="log" aria-label="聊天消息" aria-live="polite" tabIndex={0}>
      <div className={styles.messageList} ref={scroll.contentRef}>
      <QueryState isPending={query.isPending} error={query.error} onRetry={() => query.refetch()} />
      {!query.isPending && !query.error && messages.length === 0 && <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无消息，发送第一条消息吧" />}
      {messages.map((message: AiChatMessage) => <MessageRecord key={message.id} message={message} askDocumentDisabled={writing || sending || !!draft.trim() || feedback?.state === 'unknown' || hasRunning} feedbackContext={canSend ? { conversationId: id, scope, isCurrent } : undefined} />)}
      {streamState && <article className={styles.message}>
        <div className={styles.messageMeta}>
          <strong>AI 助手</strong>
          <Tag color="processing">回复中</Tag>
        </div>
        {streamState.tools.map(tool => <div className={styles.tool} key={tool.key}>
          <Typography.Text type="secondary">工具 {tool.name} {tool.status === 'start' ? '调用中…' : '已完成'}</Typography.Text>
        </div>)}
        {streamState.content
          ? <SafeMarkdown content={streamState.content} />
          : <Typography.Text type="secondary">正在生成…</Typography.Text>}
      </article>}
      </div>
    </div>
    {scroll.showLatest && <Button className={styles.jumpToLatest} size="small" icon={<DownOutlined />} onClick={scroll.scrollToLatest} aria-label="回到最新消息">回到最新</Button>}
    </div>
    {feedback?.state === 'sending' && <Alert className={styles.feedback} type="info" showIcon
      title={streamState ? 'AI 正在回复，内容实时生成中…' : 'AI 正在回复，请稍候…'}
      description="请求可能需要较长时间，请勿重复提交。" />}
    {feedback?.state === 'unknown' && <Alert className={styles.feedback} type="warning" showIcon title={feedback.error}
      description={<Space direction="vertical">
        <span>本次请求未确认成功，请查看上方消息状态。失败记录表示调用失败；未发现记录时结果仍可能未知。草稿已保留。</span>
        <span>不会自动重新发送。重查复用原请求 ID；若此前未送达，服务端可能首次开始处理。重新发送会创建新的请求，可能再次计费。</span>
        {!canUseScope(feedback.request.kb_id) && <span>恢复原知识库权限后可重查本次结果。</span>}
        {canSend && <Button size="small" disabled={writing || query.isError || !query.data || !canUseScope(feedback.request.kb_id)} onClick={recheck}>重查本次结果</Button>}
      </Space>} />}
    {hasRunning && feedback?.state !== 'sending' && <Alert className={styles.feedback} type="info" showIcon title="会话有处理中或执行中断的消息"
      description="可刷新查看状态。服务端不会自动重放；再次发送会发起新请求，可能产生新的用量。" />}
    {canSend ? <Form<ComposerValues> form={form} className={styles.composer} onFinish={sendMessage} initialValues={{ content: '' }}>
      {head.isError && <Alert type="warning" title="未能核对最新的提问范围" action={<Button size="small" loading={head.isFetching} onClick={() => void head.refetch()}>重试</Button>} />}
      {legacyConflict && <Alert type="warning" title="此标签保存的资料范围与最新消息不一致" description="请选择下一问使用的范围。"
        action={<Space wrap><Button size="small" disabled={writing || head.isFetching || head.isError} onClick={() => onKnowledgeChange?.(knowledge)}>沿用本标签范围</Button>
          <Button size="small" disabled={writing || head.isFetching || head.isError} onClick={() => onKnowledgeChange?.()}>采用最新消息范围</Button></Space>} />}
      {onKnowledgeChange && (query.data || selectedKnowledge) && <ChatKnowledgePicker scope={scope} value={knowledge} onChange={onKnowledgeChange} disabled={writing || !query.data} documentScope={documentScope} />}
      <Form.Item name="content" rules={[{ validator: (_, value: string) => {
        const error = validateMessageContent(value ?? '') || (knowledge.grounded && new TextEncoder().encode(value ?? '').length > 2000 ? '仅依据资料回答时，问题最多 2000 UTF-8 字节' : undefined);
        return error ? Promise.reject(new Error(error)) : Promise.resolve();
      } }]}>
        <Input.TextArea autoSize={{ minRows: 3, maxRows: 8 }} placeholder="输入消息，Enter 发送，Shift + Enter 换行" aria-label="消息内容"
          onChange={() => { draftRevision.current += 1; }}
          onPressEnter={event => {
            if (event.shiftKey || event.nativeEvent.isComposing) return;
            event.preventDefault();
            if (!writing) form.submit();
          }} />
      </Form.Item>
      <div className={styles.composerActions}>
        <Typography.Text type="secondary" className={styles.recordNote}>{new TextEncoder().encode(draft).length} / {knowledge.grounded ? 2000 : 16000} UTF-8 字节</Typography.Text>
        <Button type="primary" htmlType="submit" icon={<SendOutlined />} loading={sending}
          disabled={writing || scopeBlocked || query.isError || !query.data || !draft.trim() || !canSendSelection}>{feedback?.state === 'unknown' || hasRunning ? '重新发送（新请求）' : '发送'}</Button>
      </div>
    </Form> : <Typography.Text type="secondary">当前账号没有发送消息权限，可查看历史消息。</Typography.Text>}
  </Card>;
}
