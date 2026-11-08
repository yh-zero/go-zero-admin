import { useState } from 'react';
import { Alert, Button, Card, Form, Input, Result, Space, Tabs, Tag, Typography } from 'antd';
import { ArrowLeftOutlined, ReloadOutlined } from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useNavigate, useParams, useSearchParams } from 'react-router-dom';
import { QueryState } from '../../components/QueryState';
import { AIWorkspaceAccess } from './AIWorkspaceAccess';
import { useAiWorkspaceGuard } from './useAiWorkspaceGuard';
import { aiWorkspaceKeys, kbDetailOptions } from './workspaceQueries';
import type { AiWorkspaceScope } from './workspaceQueries';
import { KnowledgeDocumentsPanel } from './KnowledgeDocumentsDrawer';
import { WorkflowDraftRun } from './WorkflowDraftRun';
import { workflowTemplate } from './workflowDefinition';
import { KbSearchResults } from './KbSearchModal';
import { kbSearchOptions, validateKbSearchQuery } from './knowledgeSearch';
import { KnowledgeDocumentModal } from './KnowledgeDocumentModal';
import { KnowledgeFeedbackPanel } from './KnowledgeFeedbackPanel';
import { parseTaskId } from './taskState';
import styles from './workspace.module.css';

export default function KnowledgeWorkspacePage() {
  const { id } = useParams();
  const kbId = parseTaskId(id ?? null);
  if (!kbId) return <Result status="404" title="知识库不存在" />;
  return <AIWorkspaceAccess code="sys:ai:kb:list" title="知识库">{scope => <KnowledgeWorkspaceSession key={`${scope.join(':')}:${kbId}`} scope={scope} kbId={kbId} />}</AIWorkspaceAccess>;
}

function KnowledgeWorkspaceSession({ scope, kbId }: { scope: AiWorkspaceScope; kbId: number }) {
  const navigate = useNavigate();
  const [params] = useSearchParams();
  const guard = useAiWorkspaceGuard(scope, 'sys:ai:kb:list');
  const client = useQueryClient();
  const kb = useQuery(kbDetailOptions(scope, kbId));
  const [draft] = useState(() => workflowTemplate('kb_analyze', []));
  const [tab, setTab] = useState(() => params.get('tab') === 'feedback' ? 'feedback' : 'docs');
  function refresh() {
    if (!guard.isCurrent()) return;
    void client.invalidateQueries({ queryKey: aiWorkspaceKeys.kb(scope, kbId) });
    void client.invalidateQueries({ queryKey: aiWorkspaceKeys.kbs(scope) });
  }
  return <div className={styles.page}>
    <div className={styles.heading}><div><Button type="link" icon={<ArrowLeftOutlined />} disabled={guard.writing} onClick={() => navigate('/ai/knowledge')}>全部知识库</Button><Typography.Title level={3}>{kb.data?.kb.name ?? '知识库工作区'}</Typography.Title>
      {kb.data && <Space wrap><Tag color="success">{kb.data.kb.ready_count} 篇可使用</Tag><Tag color="processing">{kb.data.kb.pending_count} 篇处理中</Tag><Typography.Text type="secondary">{kb.data.kb.description}</Typography.Text></Space>}</div><Space wrap><Button disabled={guard.writing || !kb.data || !!kb.error} onClick={() => navigate(`/ai/knowledge/${kbId}/quality`)}>质量验收</Button><Button icon={<ReloadOutlined />} loading={kb.isFetching} onClick={refresh}>刷新</Button></Space></div>
    <QueryState isPending={kb.isPending} error={kb.error} onRetry={() => kb.refetch()} />
    {kb.data && !kb.error && <div className={styles.columns}>
      <Card><Tabs activeKey={tab} onChange={next => { if (!guard.writing && guard.isCurrent()) setTab(next); }} destroyOnHidden items={[
        { key: 'docs', label: '资料', disabled: guard.writing, children: <KnowledgeDocumentsPanel key={params.get('doc_id') ?? 'docs'} scope={scope} kb={kb.data.kb} onChanged={refresh} initialDocId={parseTaskId(params.get('doc_id'))} /> },
        { key: 'search', label: '测试资料检索', disabled: guard.writing, children: <KnowledgeSearchPanel scope={scope} kbId={kbId} configured={kb.data.capabilities.retrieval_configured} /> },
        { key: 'feedback', label: '反馈与复核', disabled: guard.writing, children: <KnowledgeFeedbackPanel scope={scope} kbId={kbId} guard={guard} /> },
      ]} /></Card>
      <Card><WorkflowDraftRun scope={scope} guard={guard} draft={draft} attemptId={`knowledge:${kbId}`} fixedKb={kb.data.kb} capabilities={kb.data.capabilities} onSave={taskId => navigate(`/ai/workflows/new?from_task=${taskId}`)} /></Card>
    </div>}
  </div>;
}

function KnowledgeSearchPanel({ scope, kbId, configured }: { scope: AiWorkspaceScope; kbId: number; configured: boolean }) {
  const guard = useAiWorkspaceGuard(scope, 'sys:ai:kb:list');
  const search = useMutation(kbSearchOptions(scope, kbId));
  const [docId, setDocId] = useState<number | null>(null);
  const [form] = Form.useForm<{ query: string }>();
  async function submit(values: { query: string }) {
    if (!guard.beginWrite('sys:ai:kb:list')) return;
    try { await search.mutateAsync(values); } catch { /* 请求层反馈，表单保留。 */ } finally { guard.finishWrite(); }
  }
  return <div className={styles.inlineSearch}>
    <Typography.Paragraph type="secondary">查看与问题相关的片段，不生成分析答案。命中片段不代表资料足以回答；检索会调用查询 Embedding。</Typography.Paragraph>
    {!configured && <Alert type="warning" title="检索模型尚未配置，请联系管理员。" />}
    <Form form={form} layout="vertical" onFinish={values => void submit(values)} disabled={guard.writing}>
      <Form.Item name="query" label="试试能否找到对应资料" rules={[{ validator: (_, value: string) => { const error = validateKbSearchQuery(value ?? ''); return error ? Promise.reject(new Error(error)) : Promise.resolve(); } }]}><Input.TextArea rows={3} placeholder="输入问题或关键词" /></Form.Item>
      <Button type="primary" htmlType="submit" loading={guard.writing} disabled={!configured}>查找资料</Button>
    </Form>
    {search.isError && <Alert type="warning" title="本次检索未完成，请根据请求错误提示处理后重试。" />}
    {search.data && !search.isPending && <KbSearchResults result={search.data} onOpenDoc={setDocId} />}
    {docId && <KnowledgeDocumentModal scope={scope} kbId={kbId} docId={docId} canEdit={false} writing={false} beginWrite={() => false} finishWrite={() => undefined} isCurrent={guard.isCurrent} onClose={() => setDocId(null)} onChanged={() => undefined} />}
  </div>;
}
