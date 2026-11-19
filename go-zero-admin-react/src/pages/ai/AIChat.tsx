import { useEffect, useRef, useState } from 'react';
import { Button, Card, Dropdown, Empty, Form, Input, Modal, Pagination, Popconfirm, Result, Skeleton, Space } from 'antd';
import { DeleteOutlined, MoreOutlined, PlusOutlined, PushpinFilled, ReloadOutlined } from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useLocation } from 'react-router-dom';
import { deleteAiConversation, updateAiConversation } from '../../api/ai';
import type { AiConversationInfo, AiSendMessageResult, AiUpdateConversationReq } from '../../api/types';
import { QueryState } from '../../components/QueryState';
import { useAuthStore } from '../../stores/auth';
import { formatTime } from '../../utils/format';
import { aiChatKeys, conversationListOptions } from './chatQueries';
import type { AiChatScope } from './chatQueries';
import { Conversation } from './Conversation';
import { NewConversationComposer } from './NewConversationComposer';
import { automaticKnowledge, readChatKnowledge, writeChatKnowledge } from './chatKnowledge';
import type { ChatKnowledgeSelection, StoredChatKnowledge } from './chatKnowledge';
import type { ChatScrollPosition } from './useChatScroll';
import styles from './chat.module.css';

export default function AIChat() {
  const auth = useAuthStore();
  const { key, state } = useLocation();
  const scope: AiChatScope = [auth.sessionId, auth.sessionVersion, auth.departmentVersion];
  const selected = state?.knowledge;
  const initialKbId = selected?.scope === JSON.stringify(scope) && Number.isSafeInteger(selected.id) && selected.id > 0 ? selected.id as number : 0;
  const initialDocId = initialKbId && Number.isSafeInteger(selected.doc_id) && selected.doc_id > 0 ? selected.doc_id as number : undefined;
  return <AIChatAccess scope={scope} entryKey={initialKbId ? key : undefined} initialKbId={initialKbId} initialDocId={initialDocId} permissionsReady={auth.permissionsReady} permCodes={auth.permCodes} />;
}

interface AccessProps { scope: AiChatScope; initialKbId?: number; initialDocId?: number; entryKey?: string; permissionsReady: boolean; permCodes: readonly string[] }

export function AIChatAccess({ scope, initialKbId = 0, initialDocId, entryKey, permissionsReady, permCodes }: AccessProps) {
  if (!permissionsReady) return <div className={styles.workspace} role="status" aria-label="AI 对话加载中"><Skeleton active /></div>;
  if (!permCodes.includes('sys:ai:conversation:list')) return <Result status="403" title="无权查看 AI 对话" subTitle="请联系管理员授予查看会话权限。" />;
  // 身份或部门切换时一并销毁选择、草稿与本次提交引用，避免旧会话回调影响新身份。
  return <AIChatSession key={JSON.stringify([scope, entryKey])} scope={scope} initialKbId={initialKbId} initialDocId={initialDocId} entryKey={entryKey} />;
}

function AIChatSession({ scope, initialKbId, initialDocId, entryKey }: { scope: AiChatScope; initialKbId: number; initialDocId?: number; entryKey?: string }) {
  const auth = useAuthStore();
  const canRead = auth.permissionsReady && auth.permCodes.includes('sys:ai:conversation:list');
  const canCreate = auth.permissionsReady && auth.permCodes.includes('sys:ai:conversation:create');
  const canDelete = auth.permissionsReady && auth.permCodes.includes('sys:ai:conversation:delete');
  const canSend = auth.permissionsReady && auth.permCodes.includes('sys:ai:message:send');
  const queryClient = useQueryClient();
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState('');
  const [renaming, setRenaming] = useState<AiConversationInfo | null>(null);
  const [selectedId, setSelectedId] = useState<number | null>(null);
  const [readLatestOnOpen, setReadLatestOnOpen] = useState(false);
  const sessionId = scope[0], departmentId = auth.selectedDeptId;
  const [initialDraft] = useState(() => {
    const stored = readChatKnowledge(sessionId, departmentId, 0);
    return initialKbId && (!stored || stored.entryKey !== entryKey) ? { kb_id: initialKbId, doc_id: initialDocId, grounded: true, entryKey } : stored ?? automaticKnowledge;
  });
  const [knowledge, setSelection] = useState<StoredChatKnowledge | undefined>(initialDraft);
  const [knowledgeRevision, setKnowledgeRevision] = useState(0);
  const assignedConversation = useRef<number | null>(null);
  const draftEntryKey = entryKey ?? ('entryKey' in initialDraft ? initialDraft.entryKey : undefined);
  useEffect(() => {
    if (initialKbId) writeChatKnowledge(sessionId, departmentId, 0, initialDraft, draftEntryKey);
  }, [sessionId, departmentId, initialKbId, initialDraft, draftEntryKey]);
  function setKnowledge(value?: ChatKnowledgeSelection) {
    if (!isCurrent()) return;
    setKnowledgeRevision(revision => revision + 1);
    writeChatKnowledge(sessionId, departmentId, selectedId ?? 0, value, selectedId === null ? draftEntryKey : undefined);
    if (selectedId === null && assignedConversation.current !== null) {
      writeChatKnowledge(sessionId, departmentId, assignedConversation.current, value);
    }
    setSelection(readChatKnowledge(sessionId, departmentId, selectedId ?? 0));
  }
  function selectConversation(id: number | null, reopen = false) {
    assignedConversation.current = null;
    if (reopen && id !== null && id === selectedId) void queryClient.invalidateQueries({ queryKey: aiChatKeys.latest(scope, id), exact: true });
    setReadLatestOnOpen(reopen);
    setSelection(readChatKnowledge(sessionId, departmentId, id ?? 0) ?? (id === null ? automaticKnowledge : undefined));
    setSelectedId(id);
  }
  function rememberAssignedConversation(id: number) {
    if (!isCurrent()) return;
    assignedConversation.current = id;
    // 重查仍发送原请求，已分配会话接续的则是编辑器当前选择。
    writeChatKnowledge(sessionId, departmentId, id, knowledge ?? automaticKnowledge);
  }
  const [composerVersion, setComposerVersion] = useState(0);
  const [writing, setWriting] = useState(false);
  const busy = useRef(false);
  // 显式导航更换容器；自动创建完成时复用，避免旧面板清理覆盖新面板的位置。
  const [scrollPosition, setScrollPosition] = useState<{ current: ChatScrollPosition | null }>(() => ({ current: null }));
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
  const conversations = useQuery({ ...conversationListOptions(scope, page, search), enabled: canRead });
  const remove = useMutation({ mutationFn: deleteAiConversation, retry: false });
  const update = useMutation({ mutationFn: ({ id, data }: { id: number; data: AiUpdateConversationReq }) => updateAiConversation(id, data), retry: false });

  async function organize(id: number, data: AiUpdateConversationReq) {
    if (!beginWrite('sys:ai:conversation:create')) return;
    try {
      await update.mutateAsync({ id, data });
      if (!isCurrent()) return;
      setRenaming(null);
      await Promise.all([refreshLists(), queryClient.invalidateQueries({ queryKey: aiChatKeys.messages(scope, id), exact: true })]);
    } catch { /* 表单与原列表保留，允许用户修正或重试。 */ }
    finally { finishWrite(); }
  }

  async function refreshLists() { await queryClient.invalidateQueries({ queryKey: aiChatKeys.lists(scope) }); }
  function onSendCompleted(result: AiSendMessageResult) {
    if (!isCurrent()) return;
    if (selectedId === null) {
      writeChatKnowledge(sessionId, departmentId, 0, automaticKnowledge, draftEntryKey);
    }
    if (!result.conversation) {
      writeChatKnowledge(sessionId, departmentId, result.conversation_id);
      setRemovedIds(current => new Set(current).add(result.conversation_id));
      setScrollPosition({ current: null });
      selectConversation(null);
      return;
    }
    selectConversation(result.conversation_id);
    setPage(1);
  }
  async function deleteConversation(id: number) {
    if (removedIds.has(id) || !beginWrite('sys:ai:conversation:delete')) return;
    try {
      await remove.mutateAsync(id);
      if (!isCurrent()) return;
      writeChatKnowledge(sessionId, departmentId, id);
      setRemovedIds(current => new Set(current).add(id));
      if (selectedId === id) { setScrollPosition({ current: null }); selectConversation(null); }
      const queryKey = aiChatKeys.messages(scope, id);
      await queryClient.cancelQueries({ queryKey });
      queryClient.removeQueries({ queryKey });
      if ((conversations.data?.list.length ?? 0) <= 1 && page > 1) setPage(page - 1);
      await refreshLists();
    } catch { /* 不移除失败的会话；统一层显示失败原因。 */ }
    finally { finishWrite(); }
  }

  return <div className={styles.workspace}>
      <Card className={styles.sidebar} title="我的会话" extra={<Space size={4}>
      <Button size="small" icon={<ReloadOutlined />} aria-label="刷新会话" title="刷新会话" disabled={writing} onClick={() => {
        void refreshLists();
        if (selectedId !== null) void queryClient.invalidateQueries({ queryKey: aiChatKeys.latest(scope, selectedId), exact: true });
      }} />
      {canCreate && <Button size="small" type="primary" icon={<PlusOutlined />} disabled={writing} onClick={() => { setScrollPosition({ current: null }); writeChatKnowledge(sessionId, departmentId, 0, automaticKnowledge, draftEntryKey); selectConversation(null); setComposerVersion(version => version + 1); }}>新建会话</Button>}
    </Space>}>
        <Input.Search className={styles.conversationSearch} placeholder="搜索会话标题或正文" aria-label="搜索会话" maxLength={128} allowClear disabled={writing} onSearch={value => { setPage(1); setSearch(value.trim()); }} />
        <div className={styles.list}>
          <QueryState isPending={conversations.isPending} error={conversations.error} onRetry={() => conversations.refetch()} />
          {!conversations.isPending && !conversations.error && !conversations.data?.list.length && <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无会话" />}
          {(conversations.data?.list ?? []).filter(row => !removedIds.has(row.id)).map(row => <div key={row.id}
            className={`${styles.conversationItem} ${selectedId === row.id ? styles.selected : ''}`}>
            <Button type="text" className={styles.selectConversation} disabled={writing} aria-pressed={selectedId === row.id}
              onClick={() => { if (!busy.current && !removedIds.has(row.id)) { if (selectedId !== row.id) setScrollPosition({ current: null }); selectConversation(row.id, true); } }}>
              <span className={styles.title} title={row.title}>{row.pinned && <PushpinFilled />} {row.title || '未命名会话'}</span>
              <span className={styles.small}>{row.message_count} 条消息 · {formatTime(row.updated_at)}</span>
            </Button>
            {canCreate && <Dropdown menu={{ items: [{ key: 'rename', label: '重命名' }, { key: 'pin', label: row.pinned ? '取消置顶' : '置顶' }], onClick: ({ key }) => { if (writing) return; if (key === 'rename') setRenaming(row); else void organize(row.id, { pinned: !row.pinned }); } }}><Button type="text" icon={<MoreOutlined />} disabled={writing} aria-label={`整理会话 ${row.title || row.id}`} /></Dropdown>}
            {canDelete && <Popconfirm title="删除此会话？" description="会话与消息删除后无法继续使用。" disabled={writing}
              onConfirm={() => deleteConversation(row.id)} okButtonProps={{ danger: true }}>
              <Button type="text" danger icon={<DeleteOutlined />} disabled={writing} aria-label={`删除会话 ${row.title || row.id}`} />
            </Popconfirm>}
          </div>)}
        </div>
        <Pagination simple current={page} pageSize={20} total={conversations.data?.total ?? 0} disabled={writing}
          onChange={next => { if (!busy.current) setPage(next); }} showSizeChanger={false} />
      </Card>
      {renaming && <RenameConversationDialog conversation={renaming} writing={writing} saving={update.isPending} onCancel={() => setRenaming(null)} onSave={values => organize(renaming.id, values)} />}
      {selectedId === null ? <NewConversationComposer key={composerVersion}
          scope={scope} canSend={canSend} writing={writing} scrollPosition={scrollPosition}
          knowledge={knowledge} knowledgeRevision={knowledgeRevision} onKnowledgeChange={setKnowledge}
          isCurrent={isCurrent} beginWrite={beginWrite} finishWrite={finishWrite}
          onConversationAssigned={rememberAssignedConversation} onCompleted={onSendCompleted} />
        : <Conversation key={selectedId} id={selectedId} scope={scope} canRead={canRead} canSend={canSend} writing={writing} scrollPosition={scrollPosition} readLatestOnOpen={readLatestOnOpen}
          knowledge={knowledge} knowledgeRevision={knowledgeRevision} onKnowledgeChange={setKnowledge}
          isCurrent={() => isCurrent() && !removedIds.has(selectedId)} beginWrite={beginWrite} finishWrite={finishWrite}
          onCompleted={onSendCompleted} />}
  </div>;
}

function RenameConversationDialog({ conversation, writing, saving, onCancel, onSave }: {
  conversation: AiConversationInfo; writing: boolean; saving: boolean; onCancel: () => void; onSave: (values: { title: string }) => Promise<void>;
}) {
  const [form] = Form.useForm<{ title: string }>();
  return <Modal open title="重命名会话" confirmLoading={saving} closable={!writing} mask={{ closable: !writing }} onCancel={() => { if (!writing) onCancel(); }} onOk={() => form.submit()}>
    <Form form={form} initialValues={{ title: conversation.title }} onFinish={onSave}>
      <Form.Item name="title" label="名称" rules={[{ required: true, whitespace: true }, { validator: (_, value: string) => Array.from(value ?? '').length <= 128 ? Promise.resolve() : Promise.reject(new Error('名称最多 128 个字符')) }]}><Input autoFocus /></Form.Item>
    </Form>
  </Modal>;
}
