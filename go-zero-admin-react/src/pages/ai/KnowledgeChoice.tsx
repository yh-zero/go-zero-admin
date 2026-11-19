import { useEffect, useMemo, useState } from 'react';
import { Alert, Button, Pagination, Select, Space, Typography } from 'antd';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import type { AiWorkspaceScope } from './workspaceQueries';
import { createKnowledgeSearchDebouncer, kbChoiceOptions } from './knowledgeSearch';
import styles from './workflow.module.css';

export interface WorkflowKnowledgeAccess { scope: AiWorkspaceScope; canRead: boolean }

export function KnowledgeChoice({ value, onChange, knowledge, optional, disabled, label = '知识库', allLabel = '全部本人知识库' }: {
  value?: unknown; onChange?: (value: unknown) => void; knowledge?: WorkflowKnowledgeAccess;
  optional?: boolean; disabled?: boolean; label?: string; allLabel?: string;
}) {
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState('');
  const [open, setOpen] = useState(false);
  const client = useQueryClient();
  const scope: AiWorkspaceScope = knowledge?.scope ?? ['', 0, 0];
  const options = kbChoiceOptions(scope, { page, page_size: 20, query: search });
  const [sessionId, sessionVersion, departmentVersion] = scope;
  const debouncer = useMemo(() => {
    const key = kbChoiceOptions([sessionId, sessionVersion, departmentVersion], { page, page_size: 20, query: search }).queryKey;
    return createKnowledgeSearchDebouncer(value => { setSearch(value); setPage(1); }, () => {
      // 同一流程可能同时展示多个选择器；关闭一个不能中断其他选择器共用的读取。
      const shared = client.getQueryCache().find({ queryKey: key, exact: true });
      if ((shared?.getObserversCount() ?? 0) <= 1) void client.cancelQueries({ queryKey: key, exact: true });
    });
  }, [client, sessionId, sessionVersion, departmentVersion, page, search]);
  useEffect(() => () => debouncer.cancel(), [debouncer]);
  const query = useQuery({ ...options, enabled: open && !!knowledge?.canRead });
  const selectedId = typeof value === 'number' && Number.isSafeInteger(value) && value > 0 ? value : 0;
  const selectedInPage = query.data?.list.find(kb => kb.id === selectedId);
  const selected = useQuery({ ...kbChoiceOptions(scope, { id: selectedId }), enabled: !!knowledge?.canRead && selectedId > 0 && !selectedInPage });
  const selectedName = selectedInPage?.name ?? selected.data?.list.find(kb => kb.id === selectedId)?.name;
  const unavailable = selectedId > 0 && !selectedInPage && (selected.isError || selected.isSuccess && !selectedName);
  if (!knowledge?.canRead) return <Alert type="warning" showIcon title="需要查看知识库权限，才能选择自己的知识库。请联系管理员授权。" />;
  return <Space orientation="vertical" className={styles.fullWidth}>
    <Select aria-label={label} className={styles.fullWidth} value={typeof value === 'number' ? value : optional ? 0 : undefined} allowClear={optional} disabled={disabled} loading={query.isFetching} placeholder="按名称搜索知识库" onChange={next => {
      const choice = query.data?.list.find(kb => kb.id === next);
      if (choice) client.setQueryData(kbChoiceOptions(scope, { id: choice.id }).queryKey, { list: [choice], total: 1 });
      onChange?.(next);
    }}
      showSearch={{ filterOption: false, onSearch: debouncer.search }}
      onOpenChange={next => { setOpen(next); if (!next) debouncer.cancel(); }}
      options={[...(optional ? [{ value: 0, label: allLabel }] : []), ...(query.data?.list ?? []).map(kb => ({ value: kb.id, label: kb.name }))]}
      labelRender={option => option.value === 0 ? allLabel : selectedName ?? (unavailable ? '原知识库不可用，请重新选择' : '正在核对已选知识库…')}
      notFoundContent={query.isError ? '知识库加载失败' : query.isFetching ? '正在加载…' : '暂无知识库，请先在知识库页面上传资料'}
      popupRender={menu => <>{menu}{(query.data?.total ?? 0) > 20 && <div className={styles.choicePagination} onMouseDown={event => event.preventDefault()}><Pagination simple size="small" current={page} pageSize={20} total={query.data?.total} onChange={setPage} /></div>}</>} />
    {query.isError && <Alert type="warning" title="知识库加载失败，现有选择已保留。" action={<Button size="small" onClick={() => void query.refetch()}>重试</Button>} />}
    {unavailable && <Alert type="warning" title="无法读取已选知识库，可能已删除或无权访问。原选择已保留，请重新选择或重试。" action={<Button size="small" onClick={() => void selected.refetch()}>重试</Button>} />}
    {open && query.isSuccess && query.data.total === 0 && !search && <Typography.Text type="secondary">还没有知识库，可到知识库页面添加资料。</Typography.Text>}
  </Space>;
}
