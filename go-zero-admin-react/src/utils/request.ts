import axios from 'axios';

/**
 * HTTP 封装（骨架）
 *
 * 后端约定：HTTP 200 + body { code, msg, result }
 * - code === 200 视为成功
 * - 401 视为登录过期
 *
 * 联调时在此处补：错误提示、401 跳登录、token 刷新等
 */
export const request = axios.create({
  baseURL: '/v1',
  timeout: 30000,
});

// 请求拦截：注入 JWT（联调时启用）
// request.interceptors.request.use((config) => {
//   const token = localStorage.getItem('go-zero-auth-token');
//   if (token) config.headers.Authorization = 'Bearer ' + token;
//   return config;
// });

// 响应拦截：统一处理 { code, msg }（联调时启用）
// request.interceptors.response.use(
//   (resp) => (resp.data?.code === 200 ? resp.data : Promise.reject(resp.data)),
//   (error) => Promise.reject(error),
// );
