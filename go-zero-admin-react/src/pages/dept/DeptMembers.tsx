import { useState } from 'react';
import { Alert, Avatar, Button, Card, Form, Input, Space, Table, Typography } from 'antd';
import { MailOutlined, PhoneOutlined, ReloadOutlined, SearchOutlined, TeamOutlined, UserOutlined } from '@ant-design/icons';
import { fetchUsers } from '../../api/user';
import type { DeptInfo, GetUserListReq, UserInfo } from '../../api/types';
import { QueryState } from '../../components/QueryState';
import { StatusTag } from '../../components/StatusTag';
import { usePagedQuery } from '../../hooks/usePagedQuery';
import { queryKeys } from '../../utils/queryKeys';
import { useTablePagination } from '../../hooks/useTablePagination';

interface DeptMembersProps { dept: DeptInfo; canReadUsers: boolean }

function MemberIdentity({ user }: { user: UserInfo }) {
  return <div className="dept-member-identity">
    <Avatar className={`dept-member-avatar dept-member-avatar--${user.id % 3}`} icon={<UserOutlined />} />
    <div className="dept-member-copy"><span className="dept-member-name">{user.real_name || user.nickname || user.username}</span>
      <span className="dept-member-username">{user.username}</span></div>
  </div>;
}

function MemberContact({ user }: { user: UserInfo }) {
  return <div className="dept-member-contact">
    {user.phone && <span className="dept-member-contact-line"><PhoneOutlined aria-hidden />{user.phone}</span>}
    {user.email && <span className="dept-member-contact-line"><MailOutlined aria-hidden />{user.email}</span>}
    {!user.phone && !user.email && <span className="dept-member-contact-empty">—</span>}
  </div>;
}

export function DeptMembers({ dept, canReadUsers }: DeptMembersProps) {
  const [form] = Form.useForm<Pick<GetUserListReq, 'username'>>();
  const [params, setParams] = useState<GetUserListReq>({ dept_id: dept.id, page: 1, page_size: 20 });
  const members = usePagedQuery({ params, setParams, queryKey: [queryKeys.users, 'dept-members', params],
    queryFn: () => fetchUsers(params), enabled: canReadUsers });
  const pagination = useTablePagination(params, setParams, members.data?.total ?? 0);

  return <Card className="data-card dept-members" title={<div className="dept-section-heading">
    <span className="dept-section-icon"><TeamOutlined aria-hidden /></span><span>直属成员</span>
    {canReadUsers && members.isSuccess && <span className="dept-section-count">{members.data.total}</span>}
  </div>} extra={canReadUsers && <Button type="text" size="small"
    aria-label="刷新部门成员" icon={<ReloadOutlined />} onClick={() => void members.refetch()}>刷新</Button>}>
    {!canReadUsers ? <Alert type="info" showIcon title="当前身份无成员查看权限" description="查看成员需具备用户查询权限，并受当前身份的数据范围限制。" /> : <>
      <div className="dept-member-toolbar">
        <Typography.Text type="secondary" className="dept-member-note">仅显示当前身份可查看的直属成员，含停用账号</Typography.Text>
        <Form form={form} name="dept-member-filter" layout="inline" className="dept-member-search" onFinish={values => setParams({ ...params, ...values, page: 1 })}>
          <Form.Item name="username"><Input allowClear prefix={<SearchOutlined />} aria-label="搜索直属成员" placeholder="搜索用户名" /></Form.Item>
          <Form.Item><Space><Button type="primary" htmlType="submit">查询</Button>
            <Button onClick={() => { form.resetFields(); setParams({ dept_id: dept.id, page: 1, page_size: params.page_size }); }}>重置</Button></Space></Form.Item>
        </Form>
      </div>
      <QueryState isPending={members.isPending} error={members.error} onRetry={() => void members.refetch()} />
      {members.isSuccess && <Table<UserInfo> rowKey="id" dataSource={members.data.list} loading={members.isFetching} scroll={{ x: 590 }}
        pagination={pagination} columns={[
          { title: '成员', key: 'member', width: '42%', render: (_, user) => <MemberIdentity user={user} /> },
          { title: '联系方式', key: 'contact', render: (_, user) => <MemberContact user={user} /> },
          { title: '状态', dataIndex: 'status', width: 110, render: (status: number) => <StatusTag status={status} /> },
        ]} />}
    </>}
  </Card>;
}
