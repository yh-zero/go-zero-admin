import { useState } from 'react';
import { App, Button, Card, Form, Input, Popconfirm, Select, Space, Table } from 'antd';
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { deleteRole, fetchRoles } from '../../api/role';
import type { GetRoleListReq, RoleInfo } from '../../api/types';
import { Auth } from '../../components/Auth';
import { PageHeader } from '../../components/PageHeader';
import { QueryState } from '../../components/QueryState';
import { StatusTag } from '../../components/StatusTag';
import { usePagedQuery } from '../../hooks/usePagedQuery';
import { RoleEditor } from './RoleEditor';
import { RoleDetails } from './RoleDetails';
import { RoleMenuDialog } from './RoleMenuDialog';
import { dataScopeOptions } from './roleOptions';

export default function RoleList() {
  const [form] = Form.useForm<GetRoleListReq>();
  const { message } = App.useApp();
  const [params, setParams] = useState<GetRoleListReq>({ page: 1, page_size: 20 });
  const [editor, setEditor] = useState<{ id?: number } | null>(null);
  const [detailId, setDetailId] = useState<number | null>(null);
  const [menuRoleId, setMenuRoleId] = useState<number | null>(null);
  const client = useQueryClient();
  const roles = usePagedQuery({ params, setParams, queryKey: ['roles', params], queryFn: () => fetchRoles(params) });
  const remove = useMutation({
    mutationFn: deleteRole,
    onSuccess: async () => { await client.invalidateQueries({ queryKey: ['roles'] }); message.success('角色已删除'); },
  });
  return <Space orientation="vertical" className="page-stack" size="large">
    <PageHeader title="角色管理" description="配置角色资料、数据范围和菜单授权" extra={<Auth code="sys:role:add">
      <Button type="primary" icon={<PlusOutlined />} onClick={() => setEditor({})}>新增角色</Button>
    </Auth>} />
    <Card><Form name="role-filter" form={form} layout="inline" onFinish={(values) => setParams({ ...values, page: 1, page_size: params.page_size })}>
      <Form.Item name="name" label="角色名称"><Input allowClear placeholder="搜索角色名称" /></Form.Item>
      <Form.Item name="code" label="角色编码"><Input allowClear placeholder="搜索角色编码" /></Form.Item>
      <Form.Item name="status" label="状态"><Select className="filter-select" allowClear placeholder="全部" options={[
        { label: '启用', value: 1 }, { label: '停用', value: 2 },
      ]} /></Form.Item>
      <Form.Item><Space><Button type="primary" htmlType="submit">查询</Button><Button onClick={() => {
        form.resetFields(); setParams({ page: 1, page_size: params.page_size });
      }}>重置</Button><Button aria-label="刷新角色列表" icon={<ReloadOutlined />} onClick={() => void roles.refetch()} /></Space></Form.Item>
    </Form></Card>
    <Card>
      <QueryState isPending={roles.isPending} error={roles.error} onRetry={() => void roles.refetch()} />
      <Table<RoleInfo> rowKey="id" dataSource={roles.data?.list ?? []} loading={roles.isFetching} scroll={{ x: 1050 }}
        pagination={{ current: params.page, pageSize: params.page_size, total: roles.data?.total ?? 0, showSizeChanger: true,
          showTotal: (total) => `共 ${total} 个角色`, onChange: (page, page_size) => setParams({ ...params, page, page_size }) }} columns={[
          { title: '角色名称', dataIndex: 'name', key: 'name', render: (value: string, row) => <Button type="link" onClick={() => setDetailId(row.id)}>{value}</Button> },
          { title: '角色编码', dataIndex: 'code', key: 'code' },
          { title: '说明', dataIndex: 'description', key: 'description', ellipsis: true, render: (value: string) => value || '—' },
          { title: '数据范围', dataIndex: 'data_scope', key: 'data_scope', render: (value: string) => dataScopeOptions.find((item) => item.value === value)?.label ?? value },
          { title: '排序', dataIndex: 'sort', key: 'sort' },
          { title: '状态', dataIndex: 'status', key: 'status', render: (status: number) => <StatusTag status={status} /> },
          { title: '操作', key: 'actions', width: 280, render: (_, row) => <Space wrap>
            <Button type="link" onClick={() => setDetailId(row.id)}>详情</Button>
            <Auth code="sys:role:edit"><Button type="link" onClick={() => setEditor({ id: row.id })}>编辑</Button></Auth>
            <Auth code="sys:role:menus"><Button type="link" onClick={() => setMenuRoleId(row.id)}>分配菜单</Button></Auth>
            <Auth code="sys:role:delete"><Popconfirm title={`确认删除角色 ${row.name}？`} description="关联的菜单授权和权限规则也会移除。"
              onConfirm={() => remove.mutateAsync(row.id).catch(() => undefined)} okButtonProps={{ loading: remove.isPending }}>
              <Button type="link" danger disabled={remove.isPending}>删除</Button>
            </Popconfirm></Auth>
          </Space> },
        ]} />
    </Card>
    {editor && <RoleEditor key={editor.id ?? 'new'} id={editor.id} onClose={() => setEditor(null)} />}
    {detailId !== null && <RoleDetails id={detailId} onClose={() => setDetailId(null)} />}
    {menuRoleId !== null && <RoleMenuDialog id={menuRoleId} onClose={() => setMenuRoleId(null)} />}
  </Space>;
}
