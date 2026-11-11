import { useState } from 'react';
import { Alert, Button } from 'antd';
import { PlayCircleOutlined } from '@ant-design/icons';
import { useAuthStore } from '../../stores/auth';
import { AsyncContent } from '../../components/AsyncContent';
import { draftRunAttemptKey } from './draftRunAttempts';
import { readDeferredRunState } from './deferredRunState';
import type { WorkflowDraftRunProps } from './WorkflowDraftRun';
import styles from './deferredWorkflowRun.module.css';

const loadRun = () => import('./WorkflowDraftRun').then(module => module.WorkflowDraftRun);

export function DeferredWorkflowRun(props: WorkflowDraftRunProps) {
  const departmentId = useAuthStore(state => state.selectedDeptId);
  const [state, setState] = useState(() => {
    try { return readDeferredRunState(typeof sessionStorage === 'undefined' ? undefined : sessionStorage, draftRunAttemptKey([props.scope[0], departmentId], props.attemptId)); }
    catch { return { kind: 'unknown' as const, taskId: null }; }
  });
  const [opened, setOpened] = useState(false);
  const [visited, setVisited] = useState(false);
  const label = props.fixedKb ? '用资料分析' : '试运行';
  return <section className={styles.panel} aria-label={label}>
    <Button icon={<PlayCircleOutlined />} aria-expanded={opened} disabled={props.guard.writing}
      onClick={() => { setVisited(true); setOpened(value => !value); }}>{opened ? `收起${props.fixedKb ? '分析' : '试运行'}` : label}</Button>
    {!opened && state.kind !== 'idle' && <Alert className={styles.notice} type={state.kind === 'unknown' ? 'warning' : 'info'} showIcon
      title={state.kind === 'unknown' ? '上次运行结果待确认，展开后核对原请求' : state.kind === 'running' ? '正在提交运行，请勿重复提交' : `已有运行记录 #${state.taskId}，展开查看结果`} />}
    {visited && <div className={styles.content} hidden={!opened}><AsyncContent load={loadRun} componentProps={{ ...props, onStateChange: setState }} /></div>}
  </section>;
}
