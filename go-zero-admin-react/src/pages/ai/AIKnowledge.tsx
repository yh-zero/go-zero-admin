import { useEffect, useState } from 'react';
import { Button, Card, Empty, Form, Input, Modal, Popconfirm, Space, Table, Tag } from 'antd';
import { DatabaseOutlined, DeleteOutlined, EditOutlined, PlusOutlined, ReloadOutlined } from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import type { AiKbInfo } from '../../api/types';
import { createAiKb, deleteAiKb, updateAiKb } from '../../api/ai';
import { QueryState } from '../../components/QueryState';
import { AIWorkspaceAccess } from './AIWorkspaceAccess';
import { useAiWorkspaceGuard } from './useAiWorkspaceGuard';
import { aiWorkspaceKeys, kbListOptions } from './workspaceQueries';
import type { AiWorkspaceScope } from './workspaceQueries';
import styles from './knowledgeDocuments.module.css';

const KB_LIST = 'sys:ai:kb:list';
const KB_CREATE = 'sys:ai:kb:create';
const KB_EDIT = 'sys:ai:kb:edit';
const KB_DELETE = 'sys:ai:kb:delete';

export default function AIKnowledge() {
  return <AIWorkspaceAccess code={KB_LIST} title="知识库">{scope => <AIKnowledgeSession scope={scope} />}</AIWorkspaceAccess>;
}

export function AIKnowledgeSession({ scope }: { scope: AiWorkspaceScope }) {
  const guard = useAiWorkspaceGuard(scope, KB_LIST);
  const { isCurrent } = guard;
  const client = useQueryClient();
  const navigate = useNavigate();
  const [editing, setEditing] = useState<AiKbInfo | 'create' | null>(null);
  const [query, setQuery] = useState('');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [deletingId, setDeletingId] = useState<number | null>(null);
  const kbs = useQuery({ ...kbListOptions(scope, { page, page_size: pageSize, query }), enabled: guard.can(KB_LIST) });
  const save = useMutation({ mutationFn: async ({ kb, values }: { kb: AiKbInfo | 'create'; values: KbFormValues }) => {
    if (kb === 'create') return createAiKb(values);
    else await updateAiKb(kb.id, values);
  } });
  const remove = useMutation({ mutationFn: deleteAiKb });
  useEffect(() => {
    // oxlint-disable-next-line react/set-state-in-effect -- 删除后以服务器总数修正当前分页。
    if (kbs.data && isCurrent()) setPage(current => Math.min(current, Math.max(1, Math.ceil(kbs.data.total / pageSize))));
  }, [kbs.data, pageSize, isCurrent]);

  const refresh = () => { if (guard.isCurrent()) void client.invalidateQueries({ queryKey: aiWorkspaceKeys.kbs(scope) }); };

  return <Card className={styles.knowledgePage} title={<Space><DatabaseOutlined /><span>知识库</span><Tag>资料与分析</Tag></Space>}
    extra={<Space>
      <Button icon={<ReloadOutlined />} loading={kbs.isFetching} onClick={refresh}>刷新</Button>
      {guard.can(KB_CREATE) && <Button type="primary" icon={<PlusOutlined />} disabled={guard.writing} onClick={() => setEditing('create')}>新建知识库</Button>}
    </Space>}>
    <Input.Search allowClear className={styles.kbSearch} aria-label="搜索知识库名称" placeholder="搜索知识库名称" onSearch={value => { setQuery(value.trim()); setPage(1); }} />
    <QueryState isPending={kbs.isPending} error={kbs.error} onRetry={() => kbs.refetch()} />
    {!kbs.isPending && !kbs.error && !(kbs.data?.total) && <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无知识库" />}
    {!!kbs.data?.total && <Table<AiKbInfo> rowKey="id" size="middle" scroll={{ x: 850 }}
      pagination={{ current: page, pageSize, total: kbs.data.total, showSizeChanger: true, disabled: guard.writing,
        onChange: (next, size) => { setPage(size === pageSize ? next : 1); setPageSize(size); } }}
      dataSource={kbs.data.list}
      columns={[
        { title: '名称', dataIndex: 'name', render: (_, kb) => <Button type="link" onClick={() => navigate(`/ai/knowledge/${kb.id}`)}>{kb.name}</Button> },
        { title: '描述', dataIndex: 'description', ellipsis: true },
        { title: '资料', width: 180, render: (_, kb) => <Space wrap><Tag color="success">可使用 {kb.ready_count}</Tag>{kb.pending_count > 0 && <Tag color="processing">处理 {kb.pending_count}</Tag>}{kb.failed_count > 0 && <Tag color="error">失败 {kb.failed_count}</Tag>}</Space> },
        { title: '更新时间', dataIndex: 'updated_at', width: 170, render: (v: number) => new Date(v * 1000).toLocaleString() },
        { title: '操作', width: 340, render: (_, kb) => <Space>
          <Button type="link" disabled={guard.writing} onClick={() => navigate(`/ai/knowledge/${kb.id}`)}>进入工作区</Button>
          {guard.can(KB_EDIT) && <Button type="link" icon={<EditOutlined />} disabled={guard.writing} onClick={() => setEditing(kb)}>编辑</Button>}
          {guard.can(KB_DELETE) && <Popconfirm title="删除知识库及其全部文档？" disabled={guard.writing} onConfirm={() => removeKb(kb.id)}><Button type="link" danger disabled={guard.writing} loading={deletingId === kb.id} icon={<DeleteOutlined />}>删除</Button></Popconfirm>}
        </Space> },
      ]} />}
    {editing && <KbEditModal kb={editing === 'create' ? null : editing} writing={guard.writing}
      onCancel={() => { if (!guard.writing) setEditing(null); }}
      onSubmit={async values => {
        if (!guard.beginWrite(editing === 'create' ? KB_CREATE : KB_EDIT)) return;
        try {
          const result = await save.mutateAsync({ kb: editing, values });
          if (guard.isCurrent()) { setEditing(null); refresh(); if (result) navigate(`/ai/knowledge/${result.kb.id}`); }
        } catch { /* 请求层已反馈错误，表单保留。 */ }
        finally { guard.finishWrite(); }
      }} />}
  </Card>;

  async function removeKb(id: number) {
    if (!guard.beginWrite(KB_DELETE)) return;
    setDeletingId(id);
    try { await remove.mutateAsync(id); refresh(); }
    catch { /* 请求层统一反馈。 */ }
    finally { if (guard.isCurrent()) setDeletingId(null); guard.finishWrite(); }
  }
}

interface KbFormValues { name: string; description?: string }

function KbEditModal({ kb, writing, onCancel, onSubmit }: {
  kb: AiKbInfo | null; writing: boolean;
  onCancel: () => void; onSubmit: (values: KbFormValues) => Promise<void>;
}) {
  const [form] = Form.useForm<KbFormValues>();
  return <Modal open title={kb ? '编辑知识库' : '新建知识库'} okText="保存" cancelText="取消" confirmLoading={writing}
    closable={!writing} keyboard={!writing} mask={{ closable: !writing }} cancelButtonProps={{ disabled: writing }}
    onCancel={onCancel} onOk={() => { void form.validateFields().then(onSubmit).catch(() => undefined); }} destroyOnHidden>
    <Form form={form} layout="vertical" disabled={writing} initialValues={kb ? { name: kb.name, description: kb.description } : {}} preserve={false}>
      <Form.Item name="name" label="名称" rules={[{ required: true, max: 128, message: '名称须为 1..128 字符' }, { whitespace: true, message: '名称不能为空白' }]}>
        <Input maxLength={128} placeholder="如：运维手册" />
      </Form.Item>
      <Form.Item name="description" label="描述" rules={[{ max: 512, message: '描述最多 512 字符' }]}>
        <Input.TextArea rows={2} maxLength={512} placeholder="可选" />
      </Form.Item>
    </Form>
  </Modal>;
}
