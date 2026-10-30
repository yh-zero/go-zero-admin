import { Button, Popconfirm, Space, Table, Tag } from 'antd';
import type { TableProps } from 'antd';
import type { MenuInfo } from '../../api/types';
import { Auth } from '../../components/Auth';
import { StatusTag } from '../../components/StatusTag';
import { formatTime } from '../../utils/format';
import { menuTypeOptions } from './menuOptions';

interface MenuTableProps {
  rows: MenuInfo[]; pagination: TableProps<MenuInfo>['pagination']; deleting: boolean;
  onDetails: (id: number) => void; onEdit: (id: number) => void; onDelete: (id: number) => void;
}

export function MenuTable({ rows, pagination, deleting, onDetails, onEdit, onDelete }: MenuTableProps) {
  return <Table<MenuInfo> rowKey="id" dataSource={rows} pagination={pagination} scroll={{ x: 1100 }} columns={[
    { title: '标题', dataIndex: 'title', fixed: 'left', width: 180 },
    { title: 'ID', dataIndex: 'id', width: 70 },
    { title: '类型', dataIndex: 'type', width: 90, render: (type: number) => <Tag>{menuTypeOptions.find((item) => item.value === type)?.label ?? type}</Tag> },
    { title: '页面名称', dataIndex: 'name', width: 130 },
    { title: '访问路径', dataIndex: 'path', ellipsis: true },
    { title: '权限码', dataIndex: 'perm_code', ellipsis: true },
    { title: '排序', dataIndex: 'sort', width: 70 },
    { title: '显示', dataIndex: 'visible', width: 80, render: (value: boolean) => value ? '显示' : '隐藏' },
    { title: '状态', dataIndex: 'status', width: 90, render: (status: number) => <StatusTag status={status} /> },
    { title: '创建时间', dataIndex: 'created_at', width: 170, render: formatTime },
    { title: '操作', key: 'actions', fixed: 'right', width: 210, render: (_, menu) => {
      return <Space size="small">
        <Button type="link" size="small" onClick={() => onDetails(menu.id)}>详情</Button>
        <Auth code="sys:menu:edit"><Button type="link" size="small" onClick={() => onEdit(menu.id)}>编辑</Button></Auth>
        <Auth code="sys:menu:delete"><Popconfirm title={`删除“${menu.title}”？`} description="将删除当前菜单及其全部下级，并移除相关角色菜单授权。"
            onConfirm={() => onDelete(menu.id)} okButtonProps={{ loading: deleting }}>
            <Button type="link" size="small" danger disabled={deleting}>删除</Button>
          </Popconfirm>
        </Auth>
      </Space>;
    } },
  ]} />;
}
