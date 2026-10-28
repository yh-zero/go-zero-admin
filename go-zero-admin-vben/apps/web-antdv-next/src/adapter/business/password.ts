/** Match the server's bcrypt input policy in UTF-8 bytes, not UTF-16 length. */
export const passwordValidationMessage =
  '密码须为8至72字节，中文和表情会占用多个字节';
export function passwordByteLength(value: string) {
  return new TextEncoder().encode(value).length;
}
export function isValidPassword(value: string) {
  const length = passwordByteLength(value);
  return length >= 8 && length <= 72;
}
