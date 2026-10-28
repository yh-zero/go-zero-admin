<script setup lang="ts">
import { Page } from '@vben/common-ui';
import { useUserStore } from '@vben/stores';
const userStore = useUserStore();
</script>
<template>
  <Page title="首页">
    <div class="bg-card rounded-xl p-6">
      <h2 class="text-xl font-semibold">
        欢迎，{{ userStore.userInfo?.realName }}
      </h2>
      <p class="text-muted-foreground mt-3">
        登录成功。左侧菜单由当前账号的后台授权生成。
      </p>
    </div>
  </Page>
</template>
