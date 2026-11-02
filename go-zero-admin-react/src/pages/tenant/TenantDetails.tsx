import { Descriptions, Drawer, Tag } from 'antd';
import { useQuery } from '@tanstack/react-query';
import { fetchTenant } from '../../api/tenant';
import { QueryState } from '../../components/QueryState';
import { formatTime } from '../../utils/format';

interface TenantDetailsProps { id: number; onClose: () => void }

const statusLabels: Record<string, string> = { creating: '创建中', active: '运行中', stopped: '已停用', error: '异常' };

export function TenantDetails({ id, onClose }: TenantDetailsProps) {
  const query = useQuery({ queryKey: ['tenant-list', 'detail', id], queryFn: () => fetchTenant(id) });
  const tenant = query.data?.tenant;
  return <Drawer open title="租户详情" onClose={onClose} size="large">
    <QueryState isPending={query.isPending} error={query.error} onRetry={() => query.refetch()} />
    {tenant && !query.error && <Descriptions bordered column={1} items={[
      { key: 'id', label: '租户ID', children: tenant.id },
      { key: 'slug', label: '租户标识', children: tenant.slug },
      { key: 'domain', label: '域名', children: tenant.domain || '—' },
      { key: 'compose_project', label: '服务项目', children: tenant.compose_project || '—' },
      { key: 'status', label: '状态', children: statusLabels[tenant.status] ?? tenant.status },
      { key: 'admin_account_ready', label: '管理员账号', children: <Tag color={tenant.admin_account_ready ? 'success' : 'default'}>{tenant.admin_account_ready ? '已就绪' : '未就绪'}</Tag> },
      { key: 'error_msg', label: '异常说明', children: tenant.error_msg || '—' },
      { key: 'created_by', label: '创建人ID', children: tenant.created_by || '—' },
      { key: 'created_at', label: '创建时间', children: formatTime(tenant.created_at) },
      { key: 'updated_at', label: '更新时间', children: formatTime(tenant.updated_at) },
    ]} />}
  </Drawer>;
}
