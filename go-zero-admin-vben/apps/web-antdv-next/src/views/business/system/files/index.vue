<script setup lang="ts">
import type { TableQuery } from '../shared';

import type { FileQuery, FileResource } from '#/api/business/system/files';

import { computed, ref } from 'vue';

import { Page } from '@vben/common-ui';

import { Alert, Button, message, Space, Tag } from 'antdv-next';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import { deleteFile, getFiles } from '#/api/business/system/files';

import { auditTime } from '../audit/format';
import { confirmAction, isCurrentRole, usePermission } from '../shared';
import Access from './access.vue';
import { fileSize } from './format';
import Reference from './reference.vue';
import Upload from './upload.vue';

// eslint-disable-next-line vue/component-definition-name-casing -- Match the backend menu name for route caching.
defineOptions({ name: 'files' });
const can = usePermission('files');
const canReference = computed(() => isCurrentRole(1) && can('reference'));
const upload = ref<InstanceType<typeof Upload>>();
const access = ref<InstanceType<typeof Access>>();
const reference = ref<InstanceType<typeof Reference>>();
const busy = ref<number>();
const [Grid, grid] = useVbenVxeGrid<FileResource>({
  formOptions: {
    schema: [
      { fieldName: 'name', label: '文件名称', component: 'Input' },
      {
        fieldName: 'status',
        label: '状态',
        component: 'Select',
        componentProps: {
          allowClear: true,
          options: [
            { label: '可用', value: 'active' },
            { label: '待删除，可重试', value: 'deleting' },
            { label: '已删除', value: 'deleted' },
          ],
        },
      },
    ],
  },
  gridOptions: {
    columns: [
      { field: 'id', title: 'ID', width: 85 },
      { field: 'name', title: '名称', minWidth: 200 },
      { field: 'mime', title: '类型', width: 130 },
      {
        field: 'size',
        title: '大小',
        width: 100,
        formatter: ({ cellValue }: { cellValue: number }) =>
          fileSize(cellValue),
      },
      {
        field: 'visibility',
        title: '访问范围',
        width: 105,
        slots: { default: 'visibility' },
      },
      { field: 'ownerId', title: '所有者 ID', width: 105 },
      { field: 'departmentId', title: '部门 ID', width: 100 },
      { field: 'references', title: '引用数', width: 90 },
      {
        field: 'status',
        title: '状态',
        width: 110,
        slots: { default: 'status' },
      },
      {
        field: 'createdAt',
        title: '上传时间',
        width: 185,
        formatter: ({ cellValue }: { cellValue: string }) =>
          auditTime(cellValue),
      },
      {
        title: '操作',
        width: 220,
        fixed: 'right',
        slots: { default: 'actions' },
      },
    ],
    pagerConfig: { pageSize: 20 },
    rowConfig: { keyField: 'id' },
    toolbarConfig: { refresh: true },
    proxyConfig: {
      ajax: {
        query: async ({ page }: TableQuery, values: FileQuery) => {
          const result = await getFiles({
            ...values,
            pageNo: page.currentPage,
            pageSize: page.pageSize,
          });
          return { items: result.list ?? [], total: result.total };
        },
      },
    },
  },
});

function remove(row: FileResource) {
  if (
    !can('delete') ||
    busy.value ||
    row.references > 0 ||
    row.status === 'deleted'
  )
    return;
  confirmAction(
    row.status === 'deleting' ? `重试删除 ${row.name}？` : `删除 ${row.name}？`,
    async () => {
      if (
        !can('delete') ||
        busy.value ||
        row.references > 0 ||
        row.status === 'deleted'
      )
        return;
      busy.value = row.id;
      try {
        await deleteFile(row.id);
        message.success('文件已删除');
      } finally {
        // A storage failure may already have moved the file to "deleting".
        // Reload the actual server state, so the next action is a retry.
        try {
          await grid.query();
        } finally {
          busy.value = undefined;
        }
      }
    },
    '未被引用的文件将从存储服务删除，此操作无法恢复。删除失败时保留待删除状态，可再次重试。',
  );
}
</script>

<template>
  <Page>
    <Alert
      show-icon
      type="info"
      class="mb-4"
      message="列表按当前角色的数据范围显示，默认隐藏已删除记录。公开文件允许匿名访问；私有文件通过短时授权地址访问。后台用户和 App 用户头像引用由系统自动维护，其他业务引用由管理员手动登记。"
    />
    <Grid>
      <template #toolbar-tools>
        <Button v-if="can('upload')" type="primary" @click="upload?.show()">
          上传图片
        </Button>
      </template>
      <template #visibility="{ row }">
        <Tag :color="row.visibility === 'private' ? 'blue' : 'orange'">
          {{ row.visibility === 'private' ? '私有' : '公开' }}
        </Tag>
      </template>
      <template #status="{ row }">
        <Tag
          :color="
            row.status === 'active'
              ? 'green'
              : row.status === 'deleting'
                ? 'orange'
                : 'default'
          "
        >
          {{
            row.status === 'active'
              ? '可用'
              : row.status === 'deleting'
                ? '待删除'
                : '已删除'
          }}
        </Tag>
      </template>
      <template #actions="{ row }">
        <Space>
          <Button
            v-if="row.status === 'active'"
            type="link"
            size="small"
            @click="access?.show(row)"
          >
            预览 / 下载
          </Button>
          <Button
            v-if="canReference && row.status === 'active'"
            type="link"
            size="small"
            :disabled="!!busy"
            @click="reference?.show(row)"
          >
            引用
          </Button>
          <Button
            v-if="can('delete') && row.status !== 'deleted'"
            type="link"
            danger
            size="small"
            :loading="busy === row.id"
            :disabled="!!busy || row.references > 0"
            :title="row.references > 0 ? '文件仍被引用，不能删除' : undefined"
            @click="remove(row)"
          >
            {{ row.status === 'deleting' ? '重试删除' : '删除' }}
          </Button>
        </Space>
      </template>
    </Grid>
    <Upload ref="upload" :allowed="can('upload')" @saved="grid.reload()" />
    <Access ref="access" />
    <Reference ref="reference" @saved="grid.query()" />
  </Page>
</template>
