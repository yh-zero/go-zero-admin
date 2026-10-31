import { useState } from 'react';
import { Avatar, Button, Card, Tag } from 'antd';
import { ApartmentOutlined, LockOutlined, UserOutlined } from '@ant-design/icons';
import { useQuery } from '@tanstack/react-query';
import { fetchProfile } from '../../api/profile';
import { PageHeader } from '../../components/PageHeader';
import { QueryState } from '../../components/QueryState';
import { formatGender } from '../../utils/formOptions';
import { ProfileEditor } from './ProfileEditor';
import { ChangePasswordDialog } from './ChangePasswordDialog';
import './profile.css';
export default function ProfilePage() {
  const [passwordOpen, setPasswordOpen] = useState(false);
  const query = useQuery({ queryKey: ['profile'], queryFn: fetchProfile });
  const user = query.data;
  return <div className="page-stack profile-page"><PageHeader title="个人中心" description="管理您的账号信息与个人资料" extra={<Button icon={<LockOutlined />} onClick={() => setPasswordOpen(true)}>修改密码</Button>} />
    <QueryState isPending={query.isPending} error={query.error} onRetry={() => query.refetch()} />
    {user && <div className="profile-grid">
      <Card className="profile-summary">
        <div className="profile-identity">
          <Avatar className="profile-avatar" size={80} src={user.avatar_url || undefined} icon={<UserOutlined />} />
          <h2>{user.real_name || user.nickname || user.username}</h2>
          <p className="profile-username">{user.username}</p>
          <span className="profile-account-label">当前账号</span>
        </div>
        <dl className="profile-facts">
          <div><dt><ApartmentOutlined />所属部门</dt><dd>{user.dept_name || '—'}</dd></div>
          <div><dt><UserOutlined />性别</dt><dd><Tag>{formatGender(user.gender)}</Tag></dd></div>
        </dl>
      </Card>
      <Card className="profile-editor-card" title={<div className="profile-section-heading"><span>个人资料</span><p>维护您的姓名、联系方式与基础信息</p></div>}>
        <ProfileEditor key={user.id} profile={user} />
      </Card>
    </div>}
    {passwordOpen && <ChangePasswordDialog onClose={() => setPasswordOpen(false)} />}
  </div>;
}
