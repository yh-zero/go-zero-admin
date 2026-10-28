<script setup lang="ts">
import type { TableQuery } from '../shared';

import type { Dictionary, DictionaryItem } from '#/api/business/system/types';

import { ref } from 'vue';

import { Button, Drawer, message, Modal, Space } from 'antdv-next';

import { useVbenForm, z } from '#/adapter/form';
import { useVbenVxeGrid } from '#/adapter/vxe-table';
import {
  createDictionaryItem,
  deleteDictionaryItem,
  getDictionaryItem,
  getDictionaryItems,
  updateDictionaryItem,
} from '#/api/business/system/dictionary';

import { confirmAction, statusOptions, usePermission } from '../shared';
const can = usePermission('dictionary');
const open = ref(false);
const formOpen = ref(false);
const saving = ref(false);
const dictionary = ref<Dictionary>();
const editing = ref<DictionaryItem>();
const [Form, form] = useVbenForm<DictionaryItem>({
  showDefaultActions: false,
  wrapperClass: 'grid-cols-1',
  schema: [
    {
      fieldName: 'label',
      label: '展示值',
      component: 'Input',
      rules: 'required',
    },
    {
      fieldName: 'value',
      label: '字典值',
      component: 'InputNumber',
      componentProps: { precision: 0 },
      rules: z.number().int(),
      defaultValue: 0,
    },
    { fieldName: 'extend', label: '扩展值', component: 'Input' },
    {
      fieldName: 'status',
      label: '状态',
      component: 'Select',
      componentProps: { options: statusOptions },
      rules: 'selectRequired',
      defaultValue: 1,
    },
    {
      fieldName: 'sort',
      label: '排序',
      component: 'InputNumber',
      componentProps: { precision: 0, min: 0 },
      rules: z.number().int().min(0),
      defaultValue: 0,
    },
  ],
});
const [Grid, grid] = useVbenVxeGrid<DictionaryItem>({
  formOptions: {
    schema: [
      { fieldName: 'label', label: '展示值', component: 'Input' },
      {
        fieldName: 'value',
        label: '字典值',
        component: 'InputNumber',
        componentProps: { precision: 0 },
      },
      {
        fieldName: 'status',
        label: '状态',
        component: 'Select',
        componentProps: { options: statusOptions, allowClear: true },
      },
    ],
  },
  gridOptions: {
    columns: [
      { field: 'label', title: '展示值', minWidth: 120 },
      { field: 'value', title: '字典值', width: 95 },
      { field: 'extend', title: '扩展值', minWidth: 130 },
      {
        field: 'status',
        title: '状态',
        width: 80,
        formatter: ({ cellValue }: { cellValue: number }) =>
          cellValue === 1 ? '启用' : '停用',
      },
      { field: 'sort', title: '排序', width: 70 },
      { title: '操作', width: 150, slots: { default: 'actions' } },
    ],
    pagerConfig: { pageSize: 10 },
    rowConfig: { keyField: 'ID' },
    toolbarConfig: { refresh: true },
    proxyConfig: {
      autoLoad: false,
      ajax: {
        query: async (
          { page }: TableQuery,
          values: {
            label?: string;
            status?: number;
            value?: null | number | string;
          },
        ) => {
          if (!dictionary.value) return { items: [], total: 0 };
          const result = await getDictionaryItems({
            sysDictionaryID: dictionary.value.ID,
            pageNo: page.currentPage,
            pageSize: page.pageSize,
            label: values.label,
            status: values.status ?? undefined,
            value:
              values.value === null ||
              values.value === undefined ||
              values.value === ''
                ? undefined
                : Number(values.value),
          });
          return { items: result.list ?? [], total: result.total };
        },
      },
    },
  },
});
async function show(row: Dictionary) {
  dictionary.value = row;
  open.value = true;
  await grid.formApi.reset();
  await grid.reload();
}
async function showForm(row?: DictionaryItem) {
  const detail = row ? await getDictionaryItem(row.ID) : undefined;
  editing.value = detail;
  formOpen.value = true;
  await form.reset();
  await form.setValues(
    detail
      ? { ...detail }
      : { label: '', value: 0, extend: '', status: 1, sort: 0 },
  );
}
async function save() {
  if (saving.value) return;
  saving.value = true;
  try {
    if (!(await form.validate()).valid || !dictionary.value) return;
    const values = await form.getValues();
    const data = {
      ...values,
      ID: editing.value?.ID ?? 0,
      sysDictionaryID: dictionary.value.ID,
      extend: values.extend ?? '',
    };
    await (editing.value
      ? updateDictionaryItem(data)
      : createDictionaryItem(data));
    message.success('字典项已保存');
    formOpen.value = false;
    await grid.query();
  } finally {
    saving.value = false;
  }
}
function remove(row: DictionaryItem) {
  confirmAction(`删除字典项 ${row.label}？`, async () => {
    await deleteDictionaryItem(row.ID);
    message.success('已删除');
    await grid.reload();
  });
}
defineExpose({ show });
</script>
<template>
  <Drawer
    v-model:open="open"
    :title="`${dictionary?.name ?? ''} · 字典项`"
    :size="960"
    :force-render="true"
  >
    <Grid>
      <template #toolbar-tools>
        <Button v-if="can('items')" type="primary" @click="showForm()">
          新增字典项
        </Button>
      </template>
      <template #actions="{ row }">
        <Space>
          <Button
            v-if="can('items')"
            type="link"
            size="small"
            @click="showForm(row)"
          >
            编辑
          </Button>
          <Button
            v-if="can('items')"
            type="link"
            danger
            size="small"
            @click="remove(row)"
          >
            删除
          </Button>
        </Space>
      </template>
    </Grid>
  </Drawer>
  <Modal
    v-model:open="formOpen"
    :title="editing ? '编辑字典项' : '新增字典项'"
    :confirm-loading="saving"
    :force-render="true"
    :mask-closable="false"
    :cancel-button-props="{ disabled: saving }"
    :closable="!saving"
    @ok="save"
  >
    <Form />
  </Modal>
</template>
