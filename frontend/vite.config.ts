import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  build: {
    // Build straight into the Go module so go:embed can bundle the UI
    // into the single kibo binary (embed cannot reach outside backend/)
    outDir: '../backend/webui/dist',
    emptyOutDir: true,
  },
  server: {
    // In dev, the app calls the same-origin /api and Vite forwards it
    // to the Go backend — same URLs in dev and in the built binary
    proxy: {
      '/api': 'http://localhost:8080',
    },
  },
})
