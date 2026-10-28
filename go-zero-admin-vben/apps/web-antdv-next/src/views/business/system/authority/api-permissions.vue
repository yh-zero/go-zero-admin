<script setup lang="ts">
import type { ApiResource, Policy } from '#/api/business/system/types';

import { computed, ref } from 'vue';

import {
  Alert,
  Button,
  Drawer,
  message,
  Select,
  Space,
  Spin,
  TextArea,
} from 'antdv-next';

import { savePolicies, savePolicyApiIds } from '#/api/business/system/casbin';
import {
  getPermissionEdit,
  type PermissionEdit,
} from '#/api/business/system/permissions';

import { confirmAction, refreshRoleAccess } from '../shared';
import { useEditSession } from '../use-edit-session';
import {
  parsePolicies,
  permissionChanges,
  policyKey,
  splitPolicies,
} from './authorization';
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
  apis.value = server.apis;
  originalPolicies.value = server.policies.map(policyKey);
  originalIds.value = splitPolicies(server.policies, server.apis).ids;
  conflict.value = undefined;
}
async function showConflict(ticket: ReturnType<typeof session.capture>) {
  if (!session.valid(ticket)) return;
  try {
    const server = await getPermissionEdit(
      ticket.objectId,
      'api',
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
const mode = ref('ids');
const apis = ref<ApiResource[]>([]);
const selected = ref<number[]>([]);
const unavailableIds = computed(() => {
  const available = new Set(apis.value.map((api) => api.ID));
  return selected.value.filter((id) => !available.has(id));
});
function removeUnavailable(id: number) {
  if (!saving.value && !loading.value)
    selected.value = selected.value.filter((value) => value !== id);
}
const originalIds = ref<number[]>([]);
const originalPolicies = ref<string[]>([]);
const unknown = ref<Policy[]>([]);
const text = ref('');
const options = computed(() =>
  apis.value.map((api) => ({
    group: api.apiGroup || '未分组',
    label: `${api.method} ${api.path} · ${api.description}`,
    value: api.ID,
  })),
);
const pathChanges = computed(() => {
  try {
    return {
      ...permissionChanges(
        originalPolicies.value,
        parsePolicies(text.value).map(policyKey),
      ),
      error: '',
    };
  } catch (error) {
    return {
      added: [],
      removed: [],
      total: 0,
      error: (error as Error).message,
    };
  }
});
async function show(id: number, name: string) {
  const ticket = session.begin(id);
  const version = ++loadVersion;
  saving.value = false;
  conflict.value = undefined;
  revision.value = '';
  roleId.value = id;
  title.value = `${name} · 接口授权`;
  open.value = true;
  loading.value = true;
  loaded.value = false;
  apis.value = [];
  selected.value = [];
  originalIds.value = [];
  originalPolicies.value = [];
  unknown.value = [];
  text.value = '';
  mode.value = 'ids';
  try {
    const snapshot = await getPermissionEdit(id, 'api', ticket.token);
    if (version !== loadVersion || !session.valid(ticket)) return;
    revision.value = snapshot.revision;
    apis.value = snapshot.apis ?? [];
    const policies = snapshot.policies ?? [];
    const split = splitPolicies(policies, apis.value);
    selected.value = split.ids;
    originalIds.value = [...split.ids];
    originalPolicies.value = policies.map(policyKey);
    unknown.value = split.unknown;
    text.value = policies.map(policyKey).join('\n');
    if (unknown.value.length) mode.value = 'paths';
    loaded.value = true;
  } finally {
    if (version === loadVersion && session.valid(ticket)) loading.value = false;
  }
}
function changeMode(value: unknown) {
  if (mode.value === 'ids' && unavailableIds.value.length) return;
  if (value === mode.value) return;
  if (value === 'paths') {
    const ids = new Set(selected.value);
    text.value = [
      ...apis.value.filter((api) => ids.has(api.ID)),
      ...unknown.value,
    ]
      .map(policyKey)
      .join('\n');
  } else {
    try {
      const split = splitPolicies(parsePolicies(text.value), apis.value);
      selected.value = split.ids;
      unknown.value = split.unknown;
    } catch (error) {
      message.warning((error as Error).message);
      return;
    }
  }
  mode.value = String(value);
}
async function persist(
  ticket: ReturnType<typeof session.capture>,
  ids: number[],
  policies: Policy[],
  saveMode: string,
  expectedRevision: string,
) {
  if (!session.valid(ticket)) return;
  if (
    !loaded.value ||
    loading.value ||
    saving.value ||
    conflict.value ||
    (saveMode === 'ids' && unavailableIds.value.length) ||
    !revision.value
  )
    return;
  saving.value = true;
  try {
    if (saveMode === 'ids')
      await savePolicyApiIds(
        ticket.objectId,
        ids,
        expectedRevision,
        ticket.token,
      );
    else
      await savePolicies(
        ticket.objectId,
        policies,
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
  if (
    !loaded.value ||
    loading.value ||
    saving.value ||
    conflict.value ||
    (mode.value === 'ids' && unavailableIds.value.length) ||
    !revision.value
  )
    return;
  if (mode.value === 'ids' && unknown.value.length) {
    message.warning('存在未登记规则，请使用路径模式保存，避免丢失权限');
    return;
  }
  let policies: Policy[] | undefined;
  if (mode.value === 'paths') {
    try {
      policies = parsePolicies(text.value);
    } catch (error) {
      message.warning((error as Error).message);
      return;
    }
  }
  const ticket = session.capture();
  const ids = [...selected.value];
  const expectedRevision = revision.value;
  const saveMode = mode.value;
  const action = () =>
    persist(ticket, ids, policies ?? [], saveMode, expectedRevision);
  const changes =
    mode.value === 'ids'
      ? permissionChanges(originalIds.value, selected.value)
      : permissionChanges(
          originalPolicies.value,
          (policies ?? []).map(policyKey),
        );
  if (changes.total === 0 || changes.removed.length)
    confirmAction(
      changes.total === 0 ? '清空此角色的全部接口权限？' : '确认修改接口授权？',
      action,
      `新增 ${changes.added.length} 项，撤销 ${changes.removed.length} 项，保存后共 ${changes.total} 项。被撤销的接口将无法访问，可能影响菜单加载或业务功能；菜单与按钮授权不会随之修改。`,
    );
  else void action();
}
defineExpose({ show });
</script>
<template>
  <Drawer
    v-model:open="open"
    :title="title"
    :size="760"
    :mask-closable="!saving"
    :closable="!saving"
  >
    <PermissionConflict
      :server="conflict"
      :local="{ apiIds: selected, policies: text }"
      @rebase="rebase"
    />
    <Spin :spinning="loading">
      <UnavailableSelections
        v-if="mode === 'ids'"
        :ids="unavailableIds"
        label="API"
        :disabled="saving || loading"
        @remove="removeUnavailable"
      />
      <Select
        :value="mode"
        :disabled="
          saving || loading || (mode === 'ids' && unavailableIds.length > 0)
        "
        :options="[
          { label: '按 API 清单选择', value: 'ids' },
          { label: '按路径和方法（高级）', value: 'paths' },
        ]"
        class="mb-4 w-full"
        @change="changeMode"
      />
      <Alert
        v-if="unknown.length"
        type="warning"
        show-icon
        class="mb-4"
        :message="`有 ${unknown.length} 条未登记规则，请在路径模式中维护；删除对应行表示撤销。切换到 API 清单时仍有未登记规则则禁止保存，避免丢失权限。`"
      />
      <template v-if="mode === 'ids'">
        <p class="mb-3 text-sm">
          一次保存仅替换接口权限，不修改菜单与按钮授权。
        </p>
        <PermissionOptions
          :key="`${roleId}-${loadVersion}`"
          v-model="selected"
          :options="options"
          :original="originalIds"
          :disabled="saving || loading"
        />
      </template>
      <template v-else>
        <p class="mb-3 text-sm">
          每行一条，例如 GET
          /v1/sys/menu/getMenu。已有未登记规则会保留，删除行代表明确撤销。
        </p>
        <p v-if="!pathChanges.error" class="mb-3 text-sm">
          已选 {{ pathChanges.total }} 条 · 新增
          {{ pathChanges.added.length }} 条 · 撤销
          {{ pathChanges.removed.length }} 条
        </p>
        <Alert
          v-else
          type="warning"
          class="mb-3"
          :message="pathChanges.error"
        />
        <details
          v-if="pathChanges.added.length || pathChanges.removed.length"
          class="mb-3 rounded border p-3 text-sm"
        >
          <summary class="cursor-pointer">查看本次接口权限变更</summary>
          <ul class="mt-2 space-y-1 break-all">
            <li v-for="rule in pathChanges.added" :key="`add-${rule}`">
              新增：{{ rule }}
            </li>
            <li v-for="rule in pathChanges.removed" :key="`remove-${rule}`">
              撤销：{{ rule }}
            </li>
          </ul>
        </details>
        <TextArea
          v-model:value="text"
          :rows="22"
          :disabled="saving || loading"
          class="font-mono"
        />
      </template>
    </Spin>
    <template #footer>
      <Space>
        <Button :disabled="saving" @click="open = false">取消</Button>
        <Button
          type="primary"
          :loading="saving"
          :disabled="
            !loaded ||
            loading ||
            (mode === 'ids' &&
              (unknown.length > 0 || unavailableIds.length > 0))
          "
          @click="save"
        >
          {{ mode === 'ids' ? '保存所选 API 权限' : '保存路径权限' }}
        </Button>
      </Space>
    </template>
  </Drawer>
</template>
