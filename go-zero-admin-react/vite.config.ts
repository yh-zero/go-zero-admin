import { defineConfig, loadEnv } from 'vite';
import react from '@vitejs/plugin-react';
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), 'VITE_');
  return {
    plugins: [react()],
    build: {
      rollupOptions: {
        output: {
          // antd 图标与样式运行时拆独立 chunk：antd v6 的 CSS-in-JS 运行时约 700KB，
          // 不应跟随公共组件 chunk 加载到所有页面。
          manualChunks(id: string) {
            if (id.includes('node_modules')) {
              if (id.includes('@ant-design/icons')) return 'antd-icons';
              if (id.includes('node_modules/antd/') || id.includes('node_modules/rc-') || id.includes('node_modules/@rc-component')) return 'antd-core';
              if (id.includes('node_modules/react/') || id.includes('node_modules/react-dom/') || id.includes('node_modules/scheduler/')) return 'react-vendor';
              if (id.includes('node_modules/@tanstack/')) return 'query-vendor';
              if (id.includes('node_modules/zustand/') || id.includes('node_modules/axios/') || id.includes('node_modules/dayjs/')) return 'app-vendor';
            }
            return undefined;
          },
        },
      },
    },
    server: {
      host: '127.0.0.1', port: 5999, strictPort: true,
      proxy: {
        '/v1': { target: env.VITE_API_PROXY_TARGET || 'http://127.0.0.1:7001', changeOrigin: true },
        '/ping': { target: env.VITE_API_PROXY_TARGET || 'http://127.0.0.1:7001', changeOrigin: true },
      },
    },
  };
});
