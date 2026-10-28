import type { Menu, Policy } from '../../api/business/system/types';
// Explicit dependencies are reviewable hints; they never grant API access.
export const permissionApiDependencies: Record<string, Policy[]> = {
  'user:create': [{ method: 'POST', path: '/v1/sys/register' }],
  'user:update': [
    { method: 'PUT', path: '/v1/sys/updateUserInfo' },
    { method: 'GET', path: '/v1/sys/user/resources' },
    { method: 'POST', path: '/v1/sys/user/resources/transfer' },
  ],
  'user:delete': [
    { method: 'GET', path: '/v1/sys/user/resources' },
    { method: 'DELETE', path: '/v1/sys/deleteUser' },
  ],
  'user:resetPassword': [{ method: 'PUT', path: '/v1/sys/resetUserPassword' }],
  'user:membership': [
    { method: 'GET', path: '/v1/sys/organization/membership' },
    { method: 'PUT', path: '/v1/sys/organization/membership' },
    { method: 'GET', path: '/v1/sys/organization/departments' },
    { method: 'GET', path: '/v1/sys/organization/positions' },
  ],
  'authority:create': [
    { method: 'POST', path: '/v1/sys/authority/createAuthority' },
  ],
  'authority:update': [
    { method: 'PUT', path: '/v1/sys/authority/updateAuthority' },
  ],
  'authority:delete': [
    { method: 'DELETE', path: '/v1/sys/authority/deleteAuthority' },
  ],
  'authority:menus': [
    { method: 'GET', path: '/v1/sys/permissions/edit' },
    { method: 'POST', path: '/v1/sys/authority/addAuthorityMenu' },
  ],
  'authority:buttons': [
    { method: 'GET', path: '/v1/sys/permissions/edit' },
    { method: 'PUT', path: '/v1/sys/menu/updateAuthorityButtons' },
  ],
  'authority:apis': [
    { method: 'GET', path: '/v1/sys/permissions/edit' },
    { method: 'PUT', path: '/v1/sys/casbin/updateCasbinData' },
    { method: 'PUT', path: '/v1/sys/casbin/updateCasbinDataByApiIds' },
    { method: 'GET', path: '/v1/sys/permissions/history' },
    { method: 'POST', path: '/v1/sys/permissions/rollback/preview' },
    { method: 'POST', path: '/v1/sys/permissions/rollback/apply' },
  ],
  'authority:dataScope': [
    { method: 'GET', path: '/v1/sys/permissions/edit' },
    { method: 'PUT', path: '/v1/sys/organization/dataScope' },
  ],
  'menu:create': [{ method: 'POST', path: '/v1/sys/menu/addBaseMenu' }],
  'menu:update': [
    { method: 'GET', path: '/v1/sys/menu/getBaseMenuById' },
    { method: 'POST', path: '/v1/sys/menu/previewMove' },
    { method: 'PUT', path: '/v1/sys/menu/updateBaseMenu' },
  ],
  'menu:delete': [{ method: 'DELETE', path: '/v1/sys/menu/deleteBaseMenu' }],
  'api:create': [{ method: 'POST', path: '/v1/sys/api/createApi' }],
  'api:update': [{ method: 'PUT', path: '/v1/sys/api/updateApi' }],
  'api:delete': [
    { method: 'DELETE', path: '/v1/sys/api/deleteApi' },
    { method: 'DELETE', path: '/v1/sys/api/deleteApisByIds' },
  ],
  'api:sync': [
    { method: 'GET', path: '/v1/sys/api/previewSync' },
    { method: 'POST', path: '/v1/sys/api/applySync' },
  ],
  'dictionary:create': [
    { method: 'POST', path: '/v1/sys/dictionary/createSysDictionary' },
  ],
  'dictionary:update': [
    { method: 'PUT', path: '/v1/sys/dictionary/updateSysDictionary' },
  ],
  'dictionary:delete': [
    { method: 'DELETE', path: '/v1/sys/dictionary/deleteSysDictionary' },
  ],
  'dictionary:items': [
    { method: 'GET', path: '/v1/sys/dictionary/getSysDictionaryInfoList' },
    { method: 'POST', path: '/v1/sys/dictionary/createSysDictionaryInfo' },
    { method: 'PUT', path: '/v1/sys/dictionary/updateSysDictionaryInfo' },
    { method: 'DELETE', path: '/v1/sys/dictionary/deleteSysDictionaryInfo' },
  ],
  'organization-departments:create': [
    { method: 'POST', path: '/v1/sys/organization/departments' },
  ],
  'organization-departments:update': [
    { method: 'PUT', path: '/v1/sys/organization/departments' },
  ],
  'organization-departments:delete': [
    { method: 'DELETE', path: '/v1/sys/organization/departments' },
  ],
  'organization-positions:create': [
    { method: 'POST', path: '/v1/sys/organization/positions' },
  ],
  'organization-positions:update': [
    { method: 'PUT', path: '/v1/sys/organization/positions' },
  ],
  'organization-positions:delete': [
    { method: 'DELETE', path: '/v1/sys/organization/positions' },
  ],
  'files:delete': [{ method: 'DELETE', path: '/v1/sys/files/resource' }],
  'businessTools:upload': [
    { method: 'POST', path: '/v1/sys/base/uploadFileImg' },
  ],
  'files:upload': [{ method: 'POST', path: '/v1/sys/base/uploadFileImg' }],
  'app-members-users:avatar': [
    { method: 'PUT', path: '/v1/sys/app-members/avatar' },
  ],
  'files:reference': [
    { method: 'POST', path: '/v1/sys/files/reference' },
    { method: 'DELETE', path: '/v1/sys/files/reference' },
  ],
};
export function missingPermissionApis(
  menus: Menu[],
  selected: number[],
  policies: Policy[],
) {
  const grants = new Set(
    policies.map((rule) => `${rule.method.toUpperCase()} ${rule.path}`),
  );
  const buttons = new Set(selected);
  const missing: Array<{ button: string; api: string }> = [];
  function visit(nodes: Menu[]) {
    for (const menu of nodes) {
      for (const button of menu.menuBtn ?? [])
        if (button.ID && buttons.has(button.ID)) {
          const key = button.permissionKey || `${menu.name}:${button.name}`;
          for (const api of permissionApiDependencies[key] ??
            permissionApiDependencies[`${menu.name}:${button.name}`] ??
            []) {
            const rule = `${api.method} ${api.path}`;
            if (!grants.has(rule))
              missing.push({ button: button.desc || key, api: rule });
          }
        }
      visit(menu.children ?? []);
    }
  }
  visit(menus);
  return missing;
}
