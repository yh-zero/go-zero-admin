const imageMimeTypes = new Set([
  'image/gif',
  'image/jpeg',
  'image/png',
  'image/webp',
]);

export const IMAGE_UPLOAD_ACCEPT = [...imageMimeTypes].join(',');
export const MAX_IMAGE_SIZE = 10 * 1024 * 1024;

export function validateImageUpload(file: File): void {
  if (!imageMimeTypes.has(file.type))
    throw new Error('请选择 JPG、PNG、GIF 或 WebP 图片');
  if (file.size <= 0 || file.size > MAX_IMAGE_SIZE)
    throw new Error('图片不能为空且不能超过 10 MB');
  if ([...file.name].length > 255 || !file.name.trim())
    throw new Error('文件名不能为空且不能超过 255 个字符');
}
