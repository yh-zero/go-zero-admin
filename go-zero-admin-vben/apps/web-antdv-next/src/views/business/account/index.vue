<script setup lang="ts">
import type { BackendUser } from '#/api/business/session';

import { onMounted, ref } from 'vue';

import { Page } from '@vben/common-ui';
import { useAccessStore } from '@vben/stores';

import {
  Button,
  Card,
  Descriptions,
  DescriptionsItem,
  message,
  Spin,
} from 'antdv-next';

import {
  isValidPassword,
  passwordValidationMessage,
} from '#/adapter/business/password';
import { useVbenForm, z } from '#/adapter/form';
import { changeMyPassword, getCurrentUser } from '#/api/business/account';
import DeviceSessions from '#/components/business/device-sessions.vue';
import { useAuthStore } from '#/store';

const user = ref<BackendUser>();
const loading = ref(false);
const saving = ref(false);
const auth = useAuthStore();
const access = useAccessStore();
const [PasswordForm, form] = useVbenForm({
  showDefaultActions: false,
  wrapperClass: 'grid-cols-1',
  schema: [
    {
      fieldName: 'oldPassword',
      label: '原密码',
      component: 'InputPassword',
      rules: 'required',
    },
    {
      fieldName: 'newPassword',
      label: '新密码',
      component: 'InputPassword',
      rules: z.string().refine(isValidPassword, passwordValidationMessage),
    },
    {
      fieldName: 'confirmPassword',
      label: '确认密码',
      component: 'InputPassword',
      rules: 'required',
    },
  ],
});
async function load() {
  loading.value = true;
  try {
    user.value = await getCurrentUser();
  } finally {
    loading.value = false;
  }
}
async function savePassword() {
  if (saving.value) return;
  saving.value = true;
  const token = access.accessToken;
  try {
    const validation = await form.validate();
    if (!validation.valid) return;
    const values = await form.getValues();
    if (values.newPassword !== values.confirmPassword) {
      message.error('两次新密码不一致');
      return;
    }
    if (access.accessToken !== token) return;
    await changeMyPassword({
      oldPassword: values.oldPassword,
      newPassword: values.newPassword,
    });
    if (access.accessToken !== token) return;
    await form.reset();
    if (access.accessToken !== token) return;
    message.success('密码已修改，请重新登录');
    await auth.logout(false, false);
  } finally {
    saving.value = false;
  }
}
onMounted(load);
</script>

<template>
  <Page>
    <div class="grid gap-4 lg:grid-cols-2">
      <Card title="账号信息">
        <template #extra>
          <Button :loading="loading" @click="load">刷新</Button>
        </template>
        <Spin :spinning="loading">
          <Descriptions v-if="user" :column="1" bordered>
            <DescriptionsItem label="用户名">
              {{ user.userName }}
            </DescriptionsItem>
            <DescriptionsItem label="昵称">
              {{ user.nickName || '—' }}
            </DescriptionsItem>
            <DescriptionsItem label="当前角色">
              {{ user.authority?.authorityName || user.authorityId }}
            </DescriptionsItem>
            <DescriptionsItem label="手机号">
              {{ user.phone || '—' }}
            </DescriptionsItem>
            <DescriptionsItem label="邮箱">
              {{ user.email || '—' }}
            </DescriptionsItem>
          </Descriptions>
        </Spin>
      </Card>
      <Card title="修改密码">
        <p class="text-muted-foreground mb-4">
          修改后，当前账号在所有设备的登录都会失效。
        </p>
        <PasswordForm />
        <div class="mt-4 text-right">
          <Button type="primary" :loading="saving" @click="savePassword">
            修改密码并退出
          </Button>
        </div>
      </Card>
    </div>
    <Card title="我的登录设备" class="mt-4">
      <DeviceSessions />
    </Card>
  </Page>
</template>
