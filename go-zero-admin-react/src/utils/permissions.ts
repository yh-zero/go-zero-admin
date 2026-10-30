export function requiredWritePermission(method: string, url: string): string | undefined {
  const match = /^\/sys\/(user|role|dept|menu|dict|api|file)\/(create|\d+)(?:\/(password|menus))?$/.exec(url);
  if (!match) return undefined;
  const [, module, target, operation] = match;
  const verb = method.toUpperCase();
  if (verb === 'POST' && target === 'create') return 'sys:' + module + ':add';
  if (verb === 'POST' && module === 'user' && operation === 'password') return 'sys:user:reset-password';
  if (verb === 'POST' && module === 'role' && operation === 'menus') return 'sys:role:menus';
  if (verb === 'PUT') return 'sys:' + module + ':edit';
  if (verb === 'DELETE') return 'sys:' + module + ':delete';
  return undefined;
}
