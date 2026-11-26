import { useState } from 'react';
import { Alert, Button, Card, Dropdown, Result, Space, Typography } from 'antd';
import { ArrowLeftOutlined, ReloadOutlined } from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useNavigate, useParams, useSearchParams } from 'react-router-dom';
import { QueryState } from '../../components/QueryState';
import { AsyncContent } from '../../components/AsyncContent';
import { AIWorkspaceAccess } from './AIWorkspaceAccess';
import { useAiWorkspaceGuard } from './useAiWorkspaceGuard';
import { aiWorkspaceKeys, kbDetailOptions } from './workspaceQueries';
import type { AiWorkspaceScope } from './workspaceQueries';
import { KnowledgeDocumentsPanel } from './KnowledgeDocumentsDrawer';
import { DeferredWorkflowRun } from './DeferredWorkflowRun';
import { workflowTemplate } from './workflowDefinition';
import { KbSearchResults } from './KbSearchModal';
import { kbSearchOptions } from './knowledgeSearch';
import { KnowledgeDocumentModal } from './KnowledgeDocumentModal';
import { parseTaskId } from './taskState';
import styles from './workspace.module.css';

const loadFeedback = () => import('./KnowledgeFeedbackPanel').then(module => module.KnowledgeFeedbackPanel);
const loadRun = () => import('./WorkflowDraftRun').then(module => module.WorkflowDraftRun);

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
  const [feedbackOpen, setFeedbackOpen] = useState(() => params.get('tab') === 'feedback');
  const [feedbackVisited, setFeedbackVisited] = useState(feedbackOpen);
  const [workspaceVisited, setWorkspaceVisited] = useState(!feedbackOpen);
  const [docId, setDocId] = useState<number | null>(null);
  const search = useMutation(kbSearchOptions(scope, kbId));
  async function findSources(question: string) {
    if (!kb.data?.capabilities.retrieval_configured || !guard.beginWrite('sys:ai:kb:list')) return;
    try { await search.mutateAsync({ query: question }); } catch { /* 保留问题，允许重试。 */ } finally { guard.finishWrite(); }
  }
  function refresh() {
    if (!guard.isCurrent()) return;
    void client.invalidateQueries({ queryKey: aiWorkspaceKeys.kb(scope, kbId) });
    void client.invalidateQueries({ queryKey: aiWorkspaceKeys.kbs(scope) });
  }
  return <div className={styles.page}>
    <div className={styles.heading}><div><Button type="link" icon={<ArrowLeftOutlined />} disabled={guard.writing} onClick={() => navigate('/ai/knowledge')}>全部知识库</Button><Typography.Title level={3}>{(!kb.error && kb.data?.kb.name) || '知识库工作区'}</Typography.Title>
      {kb.data && !kb.error && <Typography.Text type="secondary">{kb.data.kb.description}</Typography.Text>}</div><Space wrap>
        {kb.data && !kb.error && guard.can('sys:ai:conversation:list') && guard.can('sys:ai:message:send') && <Button disabled={guard.writing} onClick={() => navigate('/ai/chat', { state: { knowledge: { id: kbId, scope: JSON.stringify(scope) } } })}>在对话中使用</Button>}
        <Dropdown trigger={['click']} disabled={guard.writing || !kb.data || !!kb.error} menu={{ items: [{ key: 'feedback', label: '反馈与复核' }], onClick: () => {
          if (!guard.isCurrent() || guard.writing) return;
          setFeedbackVisited(true); setFeedbackOpen(true);
        } }}><Button>管理</Button></Dropdown><Button icon={<ReloadOutlined />} aria-label="刷新知识库" loading={kb.isFetching} onClick={refresh} /></Space></div>
    <QueryState isPending={kb.isPending} error={kb.error} onRetry={() => kb.refetch()} />
    {workspaceVisited && kb.data && !kb.error && kb.data.kb.ready_count > 0 && <div hidden={feedbackOpen}><Card title="向这个知识库提问">
      <AsyncContent load={loadRun} componentProps={{ scope, guard, draft, attemptId: `knowledge:${kbId}`, fixedKb: kb.data.kb,
        capabilities: kb.data.capabilities, active: !feedbackOpen, onFindSources: findSources, findingSources: search.isPending,
        onSave: taskId => navigate(`/ai/workflows/new?from_task=${taskId}`) }} />
      {search.isError && <Alert type="warning" title="查找原文失败，请核对后重试。" />}
      {search.data && !search.isPending && <KbSearchResults result={search.data} query={search.variables?.query} onOpenDoc={setDocId} />}
      {!feedbackOpen && docId && <KnowledgeDocumentModal scope={scope} kbId={kbId} docId={docId} canEdit={false} writing={false} beginWrite={() => false} finishWrite={() => undefined} isCurrent={guard.isCurrent} onClose={() => setDocId(null)} onChanged={() => undefined} />}
    </Card></div>}
    <div hidden={feedbackOpen || !kb.data || !!kb.error}><Card title="资料">
      <KnowledgeDocumentsPanel key={params.get('doc_id') ?? 'docs'} scope={scope} kbId={kbId} kb={kb.error ? undefined : kb.data?.kb}
        active={!feedbackOpen && !kb.error} onChanged={refresh} initialDocId={parseTaskId(params.get('doc_id'))} />
    </Card></div>
    {kb.data && !kb.error && kb.data.kb.ready_count === 0 && !feedbackOpen && <DeferredWorkflowRun recoveryOnly scope={scope} guard={guard} draft={draft} attemptId={`knowledge:${kbId}`}
      fixedKb={kb.data.kb} capabilities={kb.data.capabilities} onSave={taskId => navigate(`/ai/workflows/new?from_task=${taskId}`)} />}
    {kb.data && !kb.error && feedbackVisited && <div hidden={!feedbackOpen}><Card title="反馈与复核" extra={<Button disabled={guard.writing} onClick={() => { setWorkspaceVisited(true); setFeedbackOpen(false); }}>返回资料</Button>}>
      <AsyncContent load={loadFeedback} componentProps={{ scope, kbId, guard, active: feedbackOpen }} />
    </Card></div>}
  </div>;
}
