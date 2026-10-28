<script setup lang="ts">
import type { TableQuery } from '../shared';

import type {
  OrganizationQuery,
  Position,
} from '#/api/business/system/organization';

import { ref } from 'vue';

import { Page } from '@vben/common-ui';

import { Button, message, Modal, Space, Tag } from 'antdv-next';

import { useVbenForm, z } from '#/adapter/form';
import { useVbenVxeGrid } from '#/adapter/vxe-table';
import {
  createPosition,
  deletePosition,
  getPositions,
  updatePosition,
} from '#/api/business/system/organization';

import { confirmAction, statusOptions, usePermission } from '../shared';

const can = usePermission('organization-positions');
const open = ref(false);
const saving = ref(false);
const editing = ref<Position>();
const rowBusy = ref<number>();
const textRule = z
  .string()
  .trim()
  .min(1, '请输入名称或编码')
  .refine((value) => [...value].length <= 64, '最多 64 个字符');
const [Form, form] = useVbenForm<Position>({
  showDefaultActions: false,
  wrapperClass: 'grid-cols-1',
  schema: [
    {
      fieldName: 'name',
      label: '岗位名称',
      component: 'Input',
      rules: textRule,
    },
    {
      fieldName: 'code',
      label: '岗位编码',
      component: 'Input',
      rules: textRule,
      help: '编码在所有岗位中唯一，不区分大小写。',
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
const [Grid, grid] = useVbenVxeGrid<Position>({
  formOptions: {
    schema: [
      {
        fieldName: 'keyword',
        label: '搜索',
        component: 'Input',
        componentProps: { placeholder: '岗位名称或编码', allowClear: true },
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
      { field: 'name', title: '岗位名称', minWidth: 160 },
      { field: 'code', title: '编码', minWidth: 150 },
      { field: 'sort', title: '排序', width: 90 },
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
    pagerConfig: { pageSize: 10 },
    rowConfig: { keyField: 'id' },
    toolbarConfig: { refresh: true },
    proxyConfig: {
      ajax: {
        query: async ({ page }: TableQuery, values: OrganizationQuery) => {
          const all = await getPositions(values);
          const start = (page.currentPage - 1) * page.pageSize;
          return {
            items: all.slice(start, start + page.pageSize),
            total: all.length,
          };
        },
      },
    },
  },
});
async function showForm(row?: Position) {
  if (saving.value) return;
  editing.value = row;
  open.value = true;
  await form.reset();
  await form.setValues(row ?? { name: '', code: '', sort: 0, status: 1 });
}
async function save() {
  if (saving.value) return;
  saving.value = true;
  try {
    if (!(await form.validate()).valid) return;
    const values = await form.getValues();
    const data = {
      name: values.name.trim(),
      code: values.code.trim(),
      sort: values.sort ?? 0,
      status: values.status,
    };
    await (editing.value
      ? updatePosition({ ...data, id: editing.value.id })
      : createPosition(data));
    message.success('岗位已保存');
    open.value = false;
    await grid.query();
  } finally {
    saving.value = false;
  }
}
function remove(row: Position) {
  if (rowBusy.value !== undefined) return;
  confirmAction(
    `删除岗位 ${row.name}？`,
    async () => {
      if (rowBusy.value !== undefined) return;
      rowBusy.value = row.id;
      try {
        await deletePosition(row.id);
        message.success('岗位已删除');
        await grid.reload();
      } finally {
        rowBusy.value = undefined;
      }
    },
    '有人员关联的岗位不能删除，请先修改用户归属。',
  );
}
</script>
<template>
  <Page>
    <Grid>
      <template #toolbar-tools>
        <Button
          v-if="can('create')"
          type="primary"
          :disabled="saving"
          @click="showForm()"
        >
          新增岗位
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
            :disabled="saving || rowBusy !== undefined"
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
      :title="editing ? '编辑岗位' : '新增岗位'"
      :confirm-loading="saving"
      :cancel-button-props="{ disabled: saving }"
      :closable="!saving"
      :mask-closable="false"
      :force-render="true"
      @ok="save"
    >
      <Form />
    </Modal>
  </Page>
</template>
