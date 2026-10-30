import dayjs from 'dayjs';
export const formatTime = (seconds: number) => seconds > 0 ? dayjs.unix(seconds).format('YYYY-MM-DD HH:mm:ss') : '—';
export function formatBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes < 0) return '—';
  const units = ['B', 'KB', 'MB', 'GB'];
  const index = bytes > 0 ? Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1) : 0;
  return (bytes / 1024 ** index).toFixed(index ? 1 : 0) + ' ' + units[index];
}
