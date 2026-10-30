import { Card, Col, Row, Statistic, Table, Tag, Typography } from 'antd';
import { useQuery } from '@tanstack/react-query';
import { fetchUsers } from '../../api/user';
import { fetchRoles } from '../../api/role';
import { fetchFiles } from '../../api/file';
import { fetchAuditLogs } from '../../api/audit';
import type { AuditLogInfo } from '../../api/types';
import { PageHeader } from '../../components/PageHeader';
import { QueryState } from '../../components/QueryState';
import { formatTime } from '../../utils/format';
import { fetchHealth } from '../../api/health';
export default function DashboardPage() {
  const health = useQuery({ queryKey: ['health'], queryFn: fetchHealth, refetchInterval: 60_000 });
  const users = useQuery({ queryKey: ['users', { page: 1, page_size: 1 }], queryFn: () => fetchUsers({ page: 1, page_size: 1 }) });
  const roles = useQuery({ queryKey: ['roles', { page: 1, page_size: 1 }], queryFn: () => fetchRoles({ page: 1, page_size: 1 }) });
  const files = useQuery({ queryKey: ['files', { page: 1, page_size: 1 }], queryFn: () => fetchFiles({ page: 1, page_size: 1 }) });
  const audit = useQuery({ queryKey: ['audit-logs', { page: 1, page_size: 5 }], queryFn: () => fetchAuditLogs({ page: 1, page_size: 5 }) });
  const stats = [{ title: '用户总数', query: users }, { title: '角色总数', query: roles }, { title: '文件总数', query: files }, { title: '审计记录', query: audit }];
  return <div className="page-stack"><PageHeader title="工作台" description="查看当前账号可访问的管理数据" extra={<Tag color={health.data ? 'success' : 'default'}>{health.data ? '服务已连接' : health.isPending ? '连接中' : '服务暂不可用'}</Tag>} />
    <Row gutter={[16, 16]}>{stats.map(({ title, query }) => <Col xs={24} sm={12} xl={6} key={title}><Card loading={query.isPending}><Statistic title={title} value={query.isError ? '—' : query.data?.total ?? '—'} />{query.isError && <Typography.Text type="secondary">暂不可用</Typography.Text>}</Card></Col>)}</Row>
    <Card title="最近操作"><QueryState isPending={audit.isPending} error={audit.error} onRetry={() => audit.refetch()} />
      <Table<AuditLogInfo> rowKey="id" pagination={false} dataSource={audit.data?.list ?? []} scroll={{ x: 650 }} columns={[
        { title: '操作人', dataIndex: 'username' }, { title: '模块', dataIndex: 'module' }, { title: '动作', dataIndex: 'action' },
        { title: '时间', dataIndex: 'created_at', render: formatTime },
      ]} />
    </Card>
  </div>;
}
