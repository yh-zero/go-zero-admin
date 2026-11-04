import { useEffect, useRef, useState } from 'react';
import { Alert, Button, Card, Empty, Form, Input, Modal, Pagination, Popconfirm, Result, Skeleton, Space, Typography } from 'antd';
import { DeleteOutlined, PlusOutlined, ReloadOutlined, SendOutlined } from '@ant-design/icons';
import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { createAiConversation, deleteAiConversation, sendAiMessage } from '../../api/ai';
import type { AiChatMessage, AiSendMessageReq } from '../../api/types';
import { PageHeader } from '../../components/PageHeader';
import { QueryState } from '../../components/QueryState';
import { useAuthStore } from '../../stores/auth';
import { formatTime } from '../../utils/format';
import { aiChatKeys, conversationListOptions, conversationMessagesOptions } from './chatQueries';
import type { AiChatScope } from './chatQueries';
import { createMessageSender, mergeMessagePages, validateMessageContent } from './chatState';
import type { MessageAttempt } from './chatState';
import { MessageRecord } from './MessageRecord';
import styles from './chat.module.css';

type ComposerValues = { content: string };
type SendFeedback = { request: AiSendMessageReq; state: 'sending' | 'unknown' | 'succeeded'; error?: string };

function errorText(error: unknown) { return error instanceof Error ? error.message : '请求未完成，请稍后查看会话记录'; }

export default function AIChat() {
  const auth = useAuthStore();
  const scope: AiChatScope = [auth.sessionId, auth.sessionVersion, auth.departmentVersion];
  return <AIChatAccess scope={scope} permissionsReady={auth.permissionsReady} permCodes={auth.permCodes} />;
}

interface AccessProps { scope: AiChatScope; permissionsReady: boolean; permCodes: readonly string[] }

export function AIChatAccess({ scope, permissionsReady, permCodes }: AccessProps) {
  if (!permissionsReady) return <div className="page-stack"><PageHeader title="AI 对话" /><Skeleton active /></div>;
  if (!permCodes.includes('sys:ai:conversation:list')) return <Result status="403" title="无权查看 AI 对话" subTitle="请联系管理员授予查看会话权限。" />;
  // 身份或部门切换时一并销毁选择、草稿与本次提交引用，避免旧会话回调影响新身份。
  return <AIChatSession key={JSON.stringify(scope)} scope={scope} />;
}

function AIChatSession({ scope }: { scope: AiChatScope }) {
  const auth = useAuthStore();
  const canRead = auth.permissionsReady && auth.permCodes.includes('sys:ai:conversation:list');
  const canCreate = auth.permissionsReady && auth.permCodes.includes('sys:ai:conversation:create');
  const canDelete = auth.permissionsReady && auth.permCodes.includes('sys:ai:conversation:delete');
  const canSend = auth.permissionsReady && auth.permCodes.includes('sys:ai:message:send');
  const queryClient = useQueryClient();
  const [page, setPage] = useState(1);
  const [selectedId, setSelectedId] = useState<number | null>(null);
  const [createOpen, setCreateOpen] = useState(false);
  const [createForm] = Form.useForm<{ title?: string }>();
  const [writing, setWriting] = useState(false);
  const busy = useRef(false);
  const [removedIds, setRemovedIds] = useState(() => new Set<number>());
  const mounted = useRef(true);
  useEffect(() => { mounted.current = true; return () => { mounted.current = false; }; }, []);

  function isCurrent() {
    const current = useAuthStore.getState();
    return mounted.current && current.sessionId === scope[0] && current.sessionVersion === scope[1]
      && current.departmentVersion === scope[2] && current.permissionsReady
      && current.permCodes.includes('sys:ai:conversation:list');
  }
  function beginWrite(permission: string) {
    if (busy.current || !isCurrent() || !useAuthStore.getState().permCodes.includes(permission)) return false;
    busy.current = true;
    setWriting(true);
    return true;
  }
  function finishWrite() { busy.current = false; if (isCurrent()) setWriting(false); }
  const conversations = useQuery({ ...conversationListOptions(scope, page), enabled: canRead });
  const create = useMutation({ mutationFn: createAiConversation, retry: false });
  const remove = useMutation({ mutationFn: deleteAiConversation, retry: false });

  async function refreshLists() { await queryClient.invalidateQueries({ queryKey: aiChatKeys.lists(scope) }); }
  async function createConversation(values: { title?: string }) {
    if (!beginWrite('sys:ai:conversation:create')) return;
    try {
      const data = await create.mutateAsync(values.title?.trim() || undefined);
      if (!isCurrent()) return;
      setCreateOpen(false);
      createForm.resetFields();
      setSelectedId(data.conversation.id);
      setPage(1);
      await refreshLists();
    } catch { /* 请求层已提示，保留表单与错误反馈供用户处理。 */ }
    finally { finishWrite(); }
  }
  async function deleteConversation(id: number) {
    if (removedIds.has(id) || !beginWrite('sys:ai:conversation:delete')) return;
    try {
      await remove.mutateAsync(id);
      if (!isCurrent()) return;
      setRemovedIds(current => new Set(current).add(id));
      if (selectedId === id) setSelectedId(null);
      const queryKey = aiChatKeys.messages(scope, id);
      await queryClient.cancelQueries({ queryKey });
      queryClient.removeQueries({ queryKey });
      if ((conversations.data?.list.length ?? 0) <= 1 && page > 1) setPage(page - 1);
      await refreshLists();
    } catch { /* 不移除失败的会话；统一层显示失败原因。 */ }
    finally { finishWrite(); }
  }

  return <div className="page-stack">
    <PageHeader title="AI 对话" description="与 AI 助手对话，查看消息、工具调用记录与已知用量。" extra={<Space wrap>
      <Button icon={<ReloadOutlined />} disabled={writing} onClick={() => {
        void refreshLists();
        if (selectedId !== null) void queryClient.invalidateQueries({ queryKey: aiChatKeys.messages(scope, selectedId) });
      }}>刷新</Button>
      {canCreate && <Button type="primary" icon={<PlusOutlined />} disabled={writing} onClick={() => { create.reset(); setCreateOpen(true); }}>新建会话</Button>}
    </Space>} />
    <div className={styles.workspace}>
      <Card className={styles.sidebar} title="我的会话">
        <QueryState isPending={conversations.isPending} error={conversations.error} onRetry={() => conversations.refetch()} />
        {!conversations.isPending && !conversations.error && !conversations.data?.list.length && <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无会话" />}
        <div className={styles.list}>
          {(conversations.data?.list ?? []).filter(row => !removedIds.has(row.id)).map(row => <div key={row.id}
            className={`${styles.conversationItem} ${selectedId === row.id ? styles.selected : ''}`}>
            <Button type="text" className={styles.selectConversation} disabled={writing} aria-pressed={selectedId === row.id}
              onClick={() => { if (!busy.current && !removedIds.has(row.id)) setSelectedId(row.id); }}>
              <span className={styles.title} title={row.title}>{row.title || '未命名会话'}</span>
              <span className={styles.small}>{row.message_count} 条消息 · {formatTime(row.updated_at)}</span>
            </Button>
            {canDelete && <Popconfirm title="删除此会话？" description="会话与消息删除后无法继续使用。" disabled={writing}
              onConfirm={() => deleteConversation(row.id)} okButtonProps={{ danger: true }}>
              <Button type="text" danger icon={<DeleteOutlined />} disabled={writing} aria-label={`删除会话 ${row.title || row.id}`} />
            </Popconfirm>}
          </div>)}
        </div>
        <Pagination simple current={page} pageSize={20} total={conversations.data?.total ?? 0} disabled={writing}
          onChange={next => { if (!busy.current) setPage(next); }} showSizeChanger={false} />
      </Card>
      {selectedId === null ? <Card className={styles.conversation}><Empty description="选择一个会话，或新建会话开始对话" /></Card>
        : <Conversation key={selectedId} id={selectedId} scope={scope} canRead={canRead} canSend={canSend} writing={writing}
          isCurrent={() => isCurrent() && !removedIds.has(selectedId)} beginWrite={beginWrite} finishWrite={finishWrite}
          refreshLists={refreshLists} />}
    </div>
    <Modal open={createOpen} title="新建 AI 会话" onCancel={() => { if (!busy.current) setCreateOpen(false); }}
      onOk={() => createForm.submit()} confirmLoading={create.isPending} cancelButtonProps={{ disabled: writing }} destroyOnHidden>
      {create.error && <Alert className={styles.feedback} type="error" showIcon title={errorText(create.error)} />}
      <Form form={createForm} layout="vertical" onFinish={createConversation}>
        <Form.Item name="title" label="会话标题（可选）" rules={[{ validator: (_, value?: string) =>
          [...(value ?? '')].length <= 128 ? Promise.resolve() : Promise.reject(new Error('标题最多 128 个字符')) }]}>
          <Input placeholder="留空由系统命名" disabled={writing} />
        </Form.Item>
      </Form>
    </Modal>
  </div>;
}

interface ConversationProps {
  id: number; scope: AiChatScope; canRead: boolean; canSend: boolean; writing: boolean;
  isCurrent: () => boolean; beginWrite: (permission: string) => boolean; finishWrite: () => void; refreshLists: () => Promise<void>;
}

function Conversation({ id, scope, canRead, canSend, writing, isCurrent, beginWrite, finishWrite, refreshLists }: ConversationProps) {
  const queryClient = useQueryClient();
  const query = useInfiniteQuery({ ...conversationMessagesOptions(scope, id), enabled: canRead });
  const [form] = Form.useForm<ComposerValues>();
  const draft = Form.useWatch('content', form) ?? '';
  const draftRevision = useRef(0);
  const [sender] = useState(() => createMessageSender(sendAiMessage));
  const [feedback, setFeedback] = useState<SendFeedback | null>(null);
  const send = useMutation({ mutationFn: (attempt: MessageAttempt) => attempt.promise, retry: false });
  const messages = mergeMessagePages(query.data?.pages ?? []);
  const hasRunning = messages.some(row => row.status === 'running');
  const visible = useRef<HTMLDivElement>(null);
  const lastMessageId = messages.at(-1)?.id;
  useEffect(() => {
    // 仅新增消息时滚至底部；加载更早记录不会夺走用户正在阅读的位置。
    if (visible.current) visible.current.scrollTop = visible.current.scrollHeight;
  }, [lastMessageId]);

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
  function sendMessage(values: ComposerValues) {
    if (query.isError || !query.data || !beginWrite('sys:ai:message:send')) return;
    const attempt = sender.start(id, values.content);
    if (!attempt) { finishWrite(); return; }
    void settle(attempt, draftRevision.current);
  }
  function recheck() {
    if (query.isError || !query.data || !beginWrite('sys:ai:message:send')) return;
    const attempt = sender.recheck();
    if (!attempt) { finishWrite(); return; }
    void settle(attempt);
  }

  return <Card className={styles.conversation} title={query.data?.pages[0]?.conversation.title || 'AI 对话'}>
    <QueryState isPending={query.isPending} error={query.error} onRetry={() => query.refetch()} />
    {query.hasNextPage && <div className={styles.historyAction}><Button loading={query.isFetchingNextPage} disabled={writing || query.isFetching}
      onClick={() => { if (!query.isFetching) void query.fetchNextPage(); }}>加载更早消息</Button></div>}
    <div className={styles.messages} ref={visible} aria-live="polite">
      {!query.isPending && !query.error && messages.length === 0 && <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无消息，发送第一条消息吧" />}
      {messages.map((message: AiChatMessage) => <MessageRecord key={message.id} message={message} />)}
    </div>
    {feedback?.state === 'sending' && <Alert className={styles.feedback} type="info" showIcon title="AI 正在回复，请稍候…" description="请求可能需要较长时间，请勿重复提交。" />}
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
        <Button type="primary" htmlType="submit" icon={<SendOutlined />} loading={send.isPending}
          disabled={writing || query.isError || !query.data || !draft.trim()}>{feedback?.state === 'unknown' || hasRunning ? '重新发送（新请求）' : '发送'}</Button>
      </div>
    </Form> : <Typography.Text type="secondary">当前账号没有发送消息权限，可查看历史消息。</Typography.Text>}
  </Card>;
}
