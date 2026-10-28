<script setup lang="ts">
import type { Membership } from '#/api/business/system/organization';

import { ref } from 'vue';

import { useAccessStore, useUserStore } from '@vben/stores';

import { Alert, Button, message, Modal, Spin } from 'antdv-next';

import { useVbenForm, z } from '#/adapter/form';
import {
  getDepartments,
  getMembership,
  getPositions,
  updateMembership,
} from '#/api/business/system/organization';
import { useAuthStore } from '#/store';

import { confirmAction } from '../shared';
import {
  membershipDepartmentOptions,
  membershipPositionOptions,
  sameMembership,
} from './helpers';

const emit = defineEmits<{ saved: [] }>();
const auth = useAuthStore();
const accessStore = useAccessStore();
const userStore = useUserStore();
const open = ref(false);
const loading = ref(false);
const loaded = ref(false);
const saving = ref(false);
const validating = ref(false);
const userId = ref(0);
const userName = ref('');
const original = ref<Membership>();
let loadVersion = 0;
const [Form, form] = useVbenForm<Membership>({
  showDefaultActions: false,
  wrapperClass: 'grid-cols-1',
  schema: [
    {
      fieldName: 'departmentId',
      label: '部门',
      component: 'Select',
      componentProps: {
        options: [],
        showSearch: true,
        optionFilterProp: 'label',
      },
      rules: z.number().int().min(0),
      defaultValue: 0,
    },
    {
      fieldName: 'positionIds',
      label: '岗位',
      component: 'Select',
      componentProps: {
        options: [],
        mode: 'multiple',
        allowClear: true,
        optionFilterProp: 'label',
      },
      rules: z.array(z.number().int().positive()),
      defaultValue: [],
    },
  ],
});
async function show(id: number, name: string) {
  if (saving.value) return;
  const version = ++loadVersion;
  userId.value = id;
  userName.value = name;
  open.value = true;
  loading.value = true;
  loaded.value = false;
  original.value = undefined;
  try {
    const [assigned, departments, positions] = await Promise.all([
      getMembership(id),
      getDepartments(),
      getPositions(),
    ]);
    if (version !== loadVersion) return;
    if (assigned.userId !== id) throw new Error('用户归属响应不匹配');
    await form.reset();
    if (version !== loadVersion) return;
    const departmentItems = membershipDepartmentOptions(
      departments,
      assigned.departmentId,
    );
    const positionItems = membershipPositionOptions(
      positions,
      assigned.positionIds,
    );
    form.updateSchema([
      {
        fieldName: 'departmentId',
        componentProps: {
          options: departmentItems,
          showSearch: true,
          optionFilterProp: 'label',
          disabled: false,
        },
      },
      {
        fieldName: 'positionIds',
        componentProps: {
          options: positionItems,
          mode: 'multiple',
          allowClear: true,
          optionFilterProp: 'label',
          disabled: false,
        },
      },
    ]);
    await form.setValues({
      departmentId: assigned.departmentId,
      positionIds: [...assigned.positionIds],
    });
    if (version !== loadVersion) return;
    original.value = { ...assigned, positionIds: [...assigned.positionIds] };
    loaded.value = true;
  } catch {
    // A failed read must never become an empty replacement.
  } finally {
    if (version === loadVersion) loading.value = false;
  }
}
async function persist(data: Membership) {
  if (saving.value || !loaded.value || data.userId !== userId.value) return;
  saving.value = true;
  const token = accessStore.accessToken;
  try {
    form.updateSchema([
      { fieldName: 'departmentId', componentProps: { disabled: true } },
      { fieldName: 'positionIds', componentProps: { disabled: true } },
    ]);
    await updateMembership(data);
    loaded.value = false;
    message.success('归属已更新，该用户需要重新登录');
    open.value = false;
    if (
      token === accessStore.accessToken &&
      String(data.userId) === String(userStore.userInfo?.userId)
    ) {
      await auth.logout(false, false);
      return;
    }
    emit('saved');
  } finally {
    saving.value = false;
    form.updateSchema([
      { fieldName: 'departmentId', componentProps: { disabled: false } },
      { fieldName: 'positionIds', componentProps: { disabled: false } },
    ]);
  }
}
async function save() {
  if (
    !loaded.value ||
    loading.value ||
    saving.value ||
    validating.value ||
    !original.value
  )
    return;
  const version = loadVersion;
  validating.value = true;
  try {
    if (!(await form.validate()).valid) return;
    const values = await form.getValues();
    if (version !== loadVersion || !original.value) return;
    const data = {
      userId: userId.value,
      departmentId: values.departmentId ?? 0,
      positionIds: [...new Set(values.positionIds ?? [])],
    };
    if (sameMembership(original.value, data)) {
      message.info('归属没有变化');
      open.value = false;
      return;
    }
    confirmAction(
      `确认修改 ${userName.value} 的归属？`,
      () => persist(data),
      '部门或岗位的真实变更会立即撤销该用户所有旧会话；修改自己的归属后将退出登录。',
    );
  } finally {
    validating.value = false;
  }
}
defineExpose({ show });
</script>
<template>
  <Modal
    v-model:open="open"
    :title="`${userName} · 部门岗位归属`"
    :confirm-loading="saving"
    :ok-button-props="{ disabled: !loaded || loading || validating }"
    :cancel-button-props="{ disabled: saving }"
    :closable="!saving"
    :mask-closable="false"
    :force-render="true"
    @ok="save"
  >
    <Alert
      class="mb-4"
      type="info"
      show-icon
      message="可以选择未分配部门和清空岗位。保存实际变更后，该用户所有旧登录会话立即失效。"
    />
    <Spin :spinning="loading">
      <Form v-show="loaded" />
      <Alert
        v-if="!loading && !loaded"
        type="error"
        message="归属或组织选项读取失败，保存已禁用。"
      >
        <template #action>
          <Button @click="show(userId, userName)">重新读取</Button>
        </template>
      </Alert>
    </Spin>
  </Modal>
</template>
