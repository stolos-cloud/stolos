import Components from 'unplugin-vue-components/vite';
import Vue from '@vitejs/plugin-vue';
import { transformAssetUrls } from 'vite-plugin-vuetify';
import { defineConfig, loadEnv } from 'vite';
import { fileURLToPath, URL } from 'node:url';

const env = loadEnv(process.env.NODE_ENV, process.cwd(), '');
export default defineConfig({
    resolve: {
        alias: {
            '@': fileURLToPath(new URL('./src', import.meta.url)),
        },
        extensions: ['.js', '.json', '.jsx', '.mjs', '.ts', '.tsx', '.vue'],
    },
    plugins: [
        Vue({
            template: { transformAssetUrls },
        }),
        Components(),
    ],
    server: {
        port: 3000,
        proxy: {
            '/api': {
                target: env.VITE_API_BASE_URL,
                changeOrigin: true,
                rewrite: path => path.replace(/^\/api/, '/api/v1'),
                ws: true,
            },
        },
    },
});
