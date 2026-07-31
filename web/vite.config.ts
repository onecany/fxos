import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { HttpProxyAgent } from 'hpagent'

const proxyUrl = process.env.HTTP_PROXY || process.env.http_proxy
const backendUrl = process.env.FXOS_BACKEND_URL || 'http://localhost:8080'

export default defineConfig({
  plugins: [react()],
  server: {
    host: '0.0.0.0',
    port: 3000,
    proxy: {
      '/api': {
        target: backendUrl,
        changeOrigin: true,
        ...(proxyUrl
          ? { agent: new HttpProxyAgent({ proxy: proxyUrl }) }
          : {}),
      },
    },
  },
})
