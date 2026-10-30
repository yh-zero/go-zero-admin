import { useState } from 'react';
import { App, Button, Card, Form, Input, Popconfirm, Select, Space, Table } from 'antd';
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { deleteUser, fetchUsers } from '../../api/user';
import { fetchDeptTree } from '../../api/dept';
import type { GetUserListReq, UserInfo } from '../../api/types';
import { Auth } from '../../components/Auth';
import { PageHeader } from '../../components/PageHeader';
import { QueryState } from '../../components/QueryState';
import { StatusTag } from '../../components/StatusTag';
import { usePagedQuery } from '../../hooks/usePagedQuery';
import { UserDetails } from './UserDetails';
import { UserEditor } from './UserEditor';
import { ResetPasswordDialog } from './ResetPasswordDialog';

export default function UserList() {
  const [filterForm] = Form.useForm<GetUserListReq>();
  const { message } = App.useApp();
  const [params, setParams] = useState<GetUserListReq>({ page: 1, page_size: 20 });
  const [editor, setEditor] = useState<{ id?: number } | null>(null);
  const [detailId, setDetailId] = useState<number | null>(null);
  const [resetTarget, setResetTarget] = useState<UserInfo | null>(null);
  const client = useQueryClient();
  const users = usePagedQuery({ params, setParams, queryKey: ['users', params], queryFn: () => fetchUsers(params) });
  const departments = useQuery({ queryKey: ['depts', {}], queryFn: () => fetchDeptTree() });
  const remove = useMutation({
    mutationFn: deleteUser,
    onSuccess: async () => { await client.invalidateQueries({ queryKey: ['users'] }); message.success('用户已删除'); },
  });
  const submitFilters = (values: GetUserListReq) => setParams({ ...values, page: 1, page_size: params.page_size });
  return <Space orientation="vertical" size="large" className="page-stack">
    <PageHeader title="用户管理" description="管理用户资料、部门和账号状态" extra={<Auth code="sys:user:add">
      <Button type="primary" icon={<PlusOutlined />} onClick={() => setEditor({})}>新增用户</Button>
    </Auth>} />
    <Card><Form name="user-filter" form={filterForm} layout="inline" onFinish={submitFilters}>
      <Form.Item name="username" label="用户名"><Input allowClear placeholder="搜索用户名" /></Form.Item>
      <Form.Item name="nickname" label="昵称"><Input allowClear placeholder="搜索昵称" /></Form.Item>
      <Form.Item name="dept_id" label="部门"><Select className="filter-select" allowClear placeholder="全部部门" popupMatchSelectWidth={false}
        options={departments.data?.list.map((dept) => ({ label: dept.name, value: dept.id }))} showSearch optionFilterProp="label" /></Form.Item>
      <Form.Item name="status" label="状态"><Select className="filter-select" allowClear placeholder="全部" options={[{ label: '启用', value: 1 }, { label: '停用', value: 2 }]} /></Form.Item>
      <Form.Item><Space><Button type="primary" htmlType="submit">查询</Button><Button onClick={() => {
        filterForm.resetFields(); setParams({ page: 1, page_size: params.page_size });
      }}>重置</Button><Button aria-label="刷新用户列表" icon={<ReloadOutlined />} onClick={() => void users.refetch()} /></Space></Form.Item>
    </Form></Card>
    <Card>
      <QueryState isPending={users.isPending} error={users.error} onRetry={() => void users.refetch()} />
      <Table<UserInfo> rowKey="id" dataSource={users.data?.list ?? []} loading={users.isFetching} scroll={{ x: 1100 }}
        pagination={{ current: params.page, pageSize: params.page_size, total: users.data?.total ?? 0, showSizeChanger: true,
          showTotal: (total) => `共 ${total} 位用户`, onChange: (page, page_size) => setParams({ ...params, page, page_size }) }} columns={[
          { title: '用户名', dataIndex: 'username', key: 'username', render: (value: string, row) => <Button type="link" onClick={() => setDetailId(row.id)}>{value}</Button> },
          { title: '昵称', dataIndex: 'nickname', key: 'nickname', render: (value: string) => value || '—' },
          { title: '真实姓名', dataIndex: 'real_name', key: 'real_name', render: (value: string) => value || '—' },
          { title: '邮箱', dataIndex: 'email', key: 'email', render: (value: string) => value || '—' },
          { title: '手机号', dataIndex: 'phone', key: 'phone', render: (value: string) => value || '—' },
          { title: '状态', dataIndex: 'status', key: 'status', render: (status: number) => <StatusTag status={status} /> },
          { title: '操作', key: 'actions', width: 290, render: (_, row) => <Space wrap>
            <Button type="link" onClick={() => setDetailId(row.id)}>详情</Button>
            <Auth code="sys:user:edit"><Button type="link" onClick={() => setEditor({ id: row.id })}>编辑</Button></Auth>
            <Auth code="sys:user:reset-password"><Button type="link" onClick={() => setResetTarget(row)}>重置密码</Button></Auth>
            <Auth code="sys:user:delete"><Popconfirm title={`确认删除用户 ${row.username}？`} description="删除后该用户将从列表移除。"
              onConfirm={() => remove.mutateAsync(row.id).catch(() => undefined)} okButtonProps={{ loading: remove.isPending }}>
              <Button danger type="link" disabled={remove.isPending}>删除</Button>
            </Popconfirm></Auth>
          </Space> },
        ]} />
    </Card>
    {editor && <UserEditor key={editor.id ?? 'new'} id={editor.id} depts={departments.data?.list ?? []} onClose={() => setEditor(null)} />}
    {detailId !== null && <UserDetails id={detailId} onClose={() => setDetailId(null)} />}
    {resetTarget && <ResetPasswordDialog user={resetTarget} onClose={() => setResetTarget(null)} />}
  </Space>;
}
