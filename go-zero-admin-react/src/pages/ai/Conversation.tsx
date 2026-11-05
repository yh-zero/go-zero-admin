import { useEffect, useMemo, useRef, useState } from 'react';
import { Alert, Button, Card, Dropdown, Empty, Form, Input, Space, Tag, Typography } from 'antd';
import { DownloadOutlined, SendOutlined } from '@ant-design/icons';
import { useInfiniteQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { sendAiMessage } from '../../api/ai';
import { sendAiMessageStream } from '../../api/aiStream';
import type { AiChatMessage, AiSendMessageReq } from '../../api/types';
import { QueryState } from '../../components/QueryState';
import { aiChatKeys, conversationMessagesOptions } from './chatQueries';
import type { AiChatScope } from './chatQueries';
import { createMessageSender, exportConversation, mergeMessagePages, validateMessageContent } from './chatState';
import type { MessageAttempt } from './chatState';
import { MessageRecord } from './MessageRecord';
import styles from './chat.module.css';

type ComposerValues = { content: string };
type SendFeedback = { request: AiSendMessageReq; state: 'sending' | 'unknown' | 'succeeded'; error?: string };

function errorText(error: unknown) { return error instanceof Error ? error.message : '请求未完成，请稍后查看会话记录'; }

export interface ConversationProps {
  id: number; scope: AiChatScope; canRead: boolean; canSend: boolean; writing: boolean;
  isCurrent: () => boolean; beginWrite: (permission: string) => boolean; finishWrite: () => void; refreshLists: () => Promise<void>;
}

export function Conversation({ id, scope, canRead, canSend, writing, isCurrent, beginWrite, finishWrite, refreshLists }: ConversationProps) {
  const queryClient = useQueryClient();
  const query = useInfiniteQuery({ ...conversationMessagesOptions(scope, id), enabled: canRead });
  const [form] = Form.useForm<ComposerValues>();
  const draft = Form.useWatch('content', form) ?? '';
  const draftRevision = useRef(0);
  const [sender] = useState(() => createMessageSender(sendAiMessage));
  const [feedback, setFeedback] = useState<SendFeedback | null>(null);
  const send = useMutation({ mutationFn: (attempt: MessageAttempt) => attempt.promise, retry: false });
  // 流式回复草稿：仅本组件本地状态，不写查询缓存；done 后以服务端持久化数据为准。
  const [streamState, setStreamState] = useState<{ content: string; tools: { key: number; name: string; status: string }[] } | null>(null);
  const abortRef = useRef<AbortController | null>(null);
  useEffect(() => () => { abortRef.current?.abort(); }, []);
  // 历史消息只在分页数据变化时重算，避免流式增量触发全量重建。
  const messages = useMemo(() => mergeMessagePages(query.data?.pages ?? []), [query.data?.pages]);
  const hasRunning = messages.some(row => row.status === 'running');
  const visible = useRef<HTMLDivElement>(null);
  const lastMessageId = messages.at(-1)?.id;
  const streamLength = streamState?.content.length ?? 0;
  useEffect(() => {
    // 新增消息与流式增量滚至底部；加载更早记录不会夺走用户正在阅读的位置。
    if (visible.current) visible.current.scrollTop = visible.current.scrollHeight;
  }, [lastMessageId, streamLength]);

  async function settle(attempt: MessageAttempt, submittedRevision?: number) {
    setFeedback({ request: attempt.request, state: 'sending' });
    try {
      await send.mutateAsync(attempt);
      if (!isCurrent()) return;
      setFeedback({ request: attempt.request, state: 'succeeded' });
      // 等待时允许继续编辑，成功回包不能清掉用户新写的内容。
      if (submittedRevision !== undefined && submittedRevision === draftRevision.current
        && form.getFieldValue('content') === attempt.request.content) form.setFieldValue('content', '');
    } catch (error) {
      if (!isCurrent()) return;
      setFeedback({ request: attempt.request, state: 'unknown', error: errorText(error) });
    } finally {
      if (isCurrent()) {
        await Promise.all([
          queryClient.invalidateQueries({ queryKey: aiChatKeys.messages(scope, id) }),
          refreshLists(),
        ]);
      }
      finishWrite();
    }
  }

  // 流式发送：delta 增量渲染草稿气泡；done/error 后清草稿并以服务端持久化数据为准。
  async function settleStream(request: AiSendMessageReq, submittedRevision?: number) {
    setFeedback({ request, state: 'sending' });
    setStreamState({ content: '', tools: [] });
    const abort = new AbortController();
    abortRef.current = abort;
    try {
      await sendAiMessageStream(request, {
        onDelta: content => { if (isCurrent()) setStreamState(s => (s ? { ...s, content: s.content + content } : s)); },
        onTool: (name, status) => { if (isCurrent()) setStreamState(s => (s ? { ...s, tools: [...s.tools, { key: s.tools.length, name, status }] } : s)); },
      }, abort.signal);
      if (!isCurrent()) return;
      setFeedback({ request, state: 'succeeded' });
      // 等待时允许继续编辑，成功回包不能清掉用户新写的内容。
      if (submittedRevision !== undefined && submittedRevision === draftRevision.current
        && form.getFieldValue('content') === request.content) form.setFieldValue('content', '');
    } catch (error) {
      if (error instanceof DOMException && error.name === 'AbortError') return; // 卸载或切换会话主动中断
      if (!isCurrent()) return;
      setFeedback({ request, state: 'unknown', error: errorText(error) });
    } finally {
      abortRef.current = null;
      if (isCurrent()) {
        setStreamState(null);
        await Promise.all([
          queryClient.invalidateQueries({ queryKey: aiChatKeys.messages(scope, id) }),
          refreshLists(),
        ]);
      }
      sender.finish();
      finishWrite();
    }
  }

  function sendMessage(values: ComposerValues) {
    if (query.isError || !query.data || !beginWrite('sys:ai:message:send')) return;
    const request = sender.startStream(id, values.content);
    if (!request) { finishWrite(); return; }
    void settleStream(request, draftRevision.current);
  }

  function recheck() {
    if (query.isError || !query.data || !beginWrite('sys:ai:message:send')) return;
    const attempt = sender.recheck();
    if (!attempt) { finishWrite(); return; }
    void settle(attempt);
  }

  const conversationTitle = query.data?.pages[0]?.conversation.title || 'AI 对话';
  return <Card className={styles.conversation} title={conversationTitle}
    extra={<Dropdown menu={{ items: [
      { key: 'md', label: '导出 Markdown' },
      { key: 'json', label: '导出 JSON' },
    ], onClick: ({ key }) => exportConversation(conversationTitle, messages, key as 'md' | 'json') }}>
      <Button size="small" icon={<DownloadOutlined />}>导出</Button>
    </Dropdown>}>
    <QueryState isPending={query.isPending} error={query.error} onRetry={() => query.refetch()} />
    {query.hasNextPage && <div className={styles.historyAction}><Button loading={query.isFetchingNextPage} disabled={writing || query.isFetching}
      onClick={() => { if (!query.isFetching) void query.fetchNextPage(); }}>加载更早消息</Button></div>}
    <div className={styles.messages} ref={visible} aria-live="polite">
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
        <Button type="primary" htmlType="submit" icon={<SendOutlined />} loading={send.isPending || streamState !== null}
          disabled={writing || query.isError || !query.data || !draft.trim()}>{feedback?.state === 'unknown' || hasRunning ? '重新发送（新请求）' : '发送'}</Button>
      </div>
    </Form> : <Typography.Text type="secondary">当前账号没有发送消息权限，可查看历史消息。</Typography.Text>}
  </Card>;
}
