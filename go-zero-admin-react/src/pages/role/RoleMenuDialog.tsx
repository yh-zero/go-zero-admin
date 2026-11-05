import { useMemo, useRef, useState } from 'react';
import { Alert, App, Modal, Typography } from 'antd';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type { QueryClient } from '@tanstack/react-query';
import { fetchRole, fetchRoleMenus, updateRoleMenus } from '../../api/role';
import type { ListResult, MenuInfo, RoleInfo } from '../../api/types';
import { QueryState } from '../../components/QueryState';
import { invalidateRoleData } from '../../utils/queryInvalidation';
import { RolePermissionTable } from './RolePermissionTable';
import { groupRolePermissions, togglePermission } from './rolePermissions';
import styles from './RolePermissions.module.css';
import { queryKeys } from '../../utils/queryKeys';

interface Props { id: number; onClose: () => void }

function permissionDataReady(client: QueryClient, id: number): boolean {
  const role = client.getQueryState<RoleInfo>([queryKeys.role, id]);
  const menus = client.getQueryState<ListResult<MenuInfo>>([queryKeys.roleMenuOptions]);
  return role?.status === 'success' && role.fetchStatus === 'idle' && !role.isInvalidated && role.data?.id === id
    && menus?.status === 'success' && menus.fetchStatus === 'idle' && !menus.isInvalidated && !!menus.data;
}

export function RoleMenuDialog({ id, onClose }: Props) {
  return <RolePermissionSession key={id} id={id} onClose={onClose} />;
}

function RolePermissionSession({ id, onClose }: Props) {
  const [selection, setSelection] = useState<number[] | null>(null);
  const submitting = useRef(false);
  const { message } = App.useApp();
  const client = useQueryClient();
  const role = useQuery({ queryKey: [queryKeys.role, id], queryFn: () => fetchRole(id) });
  const menus = useQuery({ queryKey: [queryKeys.roleMenuOptions], queryFn: fetchRoleMenus });
  const checkedIds = selection ?? role.data?.menu_ids ?? [];
  const checkedSet = new Set(checkedIds);
  const groups = useMemo(() => groupRolePermissions(menus.data?.list ?? []), [menus.data]);
  const unknownCount = [...checkedSet].filter(menuId => !groups.resourceIds.has(menuId)).length;
  const available = role.isSuccess && menus.isSuccess && role.fetchStatus === 'idle' && menus.fetchStatus === 'idle' && role.data.id === id;
  const save = useMutation({
    mutationFn: (menuIds: number[]) => updateRoleMenus(id, menuIds),
    onSuccess: async () => {
      await invalidateRoleData(client);
      message.success('功能授权已更新'); onClose();
    },
    onSettled: () => { submitting.current = false; },
  });
  const onSave = () => {
    if (submitting.current || save.isPending) return;
    if (!available || !permissionDataReady(client, id)) { message.info('授权数据尚未就绪，请等待加载完成后保存。'); return; }
    submitting.current = true;
    save.mutate([...checkedIds]);
  };
  const onToggle = (menuId: number, checked: boolean) => {
    if (!available || submitting.current || !permissionDataReady(client, id)) return;
    setSelection(current => togglePermission(current ?? role.data?.menu_ids ?? [], menuId, checked));
  };
  return <Modal open width={1000} title={`功能授权${role.data ? ` · ${role.data.name}` : ''}`}
    onCancel={() => { if (!submitting.current) onClose(); }} onOk={onSave} okText="保存授权"
    confirmLoading={save.isPending} okButtonProps={{ disabled: !available || save.isPending }}
    cancelButtonProps={{ disabled: save.isPending }} closable={!save.isPending} keyboard={!save.isPending} mask={{ closable: !save.isPending }}>
    <div className="page-stack">
      <QueryState isPending={role.isPending || menus.isPending} error={role.error ?? menus.error}
        onRetry={() => { if (!submitting.current) { void role.refetch(); void menus.refetch(); } }} />
      <Alert type="info" showIcon title="目录入口、页面入口和操作权限独立勾选"
        description="勾选入口不会自动授予操作，也不会更改父级授权。此处配置页面和操作资源授权；接口访问策略另行维护。" />
      {(role.isError || menus.isError) && <Alert type="warning" showIcon title="授权数据暂时不可用，当前选择已保留。请重新加载后再保存。" />}
      {role.data && menus.data && <>
        {unknownCount > 0 && <Alert type="warning" showIcon title={`有 ${unknownCount} 项既有授权未包含在当前资源列表中`}
          description="这些授权在保存时会原样保留，不受下方勾选影响。" />}
        <div className={styles.summary}><Typography.Text strong>页面入口与操作权限</Typography.Text>
          <Typography.Text type="secondary">已选 {checkedSet.size} 项{unknownCount > 0 ? `（含 ${unknownCount} 项保留授权）` : ''}
            {role.isFetching || menus.isFetching ? ' · 正在刷新，暂不可编辑或保存' : ''}</Typography.Text></div>
        <RolePermissionTable groups={groups} checkedIds={checkedSet} disabled={!available || save.isPending} onToggle={onToggle} />
      </>}
    </div>
  </Modal>;
}
