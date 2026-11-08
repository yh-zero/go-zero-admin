import { useState } from 'react';
import { Alert, Button, Card, Drawer, Empty, Form, Input, InputNumber, Pagination, Select, Space, Statistic, Switch, Table, Tabs, Tag, Typography } from 'antd';
import { ReloadOutlined, SettingOutlined } from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { fetchAdminUsageCalls, fetchAdminUsageSummary, fetchMyUsageCalls, fetchMyUsagePolicy, fetchMyUsageSummary, fetchUsagePolicies, setUsagePolicy } from '../../api/aiUsage';
import type { AiSetUsagePolicyReq, AiUsageAdminQuery, AiUsageCallInfo, AiUsagePolicyInfo, AiUsageSummaryInfo } from '../../api/types';
import { QueryState } from '../../components/QueryState';
import { formatTime } from '../../utils/format';
import { AIWorkspaceAccess } from './AIWorkspaceAccess';
import { useAiWorkspaceGuard } from './useAiWorkspaceGuard';
import type { AiWorkspaceScope } from './workspaceQueries';
import { usagePolicyPatch, usageQueryKey, usageTokenLabel } from './usageState';
import styles from './usage.module.css';

const sources = { chat: '对话', analysis: '流程分析', kb_search: '独立检索', kb_ingest: '文档摄取', evaluation: '质量验收' };
const statuses = { reserved: '预留中', running: '请求中', succeeded: '请求完成', failed: '请求失败', cancelled: '已取消', unknown: '结果未知', not_sent: '未发送' };
const usageStates = { complete: '用量完整', partial: '部分用量', unknown: '未取得用量', invalid: '用量无效' };

export default function UsagePage() {
  return <AIWorkspaceAccess code="sys:ai:usage:read" title="用量与额度">{scope => <UsageSession scope={scope} />}</AIWorkspaceAccess>;
}

export function UsageSummaryCards({ summary }: { summary?: AiUsageSummaryInfo }) {
  return <div className={styles.metrics}>
    <Card size="small"><Statistic title="已确认供应商请求" value={summary?.confirmed_requests ?? '—'} /></Card>
    <Card size="small"><Statistic title="已知聊天 token" value={summary ? summary.known_prompt_tokens + summary.known_completion_tokens : '—'} /></Card>
    <Card size="small"><Statistic title="已知 Embedding token" value={summary?.known_embedding_tokens ?? '—'} /></Card>
    <Card size="small"><Statistic title="不确定请求" value={summary?.uncertain_requests ?? '—'} /></Card>
  </div>;
}

function UsageSession({ scope }: { scope: AiWorkspaceScope }) {
  const guard = useAiWorkspaceGuard(scope, 'sys:ai:usage:read');
  const client = useQueryClient();
  const [admin, setAdmin] = useState(false);
  const activeAdmin = admin && guard.can('sys:ai:usage:admin');
  const canManage = guard.can('sys:ai:usage:manage');
  const [params, setParams] = useState<AiUsageAdminQuery>({ page: 1, page_size: 20 });
  const [policyPage, setPolicyPage] = useState(1);
  const query = { ...params };
  if (!activeAdmin) delete query.user_id;
  const summary = useQuery({ queryKey: usageQueryKey(scope, activeAdmin, 'summary', query), retry: false,
    queryFn: ({ signal }) => activeAdmin ? fetchAdminUsageSummary(query, signal) : fetchMyUsageSummary(query, signal) });
  const calls = useQuery({ queryKey: usageQueryKey(scope, activeAdmin, 'calls', query), retry: false,
    queryFn: ({ signal }) => activeAdmin ? fetchAdminUsageCalls(query, signal) : fetchMyUsageCalls(query, signal) });
  const mine = useQuery({ queryKey: usageQueryKey(scope, false, 'policy'), retry: false, queryFn: ({ signal }) => fetchMyUsagePolicy(signal) });
  const policies = useQuery({ queryKey: usageQueryKey(scope, true, 'policies', { page: policyPage }), retry: false,
    enabled: guard.can('sys:ai:usage:admin'), queryFn: ({ signal }) => fetchUsagePolicies({ page: policyPage, page_size: 20 }, signal) });
  const [drawer, setDrawer] = useState(false);
  const [selected, setSelected] = useState<AiUsagePolicyInfo | null>(null);
  const [ownerInput, setOwnerInput] = useState<number>(0);
  const [loadingPolicy, setLoadingPolicy] = useState(false);
  const [notice, setNotice] = useState('');
  const [policyForm] = Form.useForm<Omit<AiSetUsagePolicyReq, 'expected_version'>>();
  const inherit = Form.useWatch('inherit_default', policyForm);
  const save = useMutation({
    mutationFn: ({ policy, values }: { policy: AiUsagePolicyInfo; values: Omit<AiSetUsagePolicyReq, 'expected_version'> }) => setUsagePolicy(policy.owner_id, usagePolicyPatch(policy, values)),
    onSuccess: result => {
      if (!guard.isCurrent()) return;
      showPolicy(result.policy);
      setNotice('策略已保存，正在执行的请求沿用已领取额度，后续准入使用新策略。');
      void client.invalidateQueries({ queryKey: ['ai-usage', ...scope] });
    },
    onError: () => { if (guard.isCurrent()) setNotice('保存未确认或版本已变化。草稿已保留，请先读取当前策略核对，勿直接重复提交。'); },
    onSettled: () => guard.finishWrite(),
  });

  function showPolicy(policy: AiUsagePolicyInfo) {
    setSelected(policy); setOwnerInput(policy.owner_id);
    policyForm.setFieldsValue({ daily_request_limit: policy.daily_request_limit, daily_token_warning: policy.daily_token_warning,
      inherit_default: policy.owner_id > 0 && policy.inherited, global_concurrent_limit: policy.owner_id === 0 ? policy.global_concurrent_limit : undefined });
  }
  function editPolicy(policy?: AiUsagePolicyInfo) {
    setNotice(''); setDrawer(true); setSelected(null); policyForm.resetFields();
    if (policy) showPolicy(policy);
  }
  async function loadPolicy() {
    if (!guard.isCurrent() || !canManage || loadingPolicy || !Number.isSafeInteger(ownerInput) || ownerInput < 0) return;
    setLoadingPolicy(true); setNotice('');
    try {
      const result = await fetchUsagePolicies({ user_id: ownerInput, page: 1, page_size: 100 });
      if (!guard.isCurrent()) return;
      const existing = result.list.find(row => row.owner_id === ownerInput);
      if (existing) showPolicy(existing);
      else {
        const global = policies.data?.list.find(row => row.owner_id === 0);
        showPolicy({ owner_id: ownerInput, version: 0, quota_day: '', daily_request_limit: global?.daily_request_limit ?? 1000,
          daily_token_warning: 0, inherited: true, unlimited: false, used_requests: 0, remaining_requests: 0, known_tokens: 0,
          warning_reached: false, incomplete_requests: 0, global_concurrent_limit: global?.global_concurrent_limit ?? 8, active_calls: 0, active_known: false });
        setNotice('该用户尚无独立策略；保存将按版本 0 新建，保留默认额度继承。');
      }
    } catch { if (guard.isCurrent()) setNotice('读取策略失败，现有草稿已保留，请稍后重新读取。'); }
    finally { if (guard.isCurrent()) setLoadingPolicy(false); }
  }
  function refresh() { if (guard.isCurrent()) void client.invalidateQueries({ queryKey: ['ai-usage', ...scope] }); }
  const totals = summary.data?.summary;
  const policy = mine.data?.policy;
  const incomplete = totals ? totals.partial_requests + totals.unknown_requests + totals.invalid_requests : 0;

  return <div className={styles.workspace}>
    <Card title="用量与额度" extra={<Button icon={<ReloadOutlined />} onClick={refresh} disabled={guard.writing}>刷新</Button>}>
      <Typography.Paragraph type="secondary">按实际供应商请求计数，文档向量化按请求批次计。仅从账本启用后开始记录，历史消息和流程步骤不重复累加。</Typography.Paragraph>
      <Tabs activeKey={activeAdmin ? 'admin' : 'self'} onChange={key => { setAdmin(key === 'admin'); setParams(current => ({ ...current, page: 1 })); }} items={[
        { key: 'self', label: '我的用量' }, ...(guard.can('sys:ai:usage:admin') ? [{ key: 'admin', label: '全局用量' }] : []),
      ]} />
      <Form layout="inline" className={styles.filters} onFinish={values => setParams({ ...values, page: 1, page_size: 20 })}>
        <Form.Item name="from_day" label="开始日"><Input type="date" /></Form.Item>
        <Form.Item name="to_day" label="结束日"><Input type="date" /></Form.Item>
        <Form.Item name="source" label="来源"><Select allowClear className={styles.select} options={Object.entries(sources).map(([value, label]) => ({ value, label }))} /></Form.Item>
        <Form.Item name="operation" label="调用"><Select allowClear className={styles.select} options={[{ value: 'chat', label: '聊天' }, { value: 'embedding', label: 'Embedding' }]} /></Form.Item>
        <Form.Item name="status" label="请求状态"><Select allowClear className={styles.select} options={Object.entries(statuses).map(([value, label]) => ({ value, label }))} /></Form.Item>
        <Form.Item name="usage_state" label="计量"><Select allowClear className={styles.select} options={Object.entries(usageStates).map(([value, label]) => ({ value, label }))} /></Form.Item>
        <Form.Item name="provider" label="供应商"><Input maxLength={32} /></Form.Item>
        <Form.Item name="model" label="模型"><Input maxLength={128} /></Form.Item>
        {activeAdmin && <Form.Item name="user_id" label="后台用户 ID"><InputNumber min={0} precision={0} /></Form.Item>}
        <Form.Item><Button htmlType="submit">查询</Button></Form.Item>
      </Form>
      <Typography.Paragraph type="secondary">日期按上海时区，默认最近 7 日，单次最多 31 日。</Typography.Paragraph>
      <QueryState isPending={summary.isPending} error={summary.error} onRetry={() => summary.refetch()} />
      {!summary.error && <UsageSummaryCards summary={totals} />}
      {!!incomplete && <Alert type="warning" showIcon title={`有 ${incomplete} 次请求未取得完整有效用量`} description="已知 token 不是全部计费量；请求失败、流中断或取消均不代表免费。" />}
      {totals && <Typography.Paragraph type="secondary">账本启用：{formatTime(totals.recording_started_at)}。预留 {totals.reserved_requests} 次，明确未发送 {totals.not_sent_requests} 次；不确定请求保留额度待核对。</Typography.Paragraph>}
      {totals && <details><summary>按来源与模型查看汇总</summary><Table rowKey={row => JSON.stringify([row.source, row.operation, row.provider, row.model])} dataSource={totals.groups} pagination={false} size="small" scroll={{ x: 700 }} columns={[
        { title: '来源', dataIndex: 'source', render: value => sources[value as keyof typeof sources] ?? value },
        { title: '调用', dataIndex: 'operation' }, { title: '供应商', dataIndex: 'provider' }, { title: '模型', dataIndex: 'model' },
        { title: '已确认请求', dataIndex: 'confirmed_requests' }, { title: '已知 token', dataIndex: 'known_total_tokens' },
        { title: '用量不完整', key: 'incomplete', render: (_, row) => row.partial_requests + row.unknown_requests + row.invalid_requests },
      ]} /></details>}
    </Card>

    {!activeAdmin && <Card title="我的当日额度"><QueryState isPending={mine.isPending} error={mine.error} onRetry={() => mine.refetch()} />
      {policy && <Space wrap><Tag>{policy.quota_day}</Tag><span>已占用 {policy.used_requests} 次（含预留及未知）</span><span>每日额度：{policy.unlimited ? '不限次数' : policy.daily_request_limit}</span><span>{policy.unlimited ? '' : `剩余 ${policy.remaining_requests} 次`}</span><Tag>{policy.inherited ? '继承默认额度' : '独立额度'}</Tag></Space>}
      {policy?.warning_reached && <Alert type="warning" title={`当日已知 token 已达到提醒阈值 ${policy.daily_token_warning}`} description="仅提醒，不阻断未知消耗，也不代表金额封顶。" />}
    </Card>}

    <Card title="供应商请求明细"><QueryState isPending={calls.isPending} error={calls.error} onRetry={() => calls.refetch()} />
      {calls.data && <><Table<AiUsageCallInfo> rowKey="id" dataSource={calls.data.list} pagination={false} scroll={{ x: 1000 }} locale={{ emptyText: <Empty description="此范围尚无账本记录" /> }} columns={[
        { title: '时间', dataIndex: 'started_at', render: formatTime, width: 170 },
        ...(activeAdmin ? [{ title: '用户 ID', dataIndex: 'owner_id', width: 90 }] : []),
        { title: '来源 / 调用', key: 'source', render: (_, row) => `${sources[row.source as keyof typeof sources] ?? row.source} / ${row.operation}` },
        { title: '模型', key: 'model', render: (_, row) => `${row.provider} · ${row.model}` },
        { title: '请求状态', key: 'status', render: (_, row) => <Tag>{statuses[row.status]}</Tag> },
        { title: '已知 token', key: 'tokens', render: (_, row) => <span>{usageTokenLabel(row)}<small className={styles.secondary}>{usageStates[row.usage_state]}</small></span> },
        { title: '额度', key: 'quota', render: (_, row) => row.quota_charged ? '已占用' : '已释放' },
      ]} expandable={{ expandedRowRender: row => <Typography.Paragraph className={styles.references}>请求标识：{row.request_id || '未提供'}<br />供应商请求：{row.provider_request_id || '未取得'}<br />尝试标识：{row.attempt_id}<br />{row.error_code ? `安全错误码：${row.error_code}` : ''}</Typography.Paragraph> }} />
        <Pagination className={styles.pagination} current={params.page} pageSize={params.page_size} total={calls.data.total} onChange={page => setParams(current => ({ ...current, page }))} showSizeChanger={false} />
      </>}
    </Card>

    {guard.can('sys:ai:usage:admin') && <Card title="当日策略与预算提醒" extra={canManage && <Button icon={<SettingOutlined />} onClick={() => editPolicy()}>设置用户额度</Button>}>
      <QueryState isPending={policies.isPending} error={policies.error} onRetry={() => policies.refetch()} />
      {policies.data && <Table<AiUsagePolicyInfo> rowKey="owner_id" dataSource={policies.data.list} pagination={false} scroll={{ x: 800 }} columns={[
        { title: '范围', key: 'owner', render: (_, row) => row.owner_id === 0 ? '全局默认 / 全栈提醒' : `后台用户 ${row.owner_id}` },
        { title: '每日请求额度', key: 'limit', render: (_, row) => `${row.unlimited ? '不限次数' : row.daily_request_limit}${row.inherited ? '（继承）' : ''}` },
        { title: '已占用', dataIndex: 'used_requests' },
        { title: '已知 token / 提醒阈值', key: 'warning', render: (_, row) => <span>{row.known_tokens} / {row.daily_token_warning || '关闭'} {row.warning_reached && <Tag color="warning">达到提醒</Tag>}</span> },
        { title: '全局并发', key: 'active', render: (_, row) => row.owner_id === 0 ? `${row.active_known ? row.active_calls : '未确认'} / ${row.global_concurrent_limit}` : '—' },
        { title: '操作', key: 'action', render: (_, row) => canManage && <Button type="link" onClick={() => editPolicy(row)}>设置</Button> },
      ]} />}
      {policies.data && <Pagination className={styles.pagination} current={policyPage} pageSize={20} total={policies.data.total} onChange={setPolicyPage} showSizeChanger={false} />}
      <Typography.Paragraph type="secondary">全局行的请求额度是每位用户的默认值，已占用数为全栈统计；全局没有该次数硬上限。只在共享本栈 MySQL/Redis 的实例间限制并发，token 提醒不是金额封顶。</Typography.Paragraph>
    </Card>}

    <Drawer title="额度策略" open={drawer} onClose={() => { if (!guard.writing) setDrawer(false); }} size={500} destroyOnHidden>
      <Space><InputNumber aria-label="策略用户 ID" value={ownerInput} min={0} precision={0} disabled={guard.writing || loadingPolicy} onChange={value => { setOwnerInput(value ?? 0); setSelected(null); }} /><Button loading={loadingPolicy} disabled={guard.writing} onClick={() => void loadPolicy()}>读取当前策略</Button></Space>
      <Typography.Paragraph type="secondary">0 表示全局默认；其他值为后台用户 ID。读取会替换本地草稿。</Typography.Paragraph>
      {notice && <Alert className={styles.notice} type={save.isError ? 'warning' : 'info'} title={notice} />}
      {selected && <Form form={policyForm} layout="vertical" disabled={guard.writing || !canManage} onFinish={values => { if (guard.beginWrite('sys:ai:usage:manage')) save.mutate({ policy: selected, values }); }}>
        <Typography.Paragraph>范围：{selected.owner_id === 0 ? '全局默认' : `用户 ${selected.owner_id}`} · 当前版本 {selected.version}</Typography.Paragraph>
        {selected.owner_id > 0 && <Form.Item name="inherit_default" label="继承全局每日请求额度" valuePropName="checked"><Switch /></Form.Item>}
        <Form.Item name="daily_request_limit" label="每日供应商请求次数（0 不限）" rules={[{ required: !inherit }]}><InputNumber min={0} max={1_000_000} precision={0} disabled={inherit || guard.writing} /></Form.Item>
        <Form.Item name="daily_token_warning" label="当日已知 token 提醒阈值（0 关闭）" rules={[{ required: true }]}><InputNumber min={0} max={Number.MAX_SAFE_INTEGER} precision={0} /></Form.Item>
        {selected.owner_id === 0 && <Form.Item name="global_concurrent_limit" label="全局活跃请求数" rules={[{ required: true }]}><InputNumber min={1} max={64} precision={0} /></Form.Item>}
        <Button type="primary" htmlType="submit" loading={guard.writing}>保存策略</Button>
      </Form>}
    </Drawer>
  </div>;
}
