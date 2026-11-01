import type { AppUserStatus } from '../../api/types';

export const appUserStatus = {
  pending: { label: '待审核', color: 'gold' },
  active: { label: '已启用', color: 'green' },
  disabled: { label: '已停用', color: 'default' },
  rejected: { label: '已驳回', color: 'red' },
} satisfies Record<AppUserStatus, { label: string; color: string }>;

export const appUserStatusOptions = Object.entries(appUserStatus).map(([value, item]) => ({
  value: value as AppUserStatus, label: item.label,
}));
