import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import fs from 'fs'
import yaml from 'js-yaml'
import path from 'path'

const configPath = path.resolve(__dirname, '../config.yaml')
let serverPort = 8080
try {
  const config = yaml.load(fs.readFileSync(configPath, 'utf8'))
  serverPort = config?.server?.port || 8080
} catch (e) {
  console.warn('Failed to read config.yaml, using default port 8080')
}

export default defineConfig({
  plugins: [vue()],
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: `http://localhost:${serverPort}`,
        changeOrigin: true,
      }
    }
  },
  build: {
    outDir: 'dist',
    assetsDir: 'static',
  }
})
