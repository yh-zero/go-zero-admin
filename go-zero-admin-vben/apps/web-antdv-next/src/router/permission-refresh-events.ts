// Keep auth/shared entry points independent of layouts and route generation.
let actions:
  | undefined
  | {
      refresh: (force?: boolean) => Promise<void>;
      reset: () => void;
      repair: () => void;
    };
export function registerPermissionRefresh(next: typeof actions) {
  actions = next;
}
export async function refreshCurrentPermissions(force = false) {
  await actions?.refresh(force);
  actions?.repair();
}
export function resetPermissionRefresh() {
  actions?.reset();
}
export function notifyPermissionChange() {
  localStorage.setItem(
    'business-permission-change',
    `${Date.now()}-${Math.random()}`,
  );
  void refreshCurrentPermissions(true).catch(() => {});
}
