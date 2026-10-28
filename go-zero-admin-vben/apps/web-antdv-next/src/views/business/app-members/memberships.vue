<script setup lang="ts">
import type { TableQuery } from '../system/shared';

import type {
  AppGrant,
  AppPlan,
  GrantAppMembership,
} from '#/api/business/app-members';

import { ref } from 'vue';

import { Page } from '@vben/common-ui';

import { Alert, Button, message, Modal, Tag } from 'antdv-next';

import { formatSessionTime } from '#/adapter/business/device-session';
import { StaleSessionResponseError } from '#/adapter/business/session';
import { useVbenForm, z } from '#/adapter/form';
import { useVbenVxeGrid } from '#/adapter/vxe-table';
import {
  getAppGrants,
  getAppPlans,
  getAppUsers,
  grantAppMembership,
  revokeAppGrant,
} from '#/api/business/app-members';

import { usePermission } from '../system/shared';
import { grantStatuses, useRequestScope } from './shared';

type GrantValues = Omit<GrantAppMembership, 'requestId'>;
const can = usePermission('app-memberships');
const open = ref(false);
const saving = ref(false);
const loading = ref(false);
const error = ref('');
const plans = ref<AppPlan[]>([]);
const userOptions = ref<Array<{ label: string; value: number }>>([]);
const usersLoading = ref(false);
const retryPayload = ref<GrantAppMembership>();
const revokeOpen = ref(false);
const revoking = ref(false);
const revokeError = ref('');
const revokeTarget = ref<AppGrant>();
let formVersion = 0;
let searchVersion = 0;
let revokeVersion = 0;
const scope = useRequestScope(() => {
  formVersion++;
  searchVersion++;
  revokeVersion++;
  open.value = false;
  revokeOpen.value = false;
  saving.value = false;
  revoking.value = false;
  loading.value = false;
  usersLoading.value = false;
  plans.value = [];
  userOptions.value = [];
  retryPayload.value = undefined;
  revokeTarget.value = undefined;
  error.value = '';
  revokeError.value = '';
});
let formScope = scope.capture();
let revokeScope = scope.capture();
const [Form, form] = useVbenForm<GrantValues>({
  showDefaultActions: false,
  wrapperClass: 'grid-cols-1',
  schema: [
    {
      fieldName: 'userId',
      label: 'App 用户',
      component: 'Select',
      componentProps: () => ({
        options: userOptions.value,
        showSearch: true,
        filterOption: false,
        onSearch: searchUsers,
        loading: usersLoading.value,
        disabled: loading.value || saving.value || !!retryPayload.value,
        placeholder: '输入邮箱、手机号或昵称搜索',
      }),
      rules: z.number().int().positive(),
      help: '仅选择已审核开通的账号；搜索最多显示 20 条，请输入完整账号缩小范围。',
    },
    {
      fieldName: 'planId',
      label: '会员方案',
      component: 'Select',
      componentProps: () => ({
        options: plans.value.map((plan) => ({
          label: `${plan.name}（v${plan.version}）`,
          value: plan.id,
        })),
        disabled: loading.value || saving.value || !!retryPayload.value,
      }),
      rules: z.number().int().positive(),
    },
    {
      fieldName: 'days',
      label: '天数',
      component: 'InputNumber',
      defaultValue: 30,
      componentProps: () => ({
        min: 1,
        max: 3650,
        precision: 0,
        disabled: saving.value || !!retryPayload.value,
      }),
      rules: z.number().int().min(1).max(3650),
    },
    {
      fieldName: 'reason',
      label: '开通 / 续期原因',
      component: 'Textarea',
      componentProps: () => ({
        rows: 3,
        maxLength: 500,
        disabled: saving.value || !!retryPayload.value,
      }),
      rules: z.string().trim().min(1, '请填写原因').max(500),
    },
  ],
});
const [RevokeForm, revokeForm] = useVbenForm<{ reason: string }>({
  showDefaultActions: false,
  wrapperClass: 'grid-cols-1',
  schema: [
    {
      fieldName: 'reason',
      label: '撤销原因',
      component: 'Textarea',
      componentProps: () => ({
        rows: 3,
        maxLength: 500,
        disabled: revoking.value,
      }),
      rules: z.string().trim().min(1, '请填写原因').max(500),
    },
  ],
});
const [Grid, grid] = useVbenVxeGrid<AppGrant>({
  formOptions: {
    submitOnChange: false,
    schema: [
      {
        fieldName: 'userId',
        label: 'App 用户 ID',
        component: 'InputNumber',
        componentProps: { min: 1, precision: 0, placeholder: '留空查看全部' },
      },
    ],
  },
  gridOptions: {
    columns: [
      { field: 'id', title: '授予 ID', width: 95 },
      { field: 'userId', title: '用户 ID', width: 95 },
      { field: 'account', title: '账号', minWidth: 210 },
      { field: 'planCode', title: '方案', width: 95 },
      { field: 'planVersion', title: '方案版本', width: 95 },
      {
        field: 'status',
        title: '状态',
        width: 100,
        slots: { default: 'status' },
      },
      {
        field: 'startsAt',
        title: '生效时间',
        width: 185,
        formatter: ({ cellValue }: { cellValue: string }) =>
          formatSessionTime(cellValue),
      },
      {
        field: 'endsAt',
        title: '到期时间',
        width: 185,
        formatter: ({ cellValue }: { cellValue: string }) =>
          formatSessionTime(cellValue),
      },
      {
        field: 'entitlements',
        title: '授予权益',
        minWidth: 160,
        formatter: ({ cellValue }: { cellValue: string[] }) =>
          cellValue?.join('、') || '—',
      },
      { field: 'reason', title: '授予原因', minWidth: 180, showOverflow: true },
      {
        field: 'revokedAt',
        title: '撤销时间',
        width: 185,
        formatter: ({ cellValue }: { cellValue: string }) =>
          formatSessionTime(cellValue),
      },
      {
        field: 'revokeReason',
        title: '撤销原因',
        minWidth: 180,
        showOverflow: true,
      },
      {
        title: '操作',
        width: 125,
        fixed: 'right',
        slots: { default: 'actions' },
      },
    ],
    pagerConfig: { pageSize: 10 },
    rowConfig: { keyField: 'id' },
    toolbarConfig: { refresh: true },
    proxyConfig: {
      ajax: {
        query: async ({ page }: TableQuery, values: { userId?: number }) => {
          const request = scope.capture();
          const result = await getAppGrants(
            {
              pageNo: page.currentPage,
              pageSize: page.pageSize,
              ...(values.userId ? { userId: values.userId } : {}),
            },
            request.token,
          );
          if (!scope.current(request)) throw new StaleSessionResponseError();
          return { items: result.list ?? [], total: result.total };
        },
      },
    },
  },
});
async function searchUsers(keyword = '') {
  if (!open.value || !scope.current(formScope) || retryPayload.value) return;
  const request = formScope;
  const version = ++searchVersion;
  usersLoading.value = true;
  try {
    const result = await getAppUsers(
      { pageNo: 1, pageSize: 20, keyword, status: 'active' },
      request.token,
    );
    if (version !== searchVersion || !scope.current(request) || !open.value)
      return;
    userOptions.value = (result.list ?? []).map((user) => ({
      value: user.id,
      label: `${user.account} · ${user.nickname}（ID ${user.id}）`,
    }));
  } catch {
    if (version === searchVersion && scope.current(request))
      error.value = '用户搜索失败，请重新输入搜索条件。';
  } finally {
    if (version === searchVersion && scope.current(request))
      usersLoading.value = false;
  }
}
async function showGrant() {
  if (saving.value || loading.value || !can('grant')) return;
  const version = ++formVersion;
  const request = scope.capture();
  formScope = request;
  open.value = true;
  loading.value = true;
  error.value = '';
  plans.value = [];
  userOptions.value = [];
  retryPayload.value = undefined;
  try {
    await form.reset();
    if (!scope.current(request) || version !== formVersion) return;
    await form.setValues({ days: 30, reason: '' });
    const [result] = await Promise.all([
      getAppPlans(request.token),
      searchUsers(),
    ]);
    if (!scope.current(request) || version !== formVersion) return;
    plans.value = result.list ?? [];
    if (plans.value.length === 0)
      error.value = '暂无可用会员方案，请联系管理员。';
    else if (plans.value.length === 1 && plans.value[0])
      await form.setValues({ planId: plans.value[0].id });
  } catch {
    if (scope.current(request) && version === formVersion)
      error.value = '方案加载失败，请关闭后重试。';
  } finally {
    if (scope.current(request) && version === formVersion)
      loading.value = false;
  }
}
async function saveGrant() {
  const request = formScope;
  const version = formVersion;
  if (
    !open.value ||
    saving.value ||
    loading.value ||
    !can('grant') ||
    !scope.current(request)
  )
    return;
  saving.value = true;
  error.value = '';
  try {
    if (!retryPayload.value) {
      const validation = await form.validate();
      if (!validation.valid) return;
      const values = await form.getValues();
      if (!scope.current(request) || version !== formVersion) return;
      if (
        !Number.isSafeInteger(values.userId) ||
        values.userId < 1 ||
        !plans.value.some((plan) => plan.id === values.planId) ||
        !Number.isInteger(values.days) ||
        values.days < 1 ||
        values.days > 3650 ||
        !values.reason?.trim()
      ) {
        error.value = '请选择用户和方案，填写 1–3650 的整数天数及原因。';
        return;
      }
      retryPayload.value = {
        userId: values.userId,
        planId: values.planId,
        days: values.days,
        reason: values.reason.trim(),
        requestId: Array.from(
          crypto.getRandomValues(new Uint8Array(16)),
          (byte) => byte.toString(16).padStart(2, '0'),
        ).join(''),
      };
    }
    await grantAppMembership({ ...retryPayload.value }, request.token);
    if (!scope.current(request) || version !== formVersion) return;
    message.success('会员已开通 / 续期');
    open.value = false;
    await grid.query();
  } catch {
    if (scope.current(request) && version === formVersion)
      error.value =
        '操作失败，请重试原请求；重复提交不会重复延长。关闭前请先核实授予记录。';
  } finally {
    if (scope.current(request) && version === formVersion) saving.value = false;
  }
}
function closeGrant() {
  if (!saving.value) {
    open.value = false;
    formVersion++;
    searchVersion++;
    loading.value = false;
    usersLoading.value = false;
  }
}
async function showRevoke(row: AppGrant) {
  if (revoking.value || !can('revoke') || row.status === 'revoked') return;
  const version = ++revokeVersion;
  const request = scope.capture();
  revokeScope = request;
  revokeTarget.value = row;
  revokeOpen.value = true;
  revokeError.value = '';
  await revokeForm.reset();
  if (!scope.current(request) || version !== revokeVersion) return;
  await revokeForm.setValues({ reason: '' });
}
async function saveRevoke() {
  const request = revokeScope;
  const version = revokeVersion;
  const row = revokeTarget.value;
  if (
    !row ||
    !revokeOpen.value ||
    revoking.value ||
    !can('revoke') ||
    !scope.current(request)
  )
    return;
  revoking.value = true;
  revokeError.value = '';
  try {
    const validation = await revokeForm.validate();
    if (!validation.valid) return;
    const values = await revokeForm.getValues();
    if (!scope.current(request) || version !== revokeVersion) return;
    const reason = values.reason?.trim();
    if (!reason) {
      revokeError.value = '请填写撤销原因';
      return;
    }
    await revokeAppGrant({ id: row.id, reason }, request.token);
    if (!scope.current(request) || version !== revokeVersion) return;
    message.success('此笔会员授予已撤销');
    revokeOpen.value = false;
    await grid.query();
  } catch {
    if (scope.current(request) && version === revokeVersion)
      revokeError.value = '撤销失败，已保留原因，请重试。';
  } finally {
    if (scope.current(request) && version === revokeVersion)
      revoking.value = false;
  }
}
function closeRevoke() {
  if (!revoking.value) {
    revokeOpen.value = false;
    revokeVersion++;
  }
}
</script>

<template>
  <Page
    title="App VIP 会员"
    description="人工开通与续期，每次授予独立保留记录。续期从现有授予的最晚到期时间起算；到期不影响普通账号基础功能。"
  >
    <Grid>
      <template #toolbar-tools>
        <Button
          v-if="can('grant')"
          type="primary"
          :disabled="saving || loading"
          @click="showGrant"
        >
          开通 / 续期
        </Button>
      </template>
      <template #status="{ row }">
        <Tag
          :color="
            row.status === 'active'
              ? 'green'
              : row.status === 'scheduled'
                ? 'blue'
                : 'default'
          "
        >
          {{ grantStatuses[row.status] ?? row.status }}
        </Tag>
      </template>
      <template #actions="{ row }">
        <Button
          v-if="can('revoke') && row.status !== 'revoked'"
          type="link"
          size="small"
          danger
          :disabled="revoking"
          @click="showRevoke(row)"
        >
          撤销授予
        </Button>
      </template>
    </Grid>
    <Modal
      v-model:open="open"
      title="开通 / 续期 VIP"
      :confirm-loading="saving"
      :mask-closable="false"
      :closable="!saving"
      :keyboard="!saving"
      :cancel-button-props="{ disabled: saving }"
      :ok-button-props="{ disabled: loading || plans.length === 0 }"
      :force-render="true"
      @ok="saveGrant"
      @cancel="closeGrant"
    >
      <Alert v-if="error" class="mb-3" type="error" :title="error" show-icon />
      <p v-if="loading" class="mb-3">正在加载用户和会员方案…</p>
      <p v-for="plan in plans" :key="plan.id" class="mb-3">
        {{ plan.name }} · v{{ plan.version }} · 权益：{{
          plan.entitlements.join('、')
        }}
      </p>
      <Form />
    </Modal>
    <Modal
      v-model:open="revokeOpen"
      :title="`撤销授予 #${revokeTarget?.id ?? ''}`"
      :confirm-loading="revoking"
      :mask-closable="false"
      :closable="!revoking"
      :keyboard="!revoking"
      :cancel-button-props="{ disabled: revoking }"
      :force-render="true"
      @ok="saveRevoke"
      @cancel="closeRevoke"
    >
      <p class="mb-3">
        仅撤销
        {{ revokeTarget?.account }} 的此笔授予，其他授予及其生效时间保持不变。
      </p>
      <Alert
        v-if="revokeError"
        class="mb-3"
        type="error"
        :title="revokeError"
        show-icon
      />
      <RevokeForm />
    </Modal>
  </Page>
</template>
