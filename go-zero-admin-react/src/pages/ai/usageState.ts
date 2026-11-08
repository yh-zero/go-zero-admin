import type { AiSetUsagePolicyReq, AiUsageCallInfo, AiUsagePolicyInfo } from '../../api/types';
import type { AiWorkspaceScope } from './workspaceQueries';

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
