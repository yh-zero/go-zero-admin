import { App, Button, Card, Popconfirm, Space, Table, Tag, Typography } from 'antd';
import { DesktopOutlined, LogoutOutlined, ReloadOutlined } from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import dayjs from 'dayjs';
import { fetchDeviceSessions, logout, revokeDeviceSession } from '../../api/auth';
import type { DeviceSession } from '../../api/types';
import { useAuthStore } from '../../stores/auth';
import { clearSession } from '../../utils/session';
import { QueryState } from '../../components/QueryState';

export function DeviceSessions() {
  const client = useQueryClient();
  const { message } = App.useApp();
  const navigate = useNavigate();
  const isLoggingOut = useAuthStore(state => state.isLoggingOut);
  const query = useQuery({ queryKey: ['device-sessions'], queryFn: fetchDeviceSessions });
  const revoke = useMutation({ mutationFn: (session: DeviceSession) => revokeDeviceSession(session.id), onMutate: () => useAuthStore.getState(), onSuccess: async (_, session, auth) => {
    if (session.is_current) {
      if (auth && await clearSession({ sessionVersion: auth.sessionVersion, sessionId: session.id })) navigate('/login', { replace: true });
      return;
    }
    await client.invalidateQueries({ queryKey: ['device-sessions'] });
    void message.success('已退出该设备');
  } });
  const revokeAll = useMutation({ mutationFn: () => logout(true), onSettled: () => {
    if (!useAuthStore.getState().token) navigate('/login', { replace: true });
  } });
  const busy = revoke.isPending || isLoggingOut;
  return <Card title={<Space><DesktopOutlined />登录设备</Space>} extra={<Space>
    <Button icon={<ReloadOutlined />} aria-label="刷新登录设备" onClick={() => void query.refetch()} loading={query.isFetching} disabled={busy} />
    <Popconfirm title="退出全部设备？" description="包括当前设备，所有设备都需要重新登录。" onConfirm={() => revokeAll.mutate()} disabled={busy}>
      <Button danger icon={<LogoutOutlined />} loading={isLoggingOut} disabled={busy}>退出全部设备</Button>
    </Popconfirm>
  </Space>}>
    <QueryState isPending={query.isPending} error={query.error} onRetry={() => query.refetch()} />
    {!query.isError && <Table<DeviceSession> rowKey="id" dataSource={query.data ?? []} pagination={false} loading={query.isFetching} scroll={{ x: 650 }} columns={[
      { title: '设备', key: 'device', render: (_, session) => <Space direction="vertical" size={2}>
        <Space>{session.device_name || '浏览器'}{session.is_current && <Tag color="blue">当前设备</Tag>}</Space>
        <Typography.Text type="secondary" title={session.user_agent}>{session.ip || '—'}</Typography.Text>
      </Space> },
      { title: '登录时间', dataIndex: 'created_at', render: value => value ? dayjs.unix(value).format('YYYY-MM-DD HH:mm') : '—' },
      { title: '最近活动', dataIndex: 'last_seen_at', render: value => value ? dayjs.unix(value).format('YYYY-MM-DD HH:mm') : '—' },
      { title: '操作', key: 'actions', width: 100, render: (_, session) => <Popconfirm title={session.is_current ? '退出当前设备？' : '退出这台设备？'}
        onConfirm={() => revoke.mutate(session)} disabled={busy}>
        <Button type="link" danger disabled={busy} loading={revoke.isPending && revoke.variables.id === session.id}>退出</Button>
      </Popconfirm> },
    ]} />}
  </Card>;
}
