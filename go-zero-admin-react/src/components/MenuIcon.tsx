import {
  ApiOutlined, AppstoreOutlined, ApartmentOutlined, AuditOutlined, BookOutlined,
  DashboardOutlined, FileTextOutlined, FolderOpenOutlined, FolderOutlined,
  IdcardOutlined, MenuOutlined, RightOutlined, SafetyCertificateOutlined, SettingOutlined,
  TeamOutlined, UserOutlined,
} from '@ant-design/icons';
import type { MouseEvent } from 'react';
import type { MenuInfo } from '../api/types';
import styles from './MenuIcon.module.css';

type IconEntry = { icon: typeof UserOutlined; tone: string };
const icons: Record<string, IconEntry> = {
  dashboard: { icon: DashboardOutlined, tone: 'blue' },
  user: { icon: UserOutlined, tone: 'blue' },
  team: { icon: TeamOutlined, tone: 'blue' },
  role: { icon: SafetyCertificateOutlined, tone: 'violet' },
  menu: { icon: MenuOutlined, tone: 'indigo' },
  dept: { icon: ApartmentOutlined, tone: 'cyan' },
  dict: { icon: BookOutlined, tone: 'amber' },
  api: { icon: ApiOutlined, tone: 'indigo' },
  file: { icon: FileTextOutlined, tone: 'green' },
  audit: { icon: AuditOutlined, tone: 'slate' },
  profile: { icon: IdcardOutlined, tone: 'violet' },
  system: { icon: SettingOutlined, tone: 'indigo' },
};
const aliases: Record<string, string> = {
  home: 'dashboard', users: 'user', safety: 'role', safetycertificate: 'role',
  partition: 'dept', apartment: 'dept', department: 'dept', book: 'dict', dictionary: 'dict',
  filetext: 'file', filesearch: 'audit', auditlog: 'audit', idcard: 'profile', account: 'profile',
  setting: 'system', settings: 'system',
};
const normalize = (value: string) => value.replace(/outlined$/i, '').replace(/[-_\s]/g, '').toLowerCase();

interface Props {
  menu: Pick<MenuInfo, 'name' | 'icon' | 'path' | 'type'>;
  expanded?: boolean; className?: string; variant?: 'plain' | 'badge';
}

export function MenuIcon({ menu, expanded = false, className = '', variant = 'plain' }: Props) {
  const configured = normalize(menu.icon || '');
  const configuredIcon = icons[aliases[configured] ?? configured];
  const name = normalize(menu.name || menu.path.split('/').filter(Boolean).at(-1) || '');
  const semanticIcon = icons[aliases[name] ?? name];
  // 通用的 menu/file 占位优先按页面用途识别，明确配置的业务图标则继续保留。
  const entry = configuredIcon && configured !== 'menu' && configured !== 'file'
    ? configuredIcon : semanticIcon ?? configuredIcon;
  const Icon = entry?.icon ?? (menu.type === 1 ? expanded ? FolderOpenOutlined : FolderOutlined : AppstoreOutlined);
  return <span aria-hidden="true" className={`${styles[variant]} ${variant === 'badge' ? styles[entry?.tone ?? 'indigo'] : ''} ${className}`}>
    <Icon />
  </span>;
}

interface ExpandProps {
  title: string; expanded: boolean; expandable: boolean;
  onToggle: (event: MouseEvent<HTMLButtonElement>) => void;
}

export function MenuExpandToggle({ title, expanded, expandable, onToggle }: ExpandProps) {
  if (!expandable) return <span className={styles.expandSpacer} aria-hidden="true" />;
  return <button type="button" className={`${styles.expandToggle} ${expanded ? styles.expanded : ''}`}
    aria-label={`${expanded ? '收起' : '展开'}${title}`} aria-expanded={expanded}
    onClick={event => { event.stopPropagation(); onToggle(event); }}><RightOutlined /></button>;
}
