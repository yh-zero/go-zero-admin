<script setup lang="ts">
import { useRouter } from 'vue-router';

import { Button } from 'antdv-next';

import { SESSION_HOME } from '#/adapter/business/session';
import { useAuthStore } from '#/store';
const router = useRouter();
const authStore = useAuthStore();
</script>
<template>
  <div class="flex min-h-screen flex-col items-center justify-center gap-5 p-6">
    <h1 class="text-xl font-semibold">暂时无法加载菜单</h1>
    <p>请检查网络，或联系管理员确认菜单接口权限。</p>
    <div class="flex gap-4">
      <Button type="primary" @click="router.replace(SESSION_HOME)">重试</Button><Button @click="authStore.logout(false)">返回登录</Button>
    </div>
  </div>
</template>
