export function requiredWritePermission(method: string, url: string): string | undefined {
  const verb = method.toUpperCase();
  if (verb === 'POST' && url === '/sys/ai/conversations') return 'sys:ai:conversation:create';
  if (verb === 'POST' && (url === '/sys/ai/messages' || url === '/sys/ai/messages/stream')) return 'sys:ai:message:send';
  if (verb === 'DELETE' && /^\/sys\/ai\/conversations\/\d+$/.test(url)) return 'sys:ai:conversation:delete';
  if (verb === 'PUT' && /^\/sys\/app\/users\/\d+$/.test(url)) return 'sys:app:user:edit';
  if (verb === 'PUT' && /^\/sys\/app\/feedback\/\d+$/.test(url)) return 'sys:app:feedback:reply';
  if (verb === 'POST' && /^\/sys\/app\/users\/\d+\/review$/.test(url)) return 'sys:app:user:review';
  if (verb === 'PUT' && /^\/sys\/app\/users\/\d+\/avatar$/.test(url)) return 'sys:app:user:avatar';
  if (verb === 'DELETE' && /^\/sys\/app\/users\/\d+\/sessions\/[^/]+$/.test(url)) return 'sys:app:session:revoke';
  if (verb === 'POST' && url === '/sys/app/memberships') return 'sys:app:membership:grant';
  if (verb === 'DELETE' && /^\/sys\/app\/memberships\/\d+$/.test(url)) return 'sys:app:membership:revoke';
  const match = /^\/sys\/(user|role|dept|menu|dict|api|file)\/(create|\d+)(?:\/(password|menus))?$/.exec(url);
  if (!match) return undefined;
  const [, module, target, operation] = match;
  if (verb === 'POST' && target === 'create') return 'sys:' + module + ':add';
  if (verb === 'POST' && module === 'user' && operation === 'password') return 'sys:user:reset-password';
  if (verb === 'POST' && module === 'role' && operation === 'menus') return 'sys:role:menus';
  if (verb === 'PUT') return 'sys:' + module + ':edit';
  if (verb === 'DELETE') return 'sys:' + module + ':delete';
  return undefined;
}
