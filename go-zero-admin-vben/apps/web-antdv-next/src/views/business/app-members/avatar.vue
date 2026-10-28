<script setup lang="ts">
import type { AppUser } from '#/api/business/app-members';

import { ref } from 'vue';

import { Alert, Button, message, Modal } from 'antdv-next';

import { updateAppUserAvatar } from '#/api/business/app-members';
import ImageUpload from '#/components/business/image-upload.vue';

import { usePermission } from '../system/shared';
import { useRequestScope } from './shared';

const emit = defineEmits<{ saved: [] }>();
const can = usePermission('app-members-users');
const canTools = usePermission('businessTools');
const open = ref(false);
const saving = ref(false);
const uploading = ref(false);
const target = ref<AppUser>();
const avatarUrl = ref('');
const error = ref('');
const version = ref(0);
const scope = useRequestScope(() => {
  version.value++;
  open.value = false;
  saving.value = false;
  uploading.value = false;
  target.value = undefined;
  avatarUrl.value = '';
  error.value = '';
});
let request = scope.capture();

function show(user: AppUser) {
  if (saving.value || uploading.value || !can('avatar')) return;
  version.value++;
  request = scope.capture();
  target.value = user;
  avatarUrl.value = user.avatarUrl || '';
  error.value = '';
  open.value = true;
}
function close() {
  if (saving.value || uploading.value) return;
  version.value++;
  open.value = false;
}
async function save() {
  const user = target.value;
  const currentRequest = request;
  const currentVersion = version.value;
  if (
    !user ||
    !open.value ||
    saving.value ||
    uploading.value ||
    !can('avatar') ||
    !scope.current(currentRequest)
  )
    return;
  saving.value = true;
  error.value = '';
  try {
    await updateAppUserAvatar(
      { userId: user.id, avatarUrl: avatarUrl.value.trim() },
      currentRequest.token,
    );
    if (!scope.current(currentRequest) || currentVersion !== version.value)
      return;
    message.success('App 用户头像已保存');
    open.value = false;
    emit('saved');
  } catch {
    if (scope.current(currentRequest) && currentVersion === version.value)
      error.value = '头像保存失败，已保留填写内容，请重试。';
  } finally {
    if (scope.current(currentRequest) && currentVersion === version.value)
      saving.value = false;
  }
}
defineExpose({ show });
</script>

<template>
  <Modal
    v-model:open="open"
    :title="`${target?.account ?? ''} 的头像`"
    :confirm-loading="saving"
    :mask-closable="false"
    :closable="!saving && !uploading"
    :keyboard="!saving && !uploading"
    :cancel-button-props="{ disabled: saving || uploading }"
    :ok-button-props="{ disabled: uploading || !can('avatar') }"
    ok-text="保存头像"
    @ok="save"
    @cancel="close"
  >
    <ImageUpload
      v-if="open"
      :key="version"
      v-model:value="avatarUrl"
      :disabled="saving || !can('avatar')"
      :can-upload="canTools('upload') && can('avatar')"
      @uploading-change="uploading = $event"
    />
    <Button
      class="mt-3"
      :disabled="saving || uploading"
      @click="avatarUrl = ''"
    >
      清空头像
    </Button>
    <p class="text-muted-foreground my-3 text-sm">
      本系统上传的头像保存后自动关联文件，替换或清空自动解除旧引用。上传新图片需管理工具的上传图片权限。
    </p>
    <Alert v-if="error" type="error" show-icon :title="error" />
  </Modal>
</template>
