// 缓存键集中定义：页面 queryKey 与失效列表共用同一常量，
// 避免两边拼写不一致导致缓存失效静默失效。
export const queryKeys = {
  apiResources: 'api-resources',
  appFeedback: 'app-feedback',
  appMembershipPlans: 'app-membership-plans',
  appMemberships: 'app-memberships',
  appSessions: 'app-sessions',
  appUser: 'app-user',
  appUserOptions: 'app-user-options',
  appUsers: 'app-users',
  auditLogs: 'audit-logs',
  captcha: 'captcha',
  depts: 'depts',
  deviceSessions: 'device-sessions',
  dicts: 'dicts',
  fileUrl: 'file-url',
  files: 'files',
  health: 'health',
  menus: 'menus',
  myDepts: 'my-depts',
  myMenus: 'my-menus',
  permCodes: 'perm-codes',
  profile: 'profile',
  role: 'role',
  roleMenuOptions: 'role-menu-options',
  roleOptions: 'role-options',
  roles: 'roles',
  tenantList: 'tenant-list',
  user: 'user',
  users: 'users',
} as const;

export type QueryKeyName = keyof typeof queryKeys;
