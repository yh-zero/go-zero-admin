import type { AuditQuery } from '#/api/business/system/audit';

import dayjs from 'dayjs';

export interface AuditFilter extends Omit<AuditQuery, 'endTime' | 'startTime'> {
  timeRange?: [string, string];
}
export function auditQuery(filter: AuditFilter): AuditQuery {
  const { timeRange, ...query } = filter;
  if (!timeRange?.length) return query;
  const start = dayjs(timeRange[0]);
  const end = dayjs(timeRange[1]);
  if (!start.isValid() || !end.isValid() || start.isAfter(end))
    throw new Error('请选择有效的时间范围，开始时间不能晚于结束时间');
  return {
    ...query,
    startTime: start.toISOString(),
    endTime: end.toISOString(),
  };
}
export function auditTime(value: string): string {
  const time = dayjs(value);
  return time.isValid() ? time.format('YYYY-MM-DD HH:mm:ss') : '—';
}

const labels: Record<string, string> = {
  id: '对象 ID',
  ids: '对象 ID',
  userid: '用户 ID',
  authorityid: '角色 ID',
  authorityids: '关联角色',
  menuids: '菜单 ID',
  menubtnids: '按钮 ID',
  parentid: '父级 ID',
  sysbasemenuid: '菜单 ID',
  sysdictionaryid: '字典 ID',
  status: '状态',
  enable: '启用状态',
  method: '请求方法',
};
// Keep credential-like fields hidden even if a future backend returns them.
// Free-form values are excluded; audit details only display resource identifiers.
export function safeAuditParams(
  value: string,
): { label: string; value: string }[] {
  if (!value || value.length > 4096) return [];
  try {
    const data: unknown = JSON.parse(value);
    if (!data || typeof data !== 'object' || Array.isArray(data)) return [];
    return Object.entries(data).flatMap(([rawKey, entry]) => {
      const key = rawKey.toLowerCase();
      const label = labels[key];
      if (!label) return [];
      const numeric = (item: unknown) =>
        typeof item === 'number' && Number.isSafeInteger(item) && item >= 0;
      if (numeric(entry) || typeof entry === 'boolean')
        return [{ label, value: String(entry) }];
      if (
        Array.isArray(entry) &&
        entry.length > 0 &&
        entry.length <= 100 &&
        entry.every(numeric)
      )
        return [{ label, value: entry.join(', ') }];
      if (
        typeof entry === 'string' &&
        (key === 'method'
          ? /^(GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS)$/.test(entry)
          : /^\d+(,\d+)*$/.test(entry))
      )
        return [{ label, value: entry }];
      return [];
    });
  } catch {
    return [];
  }
}
