<script setup lang="ts">
import type { ApiSyncPreview } from '#/api/business/system/api';

import { computed, ref } from 'vue';
import {
  Alert,
  Button,
  CheckboxGroup,
  Drawer,
  Empty,
  message,
  Space,
  Spin,
} from 'antdv-next';

import { applyApiSync, previewApiSync } from '#/api/business/system/api';

const emit = defineEmits<{ saved: [] }>();
const open = ref(false);
const loading = ref(false);
const saving = ref(false);
const loadError = ref('');
const preview = ref<ApiSyncPreview>();
const selected = ref<string[]>([]);
let loadVersion = 0;
const sections = computed(() => [
  { title: '新增接口', items: preview.value?.added ?? [] },
  { title: '更新接口说明', items: preview.value?.changed ?? [] },
]);
const total = computed(() =>
  sections.value.reduce((sum, section) => sum + section.items.length, 0),
);

async function load() {
  const current = ++loadVersion;
  loading.value = true;
  preview.value = undefined;
  loadError.value = '';
  selected.value = [];
  try {
    const result = await previewApiSync();
    if (current === loadVersion) preview.value = result;
  } catch {
    if (current === loadVersion)
      loadError.value = '接口预览加载失败，请重新预览后再选择同步项。';
  } finally {
    if (current === loadVersion) loading.value = false;
  }
}
async function show() {
  open.value = true;
  await load();
}
function selectAll() {
  selected.value = sections.value.flatMap((section) =>
    section.items.map((item) => item.key),
  );
}
function selectSection(keys: string[], values: unknown[]) {
  const group = new Set(keys);
  selected.value = [
    ...new Set([
      ...selected.value.filter((key) => !group.has(key)),
      ...values.map(String).filter((key) => group.has(key)),
    ]),
  ];
}
async function save() {
  if (
    !preview.value ||
    loading.value ||
    saving.value ||
    selected.value.length === 0
  )
    return;
  saving.value = true;
  try {
    const result = await applyApiSync(preview.value.version, selected.value);
    message.success(
      `已新增 ${result.added} 个接口，更新 ${result.updated} 个说明`,
    );
    emit('saved');
  } catch {
    // The request client reports errors. Refresh once even when the outcome is
    // uncertain, so a version conflict or interrupted request cannot reuse a stale selection.
  } finally {
    await load();
    saving.value = false;
  }
}
defineExpose({ show });
</script>

<template>
  <Drawer
    v-model:open="open"
    title="同步后端接口"
    :size="820"
    :closable="!saving"
    :mask-closable="!saving"
  >
    <Alert
      class="mb-4"
      show-icon
      type="info"
      message="选择要同步的接口。现有角色授权保留；新增接口需要另行授权。"
    />
    <Alert
      v-if="loadError"
      class="mb-4"
      show-icon
      type="error"
      :message="loadError"
    />
    <Spin :spinning="loading">
      <Space class="mb-4">
        <Button :disabled="loading || saving || !preview" @click="selectAll"
          >全选可同步项</Button
        >
        <Button :disabled="loading || saving" @click="selected = []"
          >取消选择</Button
        >
        <Button :disabled="loading || saving" @click="load">重新预览</Button>
        <span>已选 {{ selected.length }} / {{ total }}</span>
      </Space>
      <template v-if="preview">
        <section v-for="section in sections" :key="section.title" class="mb-6">
          <h3 class="mb-3 font-medium">
            {{ section.title }}（{{ section.items.length }}）
          </h3>
          <CheckboxGroup
            :value="
              selected.filter((key) =>
                section.items.some((item) => item.key === key),
              )
            "
            :disabled="saving"
            class="flex flex-col gap-3"
            :options="
              section.items.map((item) => ({
                value: item.key,
                label: `${item.method} ${item.path} · ${item.apiGroup} · ${item.description}${item.id ? `（原：${item.currentApiGroup} · ${item.currentDescription}）` : ''}`,
              }))
            "
            @change="
              (values) =>
                selectSection(
                  section.items.map((item) => item.key),
                  values,
                )
            "
          />
          <Empty
            v-if="section.items.length === 0"
            :image="Empty.PRESENTED_IMAGE_SIMPLE"
            description="没有待同步项"
          />
        </section>
        <section v-if="preview.obsolete?.length">
          <h3 class="mb-3 font-medium">
            不在当前授权清单中（{{ preview.obsolete.length }}）
          </h3>
          <p class="mb-3 text-sm text-muted-foreground">
            包括已移除或无需接口授权的路由。这些资源保留不动，核对用途后再单独处理。
          </p>
          <ul class="space-y-2 break-all">
            <li v-for="item in preview.obsolete" :key="item.key">
              {{ item.key }} · {{ item.description }}
            </li>
          </ul>
        </section>
      </template>
    </Spin>
    <template #footer>
      <Space>
        <Button :disabled="saving" @click="open = false">关闭</Button>
        <Button
          type="primary"
          :loading="saving"
          :disabled="saving || loading || !preview || !selected.length"
          @click="save"
        >
          应用所选 {{ selected.length }} 项
        </Button>
      </Space>
    </template>
  </Drawer>
</template>
