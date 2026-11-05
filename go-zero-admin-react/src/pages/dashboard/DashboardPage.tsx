import { Card, Col, Row, Statistic, Table, Tag, Typography } from 'antd';
import { ArrowRightOutlined, FileOutlined, FileSearchOutlined, SafetyOutlined, TeamOutlined } from '@ant-design/icons';
import { Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { fetchUsers } from '../../api/user';
import { fetchRoles } from '../../api/role';
import { fetchFiles } from '../../api/file';
import { fetchAuditLogs } from '../../api/audit';
import { fetchMyMenus } from '../../api/auth';
import type { AuditLogInfo } from '../../api/types';
import { PageHeader } from '../../components/PageHeader';
import { QueryState } from '../../components/QueryState';
import { formatTime } from '../../utils/format';
import { fetchHealth } from '../../api/health';
import { getAccessiblePagePaths } from '../../router/menuRoutes';
import { DashboardShortcuts } from './DashboardShortcuts';
import './dashboard.css';
import { queryKeys } from '../../utils/queryKeys';
export default function DashboardPage() {
  const menus = useQuery({ queryKey: [queryKeys.myMenus], queryFn: fetchMyMenus });
  const allowedPaths = menus.isSuccess ? getAccessiblePagePaths(menus.data) : new Set<string>();
  const canReadAudit = allowedPaths.has('/sys/audit');
  const health = useQuery({ queryKey: [queryKeys.health], queryFn: fetchHealth, refetchInterval: 60_000 });
  const users = useQuery({ queryKey: [queryKeys.users, { page: 1, page_size: 1 }], queryFn: () => fetchUsers({ page: 1, page_size: 1 }), enabled: allowedPaths.has('/sys/user') });
  const roles = useQuery({ queryKey: [queryKeys.roles, { page: 1, page_size: 1 }], queryFn: () => fetchRoles({ page: 1, page_size: 1 }), enabled: allowedPaths.has('/sys/role') });
  const files = useQuery({ queryKey: [queryKeys.files, { page: 1, page_size: 1 }], queryFn: () => fetchFiles({ page: 1, page_size: 1 }), enabled: allowedPaths.has('/sys/file') });
  const audit = useQuery({ queryKey: [queryKeys.auditLogs, { page: 1, page_size: 5 }], queryFn: () => fetchAuditLogs({ page: 1, page_size: 5 }), enabled: canReadAudit });
  const stats = [
    { title: '用户总数', query: users, icon: <TeamOutlined />, tone: 'blue', caption: '系统账号与成员', path: '/sys/user' },
    { title: '角色总数', query: roles, icon: <SafetyOutlined />, tone: 'violet', caption: '角色与访问权限', path: '/sys/role' },
    { title: '文件总数', query: files, icon: <FileOutlined />, tone: 'teal', caption: '文件资源与访问', path: '/sys/file' },
    { title: '审计记录', query: audit, icon: <FileSearchOutlined />, tone: 'amber', caption: '可追溯的操作记录', path: '/sys/audit' },
  ].filter(item => allowedPaths.has(item.path));
  const connected = health.isSuccess && health.data;
  return <div className="page-stack dashboard-page"><PageHeader title="工作台" description="掌握系统概况，让日常管理井然有序。" extra={<Tag className="connection-status" color={connected ? 'success' : 'default'}><span className="status-tag-dot" />{connected ? '服务已连接' : health.isPending ? '连接中' : '服务暂不可用'}</Tag>} />
    <QueryState isPending={menus.isPending} error={menus.error} onRetry={() => menus.refetch()} />
    {menus.isSuccess && <>{stats.length > 0 && <Row gutter={[18, 18]}>{stats.map(({ title, query, icon, tone, caption, path }) => <Col xs={24} sm={stats.length === 1 ? 24 : 12} xl={24 / stats.length} key={title}>
      <Card className={`stat-card stat-${tone}`} loading={query.isPending}>
        <div className="stat-card-top"><span className="stat-label">{title}</span><span className="stat-icon">{icon}</span></div>
        <Statistic value={query.isError ? '—' : query.data?.total ?? '—'} />
        <div className="stat-card-bottom"><Typography.Text type="secondary">{query.isError ? '暂不可用' : caption}</Typography.Text><Link to={path} aria-label={`查看${title}`}><ArrowRightOutlined /></Link></div>
      </Card>
    </Col>)}</Row>}
    <div className={`dashboard-lower${canReadAudit ? '' : ' dashboard-lower-single'}`}>{canReadAudit && <Card className="data-card dashboard-audit" title="最近操作" extra={<Link className="section-link" to="/sys/audit">全部记录 <ArrowRightOutlined /></Link>}><QueryState isPending={audit.isPending} error={audit.error} onRetry={() => audit.refetch()} />
      <Table<AuditLogInfo> rowKey="id" pagination={false} dataSource={audit.data?.list ?? []} scroll={{ x: 650 }} columns={[
        { title: '操作人', dataIndex: 'username', render: (value: string) => <span className="table-primary">{value || '—'}</span> },
        { title: '模块', dataIndex: 'module', render: (value: string) => <span className="code-text">{value}</span> },
        { title: '动作', dataIndex: 'action', render: (value: string) => <Tag>{value}</Tag> },
        { title: '时间', dataIndex: 'created_at', render: formatTime },
      ]} />
    </Card>}<DashboardShortcuts allowedPaths={allowedPaths} /></div></>}
  </div>;
}
