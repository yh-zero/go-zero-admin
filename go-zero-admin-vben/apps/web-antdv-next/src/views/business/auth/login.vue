<script setup lang="ts">
import type { VbenFormSchema } from '@vben/common-ui';
import type { Recordable } from '@vben/types';

import { computed, markRaw, onMounted, ref } from 'vue';

import { AuthenticationLogin, z } from '@vben/common-ui';

import { getCaptchaApi } from '#/api/core/auth';
import ImageCaptcha from '#/components/business/image-captcha.vue';
import { useAuthStore } from '#/store';

defineOptions({ name: 'BusinessLogin' });
const authStore = useAuthStore();
const loginRef = ref<InstanceType<typeof AuthenticationLogin>>();
const image = ref('');
const captchaId = ref('');
const captchaLoading = ref(false);

async function refreshCaptcha() {
  if (captchaLoading.value) return;
  captchaLoading.value = true;
  image.value = '';
  captchaId.value = '';
  loginRef.value?.getFormApi().setFieldValue('captcha', '');
  try {
    const captcha = await getCaptchaApi();
    image.value = captcha.captchaImg;
    captchaId.value = captcha.captchaId;
  } catch {
    // The request client displays the error; the image button remains retryable.
  } finally {
    captchaLoading.value = false;
  }
}

const formSchema = computed((): VbenFormSchema[] => [
  {
    component: 'VbenInput',
    componentProps: { autocomplete: 'username', placeholder: '账号' },
    fieldName: 'username',
    label: '账号',
    rules: z.string().min(1, '请输入账号'),
  },
  {
    component: 'VbenInputPassword',
    componentProps: { autocomplete: 'current-password', placeholder: '密码' },
    fieldName: 'password',
    label: '密码',
    rules: z.string().min(1, '请输入密码'),
  },
  {
    component: markRaw(ImageCaptcha),
    componentProps: {
      disabled: authStore.loginLoading,
      image: image.value,
      loading: captchaLoading.value,
      onRefresh: refreshCaptcha,
    },
    fieldName: 'captcha',
    label: '图片验证码',
    modelPropName: 'value',
    rules: z.string().regex(/^\d{6}$/, '请输入6位图片验证码'),
  },
]);

async function submit(values: Recordable<any>) {
  if (
    captchaLoading.value ||
    !image.value ||
    !captchaId.value ||
    authStore.loginLoading
  )
    return;
  try {
    await authStore.authLogin({ ...values, captchaId: captchaId.value });
  } catch {
    await refreshCaptcha();
  }
}
onMounted(refreshCaptcha);
</script>

<template>
  <AuthenticationLogin
    ref="loginRef"
    :form-schema="formSchema"
    :loading="authStore.loginLoading || captchaLoading"
    :show-code-login="false"
    :show-forget-password="false"
    :show-qrcode-login="false"
    :show-register="false"
    :show-third-party-login="false"
    :show-remember-me="false"
    title="登录管理后台"
    sub-title="使用后端账号登录"
    @submit="submit"
  />
</template>
