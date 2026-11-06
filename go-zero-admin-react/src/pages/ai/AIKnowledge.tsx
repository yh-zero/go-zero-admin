import { useState } from 'react';
import { Button, Card, Drawer, Empty, Form, Input, Modal, Popconfirm, Space, Table, Tag } from 'antd';
import { DatabaseOutlined, DeleteOutlined, EditOutlined, PlusOutlined, RedoOutlined, ReloadOutlined } from '@ant-design/icons';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import type { AiKbDocInfo, AiKbInfo } from '../../api/types';
import { createAiKb, createAiKbDoc, deleteAiKb, deleteAiKbDoc, retryAiKbDoc, updateAiKb } from '../../api/ai';
import { QueryState } from '../../components/QueryState';
import { AIWorkspaceAccess } from './AIWorkspaceAccess';
import { useAiWorkspaceGuard } from './useAiWorkspaceGuard';
import { aiWorkspaceKeys, kbDocListOptions, kbListOptions } from './workspaceQueries';
import type { AiWorkspaceScope } from './workspaceQueries';

const KB_LIST = 'sys:ai:kb:list';
const KB_CREATE = 'sys:ai:kb:create';
const KB_EDIT = 'sys:ai:kb:edit';
const KB_DELETE = 'sys:ai:kb:delete';

export default function AIKnowledge() {
  return <AIWorkspaceAccess code={KB_LIST} title="知识库">{scope => <AIKnowledgeSession scope={scope} />}</AIWorkspaceAccess>;
}

export function AIKnowledgeSession({ scope }: { scope: AiWorkspaceScope }) {
  const guard = useAiWorkspaceGuard(scope, KB_LIST);
  const client = useQueryClient();
  const [editing, setEditing] = useState<AiKbInfo | 'create' | null>(null);
  const [docsOf, setDocsOf] = useState<AiKbInfo | null>(null);
  const kbs = useQuery({ ...kbListOptions(scope, { page: 1, page_size: 20 }), enabled: guard.can(KB_LIST) });

  const refresh = () => { if (guard.isCurrent()) void client.invalidateQueries({ queryKey: aiWorkspaceKeys.kbs(scope) }); };

  return <Card title={<Space><DatabaseOutlined /><span>知识库</span><Tag>向量检索</Tag></Space>}
    extra={<Space>
      <Button icon={<ReloadOutlined />} onClick={refresh}>刷新</Button>
      {guard.can(KB_CREATE) && <Button type="primary" icon={<PlusOutlined />} onClick={() => setEditing('create')}>新建知识库</Button>}
    </Space>}>
    <QueryState isPending={kbs.isPending} error={kbs.error} onRetry={() => kbs.refetch()} />
    {!kbs.isPending && !kbs.error && !(kbs.data?.list.length) && <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无知识库" />}
    {!!kbs.data?.list.length && <Table<AiKbInfo> rowKey="id" size="middle" pagination={false}
      dataSource={kbs.data.list}
      columns={[
        { title: '名称', dataIndex: 'name' },
        { title: '描述', dataIndex: 'description', ellipsis: true },
        { title: '文档数', dataIndex: 'doc_count', width: 90 },
        { title: '分块数', dataIndex: 'chunk_count', width: 90 },
        { title: '更新时间', dataIndex: 'updated_at', width: 170, render: (v: number) => new Date(v * 1000).toLocaleString() },
        { title: '操作', width: 260, render: (_, kb) => <Space>
          <Button type="link" onClick={() => setDocsOf(kb)}>文档</Button>
          {guard.can(KB_EDIT) && <Button type="link" icon={<EditOutlined />} onClick={() => setEditing(kb)}>编辑</Button>}
          {guard.can(KB_DELETE) && <Popconfirm title="删除知识库及其全部文档？" onConfirm={() => void removeKb(kb.id)}><Button type="link" danger icon={<DeleteOutlined />}>删除</Button></Popconfirm>}
        </Space> },
      ]} />}
    {editing && <KbEditModal kb={editing === 'create' ? null : editing} writing={guard.writing}
      onCancel={() => setEditing(null)}
      onSubmit={async values => {
        if (!guard.beginWrite(editing === 'create' ? KB_CREATE : KB_EDIT)) return;
        try {
          if (editing === 'create') await createAiKb(values);
          else await updateAiKb(editing.id, values);
          setEditing(null);
          refresh();
        } finally { guard.finishWrite(); }
      }} />}
    {docsOf && <KbDocsDrawer scope={scope} kb={docsOf} onClose={() => setDocsOf(null)} onChanged={refresh} />}
  </Card>;

  async function removeKb(id: number) {
    if (!guard.beginWrite(KB_DELETE)) return;
    try { await deleteAiKb(id); refresh(); } finally { guard.finishWrite(); }
  }
}

interface KbFormValues { name: string; description?: string }

function KbEditModal({ kb, writing, onCancel, onSubmit }: {
  kb: AiKbInfo | null; writing: boolean;
  onCancel: () => void; onSubmit: (values: KbFormValues) => Promise<void>;
}) {
  const [form] = Form.useForm<KbFormValues>();
  return <Modal open title={kb ? '编辑知识库' : '新建知识库'} okText="保存" cancelText="取消" confirmLoading={writing}
    onCancel={onCancel} onOk={() => void form.validateFields().then(onSubmit)} destroyOnHidden>
    <Form form={form} layout="vertical" initialValues={kb ? { name: kb.name, description: kb.description } : {}} preserve={false}>
      <Form.Item name="name" label="名称" rules={[{ required: true, max: 128, message: '名称须为 1..128 字符' }, { whitespace: true, message: '名称不能为空白' }]}>
        <Input maxLength={128} placeholder="如：运维手册" />
      </Form.Item>
      <Form.Item name="description" label="描述" rules={[{ max: 512, message: '描述最多 512 字符' }]}>
        <Input.TextArea rows={2} maxLength={512} placeholder="可选" />
      </Form.Item>
    </Form>
  </Modal>;
}

const DOC_STATUS: Record<AiKbDocInfo['status'], { color: string; text: string }> = {
  pending: { color: 'processing', text: '摄取中' },
  ready: { color: 'success', text: '已就绪' },
  failed: { color: 'error', text: '失败' },
};

function KbDocsDrawer({ scope, kb, onClose, onChanged }: {
  scope: AiWorkspaceScope; kb: AiKbInfo; onClose: () => void; onChanged: () => void;
}) {
  const guard = useAiWorkspaceGuard(scope, KB_LIST);
  const client = useQueryClient();
  const [adding, setAdding] = useState(false);
  const docs = useQuery({ ...kbDocListOptions(scope, kb.id, { page: 1, page_size: 20 }), enabled: guard.can(KB_LIST) });

  const refresh = () => {
    if (!guard.isCurrent()) return;
    void client.invalidateQueries({ queryKey: aiWorkspaceKeys.kbDocs(scope, kb.id) });
    onChanged();
  };

  return <Drawer open title={`文档管理：${kb.name}`} size={720} onClose={onClose}
    extra={guard.can(KB_EDIT) && <Button type="primary" icon={<PlusOutlined />} onClick={() => setAdding(true)}>添加文档</Button>}>
    <QueryState isPending={docs.isPending} error={docs.error} onRetry={() => docs.refetch()} />
    {!docs.isPending && !docs.error && !(docs.data?.list.length) && <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无文档，添加后即可在 AI 对话中检索" />}
    {!!docs.data?.list.length && <Table<AiKbDocInfo> rowKey="id" size="small" pagination={false}
      dataSource={docs.data.list}
      columns={[
        { title: '标题', dataIndex: 'title', ellipsis: true },
        { title: '状态', dataIndex: 'status', width: 90, render: (v: AiKbDocInfo['status']) => <Tag color={DOC_STATUS[v].color}>{DOC_STATUS[v].text}</Tag> },
        { title: '分块', dataIndex: 'chunk_count', width: 70 },
        { title: '错误', dataIndex: 'error', width: 160, ellipsis: true, render: (v: string) => v || '—' },
        { title: '操作', width: 170, render: (_, doc) => <Space>
          {guard.can(KB_EDIT) && doc.status !== 'ready' && <Button type="link" icon={<RedoOutlined />}
            onClick={() => void retryDoc(doc.id)}>重试</Button>}
          {guard.can(KB_EDIT) && <Popconfirm title="删除该文档及其分块索引？" onConfirm={() => void removeDoc(doc.id)}><Button type="link" danger>删除</Button></Popconfirm>}
        </Space> },
      ]} />}
    {adding && <DocAddModal writing={guard.writing} onCancel={() => setAdding(false)}
      onSubmit={async values => {
        if (!guard.beginWrite(KB_EDIT)) return;
        try { await createAiKbDoc(kb.id, values); setAdding(false); refresh(); } finally { guard.finishWrite(); }
      }} />}
  </Drawer>;

  async function retryDoc(docId: number) {
    if (!guard.beginWrite(KB_EDIT)) return;
    try { await retryAiKbDoc(kb.id, docId); refresh(); } finally { guard.finishWrite(); }
  }
  async function removeDoc(docId: number) {
    if (!guard.beginWrite(KB_EDIT)) return;
    try { await deleteAiKbDoc(kb.id, docId); refresh(); } finally { guard.finishWrite(); }
  }
}

function DocAddModal({ writing, onCancel, onSubmit }: {
  writing: boolean; onCancel: () => void; onSubmit: (values: { title: string; content: string }) => Promise<void>;
}) {
  const [form] = Form.useForm<{ title: string; content: string }>();
  return <Modal open title="添加文档" okText="保存并摄取" cancelText="取消" confirmLoading={writing} width={640}
    onCancel={onCancel} onOk={() => void form.validateFields().then(onSubmit)} destroyOnHidden>
    <Form form={form} layout="vertical" preserve={false}>
      <Form.Item name="title" label="标题" rules={[{ required: true, max: 128, message: '标题须为 1..128 字符' }, { whitespace: true, message: '标题不能为空白' }]}>
        <Input maxLength={128} placeholder="如：发布流程规范" />
      </Form.Item>
      <Form.Item name="content" label="正文（纯文本，≤64KB，保存后同步完成向量化）"
        rules={[{ required: true, message: '正文不能为空' }, { max: 65536, message: '正文最多 64KB' }]}>
        <Input.TextArea rows={10} placeholder="粘贴资料原文；检索片段将作为不可信资料提供给模型" />
      </Form.Item>
    </Form>
  </Modal>;
}
