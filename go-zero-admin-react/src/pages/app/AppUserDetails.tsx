import { useEffect, useState } from 'react';
import { App, Avatar, Button, Descriptions, Drawer, Popconfirm, Space, Table, Tabs, Tag, Typography } from 'antd';
import { CrownOutlined, IdcardOutlined, LaptopOutlined, PlusOutlined, ReloadOutlined, SafetyCertificateOutlined, UserOutlined } from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { fetchMyMenus } from '../../api/auth';
import { fetchAppMemberships, fetchAppUser, revokeAppMembership } from '../../api/app';
import type { AppAdminUserInfo, AppMembershipInfo, GetAppMembershipListReq } from '../../api/types';
import { Auth } from '../../components/Auth';
import { QueryState } from '../../components/QueryState';
import { usePagedQuery } from '../../hooks/usePagedQuery';
import { getAccessiblePagePaths } from '../../router/menuRoutes';
import { useAuthStore } from '../../stores/auth';
import { formatTime } from '../../utils/format';
import { invalidateAppData } from '../../utils/queryInvalidation';
import { AppMembershipGrantDialog } from './AppMembershipGrantDialog';
import { AppUserSessionsPanel } from './AppUserSessions';
import { appUserStatus } from './appOptions';
import './app.css';
import { queryKeys } from '../../utils/queryKeys';
import { useTablePagination } from '../../hooks/useTablePagination';

interface Props { id: number; initialTab?: 'profile' | 'memberships' | 'sessions'; onClose: () => void }

function AppUserProfile({ user }: { user: AppAdminUserInfo }) {
  return <div className="app-profile-content"><section className="app-profile-block">
    <h3 className="app-profile-section-title"><IdcardOutlined />账号信息</h3>
    <Descriptions size="small" layout="vertical" column={{ xs: 1, sm: 2 }} items={[
    { key: 'id', label: '用户ID', children: user.id },
    { key: 'nickname', label: '昵称', children: user.nickname || '—' },
    { key: 'identity_kind', label: '账号类型', children: user.identity_kind === 'email' ? '邮箱' : user.identity_kind === 'phone' ? '手机号' : user.identity_kind },
    { key: 'email_verified', label: '邮箱验证', children: user.email_verified ? '已验证' : '未验证' },
    { key: 'phone_verified', label: '手机验证', children: user.phone_verified ? '已验证' : '未验证' },
    { key: 'last_login_at', label: '最近登录', children: formatTime(user.last_login_at) },
    { key: 'created_at', label: '注册时间', children: formatTime(user.created_at) },
    ]} /></section><section className="app-profile-block">
    <h3 className="app-profile-section-title"><SafetyCertificateOutlined />管理记录</h3>
    <div className="app-note-block"><span className="app-secondary-text">内部备注</span><p className="app-plain-text">{user.remark || '暂无内部备注'}</p></div>
    <div className="app-note-block"><span className="app-secondary-text">审核说明</span><p className="app-plain-text">{user.review_reason || '暂无审核说明'}</p></div>
  </section></div>;
}

function membershipState(item: AppMembershipInfo, now: number) {
  if (item.revoked) return <Tag>已撤销</Tag>;
  if (item.starts_at > now) return <Tag color="processing">未生效</Tag>;
  if (item.expires_at <= now) return <Tag color="orange">已过期</Tag>;
  return <Tag color="success">有效</Tag>;
}

function AppUserMemberships({ user, onBusyChange }: { user: AppAdminUserInfo; onBusyChange: (busy: boolean) => void }) {
  const [params, setParams] = useState<GetAppMembershipListReq>({ user_id: user.id, page: 1, page_size: 20 });
  const [grantOpen, setGrantOpen] = useState(false);
  const [now, setNow] = useState(() => Math.floor(Date.now() / 1000));
  const client = useQueryClient();
  const { message } = App.useApp();
  const memberships = usePagedQuery({ params, setParams, queryKey: [queryKeys.appMemberships, params], queryFn: () => fetchAppMemberships(params) });
  const pagination = useTablePagination(params, setParams, memberships.data?.total ?? 0);
  const revoke = useMutation({ mutationFn: revokeAppMembership, onSuccess: async () => {
    await invalidateAppData(client); void message.success('会员已撤销');
  } });
  useEffect(() => {
    const timer = window.setInterval(() => setNow(Math.floor(Date.now() / 1000)), 30_000);
    return () => window.clearInterval(timer);
  }, []);
  useEffect(() => {
    onBusyChange(revoke.isPending || grantOpen);

    return () => onBusyChange(false);
  }, [onBusyChange, revoke.isPending, grantOpen]);
  return <div className="app-user-panel">
    <div className="app-panel-toolbar"><Typography.Text type="secondary">{user.account}</Typography.Text><Space>
      <Button icon={<ReloadOutlined />} aria-label="刷新用户会员" loading={memberships.isFetching} disabled={revoke.isPending}
        onClick={() => void memberships.refetch()}>刷新</Button>
      <Auth code="sys:app:membership:grant"><Button type="primary" icon={<PlusOutlined />} disabled={revoke.isPending} onClick={() => setGrantOpen(true)}>开通会员</Button></Auth>
    </Space></div>
    <QueryState isPending={memberships.isPending} error={memberships.error} onRetry={() => memberships.refetch()} />
    {memberships.isSuccess && <Table<AppMembershipInfo> rowKey="id" dataSource={memberships.data.list} loading={memberships.isFetching} scroll={{ x: 830 }}
      pagination={pagination} columns={[
        { title: '套餐', dataIndex: 'plan_code', width: 110, render: value => <span className="app-plan-pill">{value}</span> },
        { title: '会员权益', dataIndex: 'benefits', ellipsis: true, render: value => value || '—' },
        { title: '状态', key: 'status', width: 90, render: (_, item) => membershipState(item, now) },
        { title: '生效时间', dataIndex: 'starts_at', width: 170, render: formatTime },
        { title: '到期时间', dataIndex: 'expires_at', width: 170, render: formatTime },
        { title: '操作', key: 'actions', width: 80, render: (_, item) => <Auth code="sys:app:membership:revoke"><Popconfirm title="确认撤销这项会员？"
          description="撤销后对应权益立即失效，原记录会保留。" disabled={item.revoked || revoke.isPending || memberships.isFetching}
          onConfirm={() => revoke.mutateAsync(item.id).catch(() => undefined)} okButtonProps={{ loading: revoke.isPending && revoke.variables === item.id }}>
          <Button type="link" danger disabled={item.revoked || revoke.isPending || memberships.isFetching}>撤销</Button>
        </Popconfirm></Auth> },
      ]} />}
    {grantOpen && <AppMembershipGrantDialog initialUserId={user.id} onClose={() => setGrantOpen(false)} />}
  </div>;
}

export function AppUserDetails({ id, initialTab = 'profile', onClose }: Props) {
  const [tab, setTab] = useState(initialTab);
  const [busy, setBusy] = useState(false);
  const user = useQuery({ queryKey: [queryKeys.appUser, id], queryFn: () => fetchAppUser(id) });
  const menus = useQuery({ queryKey: [queryKeys.myMenus], queryFn: fetchMyMenus });
  const canReadMemberships = menus.isSuccess && getAccessiblePagePaths(menus.data).has('/app/memberships');
  const canReadSessions = useAuthStore(state => state.permissionsReady && state.permCodes.includes('sys:app:session:list'));
  const selectedTab = tab === 'memberships' && !canReadMemberships || tab === 'sessions' && !canReadSessions ? 'profile' : tab;
  return <Drawer open className="app-user-drawer" title={<span className="app-section-title"><span className="app-section-icon"><UserOutlined /></span>APP用户详情</span>}
    size="large" closable={!busy} mask={{ closable: !busy }} keyboard={!busy}
    onClose={() => { if (!busy) onClose(); }}>
    <QueryState isPending={user.isPending} error={user.error} onRetry={() => user.refetch()} />
    {user.isSuccess && <><div className="app-user-summary"><Avatar shape="square" size={58} className="app-avatar--blue" src={user.data.avatar_url || undefined} icon={<UserOutlined />} />
      <div className="app-user-summary-copy"><div className="app-user-summary-title"><h2>{user.data.nickname || user.data.account}</h2>
        <Tag color={appUserStatus[user.data.status]?.color}>{appUserStatus[user.data.status]?.label ?? user.data.status}</Tag></div>
      <span className="app-summary-account">{user.data.account}</span><span className="app-secondary-text">用户 ID · {user.data.id}</span></div></div>
    <Tabs activeKey={selectedTab} destroyOnHidden onChange={key => { if (!busy) setTab(key as typeof tab); }} items={[
      { key: 'profile', label: '基本资料', icon: <IdcardOutlined />, disabled: busy && selectedTab !== 'profile', children: selectedTab === 'profile' && <AppUserProfile user={user.data} /> },
      { key: 'memberships', label: '会员', icon: <CrownOutlined />, disabled: !canReadMemberships || busy && selectedTab !== 'memberships',
        children: selectedTab === 'memberships' && canReadMemberships && <AppUserMemberships key={id} user={user.data} onBusyChange={setBusy} /> },
      { key: 'sessions', label: '登录设备', icon: <LaptopOutlined />, disabled: !canReadSessions || busy && selectedTab !== 'sessions',
        children: selectedTab === 'sessions' && canReadSessions && <AppUserSessionsPanel key={id} user={user.data} onBusyChange={setBusy} /> },
    ]} /></>}
  </Drawer>;
}
