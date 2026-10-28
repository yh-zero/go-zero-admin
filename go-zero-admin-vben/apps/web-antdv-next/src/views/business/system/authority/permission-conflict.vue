<script setup lang="ts">
import type { PermissionEdit } from '#/api/business/system/permissions';

import { Alert, Button } from 'antdv-next';
defineProps<{ server?: PermissionEdit; local: unknown }>();
defineEmits<{ rebase: [] }>();
</script>
<template>
  <Alert
    v-if="server"
    type="warning"
    show-icon
    message="权限已变更，本地编辑已保留。请比较服务端与本地，再明确确认以最新版本保存。"
    class="mb-4"
  >
    <template #description>
      <details>
        <summary>比较服务端与本地内容（版本 {{ server.revision }}）</summary>
        <div class="grid grid-cols-2 gap-3">
          <pre class="overflow-auto">
服务端：{{
              JSON.stringify(
                {
                  menuIds: server.menuIds,
                  menuBtnIds: server.menuBtnIds,
                  policies: server.policies,
                  dataScope: server.dataScope,
                },
                null,
                2,
              )
            }}</pre>
          <pre class="overflow-auto">
本地：{{ JSON.stringify(local, null, 2) }}</pre>
        </div>
      </details>
      <Button @click="$emit('rebase')">
        已比较，使用服务端最新版本继续编辑
      </Button>
    </template>
  </Alert>
</template>
