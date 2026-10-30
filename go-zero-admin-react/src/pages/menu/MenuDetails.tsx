import { Descriptions, Drawer } from 'antd';
import { useQuery } from '@tanstack/react-query';
import { fetchMenu } from '../../api/menu';
import { QueryState } from '../../components/QueryState';
import { StatusTag } from '../../components/StatusTag';
import { formatTime } from '../../utils/format';
import { menuTypeOptions } from './menuOptions';

interface MenuDetailsProps { id: number | null; onClose: () => void }

export function MenuDetails({ id, onClose }: MenuDetailsProps) {
  const query = useQuery({ queryKey: ['menus', 'detail', id], queryFn: () => fetchMenu(id!), enabled: id !== null });
  const menu = query.data;
  return <Drawer title="菜单详情" open={id !== null} onClose={onClose} size="large">
    <QueryState isPending={query.isPending} error={query.error} onRetry={() => { void query.refetch(); }} />
    {menu && !query.error && <Descriptions bordered column={1} items={[
      { key: 'id', label: 'ID / 父 ID', children: `${menu.id} / ${menu.parent_id}` },
      { key: 'title', label: '标题', children: menu.title },
      { key: 'type', label: '类型', children: menuTypeOptions.find((item) => item.value === menu.type)?.label ?? menu.type },
      { key: 'name', label: '页面名称', children: menu.name || '—' },
      { key: 'path', label: '路径', children: menu.path || '—' },
      { key: 'component', label: '组件标识', children: menu.component || '—' },
      { key: 'perm_code', label: '权限码', children: menu.perm_code || '—' },
      { key: 'icon', label: '图标', children: menu.icon || '—' },
      { key: 'sort', label: '排序', children: menu.sort },
      { key: 'visible', label: '侧栏显示', children: menu.visible ? '显示' : '隐藏' },
      { key: 'status', label: '状态', children: <StatusTag status={menu.status} /> },
      { key: 'created', label: '创建时间', children: formatTime(menu.created_at) },
    ]} />}
  </Drawer>;
}
