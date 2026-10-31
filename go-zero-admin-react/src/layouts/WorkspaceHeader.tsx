import { Avatar, Breadcrumb, Button, Dropdown, Layout } from 'antd';
import { DownOutlined, HomeOutlined, LogoutOutlined, MenuFoldOutlined, MenuUnfoldOutlined, UserOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { clearSession } from '../utils/session';

interface Props { title: string; name: string; navigationOpen: boolean; onToggle: () => void }

export function WorkspaceHeader({ title, name, navigationOpen, onToggle }: Props) {
  const navigate = useNavigate();
  return <Layout.Header className="app-header">
    <div className="header-location">
      <Button type="text" className="nav-toggle" aria-label={navigationOpen ? '收起导航' : '展开导航'}
        aria-expanded={navigationOpen} icon={navigationOpen ? <MenuFoldOutlined /> : <MenuUnfoldOutlined />} onClick={onToggle} />
      <span className="header-divider" />
      <Breadcrumb items={[{ title: <><HomeOutlined /><span className="breadcrumb-home">控制台</span></> }, { title }]} />
    </div>
    <Dropdown trigger={['click']} menu={{ items: [
      { key: 'profile', label: '个人中心', icon: <UserOutlined /> },
      { type: 'divider' },
      { key: 'logout', label: '退出登录', icon: <LogoutOutlined />, danger: true },
    ], onClick: ({ key }) => {
      if (key === 'logout') { clearSession(); navigate('/login', { replace: true }); }
      else navigate('/sys/profile');
    } }}>
      <Button type="text" className="header-account" aria-label={`${name}，账号菜单`}>
        <Avatar className="header-avatar" size={34} icon={<UserOutlined />} />
        <span className="header-account-name">{name}</span><DownOutlined className="account-chevron" />
      </Button>
    </Dropdown>
  </Layout.Header>;
}
