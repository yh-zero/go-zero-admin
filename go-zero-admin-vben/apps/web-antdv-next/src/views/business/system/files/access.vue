<script setup lang="ts">
import type { FileResource, FileURL } from '#/api/business/system/files';

import { onBeforeUnmount, ref } from 'vue';

import { Alert, Button, message, Modal, Space, Spin } from 'antdv-next';

import { getFileURL } from '#/api/business/system/files';

import { auditTime } from '../audit/format';
import { fileError, safeFileURL } from './format';

const open = ref(false);
const loading = ref(false);
const downloading = ref(false);
const file = ref<FileResource>();
const link = ref('');
const expiresAt = ref('');
const errorMessage = ref('');
let generation = 0;
let expiryTimer: number | undefined;
let activeDownload: AbortController | undefined;
let downloadTimeout: number | undefined;

function clearLink() {
  window.clearTimeout(expiryTimer);
  expiryTimer = undefined;
  link.value = '';
  expiresAt.value = '';
}
function setLink(result: FileURL) {
  clearLink();
  link.value = safeFileURL(result.url);
  if (result.expiresIn > 0) {
    expiresAt.value = new Date(
      Date.now() + result.expiresIn * 1000,
    ).toISOString();
    expiryTimer = window.setTimeout(() => {
      clearLink();
      errorMessage.value = '私有地址已过期，请重新获取访问地址';
    }, result.expiresIn * 1000);
  }
}
function close() {
  generation++;
  activeDownload?.abort();
  activeDownload = undefined;
  window.clearTimeout(downloadTimeout);
  downloadTimeout = undefined;
  loading.value = false;
  downloading.value = false;
  clearLink();
}
onBeforeUnmount(close);

async function refresh() {
  if (!file.value || loading.value || downloading.value) return;
  const current = ++generation;
  loading.value = true;
  errorMessage.value = '';
  clearLink();
  try {
    const result = await getFileURL(file.value.id);
    if (current !== generation || !open.value) return;
    setLink(result);
  } catch (error) {
    if (current === generation)
      errorMessage.value = fileError(error, '文件访问地址加载失败，可重新获取');
  } finally {
    if (current === generation) loading.value = false;
  }
}
async function show(row: FileResource) {
  if (downloading.value) return;
  file.value = row;
  loading.value = false;
  open.value = true;
  await refresh();
}
async function download() {
  if (!file.value || loading.value || downloading.value) return;
  const current = generation;
  const selectedFile = file.value;
  downloading.value = true;
  errorMessage.value = '';
  const controller = new AbortController();
  activeDownload = controller;
  const timeout = window.setTimeout(() => controller.abort(), 15_000);
  downloadTimeout = timeout;
  try {
    // Refresh the authorization for every download and keep the application JWT
    // out of external storage requests. Private access uses the signed URL only.
    const result = await getFileURL(selectedFile.id);
    if (current !== generation) return;
    setLink(result);
    const response = await fetch(link.value, {
      credentials: 'omit',
      referrerPolicy: 'no-referrer',
      signal: controller.signal,
    });
    if (current !== generation) return;
    if (!response.ok) throw new Error('文件读取失败，请重新获取访问地址');
    const blob = await response.blob();
    if (current !== generation) return;
    const objectURL = URL.createObjectURL(blob);
    const anchor = document.createElement('a');
    anchor.href = objectURL;
    anchor.download = selectedFile.name;
    anchor.click();
    window.setTimeout(() => URL.revokeObjectURL(objectURL), 1000);
    message.success('文件下载已开始');
  } catch (error) {
    if (current !== generation) return;
    clearLink();
    const fallback =
      '下载未完成。若存储服务未允许跨域下载，可重新获取地址后打开图片，并使用浏览器另存为。';
    errorMessage.value = (error as { response?: unknown })?.response
      ? fileError(error, fallback)
      : fallback;
  } finally {
    window.clearTimeout(timeout);
    if (activeDownload === controller) {
      activeDownload = undefined;
      downloadTimeout = undefined;
    }
    if (current === generation) downloading.value = false;
  }
}
defineExpose({ show });
</script>

<template>
  <Modal
    v-model:open="open"
    :title="file?.name ?? '文件预览'"
    :footer="null"
    :width="780"
    :mask-closable="false"
    :closable="!downloading"
    @cancel="close"
  >
    <div class="space-y-4">
      <Space wrap>
        <Button :loading="loading" :disabled="downloading" @click="refresh">
          重新获取地址
        </Button>
        <Button
          :disabled="loading || !link"
          :loading="downloading"
          @click="download"
        >
          下载
        </Button>
        <a v-if="link" :href="link" target="_blank" rel="noopener noreferrer">
          打开图片
        </a>
      </Space>
      <p v-if="expiresAt" class="text-muted-foreground text-sm">
        私有地址有效至 {{ auditTime(expiresAt) }}，过期后请重新获取。
      </p>
      <p v-else-if="link" class="text-muted-foreground text-sm">
        此文件使用公开地址。
      </p>
      <Alert
        v-if="errorMessage"
        show-icon
        type="error"
        :message="errorMessage"
      />
      <Spin :spinning="loading">
        <img
          v-if="link"
          :key="link"
          :src="link"
          :alt="file?.name"
          referrerpolicy="no-referrer"
          class="mx-auto max-h-[60vh] max-w-full object-contain"
          @error="
            errorMessage =
              '图片未能加载，地址可能已过期或存储服务暂不可用，请重新获取'
          "
        />
      </Spin>
    </div>
  </Modal>
</template>
