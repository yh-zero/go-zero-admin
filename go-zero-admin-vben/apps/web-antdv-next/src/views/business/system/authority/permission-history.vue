<script setup lang="ts">
import { ref } from 'vue';

import { Alert, Button, Drawer, Space, Spin } from 'antdv-next';

import {
  applyPermissionRollback,
  getPermissionHistory,
  type PermissionChange,
  previewPermissionRollback,
} from '#/api/business/system/permissions';

import { confirmAction, refreshRoleAccess } from '../shared';
import { useEditSession } from '../use-edit-session';
import { historyDifference } from './history-difference';
const open = ref(false);
const loaded = ref(false);
const saving = ref(false);
const loading = ref(false);
const session = useEditSession(open, loaded, saving);
const roleId = ref(0);
const roleName = ref('');
const list = ref<PermissionChange[]>([]);
const page = ref(1);
const total = ref(0);
const pageSize = 20;
const preview = ref<Awaited<ReturnType<typeof previewPermissionRollback>>>();
async function show(id: number, name: string, pageNo = 1) {
  const ticket = session.begin(id);
  saving.value = false;
  roleId.value = id;
  roleName.value = name;
  page.value = pageNo;
  open.value = true;
  loading.value = true;
  loaded.value = false;
  list.value = [];
  preview.value = undefined;
  try {
    const result = await getPermissionHistory(
      id,
      pageNo,
      pageSize,
      ticket.token,
    );
    if (!session.valid(ticket)) return;
    list.value = result.list ?? [];
    total.value = result.total;
    loaded.value = true;
  } finally {
    if (session.valid(ticket)) loading.value = false;
  }
}
async function rollback(change: PermissionChange) {
  if (saving.value || !loaded.value) return;
  const ticket = session.capture();
  saving.value = true;
  try {
    const result = await previewPermissionRollback(change.id, ticket.token);
    if (!session.valid(ticket)) return;
    preview.value = result;
    if (!result.allowed) return;
    confirmAction(
      '确认按预览撤销该次权限变更？',
      async () => {
        if (!session.valid(ticket)) return;
        saving.value = true;
        try {
          await applyPermissionRollback(
            change.id,
            change.afterRevision,
            result.version,
            ticket.token,
          );
          if (!session.valid(ticket)) return;
          refreshRoleAccess(ticket.objectId);
          await show(ticket.objectId, roleName.value, page.value);
        } finally {
          if (session.valid(ticket)) saving.value = false;
        }
      },
      `事件 #${change.id}，${change.beforeRevision} → ${change.afterRevision}。仅当当前版本仍为事件提交版本且明细完整、资源有效时执行逆向差量；预览后变更也会拒绝。`,
    );
  } finally {
    if (session.valid(ticket)) saving.value = false;
  }
}
defineExpose({ show });
</script>
<template>
  <Drawer v-model:open="open" :title="`${roleName} · 权限变更历史`" :size="800">
    <Spin :spinning="loading">
      <Alert
        v-if="preview"
        class="mb-3"
        :type="preview.allowed ? 'info' : 'warning'"
        :message="
          preview.allowed
            ? '回滚预览通过，请在确认窗口核对差异。'
            : `无法自动回滚：${preview.reason}`
        "
      />
      <p class="mb-3">
        仅展示授权集合和范围的受控明细。存在后续权限变更时需人工重新编辑。
      </p>
      <div
        v-for="change in list"
        :key="change.id"
        class="mb-4 rounded border p-3"
      >
        <!-- prettier-ignore -->
        <Space>
          <strong>#{{ change.id }} · {{ change.kind }}</strong>
          <span>
            {{ change.actorName || change.actorId }} · {{ change.createdAt }}
          </span>
        </Space>
        <p>
          版本 {{ change.beforeRevision }} → {{ change.afterRevision }} · 请求
          {{ change.traceId }}
        </p>
        <ul class="mb-3">
          <li v-for="(line, index) in historyDifference(change)" :key="index">
            {{ line }}
          </li>
        </ul>
        <details>
          <summary>比较变更前后授权</summary>
          <div class="grid grid-cols-2 gap-3">
            <pre class="overflow-auto">
变更前：{{ JSON.stringify(change.before, null, 2) }}</pre>
            <pre class="overflow-auto">
变更后：{{ JSON.stringify(change.after, null, 2) }}</pre>
          </div>
        </details>
        <Button class="mt-2" :disabled="saving" @click="rollback(change)">
          预览回滚
        </Button>
      </div>
      <p v-if="loaded && !list.length">暂无授权变更记录。</p>
      <Space v-if="loaded">
        <span>共 {{ total }} 条 · 第 {{ page }} 页</span>
        <Button
          :disabled="saving || page <= 1"
          @click="show(roleId, roleName, page - 1)"
        >
          上一页
        </Button>
        <Button
          :disabled="saving || page * pageSize >= total"
          @click="show(roleId, roleName, page + 1)"
        >
          下一页
        </Button>
      </Space>
    </Spin>
  </Drawer>
</template>
