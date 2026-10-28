// Register a business page here once: both menu editing and route resolution use it.
// value is the component identifier stored by the backend; component is a Vben view path.
export const businessPages = [
  {
    label: 'App 用户',
    value: 'views/business/app-members/users.vue',
    component: '/business/app-members/users.vue',
  },
  {
    label: 'App VIP 会员',
    value: 'views/business/app-members/memberships.vue',
    component: '/business/app-members/memberships.vue',
  },
  {
    label: 'AI Agent',
    value: 'views/business/ai-agent/index.vue',
    component: '/business/ai-agent/index.vue',
  },
  {
    label: '首页',
    value: 'views/index.vue',
    component: '/dashboard/workspace/index.vue',
  },
  {
    label: '目录',
    value: 'views/superAdmin/index.vue',
    component: '/business/access/pending.vue',
  },
  {
    label: '个人中心',
    value: 'views/business/account/index.vue',
    component: '/business/account/index.vue',
  },
  {
    label: '业务首页',
    value: 'views/business/home/index.vue',
    component: '/business/home/index.vue',
  },
  {
    label: '用户管理',
    value: 'views/superAdmin/user/user.vue',
    component: '/business/system/user/index.vue',
  },
  {
    label: '菜单管理',
    value: 'views/superAdmin/menu/menu.vue',
    component: '/business/system/menu/index.vue',
  },
  {
    label: '角色管理',
    value: 'views/superAdmin/authority/authority.vue',
    component: '/business/system/authority/index.vue',
  },
  {
    label: 'API 管理',
    value: 'views/superAdmin/api/api.vue',
    component: '/business/system/api/index.vue',
  },
  {
    label: '字典管理',
    value: 'views/superAdmin/dictionary/dictionary.vue',
    component: '/business/system/dictionary/index.vue',
  },
  {
    label: '管理工具',
    value: 'views/business/tools/index.vue',
    component: '/business/tools/index.vue',
  },
  {
    label: '部门管理',
    value: 'views/business/system/organization/departments.vue',
    component: '/business/system/organization/departments.vue',
  },
  {
    label: '岗位管理',
    value: 'views/business/system/organization/positions.vue',
    component: '/business/system/organization/positions.vue',
  },
  {
    label: '操作审计',
    value: 'views/business/system/audit/index.vue',
    component: '/business/system/audit/index.vue',
  },
  {
    label: '文件资源',
    value: 'views/business/system/files/index.vue',
    component: '/business/system/files/index.vue',
  },
  {
    label: '在线会话',
    value: 'views/business/system/sessions/index.vue',
    component: '/business/system/sessions/index.vue',
  },
];

const pageMap = new Map(
  businessPages.map((page) => [page.value, page.component]),
);

export function resolveBusinessPage(value: string) {
  return pageMap.get(value) ?? '/business/access/pending.vue';
}
