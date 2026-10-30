import { defineConfig, loadEnv } from 'vite';
import react from '@vitejs/plugin-react';
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), 'VITE_');
  return {
    plugins: [react()],
    server: {
      host: '127.0.0.1', port: 5999, strictPort: true,
      proxy: {
        '/v1': { target: env.VITE_API_PROXY_TARGET || 'http://127.0.0.1:7001', changeOrigin: true },
        '/ping': { target: env.VITE_API_PROXY_TARGET || 'http://127.0.0.1:7001', changeOrigin: true },
      },
    },
  };
});
