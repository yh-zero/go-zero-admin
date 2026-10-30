import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5999,
    proxy: {
      '/v1': { target: 'http://127.0.0.1:7001', changeOrigin: true },
    },
  },
});
