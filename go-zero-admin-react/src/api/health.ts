import { request } from '../utils/request';
export async function fetchHealth(): Promise<boolean> {
  const base = import.meta.env.VITE_API_BASE_URL || '/v1';
  const origin = /^https?:\/\//.test(base) ? new URL(base).origin : '';
  await request.get('/ping', { baseURL: origin });
  return true;
}
