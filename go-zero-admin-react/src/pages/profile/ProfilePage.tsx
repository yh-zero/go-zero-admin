import { useState } from 'react';
import { Button, Card, Descriptions, Tag } from 'antd';
import { useQuery } from '@tanstack/react-query';
import { fetchProfile } from '../../api/profile';
import { PageHeader } from '../../components/PageHeader';
import { QueryState } from '../../components/QueryState';
import { ProfileEditor } from './ProfileEditor';
import { ChangePasswordDialog } from './ChangePasswordDialog';
export default function ProfilePage() {
  const [passwordOpen, setPasswordOpen] = useState(false);
  const query = useQuery({ queryKey: ['profile'], queryFn: fetchProfile });
  const user = query.data;
  return <div className="page-stack"><PageHeader title="个人中心" description="查看账号信息并维护个人资料" extra={<Button onClick={() => setPasswordOpen(true)}>修改密码</Button>} />
    <QueryState isPending={query.isPending} error={query.error} onRetry={() => query.refetch()} />
    {user && <><Card title="账号信息"><Descriptions column={{ xs: 1, sm: 2, lg: 3 }} items={[
      { key: 'username', label: '用户名', children: user.username }, { key: 'dept', label: '所属部门', children: user.dept_name || '—' },
      { key: 'gender', label: '性别', children: <Tag>{user.gender === 1 ? '男' : user.gender === 2 ? '女' : '未知'}</Tag> },
    ]} /></Card><Card title="个人资料"><ProfileEditor key={user.id} profile={user} /></Card></>}
    {passwordOpen && <ChangePasswordDialog onClose={() => setPasswordOpen(false)} />}
  </div>;
}
