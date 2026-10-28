export const DEFAULT_AVATAR = `${import.meta.env.BASE_URL}default-avatar.svg`;

export function resolveAvatar(value?: null | string): string {
  return !value || value === '/default-avatar.svg' ? DEFAULT_AVATAR : value;
}
