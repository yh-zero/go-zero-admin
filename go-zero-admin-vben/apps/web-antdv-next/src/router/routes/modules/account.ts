import type { RouteRecordRaw } from 'vue-router';

const routes: RouteRecordRaw[] = [
  {
    name: 'BusinessAccount',
    path: '/account',
    component: () => import('#/views/business/account/index.vue'),
    meta: { hideInMenu: true, icon: 'lucide:user-round', title: '个人中心' },
  },
];
export default routes;
