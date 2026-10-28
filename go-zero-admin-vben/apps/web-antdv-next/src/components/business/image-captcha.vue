<script setup lang="ts">
import { Input } from 'antdv-next';

defineProps<{
  disabled?: boolean;
  image?: string;
  loading?: boolean;
  value?: string;
}>();
const emit = defineEmits<{ refresh: []; 'update:value': [string] }>();
</script>

<template>
  <div class="flex items-center gap-3">
    <Input
      :value="value"
      :disabled="disabled"
      aria-label="图片验证码"
      autocomplete="off"
      placeholder="请输入图片验证码"
      :maxlength="6"
      @update:value="emit('update:value', $event)"
    />
    <button
      type="button"
      class="h-10 min-w-28 rounded border px-2"
      :disabled="disabled || loading"
      aria-label="刷新图片验证码"
      @click="emit('refresh')"
    >
      <img
        v-if="image && !loading"
        :src="image"
        alt="验证码，点击刷新"
        class="h-9 w-28 rounded bg-white object-contain"
      />
      <span v-else>{{ loading ? '加载中…' : '点击重试' }}</span>
    </button>
  </div>
</template>
