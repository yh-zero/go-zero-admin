import { Alert, Button, Card, Descriptions, Empty, Space, Table, Tag, Typography } from 'antd';
import { WorkflowAnalysisResult } from './WorkflowAnalysisResult';
import { TaskAnalysisFeedback } from './TaskAnalysisFeedback';
import type { useAiWorkspaceGuard } from './useAiWorkspaceGuard';
import type { AiWorkspaceScope } from './workspaceQueries';
import { compareQualityRuns, qualityReasonLabel, qualityStatusLabels } from './knowledgeQuality';
import type { QualityRow, QualityRunView } from './knowledgeQuality';
import { formatTime } from '../../utils/format';
import styles from './knowledgeQuality.module.css';

type Guard = ReturnType<typeof useAiWorkspaceGuard>;
export type QualityFilter = 'all' | 'review' | 'execution' | 'waiting';
const millis = (value: number | null) => value === null ? '—' : `${Math.round(value)} ms`;
const percent = (value: number | null | undefined) => value == null ? '—' : `${Math.round(value * 1000) / 10}%`;
const rowLabel = (row: QualityRow) => row.executionFailed ? '执行未完成' : row.status === 'waiting' ? '未开始' : row.status === 'running' ? '执行中' : row.screen?.flagged ? '规则提示待复核' : '待人工确认';

export function KnowledgeQualityResults({ view, scope, guard, filter, canRepeat, onRepeat, onTask }: {
  view: QualityRunView; scope: AiWorkspaceScope; guard: Guard; filter: QualityFilter;
  canRepeat: (row: QualityRow) => boolean; onRepeat: (row: QualityRow) => void; onTask: (id: number) => void;
}) {
  const { run, summary } = view;
  const rows = view.rows.filter(row => filter === 'all' || filter === 'review' && !!row.screen?.flagged || filter === 'execution' && row.executionFailed || filter === 'waiting' && ['waiting', 'running'].includes(row.status));
  return <div className={styles.results}>
    <Descriptions size="small" column={{ xs: 1, sm: 2, lg: 4 }} items={[
      { key: 'status', label: '执行状态', children: <Tag>{qualityStatusLabels[run.status]}{run.stop_requested && run.status === 'running' ? ' · 当前题结束后停止' : ''}</Tag> },
      { key: 'progress', label: '进度', children: `${run.completed_count} 已完成 / ${run.failed_count} 未完成 / ${run.case_count} 题` },
      { key: 'review', label: '规则提示', children: `${summary.answer_rule_screen.flagged_count} 题待复核` },
      { key: 'time', label: '开始时间', children: run.started_at ? formatTime(run.started_at) : '等待开始' },
    ]} />
    {run.error && <Alert type={run.status === 'unknown' ? 'warning' : 'error'} showIcon title={run.error} description="执行未完成的题目单独列出，未当作空召回或错误回答。原请求和运行记录保持。" />}
    {run.status === 'cancelled' && <Alert type="info" showIcon title="批次已停止，未开始的题目没有执行。" />}
    {run.status === 'succeeded' && !run.corpus_verified && <Alert type="warning" title="结束语料未核对，不能据此比较提升。" />}
    <div className={styles.metrics}>
      <Card size="small"><Typography.Text type="secondary">直接检索的文档 Recall@5</Typography.Text><strong>{percent(summary.positive.recall_at[5])}</strong></Card>
      <Card size="small"><Typography.Text type="secondary">直接检索的证据覆盖@5</Typography.Text><strong>{percent(summary.positive.evidence_coverage_at[5])}</strong></Card>
      <Card size="small"><Typography.Text type="secondary">分析 P50 / P95</Typography.Text><strong>{millis(summary.latency.analysis_submit_to_terminal?.p50_ms ?? null)} / {millis(summary.latency.analysis_submit_to_terminal?.p95_ms ?? null)}</strong></Card>
      <Card size="small"><Typography.Text type="secondary">已知聊天用量</Typography.Text><strong>{summary.usage.known_total_tokens} tokens</strong><small>{summary.usage.incomplete_cases || run.failed_count ? '包含未取得完整用量的题目，数值不是全部计费量' : '不含查询 Embedding'}</small></Card>
    </div>
    <Alert type="info" showIcon title="规则筛查用于定位复核项，不代表语义正确率或人工验收通过。" description="检索指标用于检查直接检索结果；分析的实际引用以答案下方来源为准。耗时与质量统计只含成功题，失败项单列；预热、网络和小样本会影响结果。" />
    <Table<QualityRow> size="small" rowKey={row => row.item.id} dataSource={rows} pagination={false} scroll={{ x: 900 }}
      locale={{ emptyText: <Empty description="当前筛选没有题目" /> }}
      columns={[
        { title: '问题', key: 'query', width: 350, render: (_, row) => <div><Typography.Text>{row.item.query}</Typography.Text><div><Tag>{row.item.split === 'development' ? '开发题' : '初始预留题'}</Tag></div></div> },
        { title: '状态与复核', key: 'review', width: 160, render: (_, row) => <Tag color={row.executionFailed ? 'error' : row.screen?.flagged ? 'warning' : undefined}>{rowLabel(row)}</Tag> },
        { title: '检索 / 分析耗时', key: 'elapsed', width: 150, render: (_, row) => row.status === 'waiting' ? '—' : `${millis(row.searchServerMs)} / ${millis(row.analysisMs)}` },
        { title: '已知聊天用量', key: 'usage', width: 120, render: (_, row) => row.analysis ? `${row.analysis.usage.total_tokens}${row.analysis.usage.complete ? '' : '（不完整）'}` : '未取得' },
        { title: '操作', key: 'actions', width: 180, render: (_, row) => <Space wrap>
          <Button size="small" disabled={guard.writing || !canRepeat(row)} onClick={() => onRepeat(row)}>单题复测</Button>
          {row.taskId > 0 && guard.can('sys:ai:task:list') && <Button size="small" disabled={guard.writing} onClick={() => onTask(row.taskId)}>运行记录</Button>}
        </Space> },
      ]}
      expandable={{ expandedRowRender: row => <QualityAnswer row={row} scope={scope} guard={guard} />, rowExpandable: row => row.status !== 'waiting' || row.checks.length > 0 }} />
    <details className={styles.advanced}><summary>版本与测量范围</summary>
      <p>构建：{view.context.binary_sha}</p><p>题集：{run.dataset_sha256}</p>
      <p>模型：{view.context.model.provider} / {view.context.model.name}；检索：{view.context.embedding.provider} / {view.context.embedding.model}</p>
      <p>本批已完成分析 {summary.n} 题。比较时还会核对题目集合、知识库、语料指纹和模型预算。</p>
    </details>
  </div>;
}

export function QualityAnswer({ row, scope, guard }: { row: QualityRow; scope: AiWorkspaceScope; guard: Guard }) {
  return <section aria-label={`题目详情 ${row.item.id}`} className={styles.answer}>
    <Typography.Title level={5}>检查项</Typography.Title>
    {!!row.checks.length && <ul>{row.checks.map(check => <li key={check.requiredGroup}><Tag color={row.screen ? check.matched ? 'success' : 'warning' : undefined}>{row.screen ? check.matched ? '字符串命中' : '待复核' : '尚未筛查'}</Tag>{check.label}{check.kind === 'risk' && <Tag color="orange">关键风险</Tag>}</li>)}</ul>}
    {row.item.answer.absence_policy === 'required' && <p>本题还需人工核对：是否准确说明真实未知范围，且没有编造实例值。</p>}
    {!!row.screen?.reasons.length && <Alert type="warning" showIcon title={row.screen.reasons.map(qualityReasonLabel).join('；')} description="请对照实际引用片段确认，词法未命中也可能只是表达差异。" />}
    {row.executionFailed && <>
      <Alert type={row.status === 'unknown' ? 'warning' : 'error'} title={row.error || '本题执行未完成，结果待核对'} description={row.status === 'unknown' ? '请先按本题请求标识搜索运行记录，核对原运行结果。结果未知时不能直接复测；已发生的调用和用量不能按零解释。' : '保留原运行记录；失败费用和模型用量不能按零解释。可按本题请求标识搜索原运行记录。'} />
      <Typography.Paragraph>本题请求标识</Typography.Paragraph>
      <Typography.Paragraph code copyable={{ text: row.requestId }}>{row.requestId}</Typography.Paragraph>
    </>}
    {row.analysis && <><WorkflowAnalysisResult analysis={{ step: 1, total: 1, name: '质量验收分析', result: row.analysis }} />
      <Typography.Title level={5}>评价并加入反馈复核</Typography.Title>
      {guard.can('sys:ai:task:list') ? <TaskAnalysisFeedback key={row.taskId} scope={scope} taskId={row.taskId} stepIndex={0} guard={guard} /> : <Alert type="info" title="需要运行记录查看权限才能读取和提交已有分析评价。" />}
    </>}
  </section>;
}

export function KnowledgeQualityComparison({ baseline, candidate }: { baseline: QualityRunView; candidate: QualityRunView }) {
  const comparison = compareQualityRuns(baseline, candidate);
  if (!comparison.comparable || !comparison.delta) return <Alert type="warning" showIcon title="这两份结果不能计算提升" description={comparison.reasons.join('；')} />;
  return <Card size="small" title={`对照 #${baseline.run.id} → 当前 #${candidate.run.id}`}>
    <Descriptions size="small" column={{ xs: 1, sm: 2 }} items={[
      { key: 'flag', label: '规则提示题数变化', children: comparison.delta.flagged },
      { key: 'recall', label: '直接检索文档 Recall@5 变化', children: comparison.delta.recall5 === null ? '—' : `${Math.round(comparison.delta.recall5 * 1000) / 10} 个百分点` },
      { key: 'evidence', label: '直接检索证据覆盖@5 变化', children: comparison.delta.evidence5 === null ? '—' : `${Math.round(comparison.delta.evidence5 * 1000) / 10} 个百分点` },
      { key: 'cost', label: '已知聊天用量变化', children: `${comparison.delta.tokens} tokens` },
    ]} />
    <Typography.Paragraph type="secondary">{comparison.versionChanged ? '构建版本不同；题集、语料与模型预算一致。' : '构建版本相同。'}差值是当前减对照，规则提示减少不等于人工确认的质量提升。</Typography.Paragraph>
  </Card>;
}
