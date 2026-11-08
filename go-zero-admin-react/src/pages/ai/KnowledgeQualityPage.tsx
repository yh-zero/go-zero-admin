import { useEffect, useRef, useState } from 'react';
import { Alert, Button, Card, Empty, Pagination, Result, Select, Space, Table, Tag, Typography } from 'antd';
import { ArrowLeftOutlined, PlayCircleOutlined, ReloadOutlined } from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Link, useLocation, useNavigate, useParams, useSearchParams } from 'react-router-dom';
import { fetchAiKbEvaluation, fetchAiKbEvaluations, startAiKbEvaluation, stopAiKbEvaluation } from '../../api/ai';
import type { AiKbEvaluationInfo, AiKbEvaluationStatus, AiStartKbEvaluationReq } from '../../api/types';
import { ApiError } from '../../utils/response';
import { formatTime } from '../../utils/format';
import { useAuthStore } from '../../stores/auth';
import { QueryState } from '../../components/QueryState';
import { AIWorkspaceAccess } from './AIWorkspaceAccess';
import { useAiWorkspaceGuard } from './useAiWorkspaceGuard';
import { aiWorkspaceKeys, kbDetailOptions } from './workspaceQueries';
import type { AiWorkspaceScope } from './workspaceQueries';
import { parseTaskId } from './taskState';
import { TaskDetail } from './TaskDetail';
import { KnowledgeQualityComparison, KnowledgeQualityResults } from './KnowledgeQualityDetails';
import type { QualityFilter } from './KnowledgeQualityDetails';
import { qualityCatalogueOptions, qualityListOptions, qualityRunOptions } from './knowledgeQualityQueries';
import { assertQualityAttemptResult, assertQualityRunInfo, canRepeatQualityRow, clearQualityAttempt, createQualityAttempt, importQualityDataset, qualityAttemptKey, qualityGroupLabel, qualityStatusLabels, readQualityAttempt, writeQualityAttempt } from './knowledgeQuality';
import type { QualityCase, QualityCatalogue, QualityRow } from './knowledgeQuality';
import styles from './knowledgeQuality.module.css';
import { FeedbackDatasetPreview } from './FeedbackDatasetPreview';
import { downloadQualityDataset } from './feedbackDataset';
import { decodeCustomDataset } from '../../../../go-zero-admin/data/scripts/kb-custom-dataset.mjs';

export default function KnowledgeQualityPage() {
  const kbId = parseTaskId(useParams().id ?? null);
  if (!kbId) return <Result status="404" title="知识库不存在" />;
  return <AIWorkspaceAccess code="sys:ai:kb:list" title="质量验收">{scope => <KnowledgeQualitySession key={`${scope.join(':')}:${kbId}`} scope={scope} kbId={kbId} />}</AIWorkspaceAccess>;
}

export function KnowledgeQualitySession({ scope, kbId }: { scope: AiWorkspaceScope; kbId: number }) {
  const guard = useAiWorkspaceGuard(scope, 'sys:ai:kb:list'), client = useQueryClient();
  const location = useLocation();
  const navigate = useNavigate();
  const lookupController = useRef<AbortController | null>(null);
  useEffect(() => () => lookupController.current?.abort(), []);
  const [params, setParams] = useSearchParams();
  const selectedId = parseTaskId(params.get('evaluation_id'));
  const departmentId = useAuthStore(state => state.selectedDeptId);
  const key = qualityAttemptKey(scope, departmentId, kbId);
  const [restored] = useState(() => {
    try { return { request: readQualityAttempt(sessionStorage, key), error: '' }; }
    catch { return { request: null, error: '浏览器无法恢复上次评测标识，请先核对下方历史批次。' }; }
  });
  const [attempt, setAttempt] = useState<AiStartKbEvaluationReq | null>(restored.request);
  const attemptRef = useRef(attempt);
  const [unknown, setUnknown] = useState(!!restored.request || !!restored.error);
  const [notice, setNotice] = useState(restored.error);
  const [storageNotice, setStorageNotice] = useState('');
  const [datasetKey, setDatasetKey] = useState('');
  const [imported, setImported] = useState<QualityCatalogue | null>(null);
  const [importing, setImporting] = useState(false);
  const [groupKey, setGroupKey] = useState('');
  const [selectedCases, setSelectedCases] = useState<string[] | null>(null);
  const [page, setPage] = useState(1);
  const [historyStatus, setHistoryStatus] = useState<AiKbEvaluationStatus | 'all'>('all');
  const [filter, setFilter] = useState<QualityFilter>('all');
  const [baselineId, setBaselineId] = useState<number | null>(null);
  const [taskId, setTaskId] = useState<number | null>(null);
  const kb = useQuery(kbDetailOptions(scope, kbId));
  const catalogues = useQuery({ ...qualityCatalogueOptions(scope, kbId), enabled: !!kb.data && !kb.error && guard.isCurrent() });
  const history = useQuery({ ...qualityListOptions(scope, kbId, { page, page_size: 10, ...(historyStatus === 'all' ? {} : { status: historyStatus }) }), enabled: !!kb.data && !kb.error && guard.isCurrent() });
  const detail = useQuery({ ...qualityRunOptions(scope, kbId, selectedId ?? 0), enabled: !!selectedId && !!kb.data && !kb.error && guard.isCurrent() });
  const baseline = useQuery({ ...qualityRunOptions(scope, kbId, baselineId ?? 0), enabled: !!baselineId && baselineId !== selectedId && !!kb.data && !kb.error && guard.isCurrent() });
  const start = useMutation({ mutationFn: (request: AiStartKbEvaluationReq) => startAiKbEvaluation(kbId, request), retry: false });
  const stop = useMutation({ mutationFn: (id: number) => stopAiKbEvaluation(kbId, id), retry: false });
  const available = [...(catalogues.error ? [] : catalogues.data ?? []), ...(imported ? [imported] : [])];
  const chosen = available.find(d => d.key === datasetKey) ?? available[0];
  const groupCases = chosen?.dataset.cases.filter(item => item.kb === groupKey) ?? [];
  const caseIds = selectedCases ?? groupCases.filter(item => item.split === 'development').map(item => item.id);
  const busy = guard.writing || start.isPending || stop.isPending || importing;
  const canEvaluate = guard.can('sys:ai:kb:evaluate');
  const configured = kb.data?.capabilities.analysis_configured && kb.data.capabilities.retrieval_configured;
  const total = history.data?.total;
  const historyReady = history.isSuccess && !history.isFetching;
  const currentWorkspace = guard.isCurrent;
  useEffect(() => {
    const incoming = location.state as { datasetRaw?: unknown; datasetScope?: unknown } | null;
    if (typeof incoming?.datasetRaw !== 'string' || !Array.isArray(incoming.datasetScope) || JSON.stringify(incoming.datasetScope) !== JSON.stringify(scope)) return;
    let active = true;
    void importQualityDataset(incoming.datasetRaw).then(value => {
      if (!active || !currentWorkspace()) return;
      setImported(value); setDatasetKey(value.key); setGroupKey(value.dataset.cases[0].kb); setSelectedCases(null);
      setNotice('已导入冻结题集，尚未开始分析。请核对预览后明确开始评测。');
      // 题集移交完成后清除路由历史中的全文，后续恢复只使用原请求的会话存储。
      navigate(`${location.pathname}${location.search}`, { replace: true, state: null });
    }).catch(error => { if (active && currentWorkspace()) setNotice(error instanceof Error ? error.message : '题集导入失败'); });
    return () => { active = false; };
  }, [location.key, location.state, location.pathname, location.search, navigate, scope, currentWorkspace]);
  useEffect(() => {
    if (!guard.isCurrent() || !historyReady || total === undefined) return;
    const maximum = Math.max(1, Math.ceil(total / 10));
    // oxlint-disable-next-line react/set-state-in-effect
    if (page > maximum) setPage(current => current === page ? maximum : current);
  }, [guard, historyReady, page, total]);

  function selectRun(id: number) {
    if (busy || !guard.isCurrent()) return;
    const next = new URLSearchParams(params); next.set('evaluation_id', String(id)); setParams(next);
    setFilter('all'); setNotice('');
  }
  function discardAttempt() {
    if (!guard.isCurrent()) return;
    attemptRef.current = null; setAttempt(null); setUnknown(false);
    try { clearQualityAttempt(sessionStorage, key); setStorageNotice(''); } catch { setStorageNotice('浏览器仍保留旧标识。刷新后请核对原批次，不会自动发起运行。'); }
  }
  async function send(request: AiStartKbEvaluationReq) {
    try {
      if (request.dataset_snapshot !== undefined) {
        const verified = await importQualityDataset(request.dataset_snapshot);
        if (verified.key !== request.dataset_key || verified.sha256 !== request.dataset_sha256) throw new Error('原题集快照指纹不符，不能更换原请求内容');
        if (!guard.isCurrent()) return;
      }
      const response = await start.mutateAsync(request);
      if (!guard.isCurrent()) return;
      assertQualityRunInfo(response.evaluation, kbId);
      assertQualityAttemptResult(response.evaluation, request);
      discardAttempt();
      const next = new URLSearchParams(params); next.set('evaluation_id', String(response.evaluation.id)); setParams(next);
      setPage(1); setFilter('all');
      void client.invalidateQueries({ queryKey: aiWorkspaceKeys.qualityRuns(scope, kbId) });
      void client.invalidateQueries({ queryKey: aiWorkspaceKeys.qualityRun(scope, kbId, response.evaluation.id) });
    } catch (error) {
      if (!guard.isCurrent()) return;
      const uncertain = !(error instanceof ApiError) || error.code >= 500;
      setUnknown(uncertain);
      if (!uncertain) discardAttempt();
      setNotice(uncertain ? '提交结果尚未确认。请核对原请求，原标识与题目保持。' : '本次未能开始，所选题目已保留，请根据请求提示处理后再运行。');
    }
  }
  async function run(catalogue: QualityCatalogue | undefined, group: string, ids: string[]) {
    if (!catalogue || unknown || !canEvaluate || !configured || !guard.beginWrite('sys:ai:kb:evaluate')) return;
    try {
      const request = createQualityAttempt(catalogue, group, ids, crypto.randomUUID());
      attemptRef.current = request; setAttempt(request); setNotice('');
      try { writeQualityAttempt(sessionStorage, key, request); }
      catch { setStorageNotice('浏览器无法保存本次标识，当前页面仍保留。离开前请先核对本次批次。'); }
      await send(request);
    } catch (error) { if (guard.isCurrent()) setNotice(error instanceof Error ? error.message : '请检查题目选择'); }
    finally { guard.finishWrite(); }
  }
  async function lookup() {
    const request = attemptRef.current;
    if (!request || !guard.beginWrite('sys:ai:kb:list')) return;
    const controller = new AbortController(); lookupController.current = controller;
    try {
      const response = await fetchAiKbEvaluations(kbId, { request_id: request.request_id, page: 1, page_size: 1 }, controller.signal);
      if (!guard.isCurrent()) return;
      if (!Array.isArray(response.list) || response.total > 1 || response.list.length > 1) throw new Error('原请求查询结果无效');
      if (!response.list.length) { setNotice('尚未查到匹配批次，请稍后再核对；也可使用原标识继续原提交。'); return; }
      let result = response.list[0]; assertQualityRunInfo(result, kbId);
      if (request.dataset_snapshot !== undefined) {
        result = (await fetchAiKbEvaluation(kbId, result.id, controller.signal)).evaluation;
        if (!guard.isCurrent()) return;
        assertQualityRunInfo(result, kbId);
      }
      assertQualityAttemptResult(result, request);
      discardAttempt(); const next = new URLSearchParams(params); next.set('evaluation_id', String(result.id)); setParams(next); setNotice('已找到原批次。');
      void client.invalidateQueries({ queryKey: aiWorkspaceKeys.qualityRuns(scope, kbId) });
    } catch { if (guard.isCurrent()) setNotice('原请求暂时未能核对，标识仍保留，请稍后重查。'); }
    finally { if (lookupController.current === controller) lookupController.current = null; guard.finishWrite(); }
  }
  async function continueOriginal() {
    const request = attemptRef.current;
    if (!request || !guard.beginWrite('sys:ai:kb:evaluate')) return;
    try { await send(request); } finally { guard.finishWrite(); }
  }
  async function stopSelected() {
    const id = selectedId;
    if (!id || !guard.beginWrite('sys:ai:kb:evaluate')) return;
    try {
      const response = await stop.mutateAsync(id);
      if (!guard.isCurrent()) return;
      assertQualityRunInfo(response.evaluation, kbId);
      if (response.evaluation.id !== id) throw new Error('停止响应标识不符');
      setNotice(`已提交批次 #${id} 的停止请求，当前题结束后不再开始后续题目。`);
    } catch { if (guard.isCurrent()) setNotice(`批次 #${id} 的停止结果尚未确认，请刷新原批次状态。`); }
    finally {
      if (guard.isCurrent()) {
        void client.invalidateQueries({ queryKey: aiWorkspaceKeys.qualityRun(scope, kbId, id) });
        void client.invalidateQueries({ queryKey: aiWorkspaceKeys.qualityRuns(scope, kbId) });
      }
      guard.finishWrite();
    }
  }
  function canRepeatRow(row: QualityRow): boolean {
    const view = detail.data;
    if (!view || detail.error || busy || unknown || !configured || !canEvaluate) return false;
    const current = view.rows.find(item => item.item.id === row.item.id && item.requestId === row.requestId);
    return !!current && canRepeatQualityRow(view.run.status, current) && (view.run.dataset_key.startsWith('custom-') || available.some(d => d.key === view.run.dataset_key && d.sha256 === view.run.dataset_sha256 && d.dataset.cases.some(c => c.id === current.item.id)));
  }
  function repeat(row: QualityRow) {
    const view = detail.data;
    if (!view || !canRepeatRow(row)) return;
    const source = available.find(d => d.key === view.run.dataset_key && d.sha256 === view.run.dataset_sha256);
    if (source) void run(source, view.run.group_key, [row.item.id]);
    else if (view.run.dataset_key.startsWith('custom-')) void importQualityDataset(view.run.snapshot).then(value => {
      if (guard.isCurrent()) void run(value, view.run.group_key, [row.item.id]);
    }).catch(error => { if (guard.isCurrent()) setNotice(error instanceof Error ? error.message : '原题集无法恢复'); });
  }
  function refresh() {
    if (!guard.isCurrent()) return;
    void kb.refetch(); void catalogues.refetch(); void history.refetch(); if (selectedId) void detail.refetch();
  }

  async function importRaw(raw: string, group?: string, ids?: string[]) {
    if (busy || unknown || !guard.isCurrent()) return;
    setImporting(true);
    try {
      const value = await importQualityDataset(raw);
      if (!guard.isCurrent()) return;
      setImported(value); setDatasetKey(value.key); setGroupKey(group ?? value.dataset.cases[0].kb); setSelectedCases(ids ?? null);
      setNotice('已导入冻结题集，尚未开始分析。请核对预览后明确开始评测。');
    } catch (error) { if (guard.isCurrent()) setNotice(error instanceof Error ? error.message : '题集导入失败'); }
    finally { if (guard.isCurrent()) setImporting(false); }
  }

  async function importFile(file: File | undefined) {
    if (!file || busy || unknown || !guard.isCurrent()) return;
    if (file.size > 262_144) { setNotice('冻结题集最多256 KiB，请缩小资料范围'); return; }
    try { await importRaw(decodeCustomDataset(await file.arrayBuffer())); }
    catch { if (guard.isCurrent()) setNotice('文件不是有效UTF-8题集'); }
  }

  function reuseSnapshot() {
    const view = detail.data;
    if (!view || detail.error || busy || unknown || !guard.isCurrent()) return;
    const builtIn = catalogues.data?.find(value => value.key === view.run.dataset_key && value.sha256 === view.run.dataset_sha256 && value.raw === view.run.snapshot);
    if (builtIn) {
      setDatasetKey(builtIn.key); setGroupKey(view.run.group_key); setSelectedCases(view.rows.map(row => row.item.id)); setNotice('已选择此批次的冻结题集，尚未重新运行。');
    } else void importRaw(view.run.snapshot, view.run.group_key, view.rows.map(row => row.item.id));
  }

  return <div className={styles.page}>
    <div className={styles.heading}><div><Link to={`/ai/knowledge/${kbId}`}><Button type="link" icon={<ArrowLeftOutlined />}>返回知识库</Button></Link><Typography.Title level={3}>{kb.data?.kb.name ?? '知识库'} · 质量验收</Typography.Title><Typography.Text type="secondary">核对答案是否完整、有据，以及本次分析花了多久。</Typography.Text></div><Button icon={<ReloadOutlined />} disabled={busy} onClick={refresh}>刷新</Button></div>
    <QueryState isPending={kb.isPending} error={kb.error} onRetry={() => kb.refetch()} />
    {kb.data && !kb.error && <>
      {notice && <Alert type="info" showIcon title={notice} />}
      {storageNotice && <Alert type="warning" showIcon title={storageNotice} />}
      {unknown && <Alert type="warning" showIcon title="上次提交结果待确认" description={<div className={styles.unknown}><span>先核对原批次，页面不会自动重新运行。</span>{attempt && <Typography.Text copyable>请求标识：{attempt.request_id}</Typography.Text>}</div>}
        action={<Space wrap>{attempt ? <><Button disabled={busy} onClick={() => void lookup()}>核对原请求</Button><Button disabled={busy || !canEvaluate} onClick={() => void continueOriginal()}>使用原标识继续</Button></> : <Button disabled={busy} onClick={() => { discardAttempt(); setNotice('已清除本地恢复提示。请先在历史中核对已提交批次，再开始新的评测。'); }}>已核对历史，清除本地提示</Button>}</Space>} />}
      <Card title="开始一批评测">
        <Space wrap><label className={styles.field}>导入冻结题集 JSON<input type="file" accept=".json,application/json" aria-label="导入冻结题集JSON" disabled={busy || unknown} onChange={event => { const file = event.target.files?.[0]; event.target.value = ''; void importFile(file); }} /></label>
          {imported && <><Button disabled={busy || unknown} onClick={() => { setImported(null); setDatasetKey(''); setGroupKey(''); setSelectedCases(null); }}>移除导入题集</Button><Button onClick={() => { if (guard.isCurrent()) downloadQualityDataset(imported.raw, `${imported.key}.json`); }}>下载当前冻结题集</Button></>}</Space>
        <QueryState isPending={catalogues.isPending} error={catalogues.error} onRetry={() => catalogues.refetch()} />
        {available.length === 0 && !catalogues.isPending && <Empty description="暂无可用题集，可导入冻结JSON" />}
        {chosen && <>
          <div className={styles.setup}>
            <label className={styles.field}>选择题集<Select aria-label="选择评测题集" value={chosen.key} disabled={busy || unknown} options={available.map(d => ({ value: d.key, label: d.name }))} onChange={value => { setDatasetKey(value); setGroupKey(''); setSelectedCases(null); }} /></label>
            <label className={styles.field}>资料对应题组<Select aria-label="选择资料题组" value={groupKey || undefined} placeholder="选择与当前资料对应的题组" disabled={busy || unknown} options={[...new Set(chosen.dataset.cases.map(c => c.kb))].map(value => ({ value, label: qualityGroupLabel(value) }))} onChange={value => { setGroupKey(value); setSelectedCases(null); }} /></label>
          </div>
          <div className={styles.selection}>
            <Typography.Paragraph type="secondary">{chosen.key.startsWith('custom-') ? '当前为自定义冻结题集；采纳反馈不等于质量签署，预期答案与检查项只供评分预览，不送给模型。' : '题目基于仓库实际项目文档，由AI辅助编制，并非真实用户历史。初始预留题一经试跑即为已使用数据。'}开始前会核对当前资料的正文、版本与就绪状态。</Typography.Paragraph>
            {chosen.key.startsWith('custom-') && <FeedbackDatasetPreview catalogue={chosen} />}
            {!!groupKey && <><Space wrap><Button size="small" disabled={busy || unknown} onClick={() => setSelectedCases(groupCases.filter(c => c.split === 'development').map(c => c.id))}>选择开发题</Button><Button size="small" disabled={busy || unknown} onClick={() => setSelectedCases(groupCases.map(c => c.id))}>选择全部题</Button><Typography.Text>已选 {caseIds.length} 题</Typography.Text></Space>
              <Table<QualityCase> size="small" rowKey="id" dataSource={groupCases} pagination={{ pageSize: 6, showSizeChanger: false }}
                rowSelection={{ selectedRowKeys: caseIds, onChange: values => setSelectedCases(values.map(String)), getCheckboxProps: () => ({ disabled: busy || unknown }) }}
                columns={[{ title: '问题', dataIndex: 'query' }, { title: '题目范围', width: 130, render: (_, item) => <Tag>{item.split === 'development' ? '开发题' : '初始预留题'}</Tag> }]} />
            </>}
            {!configured && <Alert type="warning" title="检索或分析模型尚未配置，请联系管理员。" />}
            {!canEvaluate && <Alert type="info" title="当前可查看结果，需要质量验收运行权限才能开始或停止评测。" />}
            <Space wrap><Button type="primary" aria-label="开始评测" icon={<PlayCircleOutlined />} loading={start.isPending} disabled={busy || unknown || !canEvaluate || !configured || !caseIds.length || caseIds.length > 30} onClick={() => void run(chosen, groupKey, caseIds)}>开始评测</Button><Typography.Text type="secondary">本批最多 {caseIds.length} 次分析、{caseIds.length * 2} 次查询 Embedding；执行失败后停止余题。</Typography.Text></Space>
          </div>
        </>}
      </Card>
      <Card title="历史批次" extra={<Select aria-label="筛选评测批次状态" value={historyStatus} disabled={busy} options={[{ value: 'all', label: '全部状态' }, ...Object.entries(qualityStatusLabels).map(([value, label]) => ({ value, label }))]} onChange={value => { setHistoryStatus(value); setPage(1); }} />}>
        <QueryState isPending={history.isPending} error={history.error} onRetry={() => history.refetch()} />
        {history.data && !history.error && <><Table<AiKbEvaluationInfo> rowKey="id" size="small" dataSource={history.data.list} pagination={false} scroll={{ x: 750 }} columns={[
          { title: '批次', dataIndex: 'id', width: 80, render: id => `#${id}` },
          { title: '题集', key: 'dataset', render: (_, r) => <div>{catalogues.data?.find(d => d.key === r.dataset_key)?.name ?? '历史题集'}<div><Typography.Text type="secondary">{qualityGroupLabel(r.group_key)}</Typography.Text></div></div> },
          { title: '状态', key: 'status', render: (_, r) => <Tag color={r.status === 'failed' || r.status === 'unknown' ? 'error' : undefined}>{qualityStatusLabels[r.status]}{r.stop_requested && r.status === 'running' ? ' · 停止中' : ''}</Tag> },
          { title: '进度', key: 'progress', render: (_, r) => `${r.completed_count + r.failed_count}/${r.case_count}` },
          { title: '创建时间', dataIndex: 'created_at', render: value => formatTime(value) },
          { title: '操作', key: 'action', render: (_, r) => <Space><Button size="small" aria-label={`查看批次 #${r.id}`} disabled={busy} onClick={() => selectRun(r.id)}>查看</Button><Button size="small" disabled={busy || r.id === selectedId || r.status !== 'succeeded' || !r.corpus_verified} onClick={() => setBaselineId(r.id)}>设为对照</Button></Space> },
        ]} /><Pagination current={page} pageSize={10} total={history.data.total} showSizeChanger={false} onChange={setPage} disabled={busy} showTotal={total => `共 ${total} 批`} /></>}
      </Card>
      {selectedId && <Card title={`批次 #${selectedId}`} extra={<Space wrap><Button disabled={busy || unknown || !detail.data || !!detail.error} onClick={reuseSnapshot}>使用此批次冻结题集</Button><Button disabled={busy} onClick={() => void detail.refetch()}>刷新状态</Button>{detail.data && ['pending', 'running'].includes(detail.data.run.status) && <Button danger disabled={busy || !canEvaluate || detail.data.run.stop_requested} loading={stop.isPending} onClick={() => void stopSelected()}>停止后续题目</Button>}</Space>}>
        <QueryState isPending={detail.isPending} error={detail.error} onRetry={() => detail.refetch()} />
        {detail.data && !detail.error && <>
          <div className={styles.toolbar}><Select aria-label="筛选评测题目" value={filter} onChange={setFilter} options={[{ value: 'all', label: '全部题目' }, { value: 'review', label: '规则提示待复核' }, { value: 'execution', label: '执行未完成' }, { value: 'waiting', label: '未执行或执行中' }]} />{baselineId && <Button disabled={busy} onClick={() => setBaselineId(null)}>取消对照 #{baselineId}</Button>}</div>
          {baselineId && baselineId !== selectedId && <><QueryState isPending={baseline.isPending} error={baseline.error} onRetry={() => baseline.refetch()} />{baseline.data && !baseline.error && <KnowledgeQualityComparison baseline={baseline.data} candidate={detail.data} />}</>}
          <KnowledgeQualityResults view={detail.data} scope={scope} guard={guard} filter={filter} canRepeat={canRepeatRow} onRepeat={repeat} onTask={setTaskId} />
          <Typography.Paragraph type="secondary">模型执行成功仍需复核内容。停止只影响未开始题目，已发生的调用不会撤回。</Typography.Paragraph>
        </>}
      </Card>}
      {taskId && <TaskDetail scope={scope} taskId={taskId} guard={guard} onClose={() => setTaskId(null)} />}
    </>}
  </div>;
}
