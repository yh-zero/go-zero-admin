import { Alert, Button, Drawer, Popconfirm, Space, Table } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import type { MenuInfo } from '../../api/types';
import { Auth } from '../../components/Auth';
import { StatusTag } from '../../components/StatusTag';
import type { MenuEditorState } from './MenuForm';
import { getDescendantIds, getPagePermissions, getUnassignedPermissions } from './menuResources';
import styles from './MenuPage.module.css';

interface Props {
  pageId: number | null; resources: MenuInfo[]; ready: boolean; deleting: boolean;
  onClose: () => void; onEdit: (editor: MenuEditorState) => void; onDelete: (id: number) => void;
}

export function OperationPermissions({ pageId, resources, ready, deleting, onClose, onEdit, onDelete }: Props) {
  const page = resources.find(resource => resource.id === pageId && resource.type === 2);
  const unassigned = pageId === 0;
  const available = ready && (unassigned || !!page);
  const permissions = unassigned ? getUnassignedPermissions(resources) : getPagePermissions(resources, pageId ?? -1);
  return <Drawer title={unassigned ? '其他操作权限' : `操作权限 · ${page?.title ?? '页面'}`} open={pageId !== null}
    onClose={onClose} size={520} mask={false} rootClassName={styles.permissionsDrawer} styles={{ wrapper: { maxWidth: '100vw' } }}
    footer={<div className={styles.permissionFooter}><Button onClick={onClose}>关闭</Button></div>}>
    <div className={styles.permissionHeader}>
      <div className={styles.permissionContext}>
        <span>{unassigned ? '未归属于有效页面的既有操作权限' : `页面：${page?.path || '—'}`}</span>
        <span className={styles.permissionCaption}>{unassigned ? '编辑时可重新选择所属页面。' : '操作权限归属于当前页面。'}</span>
      </div>
      {!unassigned && <Auth code="sys:menu:add"><Button type="primary" icon={<PlusOutlined />} disabled={!available || deleting}
        onClick={() => onEdit({ id: null, kind: 'permission', parentId: pageId! })}>新增权限</Button></Auth>}
    </div>
    {!available && <Alert type="warning" showIcon title={ready ? '页面已不存在或类型已变更，请关闭后刷新。' : '资源正在刷新或加载失败，请刷新菜单后再操作。'} />}
    <Table<MenuInfo> className={styles.permissionTable} rowKey="id" dataSource={permissions} pagination={false} columns={[
      { title: '权限名称 / 权限码', dataIndex: 'title', render: (title: string, permission) => <div>
        <span className={styles.permissionName}>{title}</span><span className={styles.permissionCode}>{permission.perm_code || '未配置权限码'}</span>
      </div> },
      { title: '状态', dataIndex: 'status', width: 80, render: (status: number) => <StatusTag status={status} /> },
      { title: '操作', key: 'actions', width: 116, render: (_, permission) => <Space size={2}>
        <Auth code="sys:menu:edit"><Button type="link" size="small" disabled={!available || deleting}
          onClick={() => onEdit({ id: permission.id, kind: 'permission', parentId: unassigned ? undefined : pageId! })}>编辑</Button></Auth>
        <Auth code="sys:menu:delete"><Popconfirm title={`删除“${permission.title}”？`}
          description={`将移除此权限及其下级，共 ${getDescendantIds(resources, permission.id).size} 项，并移除相关角色授权。`}
          onConfirm={() => onDelete(permission.id)} okButtonProps={{ loading: deleting }}>
          <Button type="link" size="small" danger disabled={!available || deleting}>删除</Button>
        </Popconfirm></Auth>
      </Space> },
    ]} />
  </Drawer>;
}
