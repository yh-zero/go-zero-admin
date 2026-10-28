<script setup lang="ts">
import type { DeviceSession } from '#/api/business/device-sessions';
import type { TableQuery } from '#/views/business/system/shared';

import { computed, ref } from 'vue';

import { useAccessStore, useUserStore } from '@vben/stores';

import { Alert, Button, message, Tag } from 'antdv-next';

import {
  formatSessionTime,
  revokeListedDevice,
} from '#/adapter/business/device-session';
import { useVbenVxeGrid } from '#/adapter/vxe-table';
import {
  getDeviceSessions,
  getMyDeviceSessions,
  revokeDeviceSession,
  revokeMyDeviceSession,
} from '#/api/business/device-sessions';
import { useAuthStore } from '#/store';
import { confirmAction, usePermission } from '#/views/business/system/shared';

const props = withDefaults(defineProps<{ admin?: boolean }>(), {
  admin: false,
});
const user = useUserStore();
const auth = useAuthStore();
const access = useAccessStore();
const can = usePermission('sessions');
const revoking = ref<string>();
const allowed = computed(
  () => !props.admin || !!user.userInfo?.roles?.includes('1'),
);
const [Grid, grid] = useVbenVxeGrid<DeviceSession>({
  ...(props.admin
    ? {
        formOptions: {
          schema: [
            {
              fieldName: 'userId',
              label: '用户ID',
              component: 'InputNumber',
              componentProps: {
                min: 1,
                precision: 0,
                placeholder: '留空查看全部用户',
              },
            },
          ],
          submitOnChange: false,
        },
      }
    : {}),
  gridOptions: {
    columns: [
      ...(props.admin
        ? [
            { field: 'userId', title: '用户ID', width: 100 },
            { field: 'username', title: '用户名', minWidth: 130 },
          ]
        : []),
      {
        field: 'userAgent',
        title: '设备 / 浏览器',
        minWidth: 240,
        showOverflow: true,
      },
      { field: 'ip', title: '来源IP', minWidth: 135 },
      {
        field: 'createdAt',
        title: '登录时间',
        width: 185,
        formatter: ({ cellValue }: { cellValue: string }) =>
          formatSessionTime(cellValue),
      },
      {
        field: 'expiresAt',
        title: '到期时间',
        width: 185,
        formatter: ({ cellValue }: { cellValue: string }) =>
          formatSessionTime(cellValue),
      },
      {
        field: 'current',
        title: '当前设备',
        width: 110,
        slots: { default: 'current' },
      },
      {
        title: '操作',
        width: 125,
        fixed: 'right',
        slots: { default: 'actions' },
      },
    ],
    rowConfig: { keyField: 'id' },
    pagerConfig: { pageSize: 10 },
    toolbarConfig: { refresh: true },
    proxyConfig: {
      ajax: {
        query: async (
          { page }: TableQuery,
          values: { userId?: number } = {},
        ) => {
          const params = { pageNo: page.currentPage, pageSize: page.pageSize };
          const result = props.admin
            ? await getDeviceSessions({
                ...params,
                ...(values.userId ? { userId: values.userId } : {}),
              })
            : await getMyDeviceSessions(params);
          return { items: result.list ?? [], total: result.total };
        },
      },
    },
  },
});

function revoke(row: DeviceSession) {
  if (revoking.value) return;
  confirmAction(
    row.current ? '撤销当前设备并退出？' : '撤销此设备的登录？',
    async () => {
      if (revoking.value) return;
      const token = access.accessToken;
      revoking.value = row.id;
      try {
        await revokeListedDevice(
          row,
          props.admin ? revokeDeviceSession : revokeMyDeviceSession,
          async () => {
            if (access.accessToken !== token) return;
            message.success('当前设备已撤销，请重新登录');
            await auth.logout(false, false);
          },
          async () => {
            if (access.accessToken !== token) return;
            message.success('设备会话已撤销');
            await grid.reload();
          },
        );
      } finally {
        revoking.value = undefined;
      }
    },
    row.current
      ? '此操作会立即退出当前页面，其他设备仍保持登录。'
      : '该设备下次请求需要重新登录，其他设备不受影响。',
  );
}
</script>

<template>
  <div>
    <Alert
      v-if="!allowed"
      type="info"
      show-icon
      title="在线会话管理仅供内置管理员使用。"
    />
    <template v-else>
      <p class="text-muted-foreground mb-4">
        这里只显示仍然有效的设备会话。撤销单个设备不会退出其他设备；页面顶部的退出登录会退出全部设备。
      </p>
      <Grid>
        <template #current="{ row }">
          <Tag v-if="row.current" color="green">当前设备</Tag>
          <span v-else>—</span>
        </template>
        <template #actions="{ row }">
          <Button
            v-if="!admin || can('revoke')"
            type="link"
            danger
            size="small"
            :disabled="!!revoking"
            :loading="revoking === row.id"
            @click="revoke(row)"
          >
            {{ row.current ? '撤销并退出' : '撤销会话' }}
          </Button>
          <span v-else class="text-muted-foreground">无撤销权限</span>
        </template>
      </Grid>
    </template>
  </div>
</template>
