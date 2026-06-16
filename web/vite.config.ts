import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    proxy: {
      '/api': 'http://localhost:8080',
    },
  },
  build: {
    rollupOptions: {
      output: {
        manualChunks: {
          'vendor-markdown': ['react-markdown', 'remark-gfm', 'remark-breaks'],
          'vendor-dnd': ['@dnd-kit/core', '@dnd-kit/sortable', '@dnd-kit/utilities'],
          'vendor-editor': ['@tiptap/react', '@tiptap/starter-kit', 'tiptap-markdown', '@tiptap/extension-task-list', '@tiptap/extension-task-item'],
          'vendor-charts': ['recharts'],
        },
      },
    },
  },
})
