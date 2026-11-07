import { useEffect, useState } from 'react';
import { Alert, Button, Form, Modal, Pagination, Select, Space, Table, Tag, Typography } from 'antd';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { createAiKbDoc, stopAiKbImport } from '../../api/ai';
import type { AiCreateKbDocReq, AiKbIngestJobInfo } from '../../api/types';
import { QueryState } from '../../components/QueryState';
import { formatTime } from '../../utils/format';
import { useAuthStore } from '../../stores/auth';
import { KNOWLEDGE_IMPORT_MAX_FILES, advanceKnowledgeImportAttempt, knowledgeBatchPollInterval, prepareKnowledgeBatch, readKnowledgeFile, readKnowledgeImportAttempt, runKnowledgeImport, subscribeKnowledgeImportAttempt, updateKnowledgeImportAttempt, writeKnowledgeImportAttempt } from './knowledgeImport';
import type { KnowledgeImportAttempt, KnowledgeImportState } from './knowledgeImport';
import { aiWorkspaceKeys, kbImportListOptions, kbImportOptions } from './workspaceQueries';
import type { AiWorkspaceScope } from './workspaceQueries';
import { KnowledgeDocumentFields } from './KnowledgeDocumentFields';
import styles from './workspace.module.css';

type DocumentValues = Pick<AiCreateKbDocReq, 'title' | 'content'>;
interface ImportRow { id: string; filename: string; values?: DocumentValues; state: KnowledgeImportState | 'invalid'; error?: string }
const labels = { waiting: '等待接收', submitting: '提交资料', ready: '可使用', failed: '处理失败', pending: '已接收，后台处理', unknown: '结果待核对', invalid: '文件不符合要求' };
const jobLabels = { queued: '等待后台处理', running: '后台处理中', succeeded: '处理完成', failed: '处理失败', cancelled: '已停止', reused: '原文已存在' };

export function KnowledgeImportModal({ scope, kbId, initialMode = 'files', initialBatchId, writing, beginWrite, finishWrite, isCurrent, onClose, onChanged, onPreview, canEdit = true }: {
  scope: AiWorkspaceScope; kbId: number; initialMode?: 'files' | 'paste'; initialBatchId?: string;
  writing: boolean; beginWrite: () => boolean; finishWrite: () => void; isCurrent: () => boolean;
  onClose: () => void; onChanged: () => void; onPreview: (docId: number) => void;
  canEdit?: boolean;
}) {
  const departmentId = useAuthStore(state => state.selectedDeptId);
  const storageKey = `ai-knowledge-import:${JSON.stringify([scope[0], departmentId, kbId])}`;
  const pendingKey = `${storageKey}:pending`;
  const [restored] = useState(() => { try { return { attempt: readKnowledgeImportAttempt(sessionStorage, pendingKey), error: '' }; } catch { return { attempt: null, error: '原导入标识无法读取，请先核对已有批次。' }; } });
  const [pending, setPending] = useState<KnowledgeImportAttempt | null>(restored.attempt);
  const [rows, setRows] = useState<ImportRow[]>([]);
  const [reading, setReading] = useState(false);
  const [error, setError] = useState(restored.error);
  const [page, setPage] = useState(1);
  const [batchId, setBatchId] = useState<string | null>(() => {
    if (restored.attempt) return restored.attempt.batchId;
    if (initialBatchId) return initialBatchId;
    try { return sessionStorage.getItem(storageKey); } catch { return null; }
  });
  const [unknown, setUnknown] = useState(!!restored.attempt?.requestIds.length || !!restored.error);
  const [form] = Form.useForm<DocumentValues>();
  const content = Form.useWatch('content', form) ?? '';
  const active = writing || reading;
  const client = useQueryClient();
  const batches = useQuery({ ...kbImportListOptions(scope, kbId, { page, page_size: 10 }), enabled: isCurrent() });
  const batch = useQuery({ ...kbImportOptions(scope, kbId, batchId ?? ''), enabled: !!batchId && !active && isCurrent(),
    refetchInterval: query => knowledgeBatchPollInterval(query.state.data?.batch, !active && isCurrent() && !query.state.error), refetchIntervalInBackground: false });
  const stop = useMutation({ mutationFn: () => stopAiKbImport(kbId, batchId!), retry: false });
  const create = useMutation({ mutationFn: (values: AiCreateKbDocReq) => createAiKbDoc(kbId, values), retry: false });
  const waiting = rows.filter(row => row.state === 'waiting' && row.values);
  useEffect(() => subscribeKnowledgeImportAttempt(sessionStorage, pendingKey, latest => {
    if (!isCurrent()) return;
    setPending(latest); setUnknown(!!latest?.requestIds.length);
  }), [pendingKey, isCurrent]);
  useEffect(() => {
    if (!pending?.requestIds.length || !batch.data || batch.data.batch.id !== pending.batchId || !isCurrent()) return;
    const received = new Set(batch.data.list.map(job => job.request_id));
    if (!pending.requestIds.some(id => received.has(id))) return;
    try {
      const remaining = updateKnowledgeImportAttempt(sessionStorage, pendingKey, pending.batchId, latest => {
        const requestIds = latest.requestIds.filter(id => !received.has(id));
        return requestIds.length === latest.requestIds.length ? latest : { ...latest, requestIds };
      });
      if (remaining && remaining.batchId !== pending.batchId) return;
      // oxlint-disable-next-line react/set-state-in-effect -- 原批次读取仅推进当前仍存在的同批次回执。
      setUnknown(!!remaining?.requestIds.length); setPending(remaining);
    } catch { /* 存储失败时共享内存保留最新决定，订阅继续维持原标识。 */ }
  }, [pending, batch.data, isCurrent, pendingKey]);

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
      if (isCurrent()) { setRows(next); setUnknown(false); }
    } finally { if (isCurrent()) setReading(false); }
  }

  async function submit(values: DocumentValues[]) {
    if (!values.length || unknown || pending?.unsubmittedIds?.length || !beginWrite()) return;
    const prepared = prepareKnowledgeBatch(values);
    const original: KnowledgeImportAttempt = { batchId: prepared.batchId, requestIds: [], unsubmittedIds: prepared.items.map(item => item.id) };
    setPending(original);
    setBatchId(prepared.batchId); setError(''); setUnknown(false);
    const nextRows = prepared.items.map(item => ({ id: item.id, filename: item.values.title, values: item.values, state: 'waiting' as const }));
    setRows(nextRows);
    try { writeKnowledgeImportAttempt(sessionStorage, pendingKey, original); }
    catch { setError(`请保留本次批次标识：${prepared.batchId}。浏览器暂时无法保存返回入口。`); }
    try { sessionStorage.setItem(storageKey, prepared.batchId); }
    catch { setError(`请保留本次批次标识：${prepared.batchId}。浏览器暂时无法保存返回入口。`); }
    try {
      await runKnowledgeImport(prepared.items, values => create.mutateAsync(values), change => {
        if (!isCurrent()) return;
        setRows(current => current.map(row => row.id === change.id ? { ...row, ...change } : row));
      }, isCurrent, change => {
        try { updateKnowledgeImportAttempt(sessionStorage, pendingKey, prepared.batchId, latest => advanceKnowledgeImportAttempt(latest, change)); }
        catch { if (isCurrent()) setError('浏览器无法更新导入回执，请保留原批次与逐项请求标识。'); }
        if (isCurrent()) {
          try { const latest = readKnowledgeImportAttempt(sessionStorage, pendingKey); setPending(latest); setUnknown(!!latest?.requestIds.length); }
          catch { setError('无法读取原导入回执，请先核对原批次。'); }
        }
      });
    } finally {
      finishWrite();
      if (isCurrent()) {
        try { const latest = readKnowledgeImportAttempt(sessionStorage, pendingKey); setPending(latest); setUnknown(!!latest?.requestIds.length); }
        catch { setError('无法读取原导入回执，请先核对原批次。'); }
        void client.invalidateQueries({ queryKey: aiWorkspaceKeys.imports(scope, kbId) });
        void client.invalidateQueries({ queryKey: aiWorkspaceKeys.import(scope, kbId, prepared.batchId) });
        onChanged();
      }
    }
  }
  async function stopWaiting() {
    if (!batchId || !beginWrite()) return;
    try { const result = await stop.mutateAsync(); if (isCurrent()) client.setQueryData(aiWorkspaceKeys.import(scope, kbId, batchId), result); }
    catch { if (isCurrent()) setError('停止结果待核对，请刷新原批次；不会自动重复操作。'); }
    finally { finishWrite(); onChanged(); }
  }
  const serverJobs = new Set(batch.data?.list.map(job => job.request_id) ?? []);
  const unresolved = rows.filter(row => row.state === 'unknown' && !serverJobs.has(row.id));
  return <Modal open title="资料导入与后台批次" width={900} footer={null} onCancel={() => { if (!active) onClose(); }} closable={!active} keyboard={!active} mask={{ closable: !active }} destroyOnHidden>
    <Space orientation="vertical" className={styles.fullWidth} size="middle">
      <Typography.Paragraph>TXT / Markdown，每篇最多64 KiB，每批最多20篇。资料接收后由后台处理，可以关闭并稍后查看批次；停止批次只取消尚未开始的资料。</Typography.Paragraph>
      {canEdit && (initialMode === 'paste' ? <Form form={form} layout="vertical" onFinish={values => void submit([values])} disabled={active || unknown || !!pending?.unsubmittedIds?.length}>
        <KnowledgeDocumentFields content={content} /><Button type="primary" htmlType="submit" loading={writing}>接收并后台处理</Button>
      </Form> : <>
        <input aria-label="选择知识库资料文件" type="file" accept=".txt,.md,.markdown" multiple disabled={active || unknown || !!pending?.unsubmittedIds?.length} onChange={event => { void selectFiles(Array.from(event.target.files ?? [])); event.target.value = ''; }} />
        {!!rows.length && <Table<ImportRow> rowKey="id" size="small" pagination={false} dataSource={rows} columns={[
          { title: '资料', dataIndex: 'filename' }, { title: '接收状态', render: (_, row) => <Tag color={row.state === 'unknown' ? 'warning' : row.state === 'invalid' || row.state === 'failed' ? 'error' : 'processing'}>{labels[row.state]}</Tag> }, { title: '说明', dataIndex: 'error' },
        ]} />}
        <Button type="primary" disabled={!waiting.length || active || unknown || !!pending?.unsubmittedIds?.length} loading={writing} onClick={() => void submit(waiting.map(row => row.values!))}>接收并后台处理{waiting.length ? `（${waiting.length}篇）` : ''}</Button>
      </>)}
      {error && <Alert type="warning" showIcon title={error} />}
      {!!pending?.unsubmittedIds?.length && !active && <Alert type="info" showIcon title={`${pending.unsubmittedIds.length}篇资料尚未发出`} description="离开页面后已停止发送剩余资料。这些项没有提交到后台；可以明确放弃未提交项，再重新选择资料。已发出但结果未知的项会继续保留原标识核对。" action={<Button onClick={() => {
        try {
          const remaining = updateKnowledgeImportAttempt(sessionStorage, pendingKey, pending.batchId, latest => ({ ...latest, unsubmittedIds: [] }));
          setPending(remaining); setUnknown(!!remaining?.requestIds.length);
        } catch { setError('浏览器无法保存放弃状态，原标识仍保留，请先核对。'); }
      }}>放弃未提交项</Button>} />}
      {unknown && <Alert type="warning" showIcon title="部分接收结果尚未确认" description={unresolved.length ? `保留原批次和请求标识。${unresolved.length}篇资料仍需核对，页面不会自动重传。` : '已保留原批次与请求标识，已接收资料以服务端作业状态为准；不会自动重传。'} action={<Button loading={batch.isFetching} onClick={() => void batch.refetch()}>核对原批次</Button>} />}
      <div className={styles.toolbar}><Typography.Title level={5}>查看导入批次</Typography.Title><Select className={styles.search} disabled={active} value={batchId ?? undefined} placeholder="按接收时间选择批次" onChange={setBatchId} options={batches.data?.list.map(item => ({ value: item.id, label: `${formatTime(item.created_at)} · ${item.total}篇 · ${item.queued + item.running ? '处理中' : '已结束'}` }))} /><Button disabled={active} loading={batches.isFetching || batch.isFetching} onClick={() => { void batches.refetch(); if (batchId) void batch.refetch(); }}>刷新状态</Button></div>
      <QueryState error={batches.error} onRetry={() => batches.refetch()} />
      {(batches.data?.total ?? 0) > 10 && <Pagination size="small" current={page} pageSize={10} total={batches.data?.total} onChange={setPage} />}
      {batchId && <>
        <QueryState isPending={batch.isPending && !active} error={active ? undefined : batch.error} onRetry={() => batch.refetch()} />
        <details><summary>批次与原请求标识</summary><p>{batchId}</p>{pending?.requestIds.map(id => <p key={id}>{id}</p>)}{rows.map(row => <p key={row.id}>{row.filename}：{row.id}</p>)}</details>
        {batch.data && <><Space wrap><Tag>总计 {batch.data.batch.total}</Tag><Tag color="processing">等待 {batch.data.batch.queued} · 处理 {batch.data.batch.running}</Tag><Tag color="success">完成 {batch.data.batch.succeeded}</Tag><Tag color="error">失败 {batch.data.batch.failed}</Tag><Tag>原文已存在 {batch.data.batch.reused}</Tag><Tag>停止 {batch.data.batch.cancelled}</Tag></Space>
          <Table<AiKbIngestJobInfo> rowKey="id" size="small" pagination={false} dataSource={batch.data.list} columns={[
            { title: '资料', render: (_, job) => job.doc?.title ?? `资料 #${job.doc_id}` },
            { title: '作业状态', render: (_, job) => <Tag color={job.status === 'failed' ? 'error' : job.status === 'queued' || job.status === 'running' ? 'processing' : 'default'}>{jobLabels[job.status]}</Tag> },
            { title: '说明', render: (_, job) => job.error || (job.status === 'reused' ? `原文当前${job.doc?.status === 'ready' ? '可使用' : job.doc?.status === 'failed' ? '处理失败' : '处理中'}` : '') },
            { title: '操作', render: (_, job) => job.doc && <Button type="link" disabled={active} onClick={() => onPreview(job.doc_id)}>查看原文</Button> },
          ]} />
          {canEdit && batch.data.batch.queued > 0 && !batch.data.batch.stopped && <Button danger disabled={active} loading={stop.isPending} onClick={() => void stopWaiting()}>停止尚未开始的资料</Button>}
        </>}
      </>}
      <Button onClick={onClose} disabled={active}>关闭并返回工作区</Button>
    </Space>
  </Modal>;
}
