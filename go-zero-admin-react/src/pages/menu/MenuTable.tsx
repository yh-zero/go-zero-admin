import { Button, Popconfirm, Space, Table, Tag } from 'antd';
import { DeleteOutlined, EditOutlined, EyeInvisibleOutlined, EyeOutlined, SafetyCertificateOutlined } from '@ant-design/icons';
import type { MenuInfo } from '../../api/types';
import { Auth } from '../../components/Auth';
import { StatusTag } from '../../components/StatusTag';
import { MenuExpandToggle, MenuIcon } from '../../components/MenuIcon';
import { getDescendantIds, getPagePermissions } from './menuResources';
import styles from './MenuPage.module.css';

interface MenuTableProps {
  rows: MenuInfo[]; resources: MenuInfo[]; selectedId: number | null; busy: boolean;
  onDetails: (id: number) => void; onEdit: (id: number) => void;
  onDelete: (id: number) => void; onPermissions: (id: number) => void;
}

export function MenuTable({ rows, resources, selectedId, busy, onDetails, onEdit, onDelete, onPermissions }: MenuTableProps) {
  return <Table<MenuInfo> className={styles.menuTable} rowKey="id" dataSource={rows} pagination={false} scroll={{ x: 800 }}
    expandable={{ defaultExpandAllRows: true, indentSize: 22,
      expandIcon: ({ expanded, expandable, record, onExpand }) => <MenuExpandToggle title={record.title} expanded={expanded}
        expandable={expandable} onToggle={event => onExpand(record, event)} />,
    }}
    rowClassName={menu => [menu.type === 1 ? styles.directoryRow : '', menu.id === selectedId ? styles.selectedRow : ''].join(' ')} columns={[
      { title: '名称', dataIndex: 'title', width: 210, render: (title: string, menu) => <Space size={10} className={styles.resourceName}>
        <MenuIcon menu={menu} variant="badge" />
        <Button type="link" className={styles.resourceTitle} onClick={() => onDetails(menu.id)}>{title}</Button>
      </Space> },
      { title: '访问路径', dataIndex: 'path', ellipsis: true, render: (value: string) => value ? <span className={styles.routeChip}>{value}</span> : <span className={styles.muted}>—</span> },
      { title: '排序', dataIndex: 'sort', width: 56, render: (value: number) => <span className={styles.orderNumber}>{value}</span> },
      { title: '显示', dataIndex: 'visible', width: 88, render: (visible: boolean) => <Tag variant="filled" className={styles.visibilityTag}
        icon={visible ? <EyeOutlined /> : <EyeInvisibleOutlined />}>{visible ? '显示' : '隐藏'}</Tag> },
      { title: '状态', dataIndex: 'status', width: 80, render: (status: number) => <StatusTag status={status} /> },
      { title: '操作', key: 'actions', width: 236, fixed: 'right', render: (_, menu) => <Space size={4} className={styles.rowActions}>
        <Auth code="sys:menu:edit"><Button type="text" size="small" icon={<EditOutlined />} className={styles.editAction} disabled={busy} onClick={() => onEdit(menu.id)}>编辑</Button></Auth>
        {menu.type === 2 && <Button type="text" size="small" icon={<SafetyCertificateOutlined />} className={styles.permissionAction} disabled={busy} onClick={() => onPermissions(menu.id)}>
          操作权限 <span className={styles.permissionCount}>{getPagePermissions(resources, menu.id).length}</span>
        </Button>}
        <Auth code="sys:menu:delete"><Popconfirm title={`删除“${menu.title}”？`}
          description={`将删除此节点及其下级，共 ${getDescendantIds(resources, menu.id).size} 项（含操作权限），并移除相关角色授权。`}
          onConfirm={() => onDelete(menu.id)} okButtonProps={{ loading: busy }}>
          <Button type="text" size="small" icon={<DeleteOutlined />} className={styles.deleteAction} danger disabled={busy}>删除</Button>
        </Popconfirm></Auth>
      </Space> },
    ]} />;
}
