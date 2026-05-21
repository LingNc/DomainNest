import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import fs from 'fs'
import yaml from 'js-yaml'
import path from 'path'

const configPath = path.resolve(__dirname, '../config.yaml')
let serverPort = 8080
let frontendPort = 3000
let allowedHosts = []
try {
  const config = yaml.load(fs.readFileSync(configPath, 'utf8'))
  serverPort = config?.server?.port || 8080
  frontendPort = config?.server?.frontend_port || 3000
  allowedHosts = config?.server?.allowed_hosts || []
} catch (e) {
  console.warn('Failed to read config.yaml, using default ports')
}

export default defineConfig({
  plugins: [vue()],
  server: {
    port: frontendPort,
    allowedHosts,
    hmr: {
      overlay: false,
      timeout: 60000,
    },
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
