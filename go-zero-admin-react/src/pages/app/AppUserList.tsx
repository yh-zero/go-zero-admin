import { useEffect, useState } from 'react';
import { Avatar, Button, Card, Empty, Form, Input, Select, Space, Table, Tag } from 'antd';
import { AuditOutlined, CheckCircleOutlined, CrownOutlined, EditOutlined, EyeOutlined, LaptopOutlined,
  MailOutlined, MobileOutlined, PictureOutlined, ReloadOutlined, SearchOutlined, StopOutlined, TeamOutlined, UndoOutlined, UserOutlined } from '@ant-design/icons';
import { useQuery } from '@tanstack/react-query';
import { fetchAppUsers } from '../../api/app';
import { fetchMyMenus } from '../../api/auth';
import type { AppAdminUserInfo, GetAppUserListReq } from '../../api/types';
import { Auth } from '../../components/Auth';
import { PageHeader } from '../../components/PageHeader';
import { QueryState } from '../../components/QueryState';
import { usePagedQuery } from '../../hooks/usePagedQuery';
import { formatTime } from '../../utils/format';
import { AppUserAvatarDialog } from './AppUserAvatarDialog';
import { AppUserDetails } from './AppUserDetails';
import { AppUserReviewDialog } from './AppUserReviewDialog';
import type { AppUserReviewIntent } from './AppUserReviewDialog';
import { AppUserEditDialog } from './AppUserEditDialog';
import { getAccessiblePagePaths } from '../../router/menuRoutes';
import { appUserStatus, appUserStatusOptions } from './appOptions';
import './app.css';
import { queryKeys } from '../../utils/queryKeys';
import { useTablePagination } from '../../hooks/useTablePagination';

type UserAction = { id: number; mode: 'details' | 'review' | 'avatar' | 'sessions' | 'memberships' | 'edit'; intent?: AppUserReviewIntent };

export default function AppUserList() {
  const [form] = Form.useForm<GetAppUserListReq>();
  const [params, setParams] = useState<GetAppUserListReq>({ page: 1, page_size: 20 });
  const [action, setAction] = useState<UserAction | null>(null);
  const menus = useQuery({ queryKey: [queryKeys.myMenus], queryFn: fetchMyMenus });
  const canReadMemberships = menus.isSuccess && getAccessiblePagePaths(menus.data).has('/app/memberships');
  const users = usePagedQuery({ params, setParams, queryKey: [queryKeys.appUsers, params], queryFn: () => fetchAppUsers(params) });
  const pagination = useTablePagination(params, setParams, users.data?.total ?? 0);
  const selectedUser = users.data?.list.find(user => user.id === action?.id);
  useEffect(() => {
    // 必须结束已失效的弹窗会话，否则用户日后重新进入筛选结果时会重开旧弹窗。
    // oxlint-disable-next-line react/set-state-in-effect
    if (action && (action.mode === 'review' || action.mode === 'avatar') && users.isSuccess && !users.isFetching && !selectedUser) setAction(null);
  }, [action, selectedUser, users.isSuccess, users.isFetching]);
  const unavailable = users.isFetching || users.isError;

  return <div className="page-stack app-page">
    <div className="app-page-heading"><span className="app-heading-icon"><TeamOutlined /></span>
      <PageHeader title="APP用户" description="管理用户资料与账号状态，让每一次访问清晰可控" /></div>
    <Card className="filter-card app-filter-card"><Form name="app-user-filter" form={form} className="filter-form" layout="inline"
      onFinish={values => setParams({ ...values, account: values.account?.trim() || undefined, page: 1, page_size: params.page_size })}>
      <Form.Item name="account" label="账号" rules={[{ validator: (_, value?: string) =>
        new TextEncoder().encode(value?.trim() ?? '').length <= 128 ? Promise.resolve() : Promise.reject(new Error('账号筛选最多128字节')) }]}>
        <Input prefix={<SearchOutlined />} allowClear maxLength={128} placeholder="搜索邮箱或手机号" />
      </Form.Item>
      <Form.Item name="status" label="状态"><Select allowClear className="filter-select" placeholder="全部状态" options={appUserStatusOptions} /></Form.Item>
      <Form.Item><Space><Button type="primary" icon={<SearchOutlined />} htmlType="submit">查询</Button>
        <Button icon={<UndoOutlined />} onClick={() => { form.resetFields(); setParams({ page: 1, page_size: params.page_size }); }}>重置</Button>
        <Button icon={<ReloadOutlined />} title="刷新" aria-label="刷新APP用户" loading={users.isFetching} onClick={() => void users.refetch()} />
      </Space></Form.Item>
    </Form></Card>
    <Card className="data-card app-data-card" title={<span className="app-section-title"><span className="app-section-icon"><TeamOutlined /></span>
      用户列表{users.isSuccess && <span className="app-count">{users.data.total}</span>}</span>} extra={<span className="app-card-hint">用户资料与访问管理</span>}>
      <QueryState isPending={users.isPending} error={users.error} onRetry={() => users.refetch()} />
      {!users.isError && <Table<AppAdminUserInfo> rowKey="id" dataSource={users.data?.list ?? []} loading={users.isFetching}
        locale={{ emptyText: <Empty className="app-empty" image={<TeamOutlined />} description={<><strong>暂无匹配用户</strong><p>调整账号或状态后再试</p></>} /> }}
        scroll={{ x: 1250 }} pagination={pagination} columns={[
          { title: '用户账号', dataIndex: 'account', key: 'account', width: 265, render: (value: string, row) => <div className="app-table-identity">
            <Avatar shape="square" size={38} className="app-avatar--blue" src={row.avatar_url || undefined} icon={<UserOutlined />} />
            <div className="app-identity-copy"><Button type="link" className="app-account-link" disabled={unavailable}
              onClick={() => setAction({ id: row.id, mode: 'details' })}>{value}</Button><span className="app-secondary-text">用户 ID · {row.id}</span></div>
          </div> },
          { title: '昵称', dataIndex: 'nickname', width: 130, ellipsis: true, render: (value: string) => <span className="app-primary-text">{value || '—'}</span> },
          { title: '内部备注', dataIndex: 'remark', ellipsis: true, width: 160, render: (value: string) => value || '—' },
          { title: '账号验证', key: 'verification', width: 120, render: (_, row) => <Tag icon={row.email_verified ? <MailOutlined /> : row.phone_verified ? <MobileOutlined /> : undefined} color={row.email_verified || row.phone_verified ? 'blue' : 'default'}>
            {row.email_verified ? '邮箱已验证' : row.phone_verified ? '手机已验证' : '未验证'}
          </Tag> },
          { title: '状态', dataIndex: 'status', width: 100, render: (value: AppAdminUserInfo['status']) => <Tag color={appUserStatus[value]?.color}>{appUserStatus[value]?.label ?? value}</Tag> },
          { title: '最近登录', dataIndex: 'last_login_at', width: 165, render: value => <span className="app-date-text">{formatTime(value)}</span> },
          { title: '注册时间', dataIndex: 'created_at', width: 165, render: value => <span className="app-date-text">{formatTime(value)}</span> },
          { title: '操作', key: 'actions', width: 265, render: (_, row) => <Space className="app-row-actions" wrap>
            <Button type="link" icon={<EyeOutlined />} disabled={unavailable} onClick={() => setAction({ id: row.id, mode: 'details' })}>详情</Button>
            <Auth code="sys:app:user:edit"><Button type="link" icon={<EditOutlined />} disabled={unavailable} onClick={() => setAction({ id: row.id, mode: 'edit' })}>编辑</Button></Auth>
            <Auth code="sys:app:user:review"><Button type="link" icon={row.status === 'active' ? <StopOutlined /> : row.status === 'disabled' ? <CheckCircleOutlined /> : <AuditOutlined />} danger={row.status === 'active'} disabled={unavailable}
              onClick={() => setAction({ id: row.id, mode: 'review', intent: row.status === 'active' ? 'disable' : row.status === 'disabled' ? 'enable' : 'review' })}>
              {row.status === 'active' ? '停用' : row.status === 'disabled' ? '启用' : '审核'}</Button></Auth>
            <Auth code="sys:app:user:avatar"><Button type="link" icon={<PictureOutlined />} disabled={unavailable} onClick={() => setAction({ id: row.id, mode: 'avatar' })}>头像</Button></Auth>
            <Auth code="sys:app:session:list"><Button type="link" icon={<LaptopOutlined />} disabled={unavailable} onClick={() => setAction({ id: row.id, mode: 'sessions' })}>登录设备</Button></Auth>
            {canReadMemberships && <Button type="link" icon={<CrownOutlined />} disabled={unavailable} onClick={() => setAction({ id: row.id, mode: 'memberships' })}>会员</Button>}
          </Space> },
        ]} />}
    </Card>
    {action && (action.mode === 'details' || action.mode === 'sessions' || action.mode === 'memberships') && <AppUserDetails
      key={`${action.id}:${action.mode}`} id={action.id} initialTab={action.mode === 'details' ? 'profile' : action.mode} onClose={() => setAction(null)} />}
    {action?.mode === 'edit' && <AppUserEditDialog key={action.id} id={action.id} onClose={() => setAction(null)} />}
    {selectedUser && action?.mode === 'review' && <AppUserReviewDialog key={selectedUser.id} user={selectedUser} intent={action.intent} disabled={unavailable} onClose={() => setAction(null)} />}
    {selectedUser && action?.mode === 'avatar' && <AppUserAvatarDialog key={selectedUser.id} user={selectedUser} disabled={unavailable} onClose={() => setAction(null)} />}
  </div>;
}
