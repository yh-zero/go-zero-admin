<script setup lang="ts">
import { useRoute } from 'vue-router';

import { Page } from '@vben/common-ui';
const route = useRoute();
</script>
<template>
  <Page :title="String(route.meta.title)">
    <div class="bg-card rounded-xl p-6">
      尚未接入
    </div>
  </Page>
</template>
