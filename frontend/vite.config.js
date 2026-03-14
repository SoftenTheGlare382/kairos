import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/account': {
        target: 'http://localhost:8081',
        changeOrigin: true,
      },
      '/video': {
        target: 'http://localhost:8082',
        changeOrigin: true,
      },
      '/social': {
        target: 'http://localhost:8083',
        changeOrigin: true,
      },
      '/feed': {
        target: 'http://localhost:8084',
        changeOrigin: true,
      },
      '/im': {
        target: 'http://localhost:8085',
        changeOrigin: true,
        ws: true,
      },
      '/comment': {
        target: 'http://localhost:8082', // Assuming comments are in video service
        changeOrigin: true,
      },
      '/like': {
        target: 'http://localhost:8082', // Assuming likes are in video service
        changeOrigin: true,
      },
    },
  },
});
