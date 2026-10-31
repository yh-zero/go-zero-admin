import { Card } from 'antd';
import { ArrowRightOutlined, FileOutlined, MenuOutlined, PartitionOutlined, SafetyOutlined, TeamOutlined, UserOutlined } from '@ant-design/icons';
import { Link } from 'react-router-dom';

const shortcuts = [
  { title: '用户管理', description: '账号与成员信息', path: '/sys/user', icon: <TeamOutlined /> },
  { title: '角色管理', description: '角色与菜单授权', path: '/sys/role', icon: <SafetyOutlined /> },
  { title: '部门管理', description: '组织层级与联系信息', path: '/sys/dept', icon: <PartitionOutlined /> },
  { title: '菜单管理', description: '导航与页面配置', path: '/sys/menu', icon: <MenuOutlined /> },
  { title: '文件管理', description: '文件资源与链接', path: '/sys/file', icon: <FileOutlined /> },
  { title: '个人中心', description: '资料与账号设置', path: '/sys/profile', icon: <UserOutlined /> },
];

export function DashboardShortcuts() {
  return <Card className="shortcut-card" title="常用入口"><div className="shortcut-list">
    {shortcuts.map(item => <Link className="shortcut-item" key={item.path} to={item.path}>
      <span className="shortcut-icon">{item.icon}</span><span className="shortcut-copy"><strong>{item.title}</strong><small>{item.description}</small></span><ArrowRightOutlined className="shortcut-arrow" />
    </Link>)}
  </div></Card>;
}
