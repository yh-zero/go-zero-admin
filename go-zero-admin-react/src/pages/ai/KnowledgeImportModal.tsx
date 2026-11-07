import { useState } from 'react';
import { Alert, Button, Modal, Space, Table, Tag, Typography } from 'antd';
import { createAiKbDoc } from '../../api/ai';
import type { AiCreateKbDocReq, AiKbDocInfo } from '../../api/types';
import { KNOWLEDGE_IMPORT_MAX_FILES, readKnowledgeFile, runKnowledgeImport } from './knowledgeImport';
import type { KnowledgeImportState } from './knowledgeImport';
import styles from './knowledgeDocuments.module.css';

interface ImportRow { id: string; filename: string; values?: AiCreateKbDocReq; state: KnowledgeImportState | 'invalid'; error?: string; doc?: AiKbDocInfo }
const LABELS = { waiting: '等待导入', submitting: '保存并处理', ready: '已就绪', failed: '处理失败', pending: '处理中', unknown: '结果待确认', invalid: '文件不符合要求' };

export function KnowledgeImportModal({ kbId, writing, beginWrite, finishWrite, isCurrent, onClose, onChanged, onPreview }: {
  kbId: number; writing: boolean; beginWrite: () => boolean; finishWrite: () => void; isCurrent: () => boolean;
  onClose: () => void; onChanged: () => void; onPreview: (docId: number) => void;
}) {
  const [rows, setRows] = useState<ImportRow[]>([]);
  const [reading, setReading] = useState(false);
  const [error, setError] = useState('');
  const active = writing || reading;
  const waiting = rows.filter(row => row.state === 'waiting' && row.values);

  async function selectFiles(files: File[]) {
    if (active) return;
    if (files.length > KNOWLEDGE_IMPORT_MAX_FILES) { setError('每批最多20个文件，请重新选择'); return; }
    setReading(true); setError('');
    try {
      const next = await Promise.all(files.map(async file => {
        const id = crypto.randomUUID();
        try { return { id, filename: file.name, values: await readKnowledgeFile(file), state: 'waiting' as const }; }
        catch (cause) { return { id, filename: file.name, state: 'invalid' as const, error: cause instanceof Error ? cause.message : '文件读取失败' }; }
      }));
      if (isCurrent()) setRows(next);
    } finally { if (isCurrent()) setReading(false); }
  }
  async function submit() {
    if (!waiting.length || !beginWrite()) return;
    try {
      await runKnowledgeImport(waiting.map(row => ({ id: row.id, values: row.values! })), values => createAiKbDoc(kbId, values), change => {
        if (isCurrent()) setRows(current => current.map(row => row.id === change.id ? { ...row, ...change } : row));
      }, isCurrent);
    } finally { finishWrite(); if (isCurrent()) onChanged(); }
  }

  return <Modal open title="批量导入资料" width={800} footer={null} onCancel={() => { if (!active) onClose(); }}
    closable={!active} keyboard={!active} mask={{ closable: !active }} destroyOnHidden>
    <Space orientation="vertical" className={styles.fullWidth} size="middle">
      <Typography.Paragraph>选择 UTF-8 编码的 TXT / Markdown 文件。每批最多20个，每个最多64 KiB，同时处理2个文件。</Typography.Paragraph>
      <input className={styles.filePicker} aria-label="选择知识库资料文件" type="file" accept=".txt,.md,.markdown" multiple disabled={active || rows.some(row => row.state !== 'waiting' && row.state !== 'invalid')}
        onChange={event => { const files = Array.from(event.target.files ?? []); void selectFiles(files); event.target.value = ''; }} />
      {error && <Alert type="error" showIcon title={error} />}
      {rows.some(row => row.state === 'unknown') && <Alert type="warning" showIcon title="部分文件的结果尚未确认" description="这些文件不会自动重传。请先到文档列表核对标题和内容，避免重复保存。" />}
      <Table<ImportRow> rowKey="id" size="small" pagination={false} dataSource={rows} scroll={{ x: 650, y: 320 }} columns={[
        { title: '文件', dataIndex: 'filename', ellipsis: true },
        { title: '状态', width: 140, render: (_, row) => <Tag color={row.state === 'ready' ? 'success' : row.state === 'failed' || row.state === 'invalid' ? 'error' : 'processing'}>{LABELS[row.state]}</Tag> },
        { title: '说明', render: (_, row) => row.error || row.doc?.error || (row.doc ? `文档 #${row.doc.id}` : '—') },
        { title: '操作', width: 100, render: (_, row) => row.doc && <Button type="link" disabled={active} onClick={() => onPreview(row.doc!.id)}>查看文档</Button> },
      ]} />
      <Space><Button type="primary" disabled={!waiting.length || active || !!error} loading={writing} onClick={() => void submit()}>开始导入{waiting.length ? `（${waiting.length}个）` : ''}</Button>
        <Button disabled={active} onClick={onClose}>完成并关闭</Button></Space>
    </Space>
  </Modal>;
}
