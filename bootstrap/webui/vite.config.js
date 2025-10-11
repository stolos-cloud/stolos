import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

export default defineConfig({
    plugins: [vue()],
    build: {
        outDir: resolve(__dirname, '../out/web/webui-dist'),
    },
    server: {
        proxy: {
            '/api': 'http://localhost:9123'
        }
    }
})
