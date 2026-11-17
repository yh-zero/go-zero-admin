import { useEffect, useMemo, useState } from 'react';
import { Alert, Button, Pagination, Select, Space, Typography } from 'antd';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { fetchAiKb } from '../../api/ai';
import { aiWorkspaceKeys, kbListOptions } from './workspaceQueries';
import type { AiWorkspaceScope } from './workspaceQueries';
import { createKnowledgeSearchDebouncer } from './knowledgeSearch';
import styles from './workflow.module.css';

export interface WorkflowKnowledgeAccess { scope: AiWorkspaceScope; canRead: boolean }

export function KnowledgeChoice({ value, onChange, knowledge, optional, disabled }: { value?: unknown; onChange?: (value: unknown) => void; knowledge?: WorkflowKnowledgeAccess; optional?: boolean; disabled?: boolean }) {
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState('');
  const client = useQueryClient();
  const scope: AiWorkspaceScope = knowledge?.scope ?? ['', 0, 0];
  const options = kbListOptions(scope, { page, page_size: 20, query: search });
  const [sessionId, sessionVersion, departmentVersion] = scope;
  const debouncer = useMemo(() => {
    const key = kbListOptions([sessionId, sessionVersion, departmentVersion], { page, page_size: 20, query: search }).queryKey;
    return createKnowledgeSearchDebouncer(value => { setSearch(value); setPage(1); }, () => {
      // 同一流程可能同时展示多个选择器；关闭一个不能中断其他选择器共用的读取。
      const shared = client.getQueryCache().find({ queryKey: key, exact: true });
      if ((shared?.getObserversCount() ?? 0) <= 1) void client.cancelQueries({ queryKey: key, exact: true });
    });
  }, [client, sessionId, sessionVersion, departmentVersion, page, search]);
  useEffect(() => () => debouncer.cancel(), [debouncer]);
  const query = useQuery({ ...options, enabled: !!knowledge?.canRead });
  const selectedId = typeof value === 'number' && Number.isSafeInteger(value) && value > 0 ? value : 0;
  const selectedInPage = query.data?.list.find(kb => kb.id === selectedId);
  const selected = useQuery({ queryKey: [...aiWorkspaceKeys.kbs(knowledge?.scope ?? ['', 0, 0]), 'editor-choice', selectedId], enabled: !!knowledge?.canRead && query.isSuccess && selectedId > 0 && !selectedInPage, staleTime: 30_000, retry: false,
    queryFn: async ({ signal }) => { signal.throwIfAborted(); const data = await fetchAiKb(selectedId, signal); signal.throwIfAborted(); return data; } });
  if (!knowledge?.canRead) return <Alert type="warning" showIcon title="需要查看知识库权限，才能选择自己的知识库。请联系管理员授权。" />;
  return <Space orientation="vertical" className={styles.fullWidth}>
    <Select aria-label="知识库" className={styles.fullWidth} value={typeof value === 'number' ? value : optional ? 0 : undefined} allowClear={optional} disabled={disabled || query.isError} loading={query.isFetching} placeholder="按名称搜索知识库" onChange={onChange}
      showSearch={{ filterOption: false, onSearch: debouncer.search }}
      onOpenChange={open => { if (!open) debouncer.cancel(); else if (!query.data) void query.refetch(); }}
      options={[...(optional ? [{ value: 0, label: '全部本人知识库' }] : []), ...(query.data?.list ?? []).map(kb => ({ value: kb.id, label: `${kb.name} · ${kb.ready_count} 篇可使用` }))]}
      labelRender={option => query.data?.list.find(kb => kb.id === option.value)?.name ?? (option.value === 0 ? '全部本人知识库' : selected.data?.kb.name ?? (selected.isError ? '原知识库不可用，请重新选择' : '正在核对已选知识库…'))}
      notFoundContent={query.isError ? '知识库加载失败' : query.isFetching ? '正在加载…' : '暂无知识库，请先在知识库页面上传资料'}
      popupRender={menu => <>{menu}{(query.data?.total ?? 0) > 20 && <div className={styles.choicePagination} onMouseDown={event => event.preventDefault()}><Pagination simple size="small" current={page} pageSize={20} total={query.data?.total} onChange={setPage} /></div>}</>} />
    {query.isError && <Alert type="warning" title="知识库加载失败，现有选择已保留。" action={<Button size="small" onClick={() => void query.refetch()}>重试</Button>} />}
    {selectedId > 0 && !selectedInPage && selected.isError && <Alert type="warning" title="无法读取已选知识库，可能已删除或无权访问。原选择已保留，请重新选择或重试。" action={<Button size="small" onClick={() => void selected.refetch()}>重试</Button>} />}
    {query.isSuccess && query.data.total === 0 && <Typography.Text type="secondary">还没有知识库，请先到知识库页面添加资料，再回来继续编辑。</Typography.Text>}
  </Space>;
}
