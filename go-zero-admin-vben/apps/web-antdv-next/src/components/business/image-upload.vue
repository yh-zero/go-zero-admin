<script setup lang="ts">
import { ref } from 'vue';

import { Button, Input, message } from 'antdv-next';

import { resolveAvatar } from '#/adapter/business/avatar';
import {
  IMAGE_UPLOAD_ACCEPT,
  validateImageUpload,
} from '#/adapter/business/image-upload';
import { uploadImage } from '#/api/business/base';
const props = defineProps<{
  value?: string;
  disabled?: boolean;
  canUpload?: boolean;
}>();
const emit = defineEmits<{
  'update:value': [value: string];
  uploadingChange: [value: boolean];
}>();
const uploading = ref(false);
const fileInput = ref<HTMLInputElement>();
async function selected(event: Event) {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0];
  if (!file || props.disabled || !props.canUpload || uploading.value) return;
  input.value = '';
  try {
    validateImageUpload(file);
  } catch (error) {
    message.error(error instanceof Error ? error.message : '图片选择失败');
    return;
  }
  uploading.value = true;
  emit('uploadingChange', true);
  try {
    const result = await uploadImage(file);
    emit('update:value', result.fileImgUrl);
    message.success('图片上传成功');
  } finally {
    uploading.value = false;
    emit('uploadingChange', false);
  }
}
</script>
<template>
  <div class="w-full space-y-2">
    <Input
      :value="value"
      :disabled="disabled || uploading"
      placeholder="图片 URL（可以清空）"
      @update:value="emit('update:value', $event)"
    />
    <input
      ref="fileInput"
      type="file"
      :accept="IMAGE_UPLOAD_ACCEPT"
      hidden
      @change="selected"
    />
    <Button
      v-if="canUpload"
      :disabled="disabled"
      :loading="uploading"
      @click="fileInput?.click()"
    >
      上传图片
    </Button>
    <img
      v-if="value"
      :src="resolveAvatar(value)"
      alt="图片预览"
      class="h-16 w-16 rounded object-cover"
    />
  </div>
</template>
