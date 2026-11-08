import { useState } from 'react';
import { Alert, Button, Drawer, Empty, Modal, Pagination, Space, Table, Tag, Typography } from 'antd';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { restoreAiWorkflowRevision } from '../../api/ai';
import type { AiWorkflowInfo, AiWorkflowRevisionInfo } from '../../api/types';
import { QueryState } from '../../components/QueryState';
import { useAuthStore } from '../../stores/auth';
import { ApiError } from '../../utils/response';
import { formatTime } from '../../utils/format';
import { aiWorkspaceKeys, workflowRevisionDetailOptions, workflowRevisionListOptions } from './workspaceQueries';
import type { AiWorkspaceScope } from './workspaceQueries';
import type { useAiWorkspaceGuard } from './useAiWorkspaceGuard';
import { WorkflowDefinitionView } from './WorkflowEditor';
import { clearWorkflowRestoreAttempt, readWorkflowRestoreAttempt, workflowRestoreAttemptKey, writeWorkflowRestoreAttempt } from './workflowRestoreAttempt';
import type { WorkflowRestoreAttempt } from './workflowRestoreAttempt';
import styles from './workflow.module.css';

interface Props {
  scope: AiWorkspaceScope; workflow: AiWorkflowInfo; currentVerified: boolean;
  guard: ReturnType<typeof useAiWorkspaceGuard>; onClose: () => void; onRefresh: () => Promise<unknown>;
}

export function WorkflowHistory({ scope, workflow, currentVerified, guard, onClose, onRefresh }: Props) {
  const client = useQueryClient();
  const departmentId = useAuthStore(state => state.selectedDeptId);
  const key = workflowRestoreAttemptKey(scope[0], departmentId, workflow.id);
  const [restored] = useState(() => {
    try { return { attempt: readWorkflowRestoreAttempt(sessionStorage, key), error: '' }; }
    catch { return { attempt: null, error: '原恢复标识无法读取，请先核对当前版本和历史。' }; }
  });
  const [attempt, setAttempt] = useState<WorkflowRestoreAttempt | null>(restored.attempt);
  const [unknown, setUnknown] = useState(!!restored.attempt || !!restored.error);
  const [notice, setNotice] = useState(restored.error);
  const [page, setPage] = useState(1);
  const [selected, setSelected] = useState<number | null>(null);
  const [confirmation, setConfirmation] = useState<WorkflowRestoreAttempt | null>(null);
  const list = useQuery({ ...workflowRevisionListOptions(scope, workflow.id, { page, page_size: 10 }), enabled: guard.can('sys:ai:workflow:list') });
  const detail = useQuery({ ...workflowRevisionDetailOptions(scope, workflow.id, selected ?? 0), enabled: !!selected && guard.can('sys:ai:workflow:list') });
  const mutation = useMutation({ mutationFn: (request: WorkflowRestoreAttempt) => restoreAiWorkflowRevision(workflow.id, request.revision, { expected_revision: request.expected_revision }), retry: false });
  const record = detail.data?.revision;
  const matching = !!record && record.workflow_id === workflow.id && record.revision === selected;
  const validCurrent = currentVerified && Number.isSafeInteger(workflow.revision) && workflow.revision >= 1;
  const canRestore = guard.can('sys:ai:workflow:edit') && validCurrent && matching && !detail.isFetching && !detail.error && !unknown;
  const busy = guard.writing || mutation.isPending;

  function clearAttempt() {
    setAttempt(null); setUnknown(false);
    try { clearWorkflowRestoreAttempt(sessionStorage, key); }
    catch { setNotice('已核对的原恢复标识暂未从存储清除；刷新后仍需核对历史，不会自动重发。'); }
  }
  async function refresh() {
    if (!guard.isCurrent() || busy) return;
    await Promise.allSettled([onRefresh(), list.refetch(), ...(selected ? [detail.refetch()] : [])]);
  }
  async function restore(request: WorkflowRestoreAttempt) {
    if (unknown || !guard.beginWrite('sys:ai:workflow:edit')) return;
    setAttempt(request); setUnknown(true); setNotice('');
    try { writeWorkflowRestoreAttempt(sessionStorage, key, request); }
    catch { setNotice('原目标和条件版本只保留在当前标签页内存中，刷新前请核对本次恢复。'); }
    try {
      const response = await mutation.mutateAsync(request);
      if (!guard.isCurrent()) return;
      if (response.workflow.id !== workflow.id || response.workflow.revision !== request.expected_revision + 1) throw new Error('恢复响应版本不一致');
      clearAttempt();
      client.setQueryData(aiWorkspaceKeys.workflow(scope, workflow.id), response);
      setNotice(`已恢复为版本 ${response.workflow.revision}。当前编辑草稿保留，请核对最新版本后继续编辑。`);
    } catch (error) {
      if (!guard.isCurrent()) return;
      if (error instanceof ApiError && error.code < 500) {
        clearAttempt();
        setNotice(error.code === 409 ? '当前版本已变化，本次恢复未覆盖它。请刷新并核对后重新选择。' : '本次恢复未完成，请根据请求提示处理后重新选择。');
      } else setNotice('恢复结果尚未确认，原目标和条件版本已保留。请核对最新版本及历史，页面不会自动重复恢复。');
    } finally {
      if (guard.isCurrent()) {
        setConfirmation(null);
        await Promise.allSettled([
          client.invalidateQueries({ queryKey: aiWorkspaceKeys.workflow(scope, workflow.id), exact: true }),
          client.invalidateQueries({ queryKey: aiWorkspaceKeys.workflowRevisions(scope, workflow.id) }),
          client.invalidateQueries({ queryKey: aiWorkspaceKeys.workflows(scope) }),
        ]);
      }
      guard.finishWrite();
    }
  }

  return <Drawer open title={`版本历史 · ${workflow.name}`} size={900} onClose={() => { if (!busy) onClose(); }} closable={!busy} keyboard={!busy} maskClosable={!busy}>
    <Space orientation="vertical" className={styles.fullWidth} size="middle">
      <Space wrap><Typography.Text>当前版本：{workflow.revision}</Typography.Text><Button disabled={busy} onClick={() => void refresh()}>刷新版本与历史</Button></Space>
      <Alert type="info" title="历史保存提交前的完整版本；恢复会创建新版本。" description="恢复不会执行任务、审批或工具，也不会覆盖已创建任务的原快照。编辑器中的未保存草稿仍会保留。" />
      {notice && <Alert type="info" showIcon title={notice} />}
      {unknown && <Alert type="warning" showIcon title="上次恢复结果待核对" description={attempt ? `原目标版本 ${attempt.revision}，原条件版本 ${attempt.expected_revision}。请先核对当前版本及历史。` : '原恢复标识不可读，请先核对历史。'}
        action={<Button disabled={busy || !validCurrent} onClick={clearAttempt}>已核对，允许后续操作</Button>} />}
      <QueryState isPending={list.isPending} error={list.error} onRetry={() => list.refetch()} />
      {list.data && !list.error && <><Table<AiWorkflowRevisionInfo> rowKey="id" size="small" pagination={false} dataSource={list.data.list} scroll={{ x: 600 }}
        locale={{ emptyText: <Empty description="尚无已归档版本；第一次修改后会出现历史。" /> }} columns={[
          { title: '版本', dataIndex: 'revision', render: value => `版本 ${value}` },
          { title: '名称', dataIndex: 'name' },
          { title: '状态', dataIndex: 'enabled', render: value => <Tag>{value === 1 ? '已启用' : '已停用'}</Tag> },
          { title: '步骤', dataIndex: 'step_count' },
          { title: '归档时间', dataIndex: 'created_at', render: value => formatTime(value) },
          { title: '操作', render: (_, row) => <Button size="small" disabled={busy} onClick={() => { setSelected(row.revision); setNotice(''); }}>查看版本 {row.revision}</Button> },
        ]} /><Pagination current={page} pageSize={10} total={list.data.total} showSizeChanger={false} disabled={busy} onChange={setPage} /></>}
      {selected && <section><QueryState isPending={detail.isPending} error={detail.error} onRetry={() => detail.refetch()} />
        {matching && record && !detail.error && <>
          <WorkflowDefinitionView name={`版本 ${record.revision} · ${record.name}`} definition={record.definition} />
          <Button danger disabled={busy || !canRestore} onClick={() => setConfirmation({ revision: record.revision, expected_revision: workflow.revision })}>恢复版本 {record.revision}</Button>
          {!guard.can('sys:ai:workflow:edit') && <Typography.Paragraph type="secondary">需要编辑工作流权限才能恢复。</Typography.Paragraph>}
        </>}
      </section>}
    </Space>
    <Modal open={!!confirmation} title="确认恢复历史版本" okText="确认恢复为新版本" cancelText="继续核对" confirmLoading={busy}
      onOk={() => { if (confirmation) void restore(confirmation); }} onCancel={() => { if (!busy) setConfirmation(null); }}
      okButtonProps={{ danger: true, disabled: busy || !canRestore }} cancelButtonProps={{ disabled: busy }} closable={!busy} keyboard={!busy} mask={{ closable: !busy }}>
      <p>将版本 {confirmation?.revision} 的名称、完整定义和启用状态恢复为新版本。</p><p>仅当当前版本仍为 {confirmation?.expected_revision} 时提交；不会执行任何任务。当前未保存草稿继续保留。</p>
    </Modal>
  </Drawer>;
}
