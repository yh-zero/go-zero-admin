<script setup lang="ts">
import type { Menu } from '#/api/business/system/types';

import { computed, ref } from 'vue';

import { Button, Drawer, Space, Spin, Tree } from 'antdv-next';

import { saveAuthorityMenus } from '#/api/business/system/authority';

import { confirmAction, flattenTree, refreshRoleAccess } from '../shared';
import { includeMenuParents, permissionChanges } from './authorization';
const emit = defineEmits<{ saved: [] }>();
import {
  getPermissionEdit,
  type PermissionEdit,
} from '#/api/business/system/permissions';

import { useEditSession } from '../use-edit-session';
import PermissionConflict from './permission-conflict.vue';
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
  original.value = [...server.menuIds];
  menus.value = server.menus;
  tree.value = toTree(server.menus);
  conflict.value = undefined;
}
async function showConflict(ticket: ReturnType<typeof session.capture>) {
  if (!session.valid(ticket)) return;
  try {
    const server = await getPermissionEdit(
      ticket.objectId,
      'menu',
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
const menus = ref<Menu[]>([]);
const unavailableIds = computed(() => {
  const available = new Set(flattenTree(menus.value).map((menu) => menu.ID));
  return checked.value.filter((id) => !available.has(id));
});
function removeUnavailable(id: number) {
  if (!saving.value && !loading.value)
    checked.value = checked.value.filter((value) => value !== id);
}
const checked = ref<number[]>([]);
const original = ref<number[]>([]);
const changes = computed(() =>
  permissionChanges(
    original.value,
    includeMenuParents(checked.value, menus.value),
  ),
);
const labels = computed(
  () =>
    new Map(flattenTree(menus.value).map((menu) => [menu.ID, menu.meta.title])),
);
const tree = ref<Array<{ key: number; title: string; children?: unknown[] }>>(
  [],
);
function toTree(
  nodes: Menu[],
): Array<{ key: number; title: string; children: ReturnType<typeof toTree> }> {
  return nodes.map((node) => ({
    key: node.ID,
    title: node.meta.title,
    children: toTree(node.children ?? []),
  }));
}
async function show(id: number, name: string) {
  const ticket = session.begin(id);
  const version = ++loadVersion;
  saving.value = false;
  conflict.value = undefined;
  revision.value = '';
  roleId.value = id;
  title.value = `${name} · 菜单授权`;
  open.value = true;
  loading.value = true;
  loaded.value = false;
  menus.value = [];
  tree.value = [];
  checked.value = [];
  original.value = [];
  try {
    const snapshot = await getPermissionEdit(id, 'menu', ticket.token);
    if (version !== loadVersion || !session.valid(ticket)) return;
    revision.value = snapshot.revision;
    menus.value = snapshot.menus ?? [];
    tree.value = toTree(menus.value);
    checked.value = snapshot.menuIds ?? [];
    original.value = [...checked.value];
    loaded.value = true;
  } finally {
    if (version === loadVersion && session.valid(ticket)) loading.value = false;
  }
}
function updateChecked(
  value: Array<number | string> | { checked: Array<number | string> },
) {
  checked.value = [
    ...new Set([
      ...unavailableIds.value,
      ...(Array.isArray(value) ? value : value.checked).map(Number),
    ]),
  ];
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
    await saveAuthorityMenus(
      ticket.objectId,
      ids,
      expectedRevision,
      ticket.token,
    );
    if (!session.valid(ticket)) return;
    emit('saved');
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
  const ids = includeMenuParents([...checked.value], menus.value);
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
  if (changes.value.total === 0 || changes.value.removed.length)
    confirmAction(
      changes.value.total === 0
        ? '清空此角色的全部业务菜单？'
        : '确认修改菜单授权？',
      action,
      `新增 ${changes.value.added.length} 项，撤销 ${changes.value.removed.length} 项，保存后共 ${changes.value.total} 项（含必要父菜单）。撤销菜单会同时清理对应按钮授权；接口权限不会随之修改。`,
    );
  else void action();
}
defineExpose({ show });
</script>
<template>
  <Drawer
    v-model:open="open"
    :title="title"
    :size="520"
    :mask-closable="!saving"
    :closable="!saving"
  >
    <PermissionConflict
      :server="conflict"
      :local="{ menuIds: checked }"
      @rebase="rebase"
    />
    <Spin :spinning="loading">
      <UnavailableSelections
        :ids="unavailableIds"
        label="菜单"
        :disabled="saving || loading"
        @remove="removeUnavailable"
      />
      <p class="mb-4 text-sm">
        保存会包含所选菜单的必要父菜单。取消父菜单时，请同时取消不需要的子菜单。
      </p>
      <p class="mb-3 text-sm">
        已选 {{ changes.total }} 项 · 新增 {{ changes.added.length }} 项 · 撤销
        {{ changes.removed.length }} 项
      </p>
      <details
        v-if="changes.added.length || changes.removed.length"
        class="mb-3 rounded border p-3 text-sm"
      >
        <summary class="cursor-pointer">查看本次菜单授权变更</summary>
        <ul class="mt-2 space-y-1">
          <li v-for="id in changes.added" :key="`add-${id}`">
            新增：{{ labels.get(id) ?? id }}
          </li>
          <li v-for="id in changes.removed" :key="`remove-${id}`">
            撤销：{{ labels.get(id) ?? id }}
          </li>
        </ul>
      </details>
      <Tree
        :tree-data="tree"
        checkable
        check-strictly
        :checked-keys="checked"
        :disabled="saving || loading"
        default-expand-all
        @check="updateChecked"
      />
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
          保存菜单授权
        </Button>
      </Space>
    </template>
  </Drawer>
</template>
