import { Descriptions, Drawer } from 'antd';
import { useQuery } from '@tanstack/react-query';
import { fetchUser } from '../../api/user';
import { QueryState } from '../../components/QueryState';
import { StatusTag } from '../../components/StatusTag';
import { formatTime } from '../../utils/format';

interface Props { id: number; onClose: () => void }

export function UserDetails({ id, onClose }: Props) {
  const user = useQuery({ queryKey: ['user', id], queryFn: () => fetchUser(id) });
  const data = user.data;
  return <Drawer open title="用户详情" onClose={onClose} size="large">
    <QueryState isPending={user.isPending} error={user.error} onRetry={() => void user.refetch()} />
    {data && <Descriptions column={1} bordered items={[
      { key: 'id', label: '用户 ID', children: data.id },
      { key: 'username', label: '用户名', children: data.username },
      { key: 'nickname', label: '昵称', children: data.nickname || '—' },
      { key: 'real_name', label: '真实姓名', children: data.real_name || '—' },
      { key: 'email', label: '邮箱', children: data.email || '—' },
      { key: 'phone', label: '手机号', children: data.phone || '—' },
      { key: 'dept', label: '部门', children: data.dept_name || (data.dept_id ? `部门 ${data.dept_id}` : '未分配') },
      { key: 'gender', label: '性别', children: ['未知', '男', '女'][data.gender] || '未知' },
      { key: 'status', label: '状态', children: <StatusTag status={data.status} /> },
      { key: 'last_login_at', label: '最近登录', children: formatTime(data.last_login_at) },
      { key: 'last_login_ip', label: '最近登录 IP', children: data.last_login_ip || '—' },
      { key: 'created_at', label: '创建时间', children: formatTime(data.created_at) },
      { key: 'updated_at', label: '更新时间', children: formatTime(data.updated_at) },
      { key: 'remark', label: '备注', children: data.remark || '—' },
    ]} />}
  </Drawer>;
}
