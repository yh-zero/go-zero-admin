export type CaptchaSource = { kind: 'image' | 'text' | 'invalid'; value: string };
export function captchaSource(encoded: string): CaptchaSource {
  const value = encoded.trim();
  const invalid: CaptchaSource = { kind: 'invalid', value: '' };
  if (!value || value.length > 2_000_000) return invalid;
  if (/^data:image\/(png|jpe?g|gif|webp);base64,[A-Za-z0-9+/=\s]+$/.test(value)) return { kind: 'image', value };
  if (value.startsWith('data:')) return invalid;
  try {
    const decoded = atob(value);
    if (/^\d{4,8}$/.test(decoded)) return { kind: 'text', value: decoded };
    const mime = value.startsWith('iVBOR') ? 'png' : value.startsWith('/9j/') ? 'jpeg' : value.startsWith('R0lGOD') ? 'gif' : value.startsWith('UklGR') ? 'webp' : '';
    return mime ? { kind: 'image', value: 'data:image/' + mime + ';base64,' + value } : invalid;
  } catch { return invalid; }
}
