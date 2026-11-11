import type { AiSetUsagePolicyReq, AiUsageAdminQuery, AiUsageCallInfo, AiUsagePolicyInfo } from '../../api/types';
import type { AiWorkspaceScope } from './workspaceQueries';

export function usageDateRange(days: 1 | 7 | 30, now = new Date()) {
  const parts = new Intl.DateTimeFormat('en-CA', { timeZone: 'Asia/Shanghai', year: 'numeric', month: '2-digit', day: '2-digit' }).formatToParts(now);
  const part = (name: string) => parts.find(value => value.type === name)?.value;
  const to_day = `${part('year')}-${part('month')}-${part('day')}`;
  const end = Date.parse(`${to_day}T00:00:00Z`);
  return { from_day: new Date(end - (days - 1) * 86_400_000).toISOString().slice(0, 10), to_day };
}

export function usageDateRangeError(fromDay?: string, toDay?: string): string | undefined {
  if (!fromDay || !toDay) return '请选择开始和结束日期';
  const timestamp = (day: string) => {
    if (!/^\d{4}-\d{2}-\d{2}$/.test(day)) return NaN;
    const value = Date.parse(`${day}T00:00:00Z`);
    return Number.isFinite(value) && new Date(value).toISOString().slice(0, 10) === day ? value : NaN;
  };
  const from = timestamp(fromDay); const to = timestamp(toDay);
  if (!Number.isFinite(from) || !Number.isFinite(to)) return '请输入有效日期';
  if (to < from) return '结束日期不能早于开始日期';
  if ((to - from) / 86_400_000 >= 31) return '单次最多查询 31 日（包含开始和结束日期）';
  return undefined;
}

export function usageFilterDefaults(now = new Date()): AiUsageAdminQuery {
  return { ...usageDateRange(7, now), page: 1, page_size: 20 };
}

export function usageFilterParams(values: AiUsageAdminQuery, admin: boolean): AiUsageAdminQuery {
  const error = usageDateRangeError(values.from_day, values.to_day);
  if (error) throw new Error(error);
  const result = { ...values, provider: values.provider?.trim(), model: values.model?.trim(), page: 1, page_size: 20 };
  if (!result.provider) delete result.provider;
  if (!result.model) delete result.model;
  if (!result.source) delete result.source;
  if (!result.operation) delete result.operation;
  if (!result.status) delete result.status;
  if (!result.usage_state) delete result.usage_state;
  if (!admin || result.user_id === undefined || result.user_id === null) delete result.user_id;
  return result;
}

export function usageTokenLabel(call: Pick<AiUsageCallInfo, 'usage_state' | 'total_tokens'>) {
  if (call.total_tokens === undefined || call.total_tokens === null) return '未取得';
  return `${call.total_tokens.toLocaleString()}${call.usage_state === 'complete' ? '' : '（不完整）'}`;
}

export function usageQueryKey(scope: AiWorkspaceScope, admin: boolean, part: string, params: unknown = {}) {
  return ['ai-usage', ...scope, admin ? 'admin' : 'self', part, params] as const;
}

export function usagePolicyPatch(policy: Pick<AiUsagePolicyInfo, 'owner_id' | 'version'>, values: Omit<AiSetUsagePolicyReq, 'expected_version'>): AiSetUsagePolicyReq {
  if (policy.owner_id === 0 && values.inherit_default) throw new Error('全局默认策略不能继承自己');
  if (values.daily_request_limit !== undefined && (!Number.isSafeInteger(values.daily_request_limit) || values.daily_request_limit < 0 || values.daily_request_limit > 1_000_000)) throw new Error('每日请求额度须为 0..1000000 的整数');
  if (values.daily_token_warning !== undefined && (!Number.isSafeInteger(values.daily_token_warning) || values.daily_token_warning < 0)) throw new Error('token 提醒阈值须为非负安全整数');
  if (values.global_concurrent_limit !== undefined && (policy.owner_id !== 0 || !Number.isInteger(values.global_concurrent_limit) || values.global_concurrent_limit < 1 || values.global_concurrent_limit > 64)) throw new Error('全局并发须为 1..64 的整数');
  const result: AiSetUsagePolicyReq = { expected_version: policy.version, ...values };
  if (values.inherit_default) delete result.daily_request_limit;
  return result;
}
