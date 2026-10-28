<script setup lang="ts">
import { ref } from 'vue';

import { Page } from '@vben/common-ui';

import { Alert, Button, Card, message } from 'antdv-next';

import { useVbenForm, z } from '#/adapter/form';
import { sendEmailCode } from '#/api/business/base';
import ImageUpload from '#/components/business/image-upload.vue';

import { usePermission } from '../system/shared';
const can = usePermission('businessTools');
const image = ref('');
const sending = ref(false);
const [Form, form] = useVbenForm({
  showDefaultActions: false,
  wrapperClass: 'grid-cols-1',
  schema: [
    {
      fieldName: 'email',
      label: '收件邮箱',
      component: 'Input',
      rules: z.string().email('请输入有效邮箱'),
    },
  ],
});
async function send() {
  if (sending.value) return;
  sending.value = true;
  try {
    if (!(await form.validate()).valid) return;
    const values = await form.getValues();
    await sendEmailCode(values.email);
    message.success('验证码已发送，请检查收件箱');
  } finally {
    sending.value = false;
  }
}
</script>
<template>
  <Page>
    <Alert
      type="info"
      show-icon
      message="外部服务未配置时会显示后端失败原因；发送验证码不代表已实现邮箱绑定或找回密码。"
      class="mb-4"
    />
    <div class="grid gap-4 lg:grid-cols-2">
      <Card title="图片上传">
        <ImageUpload
          v-model:value="image"
          :can-upload="can('upload')"
          :disabled="!can('upload')"
        />
        <p class="mt-3 text-sm">支持 JPG、PNG、GIF、WebP，最大 10 MB。</p>
      </Card>
      <Card title="邮箱验证码">
        <Form />
        <Button
          v-if="can('sendEmail')"
          type="primary"
          :loading="sending"
          @click="send"
        >
          发送验证码
        </Button>
        <p class="mt-3 text-sm">按后端频率限制发送，不强制重复发送。</p>
      </Card>
    </div>
  </Page>
</template>
