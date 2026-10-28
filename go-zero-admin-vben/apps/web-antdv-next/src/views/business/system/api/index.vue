<script setup lang="ts">
import type { TableQuery } from '../shared';

import type { ApiResource } from '#/api/business/system/types';

import { ref } from 'vue';

import { Page } from '@vben/common-ui';

import { Button, message, Modal, Space } from 'antdv-next';

import { useVbenForm, z } from '#/adapter/form';
import { useVbenVxeGrid } from '#/adapter/vxe-table';
import {
  createApi,
  deleteApi,
  deleteApis,
  getApis,
  updateApi,
} from '#/api/business/system/api';

import { confirmAction, methods, usePermission } from '../shared';
import SyncDrawer from './sync.vue';

const syncDrawer = ref<InstanceType<typeof SyncDrawer>>();
const can = usePermission('api');
const open = ref(false);
const saving = ref(false);
const editing = ref<ApiResource>();
const selected = ref<ApiResource[]>([]);
const [Form, form] = useVbenForm<ApiResource>({
  showDefaultActions: false,
  wrapperClass: 'grid-cols-1',
  schema: [
    {
      fieldName: 'path',
      label: '接口路径',
      component: 'Input',
      componentProps: { placeholder: '/v1/sys/...' },
      rules: z
        .string()
        .regex(
          /^\/v1\/[^?#\s]+$/,
          '填写 /v1/ 开头的后端路径，不含 /api 代理前缀',
        ),
    },
    {
      fieldName: 'method',
      label: '方法',
      component: 'Select',
      componentProps: { options: methods },
      rules: 'required',
      defaultValue: 'GET',
    },
    {
      fieldName: 'apiGroup',
      label: '分组',
      component: 'Input',
      rules: 'required',
    },
    {
      fieldName: 'description',
      label: '描述',
      component: 'Input',
      rules: 'required',
    },
  ],
});
const [Grid, grid] = useVbenVxeGrid<ApiResource>({
  formOptions: {
    schema: [
      { fieldName: 'path', label: '路径', component: 'Input' },
      { fieldName: 'apiGroup', label: '分组', component: 'Input' },
      {
        fieldName: 'method',
        label: '方法',
        component: 'Select',
        componentProps: { options: methods, allowClear: true },
      },
    ],
  },
  gridEvents: {
    checkboxChange: () => {
      selected.value = grid.grid.getCheckboxRecords();
    },
    checkboxAll: () => {
      selected.value = grid.grid.getCheckboxRecords();
    },
  },
  gridOptions: {
    columns: [
      { type: 'checkbox', width: 45 },
      { field: 'path', title: '路径', minWidth: 260 },
      { field: 'method', title: '方法', width: 100 },
      { field: 'apiGroup', title: '分组', minWidth: 120 },
      { field: 'description', title: '描述', minWidth: 200 },
      {
        title: '操作',
        width: 150,
        fixed: 'right',
        slots: { default: 'actions' },
      },
    ],
    pagerConfig: { pageSize: 20 },
    rowConfig: { keyField: 'ID' },
    toolbarConfig: { refresh: true },
    proxyConfig: {
      ajax: {
        query: async ({ page }: TableQuery, values: Partial<ApiResource>) => {
          selected.value = [];
          const result = await getApis({
            ...values,
            pageNo: page.currentPage,
            pageSize: page.pageSize,
            orderKey: 'id',
            desc: true,
          });
          return { items: result.list ?? [], total: result.total };
        },
      },
    },
  },
});
async function showForm(row?: ApiResource) {
  editing.value = row;
  open.value = true;
  await form.reset();
  await form.setValues(
    row
      ? { ...row }
      : { path: '', method: 'GET', apiGroup: '', description: '' },
  );
}
async function save() {
  if (saving.value) return;
  saving.value = true;
  try {
    if (!(await form.validate()).valid) return;
    const values = await form.getValues();
    const data = { ...values, ID: editing.value?.ID ?? 0 };
    await (editing.value ? updateApi(data) : createApi(data));
    message.success('保存成功');
    open.value = false;
    await grid.query();
  } finally {
    saving.value = false;
  }
}
function remove(row: ApiResource) {
  confirmAction(
    `删除 ${row.method} ${row.path}？`,
    async () => {
      await deleteApi(row);
      message.success('已删除');
      await grid.reload();
    },
    '关联的角色接口策略也会清理。',
  );
}
function removeSelected() {
  if (!selected.value.length) return;
  const ids = selected.value.map((row) => row.ID);
  confirmAction(
    `删除所选 ${ids.length} 个接口？`,
    async () => {
      await deleteApis(ids);
      message.success('已删除');
      await grid.reload();
    },
    '关联的角色接口策略也会清理。',
  );
}
</script>
<template>
  <Page>
    <Grid>
      <template #toolbar-tools>
        <Space>
          <Button v-if="can('sync')" @click="syncDrawer?.show()">同步后端接口</Button>
          <Button
            v-if="can('delete')"
            danger
            :disabled="!selected.length"
            @click="removeSelected"
          >
            批量删除
          </Button>
          <Button v-if="can('create')" type="primary" @click="showForm()">
            新增接口
          </Button>
        </Space>
      </template>
      <template #actions="{ row }">
        <Space>
          <Button
            v-if="can('update')"
            size="small"
            type="link"
            @click="showForm(row)"
          >
            编辑
          </Button>
          <Button
            v-if="can('delete')"
            size="small"
            type="link"
            danger
            @click="remove(row)"
          >
            删除
          </Button>
        </Space>
      </template>
    </Grid>
    <SyncDrawer ref="syncDrawer" @saved="grid.query()" />
    <Modal
      v-model:open="open"
      :title="editing ? '编辑 API' : '新增 API'"
      :confirm-loading="saving"
      :force-render="true"
      :mask-closable="false"
      :cancel-button-props="{ disabled: saving }"
      :closable="!saving"
      @ok="save"
    >
      <Form />
    </Modal>
  </Page>
</template>
