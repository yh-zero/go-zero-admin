<script setup lang="ts">
import type { PermissionOption } from './authorization';

import { computed, ref } from 'vue';

import { Alert, Button, Drawer, Space, Spin } from 'antdv-next';

import { missingPermissionApis } from '#/adapter/business/permission-resources';
import { saveAuthorityButtons } from '#/api/business/system/menu';
import {
  getPermissionEdit,
  type PermissionEdit,
} from '#/api/business/system/permissions';

import { confirmAction, flattenTree, refreshRoleAccess } from '../shared';
import { useEditSession } from '../use-edit-session';
import { permissionChanges } from './authorization';
import PermissionConflict from './permission-conflict.vue';
import PermissionOptions from './permission-options.vue';
import UnavailableSelections from './unavailable-selections.vue';
const open = ref(false);
const loading = ref(false);
const loaded = ref(false);
let loadVersion = 0;
const saving = ref(false);
const session = useEditSession(open, loaded, saving);
const revision = ref('');
const conflict = ref<PermissionEdit>();
function rebase() {
  if (!conflict.value) return;
  const server = conflict.value;
  session.begin(server.authorityId);
  revision.value = server.revision;
  original.value = [...server.menuBtnIds];
  snapshotResources.value = server;
  options.value = flattenTree(server.menus)
    .filter((menu) => server.menuIds.includes(menu.ID))
    .flatMap((menu) =>
      (menu.menuBtn ?? [])
        .filter((button) => button.ID)
        .map((button) => ({
          group: menu.meta.title,
          label: button.desc || button.permissionKey || button.name,
          value: button.ID!,
        })),
    );
  conflict.value = undefined;
}
async function showConflict(ticket: ReturnType<typeof session.capture>) {
  if (!session.valid(ticket)) return;
  try {
    const server = await getPermissionEdit(
      ticket.objectId,
      'button',
      ticket.token,
    );
    if (session.valid(ticket) && server.revision !== revision.value)
      conflict.value = server;
  } catch {
    /* Original error already displayed by client. */
  }
}
const roleId = ref(0);
const title = ref('');
const selected = ref<number[]>([]);
const original = ref<number[]>([]);
const options = ref<PermissionOption[]>([]);
const unavailableIds = computed(() => {
  const available = new Set(options.value.map((option) => option.value));
  return selected.value.filter((id) => !available.has(id));
});
function removeUnavailable(id: number) {
  if (!saving.value && !loading.value)
    selected.value = selected.value.filter((value) => value !== id);
}
const snapshotResources = ref<PermissionEdit>();
const missingApis = computed(() =>
  snapshotResources.value
    ? missingPermissionApis(
        snapshotResources.value.menus ?? [],
        selected.value,
        snapshotResources.value.policies ?? [],
      )
    : [],
);
async function show(id: number, name: string) {
  const ticket = session.begin(id);
  const version = ++loadVersion;
  saving.value = false;
  conflict.value = undefined;
  revision.value = '';
  roleId.value = id;
  title.value = `${name} · 按钮授权`;
  open.value = true;
  loading.value = true;
  loaded.value = false;
  selected.value = [];
  original.value = [];
  options.value = [];
  try {
    const snapshot = await getPermissionEdit(id, 'button', ticket.token);
    if (version !== loadVersion || !session.valid(ticket)) return;
    snapshotResources.value = snapshot;
    revision.value = snapshot.revision;
    selected.value = snapshot.menuBtnIds ?? [];
    original.value = [...selected.value];
    options.value = flattenTree(snapshot.menus ?? [])
      .filter((menu) => snapshot.menuIds.includes(menu.ID))
      .flatMap((menu) =>
        (menu.menuBtn ?? [])
          .filter((button) => button.ID)
          .map((button) => ({
            group: `${menu.meta.title} (${menu.name})`,
            label: `${button.desc || button.name} (${button.permissionKey || `${menu.name}:${button.name}`})`,
            value: button.ID!,
          })),
      );
    loaded.value = true;
  } finally {
    if (version === loadVersion && session.valid(ticket)) loading.value = false;
  }
}
async function persist(
  ticket: ReturnType<typeof session.capture>,
  ids: number[],
  expectedRevision: string,
) {
  if (!session.valid(ticket)) return;
  if (
    !loaded.value ||
    loading.value ||
    saving.value ||
    conflict.value ||
    unavailableIds.value.length ||
    !revision.value
  )
    return;
  saving.value = true;
  try {
    await saveAuthorityButtons(
      ticket.objectId,
      ids,
      expectedRevision,
      ticket.token,
    );
    if (!session.valid(ticket)) return;
    refreshRoleAccess(ticket.objectId);
    open.value = false;
  } catch {
    await showConflict(ticket);
  } finally {
    if (session.valid(ticket)) saving.value = false;
  }
}
function save() {
  const ticket = session.capture();
  const ids = [...selected.value];
  const expectedRevision = revision.value;
  const action = () => persist(ticket, ids, expectedRevision);
  if (
    !loaded.value ||
    loading.value ||
    saving.value ||
    conflict.value ||
    unavailableIds.value.length ||
    !revision.value
  )
    return;
  const changes = permissionChanges(original.value, selected.value);
  if (changes.removed.length)
    confirmAction(
      !selected.value.length ? '清空此角色的按钮权限？' : '确认修改按钮授权？',
      action,
      `新增 ${changes.added.length} 项，撤销 ${changes.removed.length} 项，保存后共 ${changes.total} 项。撤销后该角色无法使用对应按钮，接口权限不会随之修改。`,
    );
  else void action();
}
defineExpose({ show });
</script>
<template>
  <Drawer
    v-model:open="open"
    :title="title"
    :size="600"
    :mask-closable="!saving"
    :closable="!saving"
  >
    <PermissionConflict
      :server="conflict"
      :local="{ menuBtnIds: selected }"
      @rebase="rebase"
    />
    <Spin :spinning="loading">
      <UnavailableSelections
        :ids="unavailableIds"
        label="按钮"
        :disabled="saving || loading"
        @remove="removeUnavailable"
      />
      <p class="mb-4">
        只展示此角色已授权菜单下的按钮。接口访问权限需单独保存。
      </p>
      <Alert
        v-if="missingApis.length"
        type="warning"
        class="mb-4"
        message="已选按钮缺少依赖 API，请在接口授权中核对并单独保存。"
      >
        <template #description>
          <ul>
            <li v-for="(item, index) in missingApis" :key="index">
              {{ item.button }}：{{ item.api }}
            </li>
          </ul>
        </template>
      </Alert>
      <PermissionOptions
        :key="`${roleId}-${loadVersion}`"
        v-model="selected"
        :options="options"
        :original="original"
        :disabled="saving || loading"
      />
      <p v-if="!loading && !options.length">没有可授权按钮，请先分配菜单。</p>
    </Spin>
    <template #footer>
      <Space>
        <Button :disabled="saving" @click="open = false">取消</Button>
        <Button
          type="primary"
          :loading="saving"
          :disabled="
            !loaded || loading || !!conflict || unavailableIds.length > 0
          "
          @click="save"
        >
          保存按钮授权
        </Button>
      </Space>
    </template>
  </Drawer>
</template>
