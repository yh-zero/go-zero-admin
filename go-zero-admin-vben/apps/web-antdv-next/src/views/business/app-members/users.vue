<script setup lang="ts">
import type { TableQuery } from '../system/shared';

import type {
  AppSession,
  AppUser,
  AppUserStatus,
} from '#/api/business/app-members';

import { ref } from 'vue';

import { Page } from '@vben/common-ui';

import { Alert, Button, message, Modal, Space, Tag } from 'antdv-next';

import { resolveAvatar } from '#/adapter/business/avatar';
import { formatSessionTime } from '#/adapter/business/device-session';
import { StaleSessionResponseError } from '#/adapter/business/session';
import { useVbenForm, z } from '#/adapter/form';
import { useVbenVxeGrid } from '#/adapter/vxe-table';
import {
  getAppSessions,
  getAppUsers,
  revokeAppSession,
  updateAppUserStatus,
} from '#/api/business/app-members';

import { confirmAction, usePermission } from '../system/shared';
import Avatar from './avatar.vue';
import { useRequestScope, userStatuses } from './shared';

const can = usePermission('app-members-users');
const avatar = ref<InstanceType<typeof Avatar>>();
const open = ref(false);
const saving = ref(false);
const error = ref('');
const target = ref<AppUser>();
const nextStatus = ref<AppUserStatus>('active');
const actionTitle = ref('');
const sessionsOpen = ref(false);
const sessionUser = ref<AppUser>();
const sessionError = ref('');
const revoking = ref('');
let formVersion = 0;
let sessionsVersion = 0;
const scope = useRequestScope(() => {
  formVersion++;
  sessionsVersion++;
  open.value = false;
  sessionsOpen.value = false;
  saving.value = false;
  revoking.value = '';
  target.value = undefined;
  sessionUser.value = undefined;
  error.value = '';
  sessionError.value = '';
});
let formScope = scope.capture();
let sessionsScope = scope.capture();
const [Form, form] = useVbenForm<{ reason: string }>({
  showDefaultActions: false,
  wrapperClass: 'grid-cols-1',
  schema: [
    {
      fieldName: 'reason',
      label: '原因',
      component: 'Textarea',
      componentProps: () => ({
        rows: 3,
        maxLength: 500,
        disabled: saving.value,
      }),
      rules: z.string().trim().max(500).optional(),
      help: '拒绝或禁用必须填写原因；审核通过、恢复可选填。',
    },
  ],
});
const [Grid, grid] = useVbenVxeGrid<AppUser>({
  formOptions: {
    submitOnChange: false,
    schema: [
      {
        fieldName: 'keyword',
        label: '搜索',
        component: 'Input',
        componentProps: { placeholder: '邮箱、手机号或昵称', allowClear: true },
      },
      {
        fieldName: 'status',
        label: '状态',
        component: 'Select',
        componentProps: { options: userStatuses, allowClear: true },
      },
    ],
  },
  gridOptions: {
    columns: [
      { field: 'id', title: '用户 ID', width: 95 },
      {
        field: 'avatarUrl',
        title: '头像',
        width: 80,
        slots: { default: 'avatar' },
      },
      { field: 'account', title: '邮箱 / 手机号', minWidth: 210 },
      {
        field: 'identityVerified',
        title: '账号归属验证',
        minWidth: 170,
        slots: { default: 'identity' },
      },
      { field: 'nickname', title: '昵称', minWidth: 120 },
      {
        field: 'registrationIP',
        title: '注册 IP',
        minWidth: 200,
        showOverflow: true,
        formatter: ({ cellValue }: { cellValue: string }) =>
          cellValue || '未记录',
      },
      {
        field: 'registrationRegion',
        title: 'IP 归属地',
        minWidth: 190,
        showOverflow: true,
        formatter: ({ cellValue }: { cellValue: string }) =>
          cellValue || '未知',
      },
      {
        field: 'status',
        title: '状态',
        width: 100,
        slots: { default: 'status' },
      },
      {
        field: 'reviewReason',
        title: '审核 / 状态原因',
        minWidth: 180,
        showOverflow: true,
      },
      {
        field: 'createdAt',
        title: '注册时间',
        width: 180,
        formatter: ({ cellValue }: { cellValue: string }) =>
          formatSessionTime(cellValue),
      },
      {
        title: '操作',
        width: 380,
        fixed: 'right',
        slots: { default: 'actions' },
      },
    ],
    pagerConfig: { pageSize: 10 },
    rowConfig: { keyField: 'id' },
    toolbarConfig: { refresh: true },
    proxyConfig: {
      ajax: {
        query: async (
          { page }: TableQuery,
          values: { keyword?: string; status?: AppUserStatus },
        ) => {
          const request = scope.capture();
          const result = await getAppUsers(
            {
              pageNo: page.currentPage,
              pageSize: page.pageSize,
              keyword: values.keyword,
              status: values.status,
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
const [SessionGrid, sessionGrid] = useVbenVxeGrid<AppSession>({
  gridOptions: {
    columns: [
      {
        field: 'userAgent',
        title: '设备 / 浏览器',
        minWidth: 240,
        showOverflow: true,
      },
      { field: 'ip', title: '来源 IP', minWidth: 130 },
      {
        field: 'createdAt',
        title: '登录时间',
        width: 180,
        formatter: ({ cellValue }: { cellValue: string }) =>
          formatSessionTime(cellValue),
      },
      {
        field: 'expiresAt',
        title: '到期时间',
        width: 180,
        formatter: ({ cellValue }: { cellValue: string }) =>
          formatSessionTime(cellValue),
      },
      {
        title: '操作',
        width: 120,
        fixed: 'right',
        slots: { default: 'actions' },
      },
    ],
    rowConfig: { keyField: 'id' },
    pagerConfig: { enabled: false },
    toolbarConfig: { refresh: true },
    proxyConfig: {
      autoLoad: true,
      ajax: {
        query: async () => {
          const request = sessionsScope;
          const version = sessionsVersion;
          const user = sessionUser.value;
          if (!user || !sessionsOpen.value || !scope.current(request))
            throw new StaleSessionResponseError();
          sessionError.value = '';
          try {
            const result = await getAppSessions(user.id, request.token);
            if (version !== sessionsVersion || !scope.current(request))
              throw new StaleSessionResponseError();
            return {
              items: result.list ?? [],
              total: result.list?.length ?? 0,
            };
          } catch (error) {
            if (version === sessionsVersion && scope.current(request))
              sessionError.value = '会话加载失败，请刷新重试。';
            throw error;
          }
        },
      },
    },
  },
});
function permitted(row: AppUser, status: AppUserStatus) {
  return (
    can(
      row.status === 'pending' || row.status === 'rejected'
        ? 'review'
        : 'status',
    ) &&
    ((status === 'active' && row.status !== 'active') ||
      (status === 'rejected' && row.status === 'pending') ||
      (status === 'disabled' && row.status === 'active'))
  );
}
async function showStatus(row: AppUser, status: AppUserStatus, title: string) {
  if (saving.value || !permitted(row, status)) return;
  const version = ++formVersion;
  const request = scope.capture();
  formScope = request;
  target.value = row;
  nextStatus.value = status;
  actionTitle.value = title;
  error.value = '';
  open.value = true;
  await form.reset();
  if (version !== formVersion || !scope.current(request)) return;
  await form.setValues({ reason: '' });
}
async function saveStatus() {
  const row = target.value;
  const request = formScope;
  const version = formVersion;
  if (
    !row ||
    !open.value ||
    saving.value ||
    !scope.current(request) ||
    !permitted(row, nextStatus.value)
  )
    return;
  saving.value = true;
  error.value = '';
  try {
    const validation = await form.validate();
    if (!validation.valid) return;
    const values = await form.getValues();
    if (!scope.current(request) || version !== formVersion) return;
    const reason = values.reason?.trim() ?? '';
    if (
      (nextStatus.value === 'rejected' || nextStatus.value === 'disabled') &&
      !reason
    ) {
      error.value = '拒绝或禁用必须填写原因';
      return;
    }
    await updateAppUserStatus(
      { userId: row.id, status: nextStatus.value, reason },
      request.token,
    );
    if (!scope.current(request) || version !== formVersion) return;
    message.success('账号状态已更新');
    open.value = false;
    await grid.query();
  } catch {
    if (scope.current(request) && version === formVersion)
      error.value = '操作失败，已保留填写内容，请重试。';
  } finally {
    if (scope.current(request) && version === formVersion) saving.value = false;
  }
}
function closeStatus() {
  if (!saving.value) {
    open.value = false;
    formVersion++;
  }
}
function closeSessions() {
  if (!revoking.value) {
    sessionsOpen.value = false;
    sessionsVersion++;
  }
}
function showSessions(row: AppUser) {
  if (revoking.value) return;
  sessionsVersion++;
  sessionsScope = scope.capture();
  sessionUser.value = row;
  sessionsOpen.value = true;
  sessionError.value = '';
}
function revoke(row: AppSession) {
  if (revoking.value || !can('revoke-session')) return;
  const request = sessionsScope;
  const version = sessionsVersion;
  confirmAction(
    '撤销此 App 设备会话？',
    async () => {
      if (
        !sessionsOpen.value ||
        revoking.value ||
        !can('revoke-session') ||
        !scope.current(request) ||
        version !== sessionsVersion
      )
        return;
      revoking.value = row.id;
      sessionError.value = '';
      try {
        await revokeAppSession(row.id, request.token);
        if (!scope.current(request) || version !== sessionsVersion) return;
        message.success('App 设备会话已撤销');
        await sessionGrid.query();
      } catch {
        if (scope.current(request) && version === sessionsVersion)
          sessionError.value = '撤销失败，请重试。';
      } finally {
        if (scope.current(request) && version === sessionsVersion)
          revoking.value = '';
      }
    },
    '此设备需重新登录，其他设备不受影响。',
  );
}
</script>

<template>
  <Page
    title="App 用户"
    description="注册后待人工审核，审核通过后方可登录。VIP 由会员管理单独授予。"
  >
    <Alert
      class="mb-4"
      type="warning"
      show-icon
      title="邮箱验证仅确认归属，账号仍需人工审核；未验证的邮箱或手机号请人工核实"
    />
    <Grid>
      <template #avatar="{ row }">
        <img
          :src="resolveAvatar(row.avatarUrl)"
          alt="App 用户头像"
          class="h-9 w-9 rounded object-cover"
          referrerpolicy="no-referrer"
        />
      </template>
      <template #identity="{ row }">
        <Tag
          :color="
            row.identityKind === 'email' && row.identityVerified
              ? 'green'
              : 'orange'
          "
        >
          {{
            row.identityKind === 'email'
              ? row.identityVerified
                ? '邮箱已验证'
                : '邮箱未验证'
              : '手机号待人工核实'
          }}
        </Tag>
      </template>
      <template #status="{ row }">
        <Tag
          :color="
            row.status === 'active'
              ? 'green'
              : row.status === 'pending'
                ? 'orange'
                : 'red'
          "
        >
          {{
            userStatuses.find((item) => item.value === row.status)?.label ??
            row.status
          }}
        </Tag>
      </template>
      <template #actions="{ row }">
        <Space>
          <Button
            v-if="can('avatar')"
            type="link"
            size="small"
            @click="avatar?.show(row)"
          >
            设置头像
          </Button>
          <Button
            v-if="can('review') && ['pending', 'rejected'].includes(row.status)"
            type="link"
            size="small"
            :disabled="saving"
            @click="showStatus(row, 'active', '审核通过')"
          >
            审核通过
          </Button>
          <Button
            v-if="can('review') && row.status === 'pending'"
            type="link"
            size="small"
            danger
            :disabled="saving"
            @click="showStatus(row, 'rejected', '审核拒绝')"
          >
            审核拒绝
          </Button>
          <Button
            v-if="can('status') && row.status === 'active'"
            type="link"
            size="small"
            danger
            :disabled="saving"
            @click="showStatus(row, 'disabled', '禁用账号')"
          >
            禁用
          </Button>
          <Button
            v-if="can('status') && row.status === 'disabled'"
            type="link"
            size="small"
            :disabled="saving"
            @click="showStatus(row, 'active', '恢复账号')"
          >
            恢复
          </Button>
          <Button
            type="link"
            size="small"
            :disabled="!!revoking"
            @click="showSessions(row)"
          >
            查看会话
          </Button>
        </Space>
      </template>
    </Grid>
    <Avatar ref="avatar" @saved="grid.query()" />
    <Modal
      v-model:open="open"
      :title="`${actionTitle}：${target?.account ?? ''}`"
      :confirm-loading="saving"
      :mask-closable="false"
      :closable="!saving"
      :keyboard="!saving"
      :cancel-button-props="{ disabled: saving }"
      :force-render="true"
      @ok="saveStatus"
      @cancel="closeStatus"
    >
      <p class="mb-3">
        邮箱验证仅确认归属，账号仍需人工审核；未验证的邮箱或手机号请人工核实
      </p>
      <dl
        aria-label="注册来源"
        class="mb-4 space-y-2 rounded-md border p-3 text-sm"
      >
        <div class="flex gap-3">
          <dt class="text-muted-foreground w-20 shrink-0">注册 IP</dt>
          <dd class="min-w-0 select-all break-all font-mono">
            {{ target?.registrationIP || '未记录' }}
          </dd>
        </div>
        <div class="flex gap-3">
          <dt class="text-muted-foreground w-20 shrink-0">IP 归属地</dt>
          <dd class="min-w-0 select-all break-words">
            {{ target?.registrationRegion || '未知' }}
          </dd>
        </div>
      </dl>
      <p v-if="nextStatus === 'disabled'" class="mb-3">
        禁用将立即撤销该用户全部会话。会员时间继续计算，不会暂停。
      </p>
      <p v-if="target?.status === 'disabled'" class="mb-3">
        恢复后该用户需要重新登录。
      </p>
      <Alert v-if="error" class="mb-3" type="error" :title="error" show-icon />
      <Form />
    </Modal>
    <Modal
      v-model:open="sessionsOpen"
      :title="`${sessionUser?.account ?? ''} 的 App 会话`"
      :width="1000"
      :footer="null"
      :mask-closable="false"
      :closable="!revoking"
      :keyboard="!revoking"
      @cancel="closeSessions"
    >
      <Alert
        v-if="sessionError"
        class="mb-3"
        type="error"
        :title="sessionError"
        show-icon
      />
      <SessionGrid v-if="sessionsOpen" :key="sessionUser?.id">
        <template #actions="{ row }">
          <Button
            v-if="can('revoke-session')"
            type="link"
            size="small"
            danger
            :disabled="!!revoking"
            :loading="revoking === row.id"
            @click="revoke(row)"
          >
            撤销会话
          </Button>
        </template>
      </SessionGrid>
    </Modal>
  </Page>
</template>
