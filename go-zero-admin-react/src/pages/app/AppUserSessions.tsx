import { useEffect, useState } from 'react';
import { App, Button, Drawer, Popconfirm, Result, Space, Table, Typography } from 'antd';
import { ReloadOutlined } from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { fetchAppUserSessions, revokeAppUserSession } from '../../api/app';
import type { AppUserInfo, DeviceSession } from '../../api/types';
import { Auth } from '../../components/Auth';
import { QueryState } from '../../components/QueryState';
import { useAuthStore } from '../../stores/auth';
import { formatTime } from '../../utils/format';
import { invalidateAppData } from '../../utils/queryInvalidation';

interface Props { user: AppUserInfo; onClose: () => void }
interface PanelProps { user: AppUserInfo; onBusyChange?: (busy: boolean) => void }

export function AppUserSessions({ user, onClose }: Props) {
  const [busy, setBusy] = useState(false);
  return <Drawer open title="APP用户登录设备" size="large" closable={!busy} onClose={() => { if (!busy) onClose(); }}>
    <AppUserSessionsPanel user={user} onBusyChange={setBusy} />
  </Drawer>;
}

export function AppUserSessionsPanel({ user, onBusyChange }: PanelProps) {
  const { message } = App.useApp();
  const client = useQueryClient();
  const allowed = useAuthStore(state => state.permissionsReady && state.permCodes.includes('sys:app:session:list'));
  const sessions = useQuery({
    queryKey: ['app-sessions', user.id], queryFn: () => fetchAppUserSessions(user.id), enabled: allowed,
  });
  const revoke = useMutation({
    mutationFn: (sessionId: string) => revokeAppUserSession(user.id, sessionId),
    onSuccess: async () => {
      await invalidateAppData(client);
      void message.success('该APP登录设备已退出');
    },
  });
  useEffect(() => {
    onBusyChange?.(revoke.isPending);
    return () => onBusyChange?.(false);
  }, [onBusyChange, revoke.isPending]);
  return <div className="app-user-panel">
    <div className="app-panel-toolbar"><Typography.Text type="secondary">{user.account}</Typography.Text>
      {allowed && <Button icon={<ReloadOutlined />} aria-label="刷新APP登录设备"
        loading={sessions.isFetching} disabled={revoke.isPending} onClick={() => void sessions.refetch()}>刷新</Button>}</div>
    {!allowed ? <Result status="403" title="暂无查看登录设备权限" /> : <>
      <QueryState isPending={sessions.isPending} error={sessions.error} onRetry={() => sessions.refetch()} />
      {!sessions.isError && <Table<DeviceSession> rowKey="id" dataSource={sessions.data ?? []}
        pagination={false} loading={sessions.isFetching} scroll={{ x: 700 }} columns={[
          { title: '设备', key: 'device', render: (_, session) => <Space orientation="vertical" size={2}>
            <span>{session.device_name || '浏览器'}</span>
            <Typography.Text type="secondary" title={session.user_agent}>{session.ip || '—'}</Typography.Text>
          </Space> },
          { title: '登录时间', dataIndex: 'created_at', render: formatTime },
          { title: '最近活动', dataIndex: 'last_seen_at', render: formatTime },
          { title: '到期时间', dataIndex: 'expires_at', render: formatTime },
          { title: '操作', key: 'actions', width: 90, render: (_, session) => <Auth code="sys:app:session:revoke">
            <Popconfirm title="退出这台APP设备？" description="该设备将需要重新登录。"
              disabled={revoke.isPending || sessions.isFetching}
              onConfirm={() => revoke.mutateAsync(session.id).catch(() => undefined)}
              okButtonProps={{ loading: revoke.isPending }}>
              <Button type="link" danger disabled={revoke.isPending || sessions.isFetching}
                loading={revoke.isPending && revoke.variables === session.id}>退出</Button>
            </Popconfirm>
          </Auth> },
        ]} />}
    </>}
  </div>;
}
