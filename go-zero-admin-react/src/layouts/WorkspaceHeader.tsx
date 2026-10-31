import { Avatar, Breadcrumb, Button, Dropdown, Layout } from 'antd';
import { DownOutlined, HomeOutlined, LogoutOutlined, MenuFoldOutlined, MenuUnfoldOutlined, UserOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { useMutation } from '@tanstack/react-query';
import { logout } from '../api/auth';
import { useAuthStore } from '../stores/auth';
import { DepartmentSwitcher } from './DepartmentSwitcher';

interface Props { title: string; name: string; navigationOpen: boolean; onToggle: () => void }

export function WorkspaceHeader({ title, name, navigationOpen, onToggle }: Props) {
  const navigate = useNavigate();
  const isLoggingOut = useAuthStore(state => state.isLoggingOut);
  const mutation = useMutation({ mutationFn: () => logout(), onSettled: () => {
    if (!useAuthStore.getState().token) navigate('/login', { replace: true });
  } });
  const onLogout = () => { if (!isLoggingOut) mutation.mutate(); };
  return <Layout.Header className="app-header">
    <div className="header-location">
      <Button type="text" className="nav-toggle" aria-label={navigationOpen ? '收起导航' : '展开导航'}
        aria-expanded={navigationOpen} icon={navigationOpen ? <MenuFoldOutlined /> : <MenuUnfoldOutlined />} onClick={onToggle} />
      <span className="header-divider" />
      <Breadcrumb items={[{ title: <><HomeOutlined /><span className="breadcrumb-home">控制台</span></> }, { title }]} />
    </div>
    <div className="header-actions"><DepartmentSwitcher /><Dropdown trigger={['click']} menu={{ items: [
      { key: 'profile', label: '个人中心', icon: <UserOutlined /> },
      { type: 'divider' },
      { key: 'logout', label: '退出登录', icon: <LogoutOutlined />, danger: true, disabled: isLoggingOut },
    ], onClick: ({ key }) => {
      if (key === 'logout') onLogout();
      else navigate('/sys/profile');
    } }}>
      <Button type="text" className="header-account" aria-label={`${name}，账号菜单`}>
        <Avatar className="header-avatar" size={34} icon={<UserOutlined />} />
        <span className="header-account-name">{name}</span><DownOutlined className="account-chevron" />
      </Button>
    </Dropdown>
      <Button type="text" className="header-logout" icon={<LogoutOutlined />} aria-label="退出登录" loading={isLoggingOut} onClick={onLogout}>
        <span className="header-logout-label">退出登录</span>
      </Button>
    </div>
  </Layout.Header>;
}
