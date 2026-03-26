import { fileURLToPath, URL } from 'node:url'

import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
import vueJsx from '@vitejs/plugin-vue-jsx'
import vueDevTools from 'vite-plugin-vue-devtools'

// https://vite.dev/config/
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const enableDevtools = env.VITE_DEVTOOLS === 'true'
    return {
    plugins: [
      vue(),
      vueJsx(),
      enableDevtools && vueDevTools(),
    ].filter(Boolean) as any,
    resolve: {
      alias: {
        '@': fileURLToPath(new URL('./src', import.meta.url))
      },
    },
    server: {
      proxy: {
        '/api': {
          target: 'http://127.0.0.1:8081',
          changeOrigin: true,
        },
      },
    },
    build: {
      rollupOptions: {
        output: {
          manualChunks(id) {
            if (!id.includes('node_modules')) return undefined
            if (id.includes('echarts') || id.includes('zrender') || id.includes('vue-echarts')) return 'chart-vendor'
            if (id.includes('element-plus') || id.includes('@element-plus')) return 'ui-vendor'
            if (id.includes('vue') || id.includes('pinia') || id.includes('vue-router')) return 'core-vendor'
            return 'vendor'
          },
        },
      },
    },
  }
})
