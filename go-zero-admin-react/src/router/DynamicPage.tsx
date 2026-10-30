import { Result } from 'antd';
import { useQuery } from '@tanstack/react-query';
import { useLocation } from 'react-router-dom';
import { fetchMyMenus } from '../api/auth';
import { flattenMenus, pageNameForMenu, safeMenuPath } from './menuRoutes';
import { pages } from './registry';
import PageElement from './PageElement';
export default function DynamicPage() {
  const location = useLocation();
  const menus = useQuery({ queryKey: ['my-menus'], queryFn: fetchMyMenus });
  const menu = flattenMenus(menus.data ?? []).find(item => safeMenuPath(item.path) === location.pathname && item.type === 2 && item.status === 1);
  const name = menu && pageNameForMenu(menu);
  return name && Object.hasOwn(pages, name) ? <PageElement name={name as keyof typeof pages} /> : <Result status="404" title="页面不存在" subTitle="该页面尚未接入，或菜单未能加载。" />;
}
