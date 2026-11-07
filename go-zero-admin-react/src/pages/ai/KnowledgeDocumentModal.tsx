import { useState } from 'react';
import { Alert, Button, Descriptions, Form, Modal, Popconfirm, Space, Typography } from 'antd';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type { AiCreateKbDocReq, AiKbDocInfo, AiUpdateKbDocReq } from '../../api/types';
import { fetchAiKbDoc, updateAiKbDoc } from '../../api/ai';
import { QueryState } from '../../components/QueryState';
import { knowledgeDocumentKeys, knowledgeDocumentOptions } from './knowledgeDocumentQueries';
import type { AiWorkspaceScope } from './workspaceQueries';
import { DocumentFeedback } from './KnowledgeDocumentStatus';
import { KnowledgeDocumentFields } from './KnowledgeDocumentFields';
import { documentWriteFailure } from './knowledgeDocuments';
import styles from './knowledgeDocuments.module.css';

interface Props {
  scope: AiWorkspaceScope; kbId: number; docId: number; canEdit: boolean; writing: boolean;
  isCurrent: () => boolean; beginWrite: () => boolean; finishWrite: () => void;
  onClose: () => void; onChanged: (doc: AiKbDocInfo) => void;
}

export function KnowledgeDocumentModal({ scope, kbId, docId, canEdit, writing, isCurrent, beginWrite, finishWrite, onClose, onChanged }: Props) {
  const client = useQueryClient();
  const detail = useQuery({ ...knowledgeDocumentOptions(scope, kbId, docId), enabled: isCurrent() });
  const metadata = useQuery({ ...knowledgeDocumentOptions(scope, kbId, docId, true), enabled: isCurrent() });
  const update = useMutation({ retry: false, mutationFn: (values: AiUpdateKbDocReq) => updateAiKbDoc(kbId, docId, values) });
  const [editing, setEditing] = useState<{ values: AiCreateKbDocReq; version: number } | null>(null);
  const [uncertain, setUncertain] = useState(false);
  const [attempt, setAttempt] = useState<AiUpdateKbDocReq | null>(null);
  const [saveError, setSaveError] = useState('');
  const doc = metadata.data?.doc && detail.data && metadata.data.doc.version >= detail.data.doc.version ? metadata.data.doc : detail.data?.doc;

  async function refresh() {
    try {
      const latest = await fetchAiKbDoc(kbId, docId);
      if (!isCurrent()) return;
      client.setQueryData(knowledgeDocumentKeys.detail(scope, kbId, docId, false), latest);
      client.setQueryData(knowledgeDocumentKeys.detail(scope, kbId, docId, true), { doc: latest.doc, content: '' });
      if (uncertain && attempt && latest.doc.version > attempt.expected_version && latest.doc.title === attempt.title.trim() && latest.content === attempt.content) {
        setUncertain(false); setEditing(null); onChanged(latest.doc);
      }
    } catch { /* 统一请求层反馈，保留草稿和未知结果。 */ }
  }

  async function save(values: AiCreateKbDocReq) {
    if (!editing || uncertain || !beginWrite()) return;
    const data = { ...values, expected_version: editing.version };
    setAttempt(data); setSaveError('');
    try {
      await client.cancelQueries({ queryKey: knowledgeDocumentKeys.all(scope, kbId) });
      if (!isCurrent()) return;
      const result = await update.mutateAsync(data);
      if (!isCurrent()) return;
      setEditing(null); setUncertain(false); onChanged(result.doc);
      await client.invalidateQueries({ queryKey: knowledgeDocumentKeys.detail(scope, kbId, docId, false) });
    } catch (cause) {
      if (isCurrent()) { const failure = documentWriteFailure(cause); setUncertain(failure.uncertain); setSaveError(failure.message); }
    }
    finally { finishWrite(); }
  }

  return <Modal open title={editing ? '编辑原文并重新处理' : '文档原文'} width={800} footer={null}
    closable={!writing} keyboard={!writing} mask={{ closable: !writing }} onCancel={() => { if (!writing) onClose(); }} destroyOnHidden>
    <QueryState isPending={detail.isPending} error={detail.error} onRetry={refresh} />
    {detail.data && doc && <Space orientation="vertical" className={styles.fullWidth} size="middle">
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
          }}><Button disabled={writing}>使用最新原文重新编辑</Button></Popconfirm>}
          {uncertain && <Popconfirm title="将保留输入并解除锁定。旧版本再次保存会被拒绝，请先核对最新原文。" onConfirm={() => setUncertain(false)}><Button disabled={writing}>继续编辑草稿</Button></Popconfirm>}
          <Button disabled={writing} onClick={() => { setEditing(null); setUncertain(false); }}>取消编辑</Button></Space>
      </> : <>
        <Typography.Paragraph copyable={{ text: detail.data.content }}>原文内容</Typography.Paragraph>
        <pre className={styles.documentBody}>{detail.data.content}</pre>
        <Space wrap><Button loading={detail.isFetching} onClick={() => void refresh()}>刷新原文</Button>
          {canEdit && <Button type="primary" disabled={writing || doc.version !== detail.data.doc.version} onClick={() => {
            setUncertain(false); setSaveError(''); setEditing({ values: { title: detail.data.doc.title, content: detail.data.content }, version: detail.data.doc.version });
          }}>编辑原文</Button>}</Space>
      </>}
    </Space>}
  </Modal>;
}

function DocumentEditForm({ initialValues, writing, onSubmit }: { initialValues: AiCreateKbDocReq; writing: boolean; onSubmit: (values: AiCreateKbDocReq) => Promise<void> }) {
  const [form] = Form.useForm<AiCreateKbDocReq>();
  const content = Form.useWatch('content', form) ?? initialValues.content;
  return <Form form={form} initialValues={initialValues} layout="vertical" disabled={writing} onFinish={onSubmit} preserve={false}>
    <KnowledgeDocumentFields content={content} />
    <Button type="primary" htmlType="submit" loading={writing} disabled={writing}>保存并重新处理</Button>
  </Form>;
}
