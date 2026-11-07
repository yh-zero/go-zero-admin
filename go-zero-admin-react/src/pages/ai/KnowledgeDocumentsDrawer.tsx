import { useEffect, useState } from 'react';
import { Alert, Button, Drawer, Empty, Form, Modal, Popconfirm, Space, Table, Tag, Typography } from 'antd';
import { FileTextOutlined, PlusOutlined, RedoOutlined, ReloadOutlined, UploadOutlined } from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type { AiCreateKbDocReq, AiKbDocInfo, AiKbInfo, PageResult } from '../../api/types';
import { deleteAiKbDoc } from '../../api/ai';
import { QueryState } from '../../components/QueryState';
import { useAiWorkspaceGuard } from './useAiWorkspaceGuard';
import { aiWorkspaceKeys, kbDocListOptions } from './workspaceQueries';
import type { AiWorkspaceScope } from './workspaceQueries';
import { canRetryDocument, documentStageLabel, documentWriteFailure, documentWriteOptions } from './knowledgeDocuments';
import { knowledgeDocumentKeys, knowledgeDocumentOptions } from './knowledgeDocumentQueries';
import { KnowledgeDocumentModal } from './KnowledgeDocumentModal';
import { KnowledgeImportModal } from './KnowledgeImportModal';
import { KnowledgeDocumentFields } from './KnowledgeDocumentFields';
import { DocumentFeedback } from './KnowledgeDocumentStatus';
import styles from './knowledgeDocuments.module.css';

const KB_LIST = 'sys:ai:kb:list';
const KB_EDIT = 'sys:ai:kb:edit';
const STATUS = { pending: 'processing', ready: 'success', failed: 'error' } as const;

export function KnowledgeDocumentsDrawer({ scope, kb, onClose, onChanged }: {
  scope: AiWorkspaceScope; kb: AiKbInfo; onClose: () => void; onChanged: () => void;
}) {
  const guard = useAiWorkspaceGuard(scope, KB_LIST);
  const { isCurrent } = guard;
  const client = useQueryClient();
  const [adding, setAdding] = useState(false);
  const [importing, setImporting] = useState(false);
  const [previewId, setPreviewId] = useState<number | null>(null);
  const [addDocId, setAddDocId] = useState<number | null>(null);
  const [trackedId, setTrackedId] = useState<number | null>(null);
  const [unknown, setUnknown] = useState(false);
  const [addError, setAddError] = useState('');
  const [draft, setDraft] = useState<AiCreateKbDocReq>();
  const [rowUnknown, setRowUnknown] = useState(false);
  const [busyDocId, setBusyDocId] = useState<number | null>(null);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const listOptions = kbDocListOptions(scope, kb.id, { page, page_size: pageSize });
  const docs = useQuery({ ...listOptions, enabled: guard.can(KB_LIST),
    refetchInterval: query => guard.writing || query.state.data?.list.some(doc => doc.status === 'pending') ? 3000 : false });
  const tracked = useQuery({ ...knowledgeDocumentOptions(scope, kb.id, trackedId ?? 0, true), enabled: !!trackedId && guard.can(KB_LIST) });
  const addDoc = useQuery({ ...knowledgeDocumentOptions(scope, kb.id, addDocId ?? 0, true), enabled: !!addDocId && guard.can(KB_LIST) });
  const write = useMutation(documentWriteOptions(kb.id));
  const remove = useMutation({ mutationFn: (id: number) => deleteAiKbDoc(kb.id, id) });

  useEffect(() => {
    // oxlint-disable-next-line react/set-state-in-effect -- 删除后以服务端总数修正分页。
    if (docs.data && isCurrent()) setPage(current => Math.min(current, Math.max(1, Math.ceil(docs.data.total / pageSize))));
  }, [docs.data, pageSize, isCurrent]);
  useEffect(() => {
    if (!isCurrent() || guard.writing || !addDoc.data || addDoc.dataUpdatedAt <= write.submittedAt) return;
    // oxlint-disable-next-line react/set-state-in-effect -- 同ID新鲜查询确认当前状态，失败或处理中也不再是未知结果。
    setUnknown(false);
    if (addDoc.data.doc.status === 'ready') { setAdding(false); setDraft(undefined); }
  }, [addDoc.data, addDoc.dataUpdatedAt, isCurrent, guard.writing, write.submittedAt]);
  useEffect(() => {
    // oxlint-disable-next-line react/set-state-in-effect -- 网络未知后仅用新鲜的同ID读取确认状态。
    if (isCurrent() && !guard.writing && tracked.data && !tracked.isFetching && tracked.dataUpdatedAt > write.submittedAt) setRowUnknown(false);
  }, [tracked.data, tracked.dataUpdatedAt, tracked.isFetching, isCurrent, guard.writing, write.submittedAt]);

  function remember(doc: AiKbDocInfo) {
    client.setQueryData(knowledgeDocumentKeys.detail(scope, kb.id, doc.id, true), { doc, content: '' });
    client.setQueriesData<PageResult<AiKbDocInfo>>({ queryKey: aiWorkspaceKeys.kbDocs(scope, kb.id) }, value => value
      ? { ...value, list: value.list.map(item => item.id === doc.id ? doc : item) } : value);
    setTrackedId(doc.id);
  }
  function refresh() {
    if (!guard.isCurrent()) return;
    void client.invalidateQueries({ queryKey: aiWorkspaceKeys.kbDocs(scope, kb.id) });
    void client.invalidateQueries({ queryKey: knowledgeDocumentKeys.all(scope, kb.id) });
    onChanged();
  }

  return <Drawer className={styles.documentDrawer} open title={`文档管理：${kb.name}`} size={900} onClose={() => { if (!guard.writing) onClose(); }}
    closable={!guard.writing} keyboard={!guard.writing} maskClosable={!guard.writing}
    extra={<Space wrap><Button icon={<ReloadOutlined />} loading={docs.isFetching || tracked.isFetching} onClick={refresh}>刷新状态</Button>
      {guard.can(KB_EDIT) && <><Button icon={<UploadOutlined />} disabled={guard.writing || adding} onClick={() => setImporting(true)}>批量导入</Button>
        <Button type="primary" icon={<PlusOutlined />} disabled={guard.writing || adding} onClick={() => {
          if (addDoc.data?.doc.status === 'ready') { setAddDocId(null); setDraft(undefined); setUnknown(false); }
          setAdding(true);
        }}>{draft || unknown ? '继续添加' : '添加文档'}</Button></>}</Space>}>
    <Typography.Paragraph type="secondary">支持粘贴原文或批量导入 UTF-8 的 TXT / Markdown 文件，每篇最多64 KiB。处理状态每3秒刷新，失败时重试已有文档。</Typography.Paragraph>
    <QueryState isPending={docs.isPending} error={docs.error} onRetry={() => docs.refetch()} />
    {trackedId && tracked.error && <Alert type="warning" showIcon title="暂未能读取所提交文档的状态" description="已保留文档ID，请刷新状态。文档可能已经删除，不会自动重新创建。" />}
    {rowUnknown && <Alert type="warning" showIcon title="重试结果尚未确认" description="请刷新同一文档的状态，网络错误不代表服务端撤销。" />}
    {tracked.data && <DocumentFeedback doc={tracked.data.doc} />}
    {!docs.isPending && !docs.error && !docs.data?.total && <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无文档，添加后可用于知识库对话与分析" />}
    {!!docs.data?.total && <Table<AiKbDocInfo> rowKey="id" size="small" scroll={{ x: 680 }}
      pagination={{ current: page, pageSize, total: docs.data.total, showSizeChanger: true, disabled: guard.writing,
        onChange: (next, size) => { setPage(size === pageSize ? next : 1); setPageSize(size); } }} dataSource={docs.data.list}
      columns={[
        { title: '标题', dataIndex: 'title', ellipsis: true },
        { title: '处理阶段', width: 180, render: (_, doc) => <Space orientation="vertical" size={0}><Tag color={STATUS[doc.status]}>{documentStageLabel(doc)}</Tag>
          {doc.status === 'failed' && <Typography.Text type="danger">{doc.error || '请重试已有文档'}</Typography.Text>}</Space> },
        { title: '版本', dataIndex: 'version', width: 65, render: value => `v${value}` },
        { title: '分块', dataIndex: 'chunk_count', width: 60 },
        { title: '操作', width: 245, render: (_, doc) => <Space>
          <Button type="link" icon={<FileTextOutlined />} disabled={guard.writing} onClick={() => setPreviewId(doc.id)}>原文</Button>
          {guard.can(KB_EDIT) && doc.status !== 'ready' && <Button type="link" icon={<RedoOutlined />} disabled={guard.writing || !canRetryDocument(doc)}
            loading={busyDocId === doc.id && write.isPending} title="处理中需等待120秒保护期结束" onClick={() => void retry(doc)}>重试</Button>}
          {guard.can(KB_EDIT) && <Popconfirm title="删除该文档？" disabled={guard.writing} onConfirm={() => removeDoc(doc.id)}><Button type="link" danger disabled={guard.writing} loading={busyDocId === doc.id && remove.isPending}>删除</Button></Popconfirm>}
        </Space> },
      ]} />}
    {adding && <DocumentAddModal writing={guard.writing} doc={addDoc.data?.doc} unknown={unknown} error={addError} initialValues={draft}
      onCancel={() => { if (!guard.writing) setAdding(false); }} onRefresh={refresh}
      onEdit={() => { setAddDocId(null); setUnknown(false); }} onSubmit={submit} />}
    {previewId && <KnowledgeDocumentModal scope={scope} kbId={kb.id} docId={previewId} onClose={() => setPreviewId(null)}
      canEdit={guard.can(KB_EDIT)} beginWrite={() => guard.beginWrite(KB_EDIT)} finishWrite={guard.finishWrite} writing={guard.writing}
      isCurrent={guard.isCurrent} onChanged={doc => { remember(doc); refresh(); }} />}
    {importing && <KnowledgeImportModal kbId={kb.id} isCurrent={guard.isCurrent} writing={guard.writing}
      beginWrite={() => guard.beginWrite(KB_EDIT)} finishWrite={guard.finishWrite}
      onClose={() => setImporting(false)} onChanged={refresh} onPreview={id => { setImporting(false); setPreviewId(id); }} />}
  </Drawer>;

  async function submit(values: AiCreateKbDocReq) {
    if (!guard.beginWrite(KB_EDIT)) return;
    setDraft(values); setAddError('');
    try {
      await client.cancelQueries({ queryKey: aiWorkspaceKeys.kbDocs(scope, kb.id) });
      await client.cancelQueries({ queryKey: knowledgeDocumentKeys.all(scope, kb.id) });
      if (!guard.isCurrent()) return;
      const { doc } = await write.mutateAsync({ values, doc: addDoc.data?.doc });
      if (!guard.isCurrent()) return;
      remember(doc); setAddDocId(doc.id); setUnknown(false);
      if (doc.status === 'ready') { setAdding(false); setDraft(undefined); }
      refresh();
    } catch (cause) {
      if (guard.isCurrent()) { const failure = documentWriteFailure(cause); setUnknown(failure.uncertain); setAddError(failure.message); refresh(); }
    }
    finally { guard.finishWrite(); }
  }
  async function retry(doc: AiKbDocInfo) {
    if (!canRetryDocument(doc) || !guard.beginWrite(KB_EDIT)) return;
    setBusyDocId(doc.id); setTrackedId(doc.id);
    await client.cancelQueries({ queryKey: knowledgeDocumentKeys.detail(scope, kb.id, doc.id, true) });
    try {
      if (!guard.isCurrent()) return;
      const result = await write.mutateAsync({ values: { title: '', content: '' }, doc });
      if (guard.isCurrent()) { remember(result.doc); setRowUnknown(false); refresh(); }
    } catch { if (guard.isCurrent()) { setRowUnknown(true); refresh(); } }
    finally { if (guard.isCurrent()) setBusyDocId(null); guard.finishWrite(); }
  }
  async function removeDoc(id: number) {
    if (!guard.beginWrite(KB_EDIT)) return;
    setBusyDocId(id);
    try {
      await remove.mutateAsync(id);
      if (guard.isCurrent()) {
        if (trackedId === id) setTrackedId(null);
        if (addDocId === id) setAddDocId(null);
        client.removeQueries({ queryKey: [...knowledgeDocumentKeys.all(scope, kb.id), id] }); refresh();
      }
    } catch { /* 统一请求层反馈。 */ }
    finally { if (guard.isCurrent()) setBusyDocId(null); guard.finishWrite(); }
  }
}

function DocumentAddModal({ writing, doc, unknown, error, initialValues, onCancel, onSubmit, onEdit, onRefresh }: {
  writing: boolean; doc?: AiKbDocInfo; unknown: boolean; error: string; initialValues?: AiCreateKbDocReq;
  onCancel: () => void; onSubmit: (values: AiCreateKbDocReq) => Promise<void>; onEdit: () => void; onRefresh: () => void;
}) {
  const [form] = Form.useForm<AiCreateKbDocReq>();
  const content = Form.useWatch('content', form) ?? '';
  return <Modal open title="添加文档" width={680} okText={doc ? '重试已有文档' : '保存并处理'} confirmLoading={writing}
    closable={!writing} keyboard={!writing} mask={{ closable: !writing }} cancelButtonProps={{ disabled: writing }}
    okButtonProps={{ disabled: unknown || (!!doc && !canRetryDocument(doc)) }} onCancel={onCancel}
    onOk={() => { void form.validateFields().then(onSubmit).catch(() => undefined); }} destroyOnHidden>
    {doc && <DocumentFeedback doc={doc} />}
    {unknown && <Alert type="warning" showIcon title="结果尚未确认，草稿已保留" description="请先刷新列表确认，避免重复提交。网络错误不代表服务端撤销。" />}
    {error && !unknown && <Alert type="error" showIcon title="本次保存未完成" description={error} />}
    {(doc || unknown) && <Space><Button disabled={writing} onClick={onRefresh}>刷新状态</Button>
      <Popconfirm title="继续编辑将作为新资料提交，已有文档仍会保留。请先确认处理结果。" onConfirm={onEdit}><Button disabled={writing}>编辑新资料</Button></Popconfirm></Space>}
    <Form form={form} layout="vertical" disabled={writing || !!doc || unknown} initialValues={initialValues} preserve={false}>
      <KnowledgeDocumentFields content={content} />
    </Form>
  </Modal>;
}
