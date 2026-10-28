<script setup lang="ts">
import { Alert, Button } from 'antdv-next';
defineProps<{ ids: number[]; label: string; disabled?: boolean }>();
defineEmits<{ remove: [id: number] }>();
</script>
<template>
  <Alert
    v-if="ids.length"
    type="warning"
    show-icon
    class="mb-4"
    :message="`本地${label}选择包含已删除、停用或不再可授权的资源。编辑已保留，请逐项移除失效选择后保存。`"
  >
    <template #description>
      <ul>
        <li v-for="id in ids" :key="id" class="flex items-center gap-2">
          <span>失效{{ label }} #{{ id }}</span>
          <Button
            :data-remove-unavailable="id"
            :disabled="disabled"
            @click="$emit('remove', id)"
          >
            移除失效{{ label }} #{{ id }}
          </Button>
        </li>
      </ul>
    </template>
  </Alert>
</template>
