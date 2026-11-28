import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import fs from 'fs'
import path from 'path'

const pkg = JSON.parse(fs.readFileSync(path.resolve(__dirname, 'package.json'), 'utf-8'))

export default defineConfig({
  base: './',
  plugins: [vue()],
  define: {
    __CUBEX_PKG_VERSION__: JSON.stringify(`v${pkg.version}`),
  },
  build: {
    outDir: '../cmd/cubexd/dist',
    emptyOutDir: true,
  },
  server: {
    port: 5173,
    proxy: {
      '/api': 'http://127.0.0.1:2026',
      '/sub': 'http://127.0.0.1:2026',
    }
  }
})
