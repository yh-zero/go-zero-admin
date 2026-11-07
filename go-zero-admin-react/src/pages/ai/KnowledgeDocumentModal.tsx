import { useEffect, useState } from 'react';
import { Alert, Button, Descriptions, Form, Modal, Popconfirm, Space, Typography } from 'antd';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type { AiCreateKbDocReq, AiKbDocInfo, AiUpdateKbDocReq } from '../../api/types';
import { fetchAiKbDoc, updateAiKbDoc } from '../../api/ai';
import { QueryState } from '../../components/QueryState';
import { useAuthStore } from '../../stores/auth';
import { knowledgeDocumentKeys, knowledgeDocumentMetadataOptions, knowledgeDocumentOptions } from './knowledgeDocumentQueries';
import type { AiWorkspaceScope } from './workspaceQueries';
import { DocumentFeedback } from './KnowledgeDocumentStatus';
import { KnowledgeDocumentFields } from './KnowledgeDocumentFields';
import { documentWriteFailure } from './knowledgeDocuments';
import { readKnowledgeImportAttempt, subscribeKnowledgeImportAttempt, updateKnowledgeImportAttempt, writeKnowledgeImportAttempt } from './knowledgeImport';
import { kbImportOptions } from './workspaceQueries';
import styles from './knowledgeDocuments.module.css';

interface Props {
  scope: AiWorkspaceScope; kbId: number; docId: number; canEdit: boolean; writing: boolean;
  isCurrent: () => boolean; beginWrite: () => boolean; finishWrite: () => void;
  onClose: () => void; onChanged: (doc: AiKbDocInfo) => void;
  metadata?: AiKbDocInfo; metadataUpdatedAt?: number;
}
type DocumentValues = Pick<AiCreateKbDocReq, 'title' | 'content'>;

export function KnowledgeDocumentModal({ scope, kbId, docId, canEdit, writing, isCurrent, beginWrite, finishWrite, onClose, onChanged, metadata: parentMetadata, metadataUpdatedAt = 0 }: Props) {
  const client = useQueryClient();
  const departmentId = useAuthStore(state => state.selectedDeptId);
  const receiptKey = `ai-knowledge-update:${JSON.stringify([scope[0], departmentId, kbId, docId])}`;
  const [original, setOriginal] = useState(() => { try { return readKnowledgeImportAttempt(sessionStorage, receiptKey); } catch { return null; } });
  const detail = useQuery({ ...knowledgeDocumentOptions(scope, kbId, docId), enabled: isCurrent() });
  const metadataOptions = knowledgeDocumentMetadataOptions(scope, kbId, docId, detail.data?.doc, parentMetadata);
  const metadata = useQuery({ ...metadataOptions, enabled: metadataOptions.enabled && isCurrent() });
  const update = useMutation({ retry: false, mutationFn: (values: AiUpdateKbDocReq) => updateAiKbDoc(kbId, docId, values) });
  const [editing, setEditing] = useState<{ values: DocumentValues; version: number } | null>(null);
  const [uncertain, setUncertain] = useState(!!original);
  const [attempt, setAttempt] = useState<AiUpdateKbDocReq | null>(null);
  const [saveError, setSaveError] = useState('');
  const receipt = useQuery({ ...kbImportOptions(scope, kbId, original?.batchId ?? ''), enabled: !!original && uncertain && !writing && isCurrent() });
  useEffect(() => subscribeKnowledgeImportAttempt(sessionStorage, receiptKey, latest => {
    if (!isCurrent()) return;
    setOriginal(latest); setUncertain(!!latest);
  }), [receiptKey, isCurrent]);
  const full = detail.data?.doc;
  const latest = parentMetadata && (parentMetadata.version > (full?.version ?? 0) || metadataUpdatedAt > detail.dataUpdatedAt) ? parentMetadata
    : metadata.dataUpdatedAt > detail.dataUpdatedAt ? metadata.data?.doc : undefined;
  const doc = latest && full && latest.version >= full.version ? latest : full;
  useEffect(() => {
    if (!original || !receipt.data || !isCurrent()) return;
    const jobs = receipt.data.list;
    if (!original.requestIds.every(id => jobs.some(job => job.request_id === id))) return;
    try { if (updateKnowledgeImportAttempt(sessionStorage, receiptKey, original.batchId, () => null)) return; }
    catch { /* 共享内存已记录确认结果，存储恢复后仍只读核对。 */ }
    // oxlint-disable-next-line react/set-state-in-effect -- 原回执确认接收；恢复时只读查询，不把新表单当原请求重发。
    setUncertain(false); setOriginal(null); setEditing(null); setSaveError('');
    void client.invalidateQueries({ queryKey: knowledgeDocumentKeys.all(scope, kbId) });
    const accepted = jobs.find(job => original.requestIds.includes(job.request_id))?.doc;
    if (accepted) onChanged(accepted);
  }, [original, receipt.data, isCurrent, receiptKey, client, scope, kbId, onChanged]);

  async function refresh() {
    try {
      if (original) await receipt.refetch();
      const latest = await fetchAiKbDoc(kbId, docId);
      if (!isCurrent()) return;
      client.setQueryData(knowledgeDocumentKeys.detail(scope, kbId, docId, false), latest);
      client.setQueryData(knowledgeDocumentKeys.detail(scope, kbId, docId, true), { doc: latest.doc, content: '' });
      if (uncertain && !original && attempt && latest.doc.version > attempt.expected_version && latest.doc.title === attempt.title.trim() && latest.content === attempt.content) {
        setUncertain(false); setEditing(null); onChanged(latest.doc);
      }
    } catch { /* 统一请求层反馈，保留草稿和未知结果。 */ }
  }

  async function save(values: DocumentValues) {
    if (!editing || uncertain || original || !beginWrite()) return;
    const data = { ...values, expected_version: editing.version, request_id: crypto.randomUUID() };
    setAttempt(data); setSaveError('');
    const pending = { batchId: data.request_id, requestIds: [data.request_id] };
    try {
      await client.cancelQueries({ queryKey: knowledgeDocumentKeys.all(scope, kbId) });
      if (!isCurrent()) return;
      setOriginal(pending);
      try { writeKnowledgeImportAttempt(sessionStorage, receiptKey, pending); }
      catch { setSaveError(`浏览器暂时无法保存回执，请保留原请求标识：${data.request_id}`); }
      const result = await update.mutateAsync(data);
      try { updateKnowledgeImportAttempt(sessionStorage, receiptKey, data.request_id, () => null); } catch { /* 原回执再次查询不产生新写入。 */ }
      if (!isCurrent()) return;
      setEditing(null); setUncertain(false); setOriginal(null);
      onChanged(result.doc);
      await client.invalidateQueries({ queryKey: knowledgeDocumentKeys.detail(scope, kbId, docId, false) });
    } catch (cause) {
      const failure = documentWriteFailure(cause);
      if (!failure.uncertain) { try { updateKnowledgeImportAttempt(sessionStorage, receiptKey, data.request_id, () => null); } catch { /* 返回时只读核对原请求。 */ } }
      if (isCurrent()) { setUncertain(failure.uncertain); setSaveError(failure.message); if (!failure.uncertain) setOriginal(null); }
    }
    finally { finishWrite(); }
  }

  return <Modal open title={editing ? '编辑原文并重新处理' : '文档原文'} width={800} footer={null}
    closable={!writing} keyboard={!writing} mask={{ closable: !writing }} onCancel={() => { if (!writing) onClose(); }} destroyOnHidden>
    <QueryState isPending={detail.isPending} error={detail.error} onRetry={refresh} />
    {detail.data && doc && <Space orientation="vertical" className={styles.fullWidth} size="middle">
      {uncertain && original && <Alert type="warning" showIcon title="上次原文更新结果待核对" description={`保留原请求标识 ${original.batchId}。这里只查询原批次，不会重新发送原文；确认结果后可读取当前原文与版本，再开始新的修改。`} action={<Button loading={receipt.isFetching} disabled={writing} onClick={() => void refresh()}>核对原批次</Button>} />}
      <DocumentFeedback doc={doc} />
      <Descriptions size="small" column={{ xs: 1, sm: 2 }} items={[
        { key: 'title', label: '标题', children: detail.data.doc.title },
        { key: 'version', label: '正文版本', children: `v${detail.data.doc.version}` },
        { key: 'chunks', label: '分块数', children: doc.chunk_count },
        { key: 'model', label: 'Embedding 模型', children: doc.embedding_model || '—' },
      ]} />
      {doc.version > detail.data.doc.version && <Alert type="warning" showIcon title="文档已有新版本" description="当前展示的是读取时的原文，请刷新后查看最新正文。" />}
      {editing ? <>
        <Alert type="info" showIcon title="保存后会替换当前原文并重新处理" description="重新处理完成前，本篇资料暂停参与检索；旧版本不会继续作为新正文的依据。" />
        {uncertain && <Alert type="warning" showIcon title="尚未确认保存结果，草稿已保留" description="请先刷新核对原文和版本，避免重复重新处理；版本冲突时需重新读取再编辑。" />}
        {saveError && !uncertain && <Alert type="error" showIcon title="本次保存未完成，草稿已保留" description={saveError} />}
        <DocumentEditForm key={editing.version} initialValues={editing.values} writing={writing || uncertain} onSubmit={save} />
        <Space wrap><Button disabled={writing} loading={detail.isFetching} onClick={() => void refresh()}>刷新核对</Button>
          {detail.data.doc.version !== editing.version && <Popconfirm title="将丢弃当前编辑草稿，并载入已核对的最新原文。" onConfirm={() => {
            setEditing({ values: { title: detail.data.doc.title, content: detail.data.content }, version: detail.data.doc.version }); setUncertain(false); setSaveError('');
          }}><Button disabled={writing || !!original}>使用最新原文重新编辑</Button></Popconfirm>}
          {uncertain && !original && <Popconfirm title="将保留输入并解除锁定。旧版本再次保存会被拒绝，请先核对最新原文。" onConfirm={() => setUncertain(false)}><Button disabled={writing}>继续编辑草稿</Button></Popconfirm>}
          <Button disabled={writing} onClick={() => { setEditing(null); setUncertain(!!original); }}>取消编辑</Button></Space>
      </> : <>
        <Typography.Paragraph copyable={{ text: detail.data.content }}>原文内容</Typography.Paragraph>
        <pre className={styles.documentBody}>{detail.data.content}</pre>
        <Space wrap><Button loading={detail.isFetching} onClick={() => void refresh()}>刷新原文</Button>
          {canEdit && <Button type="primary" disabled={writing || uncertain || !!original || doc.version !== detail.data.doc.version} onClick={() => {
            setUncertain(false); setSaveError(''); setEditing({ values: { title: detail.data.doc.title, content: detail.data.content }, version: detail.data.doc.version });
          }}>编辑原文</Button>}</Space>
      </>}
    </Space>}
  </Modal>;
}

function DocumentEditForm({ initialValues, writing, onSubmit }: { initialValues: DocumentValues; writing: boolean; onSubmit: (values: DocumentValues) => Promise<void> }) {
  const [form] = Form.useForm<DocumentValues>();
  const content = Form.useWatch('content', form) ?? initialValues.content;
  return <Form form={form} initialValues={initialValues} layout="vertical" disabled={writing} onFinish={onSubmit} preserve={false}>
    <KnowledgeDocumentFields content={content} />
    <Button type="primary" htmlType="submit" loading={writing} disabled={writing}>保存并重新处理</Button>
  </Form>;
}
