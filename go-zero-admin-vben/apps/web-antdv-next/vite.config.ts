import { defineConfig } from '@vben/vite-config';

export default defineConfig(async () => {
  return {
    application: {},
    vite: {
      server: {
        proxy: {
          '/api': {
            changeOrigin: true,
            rewrite: (path) => path.replace(/^\/api/, ''),
            // 本地 Go 后端，验证码和登录共用代理
            target: 'http://127.0.0.1:7001',
            ws: true,
          },
        },
      },
    },
  };
});
