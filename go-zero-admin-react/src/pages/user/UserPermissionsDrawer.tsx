import { useState } from 'react';
import { Alert, Button, Descriptions, Drawer, Flex, Select, Space, Table, Tag, Typography } from 'antd';
import { ReloadOutlined } from '@ant-design/icons';
import { useQuery } from '@tanstack/react-query';
import type { UserInfo, UserPermissionCheck } from '../../api/types';
import { fetchUserEffectivePermissions } from '../../api/user';
import { QueryState } from '../../components/QueryState';
import { queryKeys } from '../../utils/queryKeys';

const scopeLabels: Record<string, string> = {
  ALL: '全部用户', DEPT_AND_CHILDREN: '本部门及下级', DEPT: '本部门', SELF: '仅本人',
};

interface Props { user: UserInfo; onClose: () => void }

export function UserPermissionsDrawer({ user, onClose }: Props) {
  const [deptId, setDeptId] = useState<number>();
  const defaults = useQuery({
    queryKey: [queryKeys.user, user.id, 'permissions', 'default'],
    queryFn: ({ signal }) => fetchUserEffectivePermissions(user.id, undefined, signal),
    staleTime: 0,
    gcTime: 0,
  });
  const selected = useQuery({
    queryKey: [queryKeys.user, user.id, 'permissions', 'department', deptId],
    queryFn: ({ signal }) => fetchUserEffectivePermissions(user.id, deptId, signal),
    enabled: deptId !== undefined,
    staleTime: 0,
    gcTime: 0,
  });
  const permissions = deptId === undefined ? defaults : selected;
  const data = permissions.isError ? undefined : permissions.data;
  const options = defaults.data?.depts ?? [];
  const defaultDeptId = defaults.data?.dept_id;

  return <Drawer open title={`查看权限 · ${user.username}`} size="large" onClose={onClose}>
    <Flex vertical gap="middle">
      {defaults.data?.enabled && <Flex align="center" gap="small" wrap>
        <Typography.Text>按部门核对</Typography.Text>
        <Select aria-label="核对权限的部门" className="filter-select" value={deptId ?? 'default'}
          popupMatchSelectWidth={false} showSearch optionFilterProp="label"
          options={[{ value: 'default', label: '默认部门' },
            ...options.filter(dept => dept.id !== defaultDeptId).map(dept => ({ value: dept.id, label: dept.name }))]}
          onChange={(id: number | 'default') => setDeptId(id === 'default' ? undefined : id)} />
        <Button aria-label="刷新权限结果" icon={<ReloadOutlined />} loading={permissions.isFetching}
          onClick={() => void permissions.refetch()} />
      </Flex>}
      <QueryState isPending={permissions.isPending} error={permissions.error} onRetry={() => void permissions.refetch()} />
      {data && (!data.enabled
        ? <Alert type="warning" title="该用户已停用，当前权限不生效。" showIcon />
        : <>
          <Descriptions size="small" bordered column={1} items={[
            { key: 'dept', label: '当前核对部门', children: options.find(dept => dept.id === data.dept_id)?.name
              || (data.dept_id ? `部门 ${data.dept_id}` : '未分配') },
            { key: 'roles', label: '生效角色', children: data.roles?.length
              ? <Space size={[4, 6]} wrap>{data.roles.map(role => <Tag key={`${role.code}:${role.domain}`}>
                {role.code} · {role.domain === '*' ? '全局' : '部门'} · {scopeLabels[role.data_scope] ?? '无用户数据范围'}
              </Tag>)}</Space> : '无' },
            { key: 'scope', label: '用户数据范围', children: data.data_scopes?.length
              ? data.data_scopes.map(scope => scopeLabels[scope] ?? scope).join('、') : '无' },
            { key: 'count', label: '用户列表可见人数', children: data.visible_user_count },
            { key: 'users', label: '可见账号示例', children: data.visible_users?.length
              ? <Space size={[4, 6]} wrap>
                {data.visible_users.map(account => <Tag key={account.id}>{account.username}</Tag>)}
                <Typography.Text type="secondary">最多展示 10 个</Typography.Text>
              </Space> : '暂无可展示的账号' },
          ]} />
          <Typography.Text type="secondary">
            详情、编辑、删除和重置密码以当前被查看账号为例核对接口授权。操作具体用户时，还须检查该记录是否在数据范围内。
          </Typography.Text>
          <Table<UserPermissionCheck> size="small" rowKey="key" pagination={false} scroll={{ x: 500 }}
            dataSource={data.items} columns={[
              { title: '用户管理功能', dataIndex: 'label', key: 'label', width: 115 },
              { title: '菜单或按钮', key: 'ui', render: (_, item) => <Flex vertical gap={4}>
                <div><Tag color={item.ui_granted ? 'success' : 'default'}>{item.ui_granted ? '有授权' : '无授权'}</Tag></div>
                {item.ui_roles?.length > 0 && <Typography.Text type="secondary">{item.ui_roles.join('、')}</Typography.Text>}
              </Flex> },
              { title: '接口授权', key: 'api', render: (_, item) => <Flex vertical gap={4}>
                <div><Tag color={item.api_allowed ? 'success' : 'error'}>{item.api_allowed ? '允许' : '拒绝'}</Tag></div>
                {item.api_role && <Typography.Text type="secondary">
                  {item.api_role}{item.api_domain && (item.api_domain === '*' ? ' · 全局' : ' · 部门')}
                </Typography.Text>}
              </Flex> },
            ]} />
        </>)}
    </Flex>
  </Drawer>;
}
