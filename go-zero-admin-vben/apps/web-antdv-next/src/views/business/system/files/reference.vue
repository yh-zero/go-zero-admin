<script setup lang="ts">
import type { FileResource } from '#/api/business/system/files';

import { ref } from 'vue';

import { Alert, message, Modal } from 'antdv-next';

import { useVbenForm, z } from '#/adapter/form';
import {
  addFileReference,
  removeFileReference,
} from '#/api/business/system/files';

import { fileError } from './format';

const emit = defineEmits<{ saved: [] }>();
const open = ref(false);
const busy = ref(false);
const file = ref<FileResource>();
const errorMessage = ref('');
const [Form, form] = useVbenForm({
  wrapperClass: 'grid-cols-1',
  showDefaultActions: false,
  schema: [
    {
      fieldName: 'operation',
      label: '操作',
      component: 'Select',
      defaultValue: 'add',
      componentProps: {
        options: [
          { label: '登记引用', value: 'add' },
          { label: '移除引用', value: 'remove' },
        ],
      },
      rules: 'required',
    },
    {
      fieldName: 'objectType',
      label: '业务类型',
      component: 'Input',
      componentProps: { placeholder: '例如 article，使用英文业务标识' },
      rules: z
        .string()
        .regex(
          /^[A-Za-z][\w-]{0,63}$/,
          '以字母开头，最多64个字母、数字、下划线或连字符',
        ),
    },
    {
      fieldName: 'objectId',
      label: '业务对象标识',
      component: 'Input',
      componentProps: { placeholder: '已有对象的 ID 或 UUID' },
      rules: z
        .string()
        .regex(
          /^[A-Za-z0-9][\w.:-]{0,127}$/,
          '填写最多128个字母、数字、下划线、点、冒号或连字符',
        ),
    },
  ],
});
async function show(row: FileResource) {
  if (busy.value) return;
  file.value = row;
  errorMessage.value = '';
  open.value = true;
  await form.reset();
  await form.setValues({ operation: 'add', objectType: '', objectId: '' });
}
async function save() {
  if (busy.value || !file.value) return;
  busy.value = true;
  errorMessage.value = '';
  try {
    const validation = await form.validate();
    if (!validation.valid) return;
    const values = await form.getValues();
    if (['app_user_avatar', 'sys_user_avatar'].includes(values.objectType)) {
      errorMessage.value =
        '系统头像引用由头像资料自动维护，不能在此手工登记或解除';
      return;
    }
    const data = {
      fileId: file.value.id,
      objectType: values.objectType,
      objectId: values.objectId,
    };
    await (values.operation === 'remove'
      ? removeFileReference(data)
      : addFileReference(data));
    message.success(
      values.operation === 'remove' ? '引用已移除' : '引用已登记',
    );
    open.value = false;
    emit('saved');
  } catch (error) {
    errorMessage.value = fileError(error, '引用更新失败，请核对标识后重试');
  } finally {
    busy.value = false;
  }
}
defineExpose({ show });
</script>

<template>
  <Modal
    v-model:open="open"
    :title="`维护引用 · ${file?.name ?? ''}`"
    :force-render="true"
    :mask-closable="false"
    :closable="!busy"
    :confirm-loading="busy"
    :cancel-button-props="{ disabled: busy }"
    @ok="save"
  >
    <Alert
      type="info"
      show-icon
      class="mb-4"
      message="后台用户和 App 用户头像引用由系统自动维护，不能在此手工登记或解除。其他业务引用由管理员核实后维护，移除前须确认业务已停止使用。"
    />
    <div :inert="busy"><Form /></div>
    <Alert v-if="errorMessage" type="error" show-icon :message="errorMessage" />
  </Modal>
</template>
