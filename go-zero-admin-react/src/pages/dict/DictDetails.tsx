import { Descriptions, Drawer, Table } from 'antd';
import { useQuery } from '@tanstack/react-query';
import { fetchDictByCode } from '../../api/dict';
import type { DictItem } from '../../api/types';
import { QueryState } from '../../components/QueryState';
import { StatusTag } from '../../components/StatusTag';
import { formatTime } from '../../utils/format';

interface DictDetailsProps { code: string | null; onClose: () => void }

export function DictDetails({ code, onClose }: DictDetailsProps) {
  const query = useQuery({ queryKey: ['dicts', 'detail', code], queryFn: () => fetchDictByCode(code!), enabled: code !== null });
  const dict = query.data;
  return <Drawer title="字典详情" open={code !== null} onClose={onClose} size="large">
    <QueryState isPending={query.isPending} error={query.error} onRetry={() => { void query.refetch(); }} />
    {dict && !query.error && <>
      <Descriptions bordered column={1} items={[
        { key: 'name', label: '名称', children: dict.name },
        { key: 'code', label: '编码', children: dict.code },
        { key: 'description', label: '描述', children: dict.description || '—' },
        { key: 'status', label: '状态', children: <StatusTag status={dict.status} /> },
        { key: 'created', label: '创建时间', children: formatTime(dict.created_at) },
        { key: 'updated', label: '更新时间', children: formatTime(dict.updated_at) },
      ]} />
      <Table<DictItem> rowKey={(item) => JSON.stringify([item.value, item.label, item.sort])} dataSource={dict.items ?? []} pagination={false}
        columns={[{ title: '显示文字', dataIndex: 'label' }, { title: '值', dataIndex: 'value' }, { title: '排序', dataIndex: 'sort' }]} />
    </>}
  </Drawer>;
}
