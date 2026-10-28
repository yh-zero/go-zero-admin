<script setup lang="ts">
import type { Dictionary } from '#/api/business/system/types';

import { ref } from 'vue';

import { Page } from '@vben/common-ui';

import { Button, message, Modal, Space } from 'antdv-next';

import { useVbenForm, z } from '#/adapter/form';
import { useVbenVxeGrid } from '#/adapter/vxe-table';
import {
  createDictionary,
  deleteDictionary,
  getDictionaries,
  getDictionary,
  updateDictionary,
} from '#/api/business/system/dictionary';

import { confirmAction, statusOptions, usePermission } from '../shared';
import Items from './items.vue';
const can = usePermission('dictionary');
const open = ref(false);
const saving = ref(false);
const editing = ref<Dictionary>();
const items = ref<InstanceType<typeof Items>>();
const previewOpen = ref(false);
const preview = ref<Dictionary>();
const [Form, form] = useVbenForm<Dictionary>({
  showDefaultActions: false,
  wrapperClass: 'grid-cols-1',
  schema: [
    {
      fieldName: 'name',
      label: '字典名称',
      component: 'Input',
      rules: 'required',
    },
    {
      fieldName: 'type',
      label: '字典类型',
      component: 'Input',
      rules: z
        .string()
        .regex(
          /^[A-Za-z][\w-]*$/,
          '以英文字母开头，使用字母、数字、下划线或连字符',
        ),
    },
    {
      fieldName: 'status',
      label: '状态',
      component: 'Select',
      componentProps: { options: statusOptions },
      rules: 'selectRequired',
      defaultValue: 1,
    },
    { fieldName: 'desc', label: '描述', component: 'Textarea' },
  ],
});
const [Grid, grid] = useVbenVxeGrid<Dictionary>({
  gridOptions: {
    columns: [
      { field: 'name', title: '名称', minWidth: 150 },
      { field: 'type', title: '类型', minWidth: 150 },
      {
        field: 'status',
        title: '状态',
        width: 90,
        formatter: ({ cellValue }: { cellValue: number }) =>
          cellValue === 1 ? '启用' : '停用',
      },
      { field: 'desc', title: '描述', minWidth: 200 },
      {
        title: '操作',
        width: 310,
        fixed: 'right',
        slots: { default: 'actions' },
      },
    ],
    pagerConfig: { enabled: false },
    rowConfig: { keyField: 'ID' },
    toolbarConfig: { refresh: true },
    proxyConfig: {
      ajax: {
        query: async () => ({ items: (await getDictionaries()).list ?? [] }),
      },
    },
  },
});
async function showForm(row?: Dictionary) {
  const detail = row
    ? await getDictionary({ id: row.ID, status: row.status })
    : undefined;
  editing.value = detail;
  open.value = true;
  await form.reset();
  await form.setValues(
    detail ? { ...detail } : { name: '', type: '', status: 1, desc: '' },
  );
}
async function save() {
  if (saving.value) return;
  saving.value = true;
  try {
    if (!(await form.validate()).valid) return;
    const values = await form.getValues();
    const data = {
      ...values,
      ID: editing.value?.ID ?? 0,
      desc: values.desc ?? '',
    };
    await (editing.value ? updateDictionary(data) : createDictionary(data));
    message.success('保存成功');
    open.value = false;
    await grid.query();
  } finally {
    saving.value = false;
  }
}
function remove(row: Dictionary) {
  confirmAction(
    `删除字典 ${row.name}？`,
    async () => {
      await deleteDictionary(row.ID);
      message.success('已删除');
      await grid.query();
    },
    '存在字典项时，需要先删除字典项。',
  );
}
async function showPreview(row: Dictionary) {
  preview.value = await getDictionary({ type: row.type, status: row.status });
  previewOpen.value = true;
}
</script>
<template>
  <Page>
    <Grid>
      <template #toolbar-tools>
        <Button v-if="can('create')" type="primary" @click="showForm()">
          新增字典
        </Button>
      </template>
      <template #actions="{ row }">
        <Space>
          <Button type="link" size="small" @click="showPreview(row)">
            预览
          </Button>
          <Button
            v-if="can('items')"
            type="link"
            size="small"
            @click="items?.show(row)"
          >
            字典项
          </Button>
          <Button
            v-if="can('update')"
            type="link"
            size="small"
            @click="showForm(row)"
          >
            编辑
          </Button>
          <Button
            v-if="can('delete')"
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
    <Modal
      v-model:open="open"
      :title="editing ? '编辑字典' : '新增字典'"
      :confirm-loading="saving"
      :force-render="true"
      :mask-closable="false"
      :cancel-button-props="{ disabled: saving }"
      :closable="!saving"
      @ok="save"
    >
      <Form />
    </Modal>
    <Items ref="items" />
    <Modal
      v-model:open="previewOpen"
      :title="`${preview?.name ?? ''} · 业务选项预览`"
      :footer="null"
    >
      <p class="mb-4">显示详情接口返回的业务选项；管理全部状态请打开字典项。</p>
      <ul class="space-y-2">
        <li v-for="item in preview?.sysDictionaryInfoList ?? []" :key="item.ID">
          {{ item.label }} — {{ item.value
          }}{{ item.extend ? ` (${item.extend})` : '' }}
        </li>
      </ul>
      <p v-if="!preview?.sysDictionaryInfoList?.length">暂无业务选项</p>
    </Modal>
  </Page>
</template>
