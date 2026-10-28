import type { RouteRecordStringComponent } from '@vben/types';

import type { BackendMenu } from '../../api/business/session';

import { resolveBusinessPage } from './pages';
import { SESSION_HOME } from './session';

const icons: Record<string, string> = {
  AntDesignOutlined: 'lucide:house',
  BugOutlined: 'lucide:settings',
  CheckSquareOutlined: 'lucide:shield-check',
  ColumnHeightOutlined: 'lucide:list',
  StopFilled: 'lucide:settings',
};

function resolveMenuIcon(icon = '') {
  const legacyIcon = icons[icon];
  if (typeof legacyIcon === 'string') return legacyIcon;
  return /^[a-z0-9]+(?:-[a-z0-9]+)*:[a-z0-9]+(?:-[a-z0-9]+)*$/.test(icon)
    ? icon
    : 'lucide:folder';
}

const reservedNames = new Set([
  'AccessError',
  'Authentication',
  'BusinessAccount',
  'BusinessSessionHome',
  'FallbackNotFound',
  'Login',
  'Root',
]);

export function adaptMenus(
  menus: BackendMenu[] | null,
  defaultRouter: string,
  authorityId: string,
) {
  const names = new Set<string>();
  const paths = new Set<string>();
  const codes = new Set<string>();
  const homes: { name: string; path: string }[] = [];
  const directoryHomes = new Map<string, string>();

  function visit(
    nodes: BackendMenu[] | null | undefined,
    parent = '',
    parentHidden = false,
  ): RouteRecordStringComponent[] {
    if (!nodes) return [];
    if (!Array.isArray(nodes)) throw new Error('后端菜单数据格式错误');
    return [...nodes]
      .sort((a, b) => (a.sort ?? 0) - (b.sort ?? 0))
      .map((node) => {
        if (!node.name || reservedNames.has(node.name) || names.has(node.name))
          throw new Error('后端菜单名称重复或与系统路由冲突');
        names.add(node.name);
        if (
          !node.path ||
          /[\\?#:]|(^|\/)\.{1,2}(\/|$)/.test(node.path) ||
          node.path.startsWith('//')
        )
          throw new Error('后端菜单路径无效');
        const path = (
          node.path.startsWith('/') ? node.path : `${parent}/${node.path}`
        )
          .replace(/\/{2,}/g, '/')
          .replace(/\/$/, '');
        // Vue Router matches paths without case sensitivity by default.
        const pathKey = path.toLowerCase();
        if (
          !path ||
          pathKey.startsWith('/auth') ||
          pathKey === '/account' ||
          pathKey.startsWith('/account/') ||
          pathKey.startsWith('/_session') ||
          paths.has(pathKey)
        )
          throw new Error('后端菜单路径重复或与系统路由冲突');
        paths.add(pathKey);
        const hidden = parentHidden || !!node.hidden;
        const homeStart = homes.length;
        const children = visit(node.children, path, hidden);
        const component = children.length
          ? '/business/layout/route-view.vue'
          : resolveBusinessPage(node.component || '');
        const route: RouteRecordStringComponent = {
          component,
          meta: {
            hideInMenu: !!node.hidden,
            icon: resolveMenuIcon(node.meta?.icon),
            keepAlive: !!node.meta?.keepAlive,
            order: node.sort ?? 0,
            title: node.meta?.title || node.name,
          },
          name: node.name,
          path,
        };
        if (children.length) {
          route.children = children;
          route.redirect = homes[homeStart]?.path || children[0]!.path;
          if (!hidden && homes[homeStart])
            directoryHomes.set(node.name, homes[homeStart]!.path);
        } else if (!hidden) {
          homes.push({ name: node.name, path });
        }
        for (const [button, role] of Object.entries(node.btns || {})) {
          if (button && String(role) === authorityId)
            codes.add(`${node.name}:${button}`);
        }
        return route;
      });
  }
  const routes = visit(menus);
  const homePath =
    homes.find((home) => home.name === defaultRouter)?.path ||
    directoryHomes.get(defaultRouter) ||
    homes[0]?.path ||
    SESSION_HOME;
  routes.push({
    component: '/business/access/empty.vue',
    meta: { hideInMenu: true, title: '暂无可用菜单' },
    name: 'BusinessSessionHome',
    path: SESSION_HOME,
    ...(homePath === SESSION_HOME ? {} : { redirect: homePath }),
  });
  return { codes: [...codes], homePath, routes };
}
