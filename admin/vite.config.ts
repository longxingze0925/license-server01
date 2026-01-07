import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  build: {
    rollupOptions: {
      output: {
        manualChunks: {
          // 将 React 相关库分离
          'vendor-react': ['react', 'react-dom', 'react-router-dom'],
          // 将 Ant Design 分离
          'vendor-antd': ['antd', '@ant-design/icons'],
          // 将图表库分离
          'vendor-charts': ['recharts'],
          // 将工具库分离
          'vendor-utils': ['axios', 'zustand', 'dayjs'],
        },
      },
    },
    chunkSizeWarningLimit: 1000,
  },
  server: {
    host: '127.0.0.1',
    port: 3000,
    proxy: {
      '/api': {
        target: 'http://localhost:8081',
        changeOrigin: true,
      },
    },
  },
})
