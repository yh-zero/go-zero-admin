<script setup lang="ts">
import type {
  Department,
  OrganizationQuery,
} from '#/api/business/system/organization';

import { ref } from 'vue';

import { Page } from '@vben/common-ui';

import { Alert, Button, message, Modal, Space, Spin, Tag } from 'antdv-next';

import { useVbenForm, z } from '#/adapter/form';
import { useVbenVxeGrid } from '#/adapter/vxe-table';
import {
  createDepartment,
  deleteDepartment,
  getDepartments,
  updateDepartment,
} from '#/api/business/system/organization';

import { confirmAction, statusOptions, usePermission } from '../shared';
import { departmentDescendants, departmentOptions } from './helpers';

const can = usePermission('organization-departments');
const open = ref(false);
const loading = ref(false);
const loaded = ref(false);
const saving = ref(false);
const editing = ref<Department>();
const rowBusy = ref<number>();
let loadVersion = 0;
const textRule = z
  .string()
  .trim()
  .min(1, '请输入名称或编码')
  .refine((value) => [...value].length <= 64, '最多 64 个字符');
const [Form, form] = useVbenForm<Department>({
  showDefaultActions: false,
  wrapperClass: 'grid-cols-1',
  schema: [
    {
      fieldName: 'parentId',
      label: '上级部门',
      component: 'Select',
      componentProps: { options: [] },
      rules: 'selectRequired',
      defaultValue: 0,
    },
    {
      fieldName: 'name',
      label: '部门名称',
      component: 'Input',
      rules: textRule,
    },
    {
      fieldName: 'code',
      label: '部门编码',
      component: 'Input',
      rules: textRule,
      help: '编码在所有部门中唯一，不区分大小写。',
    },
    {
      fieldName: 'leader',
      label: '负责人',
      component: 'Input',
      rules: z
        .string()
        .refine((value) => [...value].length <= 64, '最多 64 个字符')
        .optional(),
    },
    {
      fieldName: 'sort',
      label: '排序',
      component: 'InputNumber',
      componentProps: { min: 0, max: 2_147_483_647, precision: 0 },
      rules: z.number().int().min(0).max(2_147_483_647),
      defaultValue: 0,
    },
    {
      fieldName: 'status',
      label: '状态',
      component: 'Select',
      componentProps: { options: statusOptions },
      rules: 'selectRequired',
      defaultValue: 1,
    },
  ],
});
const [Grid, grid] = useVbenVxeGrid<Department>({
  formOptions: {
    schema: [
      {
        fieldName: 'keyword',
        label: '搜索',
        component: 'Input',
        componentProps: { allowClear: true, placeholder: '部门名称或编码' },
      },
      {
        fieldName: 'status',
        label: '状态',
        component: 'Select',
        componentProps: { allowClear: true, options: statusOptions },
      },
    ],
  },
  gridOptions: {
    columns: [
      { field: 'name', title: '部门', minWidth: 200, treeNode: true },
      { field: 'code', title: '编码', minWidth: 150 },
      { field: 'leader', title: '负责人', minWidth: 130 },
      { field: 'sort', title: '排序', width: 80 },
      {
        field: 'status',
        title: '状态',
        width: 90,
        slots: { default: 'status' },
      },
      {
        title: '操作',
        width: 170,
        fixed: 'right',
        slots: { default: 'actions' },
      },
    ],
    pagerConfig: { enabled: false },
    treeConfig: { rowField: 'id', childrenField: 'children', expandAll: true },
    rowConfig: { keyField: 'id' },
    toolbarConfig: { refresh: true },
    proxyConfig: {
      ajax: {
        query: async (_params: unknown, values: OrganizationQuery) => ({
          items: await getDepartments(values),
        }),
      },
    },
  },
});
async function showForm(row?: Department) {
  if (saving.value) return;
  const version = ++loadVersion;
  editing.value = row;
  open.value = true;
  loading.value = true;
  loaded.value = false;
  try {
    const all = await getDepartments();
    if (version !== loadVersion) return;
    const excluded = row
      ? departmentDescendants(all, row.id)
      : new Set<number>();
    const parents = [
      { label: '根部门', value: 0, disabled: false },
      ...departmentOptions(all).filter((item) => !excluded.has(item.value)),
    ];
    await form.reset();
    if (version !== loadVersion) return;
    form.updateSchema([
      {
        fieldName: 'parentId',
        componentProps: {
          options: parents,
          showSearch: true,
          optionFilterProp: 'label',
        },
      },
    ]);
    await form.setValues(
      row
        ? { ...row, parentId: row.parentId ?? 0 }
        : { parentId: 0, name: '', code: '', leader: '', sort: 0, status: 1 },
    );
    if (version === loadVersion) loaded.value = true;
  } catch {
    // requestClient already displays the server error. Keep retry available.
  } finally {
    if (version === loadVersion) loading.value = false;
  }
}
async function save() {
  if (!loaded.value || loading.value || saving.value) return;
  saving.value = true;
  try {
    if (!(await form.validate()).valid) return;
    const values = await form.getValues();
    const data = {
      parentId: values.parentId ?? 0,
      name: values.name.trim(),
      code: values.code.trim(),
      leader: values.leader?.trim() ?? '',
      sort: values.sort ?? 0,
      status: values.status,
    };
    await (editing.value
      ? updateDepartment({ ...data, id: editing.value.id })
      : createDepartment(data));
    message.success('部门已保存');
    open.value = false;
    await grid.query();
  } finally {
    saving.value = false;
  }
}
function remove(row: Department) {
  if (rowBusy.value !== undefined) return;
  confirmAction(
    `删除部门 ${row.name}？`,
    async () => {
      if (rowBusy.value !== undefined) return;
      rowBusy.value = row.id;
      try {
        await deleteDepartment(row.id);
        message.success('部门已删除');
        await grid.query();
      } finally {
        rowBusy.value = undefined;
      }
    },
    '有子部门、人员、自定义数据范围或未删除文件归属时，后端会拒绝删除。',
  );
}
</script>
<template>
  <Page>
    <Alert
      class="mb-4"
      type="info"
      show-icon
      message="部门按完整树展示。筛选结果保留祖先部门；停用部门及其下级不能用于新的人员归属。"
    />
    <Grid>
      <template #toolbar-tools>
        <Button
          v-if="can('create')"
          type="primary"
          :disabled="saving || loading"
          @click="showForm()"
        >
          新增部门
        </Button>
      </template>
      <template #status="{ row }">
        <Tag :color="row.status === 1 ? 'green' : 'red'">
          {{
            row.status === 1 ? '启用' : '停用'
          }}
        </Tag>
      </template>
      <template #actions="{ row }">
        <Space>
          <Button
            v-if="can('update')"
            type="link"
            size="small"
            :disabled="saving || loading || rowBusy !== undefined"
            @click="showForm(row)"
          >
            编辑
          </Button>
          <Button
            v-if="can('delete')"
            type="link"
            size="small"
            danger
            :disabled="saving || rowBusy !== undefined"
            @click="remove(row)"
          >
            删除
          </Button>
        </Space>
      </template>
    </Grid>
    <Modal
      v-model:open="open"
      :title="editing ? '编辑部门' : '新增部门'"
      :confirm-loading="saving"
      :ok-button-props="{ disabled: !loaded || loading }"
      :cancel-button-props="{ disabled: saving }"
      :closable="!saving"
      :mask-closable="false"
      :force-render="true"
      @ok="save"
    >
      <Spin :spinning="loading">
        <Form v-show="loaded" /><Alert
          v-if="!loaded && !loading"
          type="error"
          message="部门选项读取失败，保存已禁用。"
        >
          <template #action>
            <Button @click="showForm(editing)">重新读取</Button>
          </template>
        </Alert>
      </Spin>
    </Modal>
  </Page>
</template>
