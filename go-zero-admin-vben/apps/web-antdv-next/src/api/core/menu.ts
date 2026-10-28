import { adaptMenus } from '#/adapter/business/menu';
import { getPermissionSnapshot } from '#/api/business/system/permissions';
export async function getAllMenusApi() {
  const result = await getPermissionSnapshot();
  const adapted = adaptMenus(
    result.menus,
    result.defaultRouter,
    String(result.authorityId),
  );
  return adapted.routes;
}
