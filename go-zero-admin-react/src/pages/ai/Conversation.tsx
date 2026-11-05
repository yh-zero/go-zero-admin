import { useMemo, useRef } from 'react';
import { Alert, Button, Card, Dropdown, Empty, Form, Input, Space, Tag, Typography } from 'antd';
import { DownOutlined, DownloadOutlined, SendOutlined } from '@ant-design/icons';
import { useInfiniteQuery } from '@tanstack/react-query';
import type { AiChatMessage, AiSendMessageResult } from '../../api/types';
import { QueryState } from '../../components/QueryState';
import { conversationMessagesOptions } from './chatQueries';
import type { AiChatScope } from './chatQueries';
import { exportConversation, mergeMessagePages, validateMessageContent } from './chatState';
import { useMessageSend } from './useMessageSend';
import { useChatScroll } from './useChatScroll';
import type { ChatScrollPosition } from './useChatScroll';
import { MessageRecord } from './MessageRecord';
import styles from './chat.module.css';

type ComposerValues = { content: string };
export interface ConversationProps {
  id: number; scope: AiChatScope; canRead: boolean; canSend: boolean; writing: boolean;
  scrollPosition: { current: ChatScrollPosition | null };
  isCurrent: () => boolean; beginWrite: (permission: string) => boolean; finishWrite: () => void; onCompleted: (result: AiSendMessageResult) => void;
}

export function Conversation({ id, scope, canRead, canSend, writing, scrollPosition, isCurrent, beginWrite, finishWrite, onCompleted }: ConversationProps) {
  const query = useInfiniteQuery({ ...conversationMessagesOptions(scope, id), enabled: canRead });
  const [form] = Form.useForm<ComposerValues>();
  const draft = Form.useWatch('content', form) ?? '';
  const draftRevision = useRef(0);
  const { start, recheck, feedback, streamState, sending } = useMessageSend({ conversationId: id, scope, isCurrent, beginWrite, finishWrite, onCompleted });
  // 历史消息只在分页数据变化时重算，避免流式增量触发全量重建。
  const messages = useMemo(() => mergeMessagePages(query.data?.pages ?? []), [query.data?.pages]);
  const hasRunning = messages.some(row => row.status === 'running');
  const scroll = useChatScroll({ sending, firstMessageId: messages[0]?.id, position: scrollPosition });

  function sendMessage(values: ComposerValues) {
    if (query.isError || !query.data) return;
    const submittedRevision = draftRevision.current;
    start(values.content, request => {
      // 等待期间编辑的新草稿不随旧请求成功而清空。
      if (submittedRevision === draftRevision.current && form.getFieldValue('content') === request.content) form.setFieldValue('content', '');
    });
  }

  const conversationTitle = query.data?.pages[0]?.conversation.title || 'AI 对话';
  return <Card className={styles.conversation} title={conversationTitle}
    extra={<Dropdown menu={{ items: [
      { key: 'md', label: '导出 Markdown' },
      { key: 'json', label: '导出 JSON' },
    ], onClick: ({ key }) => exportConversation(conversationTitle, messages, key as 'md' | 'json') }}>
      <Button size="small" icon={<DownloadOutlined />}>导出</Button>
    </Dropdown>}>
    {query.hasNextPage && <div className={styles.historyAction}><Button loading={query.isFetchingNextPage} disabled={writing || query.isFetching}
      onClick={() => { if (!query.isFetching) { scroll.prepareForHistory(); void query.fetchNextPage().finally(scroll.finishHistory); } }}>加载更早消息</Button></div>}
    <div className={styles.messagesArea}>
    <div className={styles.messages} ref={scroll.viewportRef} onScroll={scroll.onScroll}
      onWheel={scroll.onWheel} onKeyDown={scroll.onKeyDown} onTouchStart={scroll.onTouchStart} onTouchMove={scroll.onTouchMove}
      role="log" aria-label="聊天消息" aria-live="polite" tabIndex={0}>
      <div className={styles.messageList} ref={scroll.contentRef}>
      <QueryState isPending={query.isPending} error={query.error} onRetry={() => query.refetch()} />
      {!query.isPending && !query.error && messages.length === 0 && <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无消息，发送第一条消息吧" />}
      {messages.map((message: AiChatMessage) => <MessageRecord key={message.id} message={message} />)}
      {streamState && <article className={styles.message}>
        <div className={styles.messageMeta}>
          <strong>AI 助手</strong>
          <Tag color="processing">回复中</Tag>
        </div>
        {streamState.tools.map(tool => <div className={styles.tool} key={tool.key}>
          <Typography.Text type="secondary">工具 {tool.name} {tool.status === 'start' ? '调用中…' : '已完成'}</Typography.Text>
        </div>)}
        {streamState.content
          ? <div className={styles.messageText}>{streamState.content}</div>
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
        {canSend && <Button size="small" disabled={writing || query.isError || !query.data} onClick={recheck}>重查本次结果</Button>}
      </Space>} />}
    {hasRunning && feedback?.state !== 'sending' && <Alert className={styles.feedback} type="info" showIcon title="会话有处理中或执行中断的消息"
      description="可刷新查看状态。服务端不会自动重放；再次发送会发起新请求，可能产生新的用量。" />}
    {canSend ? <Form<ComposerValues> form={form} className={styles.composer} onFinish={sendMessage} initialValues={{ content: '' }}>
      <Form.Item name="content" rules={[{ validator: (_, value: string) => {
        const error = validateMessageContent(value ?? '');
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
        <Typography.Text type="secondary" className={styles.recordNote}>{new TextEncoder().encode(draft).length} / 16000 UTF-8 字节</Typography.Text>
        <Button type="primary" htmlType="submit" icon={<SendOutlined />} loading={sending}
          disabled={writing || query.isError || !query.data || !draft.trim()}>{feedback?.state === 'unknown' || hasRunning ? '重新发送（新请求）' : '发送'}</Button>
      </div>
    </Form> : <Typography.Text type="secondary">当前账号没有发送消息权限，可查看历史消息。</Typography.Text>}
  </Card>;
}
