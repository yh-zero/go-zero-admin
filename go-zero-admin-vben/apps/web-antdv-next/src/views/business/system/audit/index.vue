<script setup lang="ts">
import type { TableQuery } from '../shared';
import type { AuditFilter } from './format';

import type { AuditLog } from '#/api/business/system/audit';

import { computed, ref } from 'vue';

import { Page } from '@vben/common-ui';

import {
  Button,
  Descriptions,
  DescriptionsItem,
  message,
  Modal,
  Tag,
} from 'antdv-next';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import { getAuditLogs } from '#/api/business/system/audit';

import { auditQuery, auditTime, safeAuditParams } from './format';

// eslint-disable-next-line vue/component-definition-name-casing -- Match the backend menu name for route caching.
defineOptions({ name: 'audit' });
const detail = ref<AuditLog>();
const open = ref(false);
const params = computed(() => safeAuditParams(detail.value?.params ?? ''));
const [Grid] = useVbenVxeGrid<AuditLog>({
  formOptions: {
    schema: [
      {
        fieldName: 'eventType',
        label: '日志类型',
        component: 'Select',
        componentProps: {
          allowClear: true,
          options: [
            { label: '操作审计', value: 'operation' },
            { label: '登录日志', value: 'login' },
          ],
        },
      },
      {
        fieldName: 'result',
        label: '结果',
        component: 'Select',
        componentProps: {
          allowClear: true,
          options: [
            { label: '成功', value: 'success' },
            { label: '失败', value: 'failure' },
          ],
        },
      },
      {
        fieldName: 'actorName',
        label: '操作者账号',
        component: 'Input',
        componentProps: { placeholder: '精确账号，含登录失败账号' },
      },
      {
        fieldName: 'actorId',
        label: '操作者 ID',
        component: 'InputNumber',
        componentProps: { min: 1, precision: 0 },
      },
      {
        fieldName: 'module',
        label: '业务模块',
        component: 'Select',
        componentProps: {
          allowClear: true,
          options: [
            { label: '用户', value: 'user' },
            { label: '菜单', value: 'menu' },
            { label: '角色', value: 'authority' },
            { label: 'API', value: 'api' },
            { label: '接口授权', value: 'casbin' },
            { label: '字典', value: 'dictionary' },
            { label: '组织', value: 'organization' },
            { label: '文件', value: 'files' },
            { label: '会话', value: 'session' },
            { label: '工具', value: 'base' },
          ],
        },
      },
      {
        fieldName: 'timeRange',
        label: '发生时间',
        component: 'RangePicker',
        componentProps: {
          showTime: true,
          valueFormat: 'YYYY-MM-DD HH:mm:ss',
          allowClear: true,
        },
      },
    ],
  },
  gridOptions: {
    columns: [
      {
        field: 'createdAt',
        title: '发生时间',
        width: 185,
        formatter: ({ cellValue }: { cellValue: string }) =>
          auditTime(cellValue),
      },
      {
        field: 'eventType',
        title: '类型',
        width: 100,
        formatter: ({ cellValue }: { cellValue: string }) =>
          cellValue === 'login' ? '登录' : '操作',
      },
      { field: 'actorName', title: '操作者账号', minWidth: 130 },
      { field: 'actorId', title: '用户 ID', width: 95 },
      { field: 'module', title: '模块', width: 120 },
      { field: 'action', title: '动作', minWidth: 175 },
      { field: 'object', title: '操作对象', minWidth: 120 },
      { field: 'ip', title: '来源 IP', minWidth: 135 },
      {
        field: 'result',
        title: '结果',
        width: 90,
        slots: { default: 'result' },
      },
      {
        title: '操作',
        width: 80,
        fixed: 'right',
        slots: { default: 'actions' },
      },
    ],
    pagerConfig: { pageSize: 20 },
    rowConfig: { keyField: 'id' },
    toolbarConfig: { refresh: true },
    proxyConfig: {
      ajax: {
        query: async ({ page }: TableQuery, values: AuditFilter) => {
          let query;
          try {
            query = auditQuery({
              ...values,
              pageNo: page.currentPage,
              pageSize: page.pageSize,
            });
          } catch (error) {
            message.warning((error as Error).message);
            throw error;
          }
          const response = await getAuditLogs(query);
          return { items: response.list ?? [], total: response.total };
        },
      },
    },
  },
});
function showDetail(row: AuditLog) {
  detail.value = row;
  open.value = true;
}
</script>

<template>
  <Page>
    <Grid>
      <template #result="{ row }">
        <Tag :color="row.result === 'success' ? 'green' : 'red'">
          {{ row.result === 'success' ? '成功' : '失败' }}
        </Tag>
      </template>
      <template #actions="{ row }">
        <Button type="link" size="small" @click="showDetail(row)">详情</Button>
      </template>
    </Grid>
    <Modal v-model:open="open" title="审计详情" :footer="null" :width="760">
      <Descriptions v-if="detail" bordered :column="2" size="small">
        <DescriptionsItem label="发生时间">
          {{ auditTime(detail.createdAt) }}
        </DescriptionsItem>
        <DescriptionsItem label="类型">
          {{ detail.eventType === 'login' ? '登录日志' : '操作审计' }}
        </DescriptionsItem>
        <DescriptionsItem label="操作者">
          {{ detail.actorName || '未识别' }}
          {{ detail.actorId ? `（${detail.actorId}）` : '' }}
        </DescriptionsItem>
        <DescriptionsItem label="角色 ID">
          {{ detail.authorityId || '—' }}
        </DescriptionsItem>
        <DescriptionsItem label="模块 / 动作">
          {{ detail.module }} / {{ detail.action }}
        </DescriptionsItem>
        <DescriptionsItem label="操作对象">
          {{ detail.object || '—' }}
        </DescriptionsItem>
        <DescriptionsItem label="结果">
          {{ detail.result === 'success' ? '成功' : '失败' }}
        </DescriptionsItem>
        <DescriptionsItem label="请求状态">
          {{ detail.statusCode || '事务内记录' }}
        </DescriptionsItem>
        <DescriptionsItem label="来源 IP">
          {{
            detail.ip || '—'
          }}
        </DescriptionsItem>
        <DescriptionsItem label="耗时">
          {{ detail.durationMs }} ms
        </DescriptionsItem>
        <DescriptionsItem label="请求路径" :span="2">
          <span class="break-all">{{ detail.method }} {{ detail.path || '—' }}</span>
        </DescriptionsItem>
        <DescriptionsItem label="追踪 ID" :span="2">
          <span class="break-all">{{ detail.traceId || '—' }}</span>
        </DescriptionsItem>
      </Descriptions>
      <Descriptions
        v-if="params.length"
        title="资源标识与状态"
        bordered
        :column="1"
        size="small"
        class="mt-4"
      >
        <DescriptionsItem
          v-for="(entry, index) in params"
          :key="index"
          :label="entry.label"
        >
          {{ entry.value }}
        </DescriptionsItem>
      </Descriptions>
      <p class="text-muted-foreground mt-4 text-sm">
        请求状态与业务结果分别记录，请以结果判断操作是否成功。时间按本地时区显示。
      </p>
    </Modal>
  </Page>
</template>
