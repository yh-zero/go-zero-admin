import { useEffect, useRef, useState } from 'react';
import { Button, Card, Empty, Pagination, Popconfirm, Result, Skeleton, Space } from 'antd';
import { DeleteOutlined, PlusOutlined, ReloadOutlined } from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { deleteAiConversation } from '../../api/ai';
import { PageHeader } from '../../components/PageHeader';
import { QueryState } from '../../components/QueryState';
import { useAuthStore } from '../../stores/auth';
import { formatTime } from '../../utils/format';
import { aiChatKeys, conversationListOptions } from './chatQueries';
import type { AiChatScope } from './chatQueries';
import { Conversation } from './Conversation';
import { NewConversationComposer } from './NewConversationComposer';
import styles from './chat.module.css';

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
  const remove = useMutation({ mutationFn: deleteAiConversation, retry: false });

  async function refreshLists() { await queryClient.invalidateQueries({ queryKey: aiChatKeys.lists(scope) }); }
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
      {canCreate && <Button type="primary" icon={<PlusOutlined />} disabled={writing} onClick={() => { setSelectedId(null); }}>新建会话</Button>}
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
      {selectedId === null ? <NewConversationComposer
          canSend={canSend} writing={writing}
          isCurrent={isCurrent} beginWrite={beginWrite} finishWrite={finishWrite}
          onCreated={async (id) => { setSelectedId(id); setPage(1); await refreshLists(); }} />
        : <Conversation key={selectedId} id={selectedId} scope={scope} canRead={canRead} canSend={canSend} writing={writing}
          isCurrent={() => isCurrent() && !removedIds.has(selectedId)} beginWrite={beginWrite} finishWrite={finishWrite}
          refreshLists={refreshLists} />}
    </div>
  </div>;
}
