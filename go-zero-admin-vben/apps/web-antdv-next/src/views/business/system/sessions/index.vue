<script setup lang="ts">
import { Page } from '@vben/common-ui';

import DeviceSessions from '#/components/business/device-sessions.vue';
</script>
<template>
  <Page
    title="在线会话"
    description="查看有效登录设备，按需撤销指定设备的访问。"
  >
    <DeviceSessions admin />
  </Page>
</template>
