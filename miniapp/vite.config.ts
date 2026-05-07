import path from 'node:path'
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    allowedHosts: [
      '.ngrok-free.app',
      '.ngrok-free.dev',
      '.ngrok.app',
      '.ngrok.io',
      '.trycloudflare.com',
      'localhost',
      '127.0.0.1',
    ],
    proxy: {
      '/api': {
        target: 'https://api.zaruchevskiy.ru',
        changeOrigin: true,
        secure: true,
      },
      '/health': {
        target: 'https://api.zaruchevskiy.ru',
        changeOrigin: true,
        secure: true,
      },
    },
  },
})
