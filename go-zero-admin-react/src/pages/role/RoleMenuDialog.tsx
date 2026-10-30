import { useState } from 'react';
import { Alert, App, Modal, Space, Tree } from 'antd';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { fetchRole, fetchRoleMenus, updateRoleMenus } from '../../api/role';
import { QueryState } from '../../components/QueryState';
import { buildTree } from '../../utils/tree';

interface Props { id: number; onClose: () => void }

export function RoleMenuDialog({ id, onClose }: Props) {
  const [selection, setSelection] = useState<number[] | null>(null);
  const { message } = App.useApp();
  const client = useQueryClient();
  const role = useQuery({ queryKey: ['role', id], queryFn: () => fetchRole(id) });
  const menus = useQuery({ queryKey: ['role-menu-options'], queryFn: fetchRoleMenus });
  const checkedIds = selection ?? role.data?.menu_ids ?? [];
  const save = useMutation({
    mutationFn: () => updateRoleMenus(id, checkedIds),
    onSuccess: async () => {
      await Promise.all([
        client.invalidateQueries({ queryKey: ['roles'] }),
        client.invalidateQueries({ queryKey: ['role', id] }),
        client.invalidateQueries({ queryKey: ['my-menus'] }),
        client.invalidateQueries({ queryKey: ['perm-codes'] }),
      ]);
      message.success('菜单授权已更新'); onClose();
    },
  });
  const available = !!role.data && !!menus.data && !role.isError && !menus.isError && !role.isFetching && !menus.isFetching;
  return <Modal open title={`分配菜单${role.data ? ` · ${role.data.name}` : ''}`} onCancel={onClose}
    onOk={() => save.mutate()} confirmLoading={save.isPending} okButtonProps={{ disabled: !available }}
    cancelButtonProps={{ disabled: save.isPending }} closable={!save.isPending} maskClosable={!save.isPending}>
    <Space orientation="vertical" className="page-stack">
      <QueryState isPending={role.isPending || menus.isPending} error={role.error ?? menus.error}
        onRetry={() => { void role.refetch(); void menus.refetch(); }} />
      {menus.isError && <Alert type="warning" showIcon title="菜单列表暂时不可用，当前授权已保留。请重试后再保存。" />}
      {available && (menus.data!.list.length > 0 ? <Tree checkable checkStrictly defaultExpandAll disabled={save.isPending}
        treeData={buildTree(menus.data!.list)} fieldNames={{ title: 'title', key: 'id', children: 'children' }} checkedKeys={checkedIds}
        onCheck={(keys) => setSelection((Array.isArray(keys) ? keys : keys.checked).map(Number))} />
        : <Alert type="info" showIcon title="暂无可分配菜单" description="保存空选择将清除该角色的全部菜单授权。" />)}
    </Space>
  </Modal>;
}
