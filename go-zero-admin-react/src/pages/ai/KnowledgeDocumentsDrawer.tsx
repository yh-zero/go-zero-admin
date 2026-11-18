import { useEffect, useRef, useState } from 'react';
import { Alert, Button, Drawer, Dropdown, Empty, Input, Modal, Select, Space, Table, Tag } from 'antd';
import { DeleteOutlined, EditOutlined, MoreOutlined, PlusOutlined, RedoOutlined, ReloadOutlined, UploadOutlined } from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type { AiKbDocInfo, AiKbInfo, AiKbResp, AiListKbDocsReq } from '../../api/types';
import { deleteAiKbDoc, retryAiKbDoc } from '../../api/ai';
import { QueryState } from '../../components/QueryState';
import { correctPageParams } from '../../hooks/usePagedQuery';
import { useAuthStore } from '../../stores/auth';
import { useAiWorkspaceGuard } from './useAiWorkspaceGuard';
import { aiWorkspaceKeys, kbDocListOptions, kbImportOptions } from './workspaceQueries';
import type { AiWorkspaceScope } from './workspaceQueries';
import { canRetryDocument, documentStageLabel, documentWriteFailure } from './knowledgeDocuments';
import { readKnowledgeImportAttempt, subscribeKnowledgeImportAttempt, updateKnowledgeImportAttempt, writeKnowledgeImportAttempt } from './knowledgeImport';
import { knowledgeDocumentKeys } from './knowledgeDocumentQueries';
import { KnowledgeDocumentModal } from './KnowledgeDocumentModal';
import { KnowledgeImportModal } from './KnowledgeImportModal';
import styles from './workspace.module.css';

interface Props { scope: AiWorkspaceScope; kb: AiKbInfo; onChanged: () => void; initialDocId?: number | null; active?: boolean }
const colors = { pending: 'processing', ready: 'success', failed: 'error' } as const;

export function KnowledgeDocumentsDrawer({ onClose, ...props }: Props & { onClose: () => void }) {
  return <Drawer open title={`资料：${props.kb.name}`} size={960} onClose={onClose}><KnowledgeDocumentsPanel {...props} kbId={props.kb.id} /></Drawer>;
}

type PanelProps = Omit<Props, 'kb'> & { kb?: AiKbInfo; kbId: number };

export function KnowledgeDocumentsPanel({ scope, kb, kbId, onChanged, initialDocId, active = true }: PanelProps) {
  const guard = useAiWorkspaceGuard(scope, 'sys:ai:kb:list');
  const { isCurrent } = guard;
  const client = useQueryClient();
  const [modal, contextHolder] = Modal.useModal();
  const deleteConfirmation = useRef<ReturnType<typeof modal.confirm> | null>(null);
  const departmentId = useAuthStore(state => state.selectedDeptId);
  const retryKey = `ai-knowledge-retry:${JSON.stringify([scope[0], departmentId, kbId])}`;
  const [retryAttempt, setRetryAttempt] = useState(() => { try { return readKnowledgeImportAttempt(sessionStorage, retryKey); } catch { return null; } });
  const [params, setParams] = useState<AiListKbDocsReq>({ page: 1, page_size: 10 });
  const [previewId, setPreviewId] = useState<number | null>(initialDocId ?? null);
  const [importing, setImporting] = useState<'files' | 'paste' | 'history' | null>(null);
  const [batchId, setBatchId] = useState<string | undefined>(retryAttempt?.batchId);
  const [feedback, setFeedback] = useState(retryAttempt ? `请先核对原批次 ${retryAttempt.batchId}，不会自动再次重试。` : '');
  const docs = useQuery({ ...kbDocListOptions(scope, kbId, params, active), enabled: active && guard.can('sys:ai:kb:list'),
    refetchInterval: query => active && !importing && !guard.writing && !query.state.error && query.state.data?.pending_count ? 3000 : false });
  const retry = useMutation({ mutationFn: ({ id, request_id }: { id: number; request_id: string }) => retryAiKbDoc(kbId, id, { request_id }), retry: false });
  const remove = useMutation({ mutationFn: (id: number) => deleteAiKbDoc(kbId, id), retry: false });
  const retryReceipt = useQuery({ ...kbImportOptions(scope, kbId, retryAttempt?.batchId ?? ''), enabled: active && !!kb && !docs.isError && !!retryAttempt && !guard.writing && !importing && isCurrent() });
  useEffect(() => {
    if (!docs.isError) return;
    deleteConfirmation.current?.destroy();
    deleteConfirmation.current = null;
    // oxlint-disable-next-line react/set-state-in-effect -- 资料访问失败后，恢复读取也须由用户重新打开操作。
    setPreviewId(null);
    setImporting(null);
  }, [docs.isError]);
  useEffect(() => subscribeKnowledgeImportAttempt(sessionStorage, retryKey, latest => {
    if (!isCurrent()) return;
    setRetryAttempt(latest); setFeedback(latest ? `请先核对原批次 ${latest.batchId}，不会自动再次重试。` : '');
  }), [retryKey, isCurrent]);
  useEffect(() => {
    if (docs.isError || !retryAttempt || !retryReceipt.isSuccess || !isCurrent() || !retryAttempt.requestIds.every(id => retryReceipt.data.list.some(job => job.request_id === id))) return;
    try { if (updateKnowledgeImportAttempt(sessionStorage, retryKey, retryAttempt.batchId, () => null)) return; }
    catch { /* 共享内存保留确认结果，不覆盖其他批次。 */ }
    // oxlint-disable-next-line react/set-state-in-effect -- 原请求回执确认接收后解除未知锁。
    setRetryAttempt(null); setFeedback('');
    void client.invalidateQueries({ queryKey: aiWorkspaceKeys.kbDocs(scope, kbId) });
  }, [docs.isError, retryAttempt, retryReceipt.isSuccess, retryReceipt.data, isCurrent, retryKey, client, scope, kbId]);
  useEffect(() => {
    if (!kb || !docs.isSuccess || docs.isFetching || !isCurrent()) return;
    const { ready_count, pending_count, failed_count } = docs.data;
    if (kb.ready_count === ready_count && kb.pending_count === pending_count && kb.failed_count === failed_count) return;
    const key = aiWorkspaceKeys.kb(scope, kbId);
    if ((client.getQueryState(key)?.dataUpdatedAt ?? 0) > docs.dataUpdatedAt) return;
    client.setQueryData<AiKbResp>(key, previous => previous ? { ...previous, kb: { ...previous.kb, ready_count, pending_count, failed_count, doc_count: ready_count + pending_count + failed_count } } : previous);
  }, [docs.isSuccess, docs.isFetching, docs.data, docs.dataUpdatedAt, kb, client, scope, kbId, isCurrent]);
  useEffect(() => {
    if (!docs.isSuccess || docs.isFetching || !isCurrent()) return;
    // oxlint-disable-next-line react/set-state-in-effect -- 删除后只校正返回本次总数的页码。
    setParams(current => current === params ? correctPageParams(current, docs.data.total) : current);
  }, [docs.isSuccess, docs.data, docs.isFetching, params, isCurrent]);
  function refresh() {
    if (!guard.isCurrent()) return;
    void client.invalidateQueries({ queryKey: aiWorkspaceKeys.kbDocs(scope, kbId) });
    void client.invalidateQueries({ queryKey: knowledgeDocumentKeys.all(scope, kbId) });
    if (retryAttempt) void retryReceipt.refetch();
    onChanged();
  }
  async function retryDocument(doc: AiKbDocInfo) {
    if (retryAttempt || !canRetryDocument(doc) || !guard.beginWrite('sys:ai:kb:edit')) return;
    const request_id = crypto.randomUUID(); setFeedback('');
    const original = { batchId: request_id, requestIds: [request_id] }; setRetryAttempt(original);
    try { writeKnowledgeImportAttempt(sessionStorage, retryKey, original); } catch { setFeedback(`请保留原请求标识：${request_id}`); }
    try { const receipt = await retry.mutateAsync({ id: doc.id, request_id }); try { updateKnowledgeImportAttempt(sessionStorage, retryKey, request_id, () => null); } catch { /* 保留的原回执不会重复提交。 */ } if (guard.isCurrent()) { setBatchId(receipt.batch_id); setRetryAttempt(null); refresh(); } }
    catch (cause) { const failure = documentWriteFailure(cause); if (!failure.uncertain) { try { updateKnowledgeImportAttempt(sessionStorage, retryKey, request_id, () => null); } catch { /* 返回时只读核对。 */ } } if (guard.isCurrent()) { if (failure.uncertain) { setBatchId(request_id); setFeedback(`请核对原批次 ${request_id}；页面不会自动再次重试。`); } else { setBatchId(undefined); setRetryAttempt(null); setFeedback(failure.message); } } }
    finally { guard.finishWrite(); }
  }
  async function deleteDocument(id: number) {
    if (!guard.beginWrite('sys:ai:kb:edit')) return;
    try { await remove.mutateAsync(id); refresh(); }
    catch { /* 请求层反馈，列表保留。 */ }
    finally { guard.finishWrite(); }
  }
  // 首次加载资料与详情并发，各自通过服务端归属校验；详情成功后才呈现资料和操作入口。
  if (!kb) return null;
  if (docs.isError) return <QueryState error={docs.error} onRetry={() => docs.refetch()} />;
  return <div>
    <div className={styles.toolbar}><Input.Search className={styles.search} allowClear aria-label="查找资料名称" placeholder="查找资料名称" onSearch={query => setParams(current => ({ ...current, page: 1, query: query.trim() || undefined }))} />
      <Select<AiKbDocInfo['status'] | ''> className={styles.statusSelect} aria-label="资料状态" value={params.status ?? ''} onChange={status => setParams(current => ({ ...current, page: 1, status: status || undefined }))} options={[{ value: '', label: '全部状态' }, { value: 'ready', label: '可使用' }, { value: 'pending', label: '处理中' }, { value: 'failed', label: '处理失败' }]} />
      <Button icon={<ReloadOutlined />} aria-label="刷新资料" loading={docs.isFetching} onClick={refresh} /></div>
    <Space wrap><Tag>全部 {(docs.data?.ready_count ?? kb.ready_count) + (docs.data?.pending_count ?? kb.pending_count) + (docs.data?.failed_count ?? kb.failed_count)}</Tag><Tag color="success">可使用 {docs.data?.ready_count ?? kb.ready_count}</Tag><Tag color="processing">处理中 {docs.data?.pending_count ?? kb.pending_count}</Tag><Tag color="error">失败 {docs.data?.failed_count ?? kb.failed_count}</Tag></Space>
    <QueryState isPending={docs.isPending} />
    {feedback && <Alert showIcon type="warning" title="操作结果待核对" description={feedback} />}
    {!docs.isPending && !docs.error && !docs.data?.total && <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={params.query || params.status ? '没有符合条件的资料' : '添加资料后，开始带来源的分析'} />}
    {!!docs.data?.total && <Table<AiKbDocInfo> rowKey="id" size="small" scroll={{ x: 580 }} dataSource={docs.data.list} pagination={{ current: params.page, pageSize: params.page_size, total: docs.data.total, showSizeChanger: true, onChange: (page, page_size) => setParams(current => ({ ...current, page: page_size === current.page_size ? page : 1, page_size })) }} columns={[
      { title: '资料', dataIndex: 'title', render: (_, doc) => <><Button type="link" onClick={() => setPreviewId(doc.id)}>{doc.title}</Button><div className={styles.help}>第 {doc.version} 版</div></> },
      { title: '状态', render: (_, doc) => <Space orientation="vertical" size={0}><Tag color={colors[doc.status]}>{documentStageLabel(doc)}</Tag>{doc.error && <span className={styles.help}>{doc.error}</span>}</Space> },
      { title: '操作', width: 190, render: (_, doc) => guard.can('sys:ai:kb:edit') && <Space size={0}>
        {canRetryDocument(doc) && <Button type="link" icon={<RedoOutlined />} aria-label={`重试${doc.title}`} disabled={guard.writing || !!retryAttempt} onClick={() => void retryDocument(doc)}>重试处理</Button>}
        <Dropdown trigger={['click']} menu={{ items: [
          { key: 'edit', label: '查看并编辑原文', icon: <EditOutlined />, disabled: guard.writing },
          { key: 'delete', label: '删除', icon: <DeleteOutlined />, danger: true, disabled: guard.writing },
        ], onClick: ({ key }) => {
          if (guard.writing || !guard.can('sys:ai:kb:edit')) return;
          if (key === 'edit') setPreviewId(doc.id);
          if (key === 'delete') deleteConfirmation.current = modal.confirm({ title: '删除这篇资料？', okText: '删除', cancelText: '取消', okButtonProps: { danger: true }, onOk: () => deleteDocument(doc.id), afterClose: () => { deleteConfirmation.current = null; } });
        } }}><Button type="text" icon={<MoreOutlined />} aria-label={`更多：${doc.title}`} disabled={guard.writing}>更多</Button></Dropdown>
      </Space> },
    ]} />}
    <div className={styles.batchLink}>{guard.can('sys:ai:kb:edit') && <Dropdown trigger={['click']} menu={{ items: [
      { key: 'files', label: '导入 TXT / Markdown', icon: <UploadOutlined /> },
      { key: 'paste', label: '粘贴资料', icon: <PlusOutlined /> },
    ], onClick: ({ key }) => { if (!guard.writing && guard.can('sys:ai:kb:edit')) setImporting(key === 'files' ? 'files' : 'paste'); } }}><Button type="primary" icon={<PlusOutlined />} disabled={guard.writing}>添加资料</Button></Dropdown>}<Button type="link" disabled={guard.writing} onClick={() => setImporting('history')}>查看导入批次</Button></div>
    {contextHolder}
    {active && docs.isSuccess && previewId && <KnowledgeDocumentModal scope={scope} kbId={kb.id} docId={previewId} metadata={docs.data.list.find(doc => doc.id === previewId)} metadataUpdatedAt={docs.dataUpdatedAt} onClose={() => setPreviewId(null)} canEdit={guard.can('sys:ai:kb:edit')} beginWrite={() => guard.beginWrite('sys:ai:kb:edit')} finishWrite={guard.finishWrite} writing={guard.writing} isCurrent={guard.isCurrent} onChanged={refresh} />}
    {active && importing && <KnowledgeImportModal scope={scope} kbId={kb.id} canEdit={guard.can('sys:ai:kb:edit')} initialMode={importing} initialBatchId={batchId} writing={guard.writing} beginWrite={() => guard.beginWrite('sys:ai:kb:edit')} finishWrite={guard.finishWrite} isCurrent={guard.isCurrent} onClose={() => setImporting(null)} onChanged={refresh} onPreview={id => { setImporting(null); setPreviewId(id); }} />}
  </div>;
}
