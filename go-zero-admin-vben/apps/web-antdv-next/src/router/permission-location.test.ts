import {
  createMemoryHistory,
  createRouter,
  type RouteRecordRaw,
} from 'vue-router';

import { commitAccessibleRoutes } from '@vben/access';

import { describe, expect, it } from 'vitest';

import {
  permissionNavigationTarget,
  synchronizePermissionTabs,
} from './permission-location';
const layout = { render: () => null };
const oldComponent = { render: () => null };
const newComponent = { render: () => null };
async function setup(previous: RouteRecordRaw[]) {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { name: 'Root', path: '/', component: layout, children: previous },
      { name: 'Home', path: '/home', component: layout },
      { name: 'FallbackNotFound', path: '/:path(.*)*', component: layout },
    ],
  });
  return router;
}
describe('permission route rebind with real memory router', () => {
  it('moves an unchanged route name to its new path preserving query/hash and drops its stale tab', async () => {
    const previous: RouteRecordRaw[] = [
      { name: 'Business', path: '/old', component: oldComponent },
    ];
    const router = await setup(previous);
    await router.replace('/old?filter=active#section');
    const old = router.currentRoute.value;
    commitAccessibleRoutes(
      router,
      [{ name: 'Business', path: '/new', component: newComponent }],
      previous,
    );
    expect(router.hasRoute('Business')).toBe(true);
    expect(router.resolve(old.fullPath).name).toBe('FallbackNotFound');
    expect(synchronizePermissionTabs(router, [old])).toEqual([]);
    const target = permissionNavigationTarget(router, old, '/home');
    expect(target).toBeDefined();
    await router.replace(target!);
    expect(router.currentRoute.value.fullPath).toBe(
      '/new?filter=active#section',
    );
    expect(
      permissionNavigationTarget(router, router.currentRoute.value, '/home'),
    ).toBeUndefined();
  });
  it('rebinds a leaf after moving its parent without losing query/hash', async () => {
    const previous: RouteRecordRaw[] = [
      {
        name: 'Group',
        path: '/old-parent',
        component: layout,
        children: [
          { name: 'Business', path: 'child', component: oldComponent },
        ],
      },
    ];
    const router = await setup(previous);
    await router.replace('/old-parent/child?q=x#item');
    const old = router.currentRoute.value;
    commitAccessibleRoutes(
      router,
      [
        {
          name: 'Group',
          path: '/new-parent',
          component: layout,
          children: [
            { name: 'Business', path: 'child', component: newComponent },
          ],
        },
      ],
      previous,
    );
    const target = permissionNavigationTarget(router, old, '/home');
    expect(target).toBeDefined();
    await router.replace(target!);
    expect(router.currentRoute.value.fullPath).toBe(
      '/new-parent/child?q=x#item',
    );
    expect(synchronizePermissionTabs(router, [old])).toEqual([]);
  });
  it('replaces stale matched records when path/name stay the same but component/meta changed', async () => {
    const previous: RouteRecordRaw[] = [
      {
        name: 'Business',
        path: '/same',
        component: oldComponent,
        meta: { title: 'Old' },
      },
    ];
    const router = await setup(previous);
    await router.replace('/same?q=x#item');
    const old = router.currentRoute.value;
    commitAccessibleRoutes(
      router,
      [
        {
          name: 'Business',
          path: '/same',
          component: newComponent,
          meta: { title: 'New' },
        },
      ],
      previous,
    );
    expect(old.matched.at(-1)?.components?.default).toBe(oldComponent);
    const target = permissionNavigationTarget(router, old, '/home');
    expect(target).toBeDefined();
    await router.replace(target!);
    expect(router.currentRoute.value.matched.at(-1)?.components?.default).toBe(
      newComponent,
    );
    const tabs = synchronizePermissionTabs(router, [old]);
    expect(tabs[0]?.meta.title).toBe('New');
    expect(tabs[0]?.matched.at(-1)?.components?.default).toBe(newComponent);
  });
  it('falls back to an available home when the current route was revoked', async () => {
    const previous: RouteRecordRaw[] = [
      { name: 'Business', path: '/old', component: oldComponent },
    ];
    const router = await setup(previous);
    await router.replace('/old');
    const old = router.currentRoute.value;
    commitAccessibleRoutes(router, [], previous);
    const target = permissionNavigationTarget(router, old, '/home');
    await router.replace(target!);
    expect(router.currentRoute.value.name).toBe('Home');
  });
});
