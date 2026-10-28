export function fileSize(size: number): string {
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
  return `${(size / (1024 * 1024)).toFixed(1)} MB`;
}
export function safeFileURL(value: string): string {
  const url = new URL(value);
  if (!['http:', 'https:'].includes(url.protocol))
    throw new Error('文件访问地址无效，请联系管理员核查');
  return url.href;
}
export function fileError(error: unknown, fallback: string): string {
  const response = (error as { response?: { data?: { message?: unknown } } })
    ?.response;
  const message = response?.data?.message;
  if (typeof message === 'string' && message) return message;
  return error instanceof Error && error.message ? error.message : fallback;
}
export { validateImageUpload as validateUpload } from '#/adapter/business/image-upload';
