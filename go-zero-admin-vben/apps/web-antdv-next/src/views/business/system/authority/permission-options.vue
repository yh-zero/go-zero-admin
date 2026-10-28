<script setup lang="ts">
import type { PermissionOption } from './authorization';

import { computed, ref } from 'vue';

import { Button, CheckboxGroup, Input, Space } from 'antdv-next';

import { permissionChanges, replaceGroupSelection } from './authorization';

const props = defineProps<{
  disabled?: boolean;
  modelValue: number[];
  options: PermissionOption[];
  original: number[];
}>();
const emit = defineEmits<{ 'update:modelValue': [value: number[]] }>();
const keyword = ref('');
const groups = computed(() => {
  const result = new Map<string, PermissionOption[]>();
  const query = keyword.value.trim().toLocaleLowerCase();
  for (const option of props.options) {
    if (
      query &&
      !`${option.group} ${option.label}`.toLocaleLowerCase().includes(query)
    )
      continue;
    const items = result.get(option.group) ?? [];
    items.push(option);
    result.set(option.group, items);
  }
  return [...result].map(([title, options]) => ({ title, options }));
});
const changes = computed(() =>
  permissionChanges(props.original, props.modelValue),
);
const labels = computed(
  () =>
    new Map(
      props.options.map((option) => [
        option.value,
        `${option.group} · ${option.label}`,
      ]),
    ),
);
function updateGroup(options: PermissionOption[], values: unknown[]) {
  emit(
    'update:modelValue',
    replaceGroupSelection(
      props.modelValue,
      options.map((item) => item.value),
      values.map(Number),
    ),
  );
}
</script>

<template>
  <div>
    <Input
      v-model:value="keyword"
      class="mb-3"
      allow-clear
      :disabled="disabled"
      placeholder="搜索分组、名称或权限标识"
      aria-label="搜索权限"
    />
    <p class="mb-3 text-sm" aria-live="polite">
      已选 {{ changes.total }} 项 · 新增 {{ changes.added.length }} 项 · 撤销
      {{ changes.removed.length }} 项
    </p>
    <details
      v-if="changes.added.length || changes.removed.length"
      class="mb-4 rounded border p-3 text-sm"
    >
      <summary class="cursor-pointer">查看本次授权变更</summary>
      <ul class="mt-2 space-y-1">
        <li v-for="id in changes.added" :key="`add-${id}`" class="break-words">
          新增：{{ labels.get(id) ?? `ID ${id}` }}
        </li>
        <li
          v-for="id in changes.removed"
          :key="`remove-${id}`"
          class="break-words"
        >
          撤销：{{ labels.get(id) ?? `ID ${id}` }}
        </li>
      </ul>
    </details>
    <p class="text-muted-foreground mb-3 text-xs">
      分组操作只影响当前搜索结果，其他已选权限会保留。
    </p>
    <section
      v-for="group in groups"
      :key="group.title"
      class="mb-4 rounded border p-3"
    >
      <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
        <span class="font-medium"
          >{{ group.title }}（{{
            group.options.filter((item) => modelValue.includes(item.value))
              .length
          }}/{{ group.options.length }}）</span
        >
        <Space>
          <Button
            size="small"
            :disabled="disabled"
            @click="
              updateGroup(
                group.options,
                group.options.map((item) => item.value),
              )
            "
            >全选本组</Button
          >
          <Button
            size="small"
            :disabled="disabled"
            @click="updateGroup(group.options, [])"
            >取消本组</Button
          >
        </Space>
      </div>
      <CheckboxGroup
        :value="
          modelValue.filter((id) =>
            group.options.some((item) => item.value === id),
          )
        "
        :options="group.options"
        :disabled="disabled"
        class="flex flex-col gap-3 break-all"
        @update:value="(values) => updateGroup(group.options, values)"
      />
    </section>
    <p v-if="!groups.length" class="text-muted-foreground">没有匹配的权限。</p>
  </div>
</template>
