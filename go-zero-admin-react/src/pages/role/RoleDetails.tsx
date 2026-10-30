import { Descriptions, Drawer, Tag } from 'antd';
import { useQuery } from '@tanstack/react-query';
import { fetchRole } from '../../api/role';
import { QueryState } from '../../components/QueryState';
import { StatusTag } from '../../components/StatusTag';
import { formatTime } from '../../utils/format';
import { dataScopeOptions } from './roleOptions';

interface Props { id: number; onClose: () => void }

export function RoleDetails({ id, onClose }: Props) {
  const role = useQuery({ queryKey: ['role', id], queryFn: () => fetchRole(id) });
  const data = role.data;
  return <Drawer open title="角色详情" onClose={onClose} size="large">
    <QueryState isPending={role.isPending} error={role.error} onRetry={() => void role.refetch()} />
    {data && <Descriptions column={1} bordered items={[
      { key: 'id', label: '角色 ID', children: data.id },
      { key: 'name', label: '角色名称', children: data.name },
      { key: 'code', label: '角色编码', children: data.code },
      { key: 'description', label: '说明', children: data.description || '—' },
      { key: 'data_scope', label: '数据范围', children: dataScopeOptions.find((item) => item.value === data.data_scope)?.label ?? data.data_scope },
      { key: 'status', label: '状态', children: <StatusTag status={data.status} /> },
      { key: 'sort', label: '排序', children: data.sort },
      { key: 'menus', label: '已分配菜单 ID', children: (data.menu_ids ?? []).length ? data.menu_ids.map((id) => <Tag key={id}>{id}</Tag>) : '未分配' },
      { key: 'created_at', label: '创建时间', children: formatTime(data.created_at) },
      { key: 'updated_at', label: '更新时间', children: formatTime(data.updated_at) },
    ]} />}
  </Drawer>;
}
