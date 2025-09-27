import Components from 'unplugin-vue-components/vite'
import Vue from '@vitejs/plugin-vue'
import { transformAssetUrls } from 'vite-plugin-vuetify'
import { defineConfig } from 'vite'
import { fileURLToPath, URL } from 'node:url'

export default defineConfig({
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    },
    extensions: ['.js', '.json', '.jsx', '.mjs', '.ts', '.tsx', '.vue'],
  },
  plugins: [
    Vue({
      template: { transformAssetUrls }
    }),
    Components()
  ],
  define: { 'process.env': {} },
  server: {
    proxy: {
      '/api': {
        target: process.env.VITE_API_BASE_URL || 'http://localhost:8000',
        changeOrigin: true,
        pathRewrite: {
          '^/api': ''
        },
      }
    },
    port: 3000,
  },
})
